// Copyright 2025 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package modelmigration

import (
	"context"
	"time"

	"github.com/juju/description/v9"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	"go.uber.org/mock/gomock"
	gc "gopkg.in/check.v1"

	coreresouces "github.com/juju/juju/core/resource"
	coreunit "github.com/juju/juju/core/unit"
	domainresource "github.com/juju/juju/domain/resource"
	"github.com/juju/juju/internal/charm/resource"
)

var fingerprint = []byte("123456789012345678901234567890123456789012345678")

type exportSuite struct {
	testing.IsolationSuite

	exportService *MockExportService
}

var _ = gc.Suite(&exportSuite{})

func (s *exportSuite) setupMocks(c *gc.C) *gomock.Controller {
	ctrl := gomock.NewController(c)

	s.exportService = NewMockExportService(ctrl)

	return ctrl
}

func (s *exportSuite) TestResourceExportEmpty(c *gc.C) {
	model := description.NewModel(description.ModelArgs{})

	exportOp := exportOperation{
		service: s.exportService,
	}

	err := exportOp.Execute(context.Background(), model)
	c.Assert(err, jc.ErrorIsNil)
}

func (s *exportSuite) TestResourceExport(c *gc.C) {
	defer s.setupMocks(c).Finish()
	// Arrange: add an app and unit to the model.
	model := description.NewModel(description.ModelArgs{})
	appName := "app-name"
	app := model.AddApplication(description.ApplicationArgs{
		Name: appName,
	})
	unitName := "app-name/0"
	app.AddUnit(description.UnitArgs{
		Name: unitName,
	})

	fp, err := resource.NewFingerprint(fingerprint)
	c.Assert(err, jc.ErrorIsNil)

	// Arrange: create resource data.
	res1Name := "resource-1"
	res1Revision := 1
	res1Time := time.Now().Truncate(time.Second).UTC()
	res1Origin := resource.OriginStore
	res1Size := int64(21)
	res1RetrievedBy := "retrieved by 1"
	res2Name := "resource-2"
	res2Revision := -1
	res2Origin := resource.OriginUpload
	res2Time := time.Now().Truncate(time.Second).Add(-time.Hour).UTC()
	res2Size := int64(12)
	res2RetrievedBy := "retrieved by 2"
	unitResName := "resource-3"
	unitResRevision := -1
	unitResOrigin := resource.OriginUpload
	unitResTime := time.Now().Truncate(time.Second).Add(-time.Hour).UTC()
	unitResSize := int64(32)
	unitResRetrievedBy := "retrieved by 3"

	// Arrange: expect ExportResources for the app.
	s.exportService.EXPECT().ExportResources(gomock.Any(), appName).Return(domainresource.ExportedResources{
		Resources: []coreresouces.Resource{{
			Resource: resource.Resource{
				Meta: resource.Meta{
					Name: res1Name,
				},
				Origin:      res1Origin,
				Revision:    res1Revision,
				Fingerprint: fp,
				Size:        res1Size,
			},
			Timestamp:   res1Time,
			RetrievedBy: res1RetrievedBy,
		}, {
			Resource: resource.Resource{
				Meta: resource.Meta{
					Name: res2Name,
				},
				Origin:      res2Origin,
				Revision:    res2Revision,
				Fingerprint: fp,
				Size:        res2Size,
			},
			Timestamp:   res2Time,
			RetrievedBy: res2RetrievedBy,
		}},
		UnitResources: []coreresouces.UnitResources{{
			Name: coreunit.Name(unitName),
			Resources: []coreresouces.Resource{{
				Resource: resource.Resource{
					Meta: resource.Meta{
						Name: unitResName,
					},
					Origin:      unitResOrigin,
					Revision:    unitResRevision,
					Fingerprint: fp,
					Size:        unitResSize,
				},
				Timestamp:   unitResTime,
				RetrievedBy: unitResRetrievedBy,
			}},
		}}},
		nil,
	)

	// Act: export the resources
	exportOp := exportOperation{
		service: s.exportService,
	}
	err = exportOp.Execute(context.Background(), model)

	// Assert: check no errors occurred.
	c.Assert(err, jc.ErrorIsNil)

	// Assert the app has resources.
	apps := model.Applications()
	c.Assert(apps, gc.HasLen, 1)
	resources := apps[0].Resources()
	c.Assert(resources, gc.HasLen, 2)
	c.Check(resources[0].Name(), gc.Equals, res1Name)

	// Assert resource 1 was exported correctly.
	res1AppRevision := resources[0].ApplicationRevision()
	c.Check(res1AppRevision.Revision(), gc.Equals, res1Revision)
	c.Check(res1AppRevision.Origin(), gc.Equals, res1Origin.String())
	c.Check(res1AppRevision.RetrievedBy(), gc.Equals, res1RetrievedBy)
	c.Check(res1AppRevision.SHA384(), gc.Equals, fp.String())
	c.Check(res1AppRevision.Size(), gc.Equals, res1Size)
	c.Check(res1AppRevision.Timestamp(), gc.Equals, res1Time)

	// Assert resource 2 was exported correctly.
	res2AppRevision := resources[1].ApplicationRevision()
	c.Check(res2AppRevision.Revision(), gc.Equals, res2Revision)
	c.Check(res2AppRevision.Origin(), gc.Equals, res2Origin.String())
	c.Check(res2AppRevision.RetrievedBy(), gc.Equals, res2RetrievedBy)
	c.Check(res2AppRevision.SHA384(), gc.Equals, fp.String())
	c.Check(res2AppRevision.Size(), gc.Equals, res2Size)
	c.Check(res2AppRevision.Timestamp(), gc.Equals, res2Time)

	// Assert the unit resource was exported correctly.
	units := app.Units()
	c.Assert(units, gc.HasLen, 1)
	unitResources := units[0].Resources()
	c.Assert(unitResources, gc.HasLen, 1)
	c.Check(unitResources[0].Name(), gc.Equals, unitResName)
	unitResourceRevision := unitResources[0].Revision()
	c.Check(unitResourceRevision.Revision(), gc.Equals, unitResRevision)
	c.Check(unitResourceRevision.Origin(), gc.Equals, unitResOrigin.String())
	c.Check(unitResourceRevision.RetrievedBy(), gc.Equals, unitResRetrievedBy)
	c.Check(unitResourceRevision.SHA384(), gc.Equals, fp.String())
	c.Check(unitResourceRevision.Size(), gc.Equals, unitResSize)
	c.Check(unitResourceRevision.Timestamp(), gc.Equals, unitResTime)
}
