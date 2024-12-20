// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/juju/names/v5"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils/v4"
	gc "gopkg.in/check.v1"

	apitesting "github.com/juju/juju/apiserver/testing"
	"github.com/juju/juju/core/permission"
	"github.com/juju/juju/core/user"
	"github.com/juju/juju/domain/access/service"
	"github.com/juju/juju/internal/auth"
	"github.com/juju/juju/internal/charm"
	"github.com/juju/juju/internal/testing/factory"
	jujutesting "github.com/juju/juju/juju/testing"
	"github.com/juju/juju/rpc/params"
	"github.com/juju/juju/state"
	"github.com/juju/juju/testcharms"
)

type baseObjectsSuite struct {
	jujutesting.ApiServerSuite

	method      string
	contentType string
}

func (s *baseObjectsSuite) assertResponse(c *gc.C, resp *http.Response, expStatus int) params.CharmsResponse {
	body := apitesting.AssertResponse(c, resp, expStatus, params.ContentTypeJSON)
	var charmResponse params.CharmsResponse
	err := json.Unmarshal(body, &charmResponse)
	c.Assert(err, jc.ErrorIsNil, gc.Commentf("body: %s", body))
	return charmResponse
}

func (s *baseObjectsSuite) assertErrorResponse(c *gc.C, resp *http.Response, expCode int, expError string) {
	charmResponse := s.assertResponse(c, resp, expCode)
	c.Check(charmResponse.Error, gc.Matches, expError)
}

func (s *baseObjectsSuite) uploadRequest(c *gc.C, url, contentType, curl string, content io.Reader) *http.Response {
	return sendHTTPRequest(c, apitesting.HTTPRequestParams{
		Method:      "PUT",
		URL:         url,
		ContentType: contentType,
		Body:        content,
		ExtraHeaders: map[string]string{
			"Juju-Curl": curl,
		},
	})
}

func (s *baseObjectsSuite) migrateObjectsCharmsURL(charmRef string) *url.URL {
	return s.URL(fmt.Sprintf("/migrate/charms/%s", charmRef), nil)
}

func (s *baseObjectsSuite) migrateObjectsCharmsURI(charmRef string) string {
	return s.migrateObjectsCharmsURL(charmRef).String()
}

func (s *baseObjectsSuite) objectsCharmsURL(charmRef string) *url.URL {
	return s.URL(fmt.Sprintf("/model-%s/charms/%s", s.ControllerModelUUID(), charmRef), nil)
}

func (s *baseObjectsSuite) objectsCharmsURI(charmRef string) string {
	return s.objectsCharmsURL(charmRef).String()
}

func (s *baseObjectsSuite) setModelImporting(c *gc.C) {
	model, err := s.ControllerModel(c).State().Model()
	c.Assert(err, jc.ErrorIsNil)
	err = model.SetMigrationMode(state.MigrationModeImporting)
	c.Assert(err, jc.ErrorIsNil)
}

func (s *baseObjectsSuite) TestObjectsCharmsServedSecurely(c *gc.C) {
	url := s.objectsCharmsURL("")
	url.Scheme = "http"
	apitesting.SendHTTPRequest(c, apitesting.HTTPRequestParams{
		Method:       "GET",
		URL:          url.String(),
		ExpectStatus: http.StatusBadRequest,
	})
}

func (s *baseObjectsSuite) TestRequiresAuth(c *gc.C) {
	resp := apitesting.SendHTTPRequest(c, apitesting.HTTPRequestParams{Method: s.method, URL: s.objectsCharmsURI("somecharm-abcd0123")})
	body := apitesting.AssertResponse(c, resp, http.StatusUnauthorized, "text/plain; charset=utf-8")
	c.Assert(string(body), gc.Equals, "authentication failed: no credentials provided\n")
}

func (s *baseObjectsSuite) TestGetFailsWithInvalidObjectSha256(c *gc.C) {
	uri := s.objectsCharmsURI("invalidsha256")
	resp := sendHTTPRequest(c, apitesting.HTTPRequestParams{Method: s.method, ContentType: s.contentType, URL: uri})
	s.assertErrorResponse(
		c, resp, http.StatusBadRequest,
		`.*"invalidsha256" is not a valid charm object path$`,
	)
}

func (s *baseObjectsSuite) TestInvalidBucket(c *gc.C) {
	wrongURL := s.URL("modelwrongbucket/charms/somecharm-abcd0123", nil)
	resp := sendHTTPRequest(c, apitesting.HTTPRequestParams{Method: s.method, URL: wrongURL.String()})
	body := apitesting.AssertResponse(c, resp, http.StatusNotFound, "text/plain; charset=utf-8")
	c.Assert(string(body), gc.Equals, "404 page not found\n")
}

func (s *baseObjectsSuite) TestInvalidModel(c *gc.C) {
	wrongURL := s.URL("model-wrongbucket/charms/somecharm-abcd0123", nil)
	resp := sendHTTPRequest(c, apitesting.HTTPRequestParams{Method: s.method, URL: wrongURL.String()})
	body := apitesting.AssertResponse(c, resp, http.StatusBadRequest, "text/plain; charset=utf-8")
	c.Assert(string(body), gc.Equals, "invalid model UUID \"wrongbucket\"\n")
}

func (s *baseObjectsSuite) TestInvalidObject(c *gc.C) {
	resp := sendHTTPRequest(c, apitesting.HTTPRequestParams{Method: s.method, ContentType: s.contentType, URL: s.objectsCharmsURI("invalidcharm")})
	body := apitesting.AssertResponse(c, resp, http.StatusBadRequest, "application/json")
	c.Assert(string(body), gc.Matches, `{"error":".*\\"invalidcharm\\" is not a valid charm object path","error-code":"bad request"}$`)
}

type getCharmObjectSuite struct {
	baseObjectsSuite
}

var _ = gc.Suite(&getCharmObjectSuite{})

func (s *getCharmObjectSuite) SetUpTest(c *gc.C) {
	s.baseObjectsSuite.SetUpTest(c)
	s.method = "GET"
}

type putCharmObjectSuite struct {
	baseObjectsSuite
}

var _ = gc.Suite(&putCharmObjectSuite{})

func (s *putCharmObjectSuite) SetUpSuite(c *gc.C) {
	s.baseObjectsSuite.SetUpSuite(c)
	s.baseObjectsSuite.method = "PUT"
	s.baseObjectsSuite.contentType = "application/zip"
}

func (s *putCharmObjectSuite) assertUploadResponse(c *gc.C, resp *http.Response, expCharmURL string) {
	charmResponse := s.assertResponse(c, resp, http.StatusOK)
	c.Check(charmResponse.Error, gc.Equals, "")
	c.Check(charmResponse.CharmURL, gc.Equals, expCharmURL)
}

func (s *putCharmObjectSuite) TestPUTRequiresUserAuth(c *gc.C) {
	f, release := s.NewFactory(c, s.ControllerModelUUID())
	defer release()
	machine, password := f.MakeMachineReturningPassword(c, &factory.MachineParams{
		Nonce: "noncy",
	})
	resp := apitesting.SendHTTPRequest(c, apitesting.HTTPRequestParams{
		Tag:         machine.Tag().String(),
		Password:    password,
		Method:      s.method,
		URL:         s.objectsCharmsURI("somecharm-abcd0123"),
		Nonce:       "noncy",
		ContentType: "foo/bar",
	})
	body := apitesting.AssertResponse(c, resp, http.StatusForbidden, "text/plain; charset=utf-8")
	c.Assert(string(body), gc.Equals, "authorization failed: tag kind machine not valid\n")

	resp = sendHTTPRequest(c, apitesting.HTTPRequestParams{Method: s.method, URL: s.objectsCharmsURI("somecharm-abcdef0")})
	s.assertErrorResponse(c, resp, http.StatusBadRequest, ".*expected Content-Type: application/zip.+")
}

func (s *putCharmObjectSuite) TestUploadFailsWithInvalidZip(c *gc.C) {
	empty := strings.NewReader("")

	// Pretend we upload a zip by setting the Content-Type, so we can
	// check the error at extraction time later.
	resp := s.uploadRequest(c, s.objectsCharmsURI("somecharm-"+getCharmHash(c, empty)), "application/zip", "local:somecharm", empty)
	s.assertErrorResponse(c, resp, http.StatusBadRequest, ".*zip: not a valid zip file$")

	// Now try with the default Content-Type.
	resp = s.uploadRequest(c, s.objectsCharmsURI("somecharm-"+getCharmHash(c, empty)), "application/octet-stream", "local:somecharm", empty)
	s.assertErrorResponse(c, resp, http.StatusBadRequest, ".*expected Content-Type: application/zip, got: application/octet-stream$")
}

func (s *putCharmObjectSuite) TestCannotUploadCharmhubCharm(c *gc.C) {
	// We should run verifications like this before processing the charm.
	empty := strings.NewReader("")
	resp := s.uploadRequest(c, s.objectsCharmsURI("somecharm-"+getCharmHash(c, empty)), "application/zip", "ch:somecharm", empty)
	s.assertErrorResponse(c, resp, http.StatusBadRequest, `.*non-local charms may only be uploaded during model migration import`)
}

func (s *putCharmObjectSuite) TestUploadBumpsRevision(c *gc.C) {
	// Add the dummy charm with revision 1.
	ch := testcharms.Repo.CharmArchive(c.MkDir(), "dummy")
	curl := fmt.Sprintf("local:%s-%d", "testcharm", ch.Revision())
	info := state.CharmInfo{
		Charm:       ch,
		ID:          curl,
		StoragePath: "testcharm-storage-path",
		SHA256:      "testcharm-1-sha256",
	}
	_, err := s.ControllerModel(c).State().AddCharm(info)
	c.Assert(err, jc.ErrorIsNil)

	// Now try uploading the same revision and verify it gets bumped,
	// and the BundleSha256 is calculated.
	f, err := os.Open(ch.Path)
	c.Assert(err, jc.ErrorIsNil)
	defer func() { _ = f.Close() }()
	resp := s.uploadRequest(c, s.objectsCharmsURI("testcharm-"+getCharmHash(c, f)), "application/zip", "local:testcharm", f)
	expectedURL := "local:testcharm-2"
	s.assertUploadResponse(c, resp, expectedURL)
	sch, err := s.ControllerModel(c).State().Charm(expectedURL)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(sch.URL(), gc.Equals, expectedURL)
	c.Assert(sch.Revision(), gc.Equals, 2)
	c.Assert(sch.IsUploaded(), jc.IsTrue)
	// No more checks for the hash here, because it is
	// verified in TestUploadRespectsLocalRevision.
	c.Assert(sch.BundleSha256(), gc.Not(gc.Equals), "")
}

func (s *putCharmObjectSuite) TestUploadVersion(c *gc.C) {
	expectedVersion := "dummy-146-g725cfd3-dirty"

	// Add the dummy charm with version "juju-2.4-beta3-146-g725cfd3-dirty".
	pathToArchive := testcharms.Repo.CharmArchivePath(c.MkDir(), "dummy")
	err := testcharms.InjectFilesToCharmArchive(pathToArchive, map[string]string{
		"version": expectedVersion,
	})
	c.Assert(err, gc.IsNil)
	ch, err := charm.ReadCharmArchive(pathToArchive)
	c.Assert(err, gc.IsNil)

	f, err := os.Open(ch.Path)
	c.Assert(err, jc.ErrorIsNil)
	defer func() { _ = f.Close() }()
	resp := s.uploadRequest(c, s.objectsCharmsURI("testcharm-"+getCharmHash(c, f)), "application/zip", "local:testcharm", f)

	expectedURL := "local:testcharm-1"
	s.assertUploadResponse(c, resp, expectedURL)
	sch, err := s.ControllerModel(c).State().Charm(expectedURL)
	c.Assert(err, jc.ErrorIsNil)

	version := sch.Version()
	c.Assert(version, gc.Equals, expectedVersion)
}

func (s *putCharmObjectSuite) TestUploadRespectsLocalRevision(c *gc.C) {
	// Make a dummy charm dir with revision 123.
	base := testcharms.Repo.ClonedDirPath(c.MkDir(), "dummy")

	// Set the disk revision to 42.
	revFile, err := os.OpenFile(filepath.Join(base, "revision"), os.O_CREATE|os.O_WRONLY, 0666)
	c.Assert(err, jc.ErrorIsNil)
	_, err = revFile.WriteString("123")
	c.Check(err, jc.ErrorIsNil)
	err = revFile.Close()
	c.Assert(err, jc.ErrorIsNil)

	dir, err := charm.ReadCharmDir(base)
	c.Assert(err, jc.ErrorIsNil)

	// Now archive the dir.
	tempFile, err := os.CreateTemp("", "charm")
	c.Assert(err, jc.ErrorIsNil)
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())
	err = dir.ArchiveTo(tempFile)
	c.Assert(err, jc.ErrorIsNil)

	expectedSHA256 := getCharmHash(c, tempFile)

	// Now try uploading it and ensure the revision persists.
	resp := s.uploadRequest(c, s.objectsCharmsURI("testcharm-"+expectedSHA256), "application/zip", "local:testcharm", tempFile)
	expectedURL := "local:testcharm-123"
	s.assertUploadResponse(c, resp, expectedURL)
	sch, err := s.ControllerModel(c).State().Charm(expectedURL)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(sch.URL(), gc.Equals, expectedURL)
	c.Assert(sch.Revision(), gc.Equals, 123)
	c.Assert(sch.IsUploaded(), jc.IsTrue)
	c.Assert(sch.BundleSha256()[0:7], gc.Equals, expectedSHA256)

	store := s.ObjectStore(c, s.ControllerModelUUID())
	reader, _, err := store.Get(context.Background(), sch.StoragePath())
	c.Assert(err, jc.ErrorIsNil)
	defer reader.Close()
	downloadedSHA256, _, err := utils.ReadSHA256(reader)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(downloadedSHA256[0:7], gc.Equals, expectedSHA256)
}

func (s *putCharmObjectSuite) TestNonLocalCharmUploadFailsIfNotMigrating(c *gc.C) {
	ch := testcharms.Repo.CharmArchive(c.MkDir(), "dummy")
	f, err := os.Open(ch.Path)
	c.Assert(err, jc.ErrorIsNil)
	defer func() { _ = f.Close() }()
	hash := getCharmHash(c, f)

	curl := fmt.Sprintf("ch:%s-%d", ch.Meta().Name, ch.Revision())
	info := state.CharmInfo{
		Charm:       ch,
		ID:          curl,
		StoragePath: "testcharm-storage-path",
		SHA256:      hash,
	}
	_, err = s.ControllerModel(c).State().AddCharm(info)
	c.Assert(err, jc.ErrorIsNil)

	resp := s.uploadRequest(c, s.objectsCharmsURI("testcharm-"+hash), "application/zip", curl, f)
	s.assertErrorResponse(c, resp, 400, ".*charms may only be uploaded during model migration import$")
}

func (s *putCharmObjectSuite) TestNonLocalCharmUpload(c *gc.C) {
	// Check that upload of charms with the "ch:" schema works (for
	// model migrations).
	s.setModelImporting(c)
	ch := testcharms.Repo.CharmArchive(c.MkDir(), "dummy")
	f, err := os.Open(ch.Path)
	c.Assert(err, jc.ErrorIsNil)
	defer func() { _ = f.Close() }()
	hash := getCharmHash(c, f)

	curl := fmt.Sprintf("ch:%s-%d", ch.Meta().Name, ch.Revision())
	info := state.CharmInfo{
		Charm:       ch,
		ID:          curl,
		StoragePath: "testcharm-storage-path",
		SHA256:      hash,
	}
	_, err = s.ControllerModel(c).State().AddCharm(info)
	c.Assert(err, jc.ErrorIsNil)

	resp := s.uploadRequest(c, s.objectsCharmsURI("testcharm-"+hash), "application/zip", "ch:testcharm-1", f)

	expectedURL := "ch:testcharm-1"
	s.assertUploadResponse(c, resp, expectedURL)
	sch, err := s.ControllerModel(c).State().Charm(expectedURL)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(sch.URL(), gc.DeepEquals, expectedURL)
	c.Assert(sch.Revision(), gc.Equals, 1)
	c.Assert(sch.IsUploaded(), jc.IsTrue)
}

func (s *putCharmObjectSuite) TestUnsupportedSchema(c *gc.C) {
	s.setModelImporting(c)
	ch := testcharms.Repo.CharmArchive(c.MkDir(), "dummy")
	f, err := os.Open(ch.Path)
	c.Assert(err, jc.ErrorIsNil)
	defer func() { _ = f.Close() }()

	resp := s.uploadRequest(c, s.objectsCharmsURI("testcharm-"+getCharmHash(c, f)), "application/zip", "zz:testcharm", f)
	s.assertErrorResponse(
		c, resp, http.StatusBadRequest,
		`cannot upload charm: "zz:testcharm" is not a valid charm url`,
	)
}

func (s *putCharmObjectSuite) TestNonLocalCharmUploadWithRevisionOverride(c *gc.C) {
	s.setModelImporting(c)
	ch := testcharms.Repo.CharmArchive(c.MkDir(), "dummy")
	f, err := os.Open(ch.Path)
	c.Assert(err, jc.ErrorIsNil)
	defer func() { _ = f.Close() }()

	resp := s.uploadRequest(c, s.objectsCharmsURI("testcharm-"+getCharmHash(c, f)), "application/zip", "ch:testcharm-99", f)

	expectedURL := "ch:testcharm-99"
	s.assertUploadResponse(c, resp, expectedURL)
	sch, err := s.ControllerModel(c).State().Charm(expectedURL)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(sch.URL(), gc.DeepEquals, expectedURL)
	c.Assert(sch.Revision(), gc.Equals, 99)
	c.Assert(sch.IsUploaded(), jc.IsTrue)
}

func (s *putCharmObjectSuite) TestMigrateCharm(c *gc.C) {
	s.setModelImporting(c)

	ch := testcharms.Repo.CharmArchive(c.MkDir(), "dummy")
	f, err := os.Open(ch.Path)
	c.Assert(err, jc.ErrorIsNil)
	defer func() { _ = f.Close() }()

	// The default user is just a normal user, not a controller admin
	url := s.migrateObjectsCharmsURI("testcharm-" + getCharmHash(c, f))
	resp := sendHTTPRequest(c, apitesting.HTTPRequestParams{
		Method:      "PUT",
		URL:         url,
		ContentType: "application/zip",
		Body:        f,
		ExtraHeaders: map[string]string{
			params.MigrationModelHTTPHeader: s.ControllerModelUUID(),
			"Juju-Curl":                     "ch:testcharm-10",
		},
	})
	expectedURL := "ch:testcharm-10"
	s.assertUploadResponse(c, resp, expectedURL)

	// The charm was added to the migrated model.
	_, err = s.ControllerModel(c).State().Charm(expectedURL)
	c.Assert(err, jc.ErrorIsNil)
}

func (s *putCharmObjectSuite) TestMigrateCharmName(c *gc.C) {
	s.setModelImporting(c)

	ch := testcharms.Repo.CharmArchive(c.MkDir(), "dummy")
	f, err := os.Open(ch.Path)
	c.Assert(err, jc.ErrorIsNil)
	defer func() { _ = f.Close() }()

	// The default user is just a normal user, not a controller admin
	url := s.migrateObjectsCharmsURI("meshuggah-" + getCharmHash(c, f))
	resp := sendHTTPRequest(c, apitesting.HTTPRequestParams{
		Method:      "PUT",
		URL:         url,
		ContentType: "application/zip",
		Body:        f,
		ExtraHeaders: map[string]string{
			params.MigrationModelHTTPHeader: s.ControllerModelUUID(),
			"Juju-Curl":                     "ch:meshuggah-1",
		},
	})
	expectedURL := "ch:meshuggah-1"
	s.assertUploadResponse(c, resp, expectedURL)

	// The charm was added to the migrated model.
	_, err = s.ControllerModel(c).State().Charm(expectedURL)
	c.Assert(err, jc.ErrorIsNil)
}

func (s *putCharmObjectSuite) TestMigrateCharmNotMigrating(c *gc.C) {
	ch := testcharms.Repo.CharmArchive(c.MkDir(), "dummy")
	f, err := os.Open(ch.Path)
	c.Assert(err, jc.ErrorIsNil)
	defer func() { _ = f.Close() }()

	// The default user is just a normal user, not a controller admin
	url := s.migrateObjectsCharmsURI("testcharm-" + getCharmHash(c, f))
	resp := sendHTTPRequest(c, apitesting.HTTPRequestParams{
		Method:      "PUT",
		URL:         url,
		ContentType: "application/zip",
		Body:        f,
		ExtraHeaders: map[string]string{
			params.MigrationModelHTTPHeader: s.ControllerModelUUID(),
			"Juju-Curl":                     "ch:testcharm-1",
		},
	})

	s.assertErrorResponse(
		c, resp, http.StatusBadRequest,
		`cannot upload charm: model migration mode is "" instead of "importing"`,
	)
}

func (s *putCharmObjectSuite) TestMigrateCharmUnauthorized(c *gc.C) {
	s.setModelImporting(c)

	userService := s.ControllerDomainServices(c).Access()
	userTag := names.NewUserTag("bobbrown")
	_, _, err := userService.AddUser(context.Background(), service.AddUserArg{
		Name:        user.NameFromTag(userTag),
		DisplayName: "Bob Brown",
		CreatorUUID: s.AdminUserUUID,
		Password:    ptr(auth.NewPassword("hunter2")),
		Permission: permission.AccessSpec{
			Access: permission.LoginAccess,
			Target: permission.ID{
				ObjectType: permission.Controller,
				Key:        s.ControllerUUID,
			},
		},
	})
	c.Assert(err, jc.ErrorIsNil)

	ch := testcharms.Repo.CharmArchive(c.MkDir(), "dummy")
	f, err := os.Open(ch.Path)
	c.Assert(err, jc.ErrorIsNil)
	defer func() { _ = f.Close() }()

	// The default user is just a normal user, not a controller admin
	url := s.migrateObjectsCharmsURI("testcharm-" + getCharmHash(c, f))
	resp := apitesting.SendHTTPRequest(c, apitesting.HTTPRequestParams{
		Method:   "PUT",
		URL:      url,
		Tag:      userTag.String(),
		Password: "hunter2",
		Body:     f,
		ExtraHeaders: map[string]string{
			params.MigrationModelHTTPHeader: s.ControllerModelUUID(),
			"Juju-Curl":                     "ch:testcharm-1",
		},
	})
	body := apitesting.AssertResponse(c, resp, http.StatusForbidden, "text/plain; charset=utf-8")
	c.Assert(string(body), gc.Matches, "authorization failed: user .* not a controller admin\n")
}

func getCharmHash(c *gc.C, stream io.ReadSeeker) string {
	_, err := stream.Seek(0, io.SeekStart)
	c.Assert(err, jc.ErrorIsNil)
	hash, _, err := utils.ReadSHA256(stream)
	c.Assert(err, jc.ErrorIsNil)
	_, err = stream.Seek(0, io.SeekStart)
	c.Assert(err, jc.ErrorIsNil)
	return hash[0:7]
}
