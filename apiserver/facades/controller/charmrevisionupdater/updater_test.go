// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charmrevisionupdater_test

import (
	"context"
	"net/http"
	"time"

	"github.com/juju/clock"
	"github.com/juju/clock/testclock"
	jc "github.com/juju/testing/checkers"
	"go.uber.org/mock/gomock"
	gc "gopkg.in/check.v1"

	commonmocks "github.com/juju/juju/apiserver/common/mocks"
	"github.com/juju/juju/apiserver/facades/controller/charmrevisionupdater"
	"github.com/juju/juju/apiserver/facades/controller/charmrevisionupdater/mocks"
	apiservertesting "github.com/juju/juju/apiserver/testing"
	"github.com/juju/juju/cloud"
	charmmetrics "github.com/juju/juju/core/charm/metrics"
	jujuversion "github.com/juju/juju/core/version"
	servicefactorytesting "github.com/juju/juju/domain/services/testing"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/internal/charm/resource"
	"github.com/juju/juju/internal/charmhub/transport"
	loggertesting "github.com/juju/juju/internal/logger/testing"
	"github.com/juju/juju/internal/testing"
	"github.com/juju/juju/state"
)

type updaterSuite struct {
	servicefactorytesting.DomainServicesSuite
	model              *mocks.MockModel
	state              *mocks.MockState
	objectStore        *mocks.MockObjectStore
	cloudService       *commonmocks.MockCloudService
	modelConfigService *mocks.MockModelConfigService
	resources          *mocks.MockResources
	client             *mocks.MockCharmhubRefreshClient

	clock clock.Clock
}

var _ = gc.Suite(&updaterSuite{})

func (s *updaterSuite) SetUpTest(c *gc.C) {
	s.DomainServicesSuite.SetUpTest(c)
	s.clock = testclock.NewClock(time.Now())
}

func (s *updaterSuite) TestNewAuthSuccess(c *gc.C) {
	authoriser := apiservertesting.FakeAuthorizer{Controller: true}
	facadeCtx := facadeContextShim{
		state:          nil,
		authorizer:     authoriser,
		logger:         loggertesting.WrapCheckLog(c),
		httpClient:     &http.Client{},
		domainServices: s.DefaultModelDomainServices(c),
	}
	updater, err := charmrevisionupdater.NewCharmRevisionUpdaterAPI(facadeCtx)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(updater, gc.NotNil)
}

func (s *updaterSuite) TestNewAuthFailure(c *gc.C) {
	authoriser := apiservertesting.FakeAuthorizer{Controller: false}
	facadeCtx := facadeContextShim{state: nil, authorizer: authoriser, logger: loggertesting.WrapCheckLog(c)}
	updater, err := charmrevisionupdater.NewCharmRevisionUpdaterAPI(facadeCtx)
	c.Assert(updater, gc.IsNil)
	c.Assert(err, gc.ErrorMatches, "permission denied")
}

func (s *updaterSuite) TestCharmhubUpdate(c *gc.C) {
	ctrl := s.setupMocks(c)
	defer ctrl.Finish()
	s.expectCharmHubModel(c)

	matcher := charmhubConfigMatcher{expected: []charmhubConfigExpected{
		{id: "charm-1", revision: 22},
		{id: "charm-2", revision: 41},
	}}
	s.client.EXPECT().RefreshWithRequestMetrics(gomock.Any(), matcher, gomock.Any()).Return([]transport.RefreshResponse{
		{
			Entity: transport.RefreshEntity{Revision: 23},
			ID:     "charm-1",
			Name:   "mysql",
		},
		{
			Entity: transport.RefreshEntity{Revision: 42},
			ID:     "charm-2",
			Name:   "postgresql",
		},
	}, nil)

	s.state.EXPECT().AllApplications().Return([]charmrevisionupdater.Application{
		makeApplication(ctrl, "ch", "mysql", "charm-1", "app-1", 22),
		makeApplication(ctrl, "ch", "postgresql", "charm-2", "app-2", 41),
	}, nil).AnyTimes()

	result, err := s.api(c).UpdateLatestRevisions(context.Background())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(result.Error, gc.IsNil)
}

func (s *updaterSuite) TestCharmhubUpdateWithMetrics(c *gc.C) {
	ctrl := s.setupMocks(c)
	defer ctrl.Finish()
	uuid := testing.ModelTag.Id()
	cfg, err := config.New(config.UseDefaults, map[string]interface{}{
		"name": "model",
		"type": "type",
		"uuid": uuid,
	})
	c.Assert(err, jc.ErrorIsNil)
	s.modelConfigService.EXPECT().ModelConfig(gomock.Any()).Return(cfg, nil).AnyTimes()
	s.model.EXPECT().ModelTag().Return(testing.ModelTag).AnyTimes()
	s.model.EXPECT().Metrics().Return(state.ModelMetrics{
		UUID:           uuid,
		ControllerUUID: "controller-1",
		CloudName:      "cloud",
	}, nil).AnyTimes()
	s.state.EXPECT().AliveRelationKeys().Return([]string{
		"app-1:end app-2:point",
	})
	matcher := charmhubConfigMatcher{expected: []charmhubConfigExpected{
		{id: "charm-1", revision: 22, relMetric: "postgresql"},
		{id: "charm-2", revision: 41, relMetric: "mysql"},
	}}
	s.testCharmhubUpdateMetrics(c, ctrl, matcher, true)
}

func (s *updaterSuite) TestCharmhubUpdateWithNoMetrics(c *gc.C) {
	ctrl := s.setupMocks(c)
	defer ctrl.Finish()
	cfg, err := config.New(config.UseDefaults, map[string]interface{}{
		"name":                     "model",
		"type":                     "type",
		"uuid":                     testing.ModelTag.Id(),
		config.DisableTelemetryKey: true,
	})
	c.Assert(err, jc.ErrorIsNil)
	s.modelConfigService.EXPECT().ModelConfig(gomock.Any()).Return(cfg, nil).AnyTimes()
	s.model.EXPECT().ModelTag().Return(testing.ModelTag).AnyTimes()
	matcher := charmhubConfigMatcher{expected: []charmhubConfigExpected{
		{id: "charm-1", revision: 22},
		{id: "charm-2", revision: 41},
	}}
	s.testCharmhubUpdateMetrics(c, ctrl, matcher, false)
}

func (s *updaterSuite) testCharmhubUpdateMetrics(c *gc.C, ctrl *gomock.Controller, matcher gomock.Matcher, exist bool) {
	s.client.EXPECT().RefreshWithRequestMetrics(gomock.Any(), matcher, charmhubMetricsMatcher{c: c, exist: exist}).Return([]transport.RefreshResponse{
		{
			Entity: transport.RefreshEntity{Revision: 23},
			ID:     "charm-1",
			Name:   "mysql",
		},
		{
			Entity: transport.RefreshEntity{Revision: 42},
			ID:     "charm-2",
			Name:   "postgresql",
		},
	}, nil)

	s.state.EXPECT().AllApplications().Return([]charmrevisionupdater.Application{
		makeApplication(ctrl, "ch", "mysql", "charm-1", "app-1", 22),
		makeApplication(ctrl, "ch", "postgresql", "charm-2", "app-2", 41),
	}, nil).AnyTimes()

	result, err := s.api(c).UpdateLatestRevisions(context.Background())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(result.Error, gc.IsNil)
}

func (s *updaterSuite) TestEmptyModelMetrics(c *gc.C) {
	ctrl := s.setupMocksNoResources(c)
	defer ctrl.Finish()
	uuid := testing.ModelTag.Id()
	s.model.EXPECT().Metrics().Return(state.ModelMetrics{
		UUID:           uuid,
		ControllerUUID: "controller-1",
		CloudName:      "cloud",
	}, nil)
	cfg, err := config.New(config.UseDefaults, map[string]interface{}{
		"name": "model",
		"type": "type",
		"uuid": uuid,
	})
	c.Assert(err, jc.ErrorIsNil)
	s.modelConfigService.EXPECT().ModelConfig(gomock.Any()).Return(cfg, nil)
	s.state.EXPECT().AllApplications().Return([]charmrevisionupdater.Application{}, nil)

	send := map[charmmetrics.MetricKey]map[charmmetrics.MetricKey]string{
		charmmetrics.Controller: {
			charmmetrics.JujuVersion: jujuversion.Current.String(),
			charmmetrics.UUID:        "controller-1",
		},
		charmmetrics.Model: {
			charmmetrics.NumApplications: "",
			charmmetrics.Cloud:           "cloud",
			charmmetrics.NumMachines:     "",
			charmmetrics.Provider:        "",
			charmmetrics.Region:          "",
			charmmetrics.NumUnits:        "",
			charmmetrics.UUID:            uuid,
		},
	}
	s.client.EXPECT().RefreshWithMetricsOnly(gomock.Any(), gomock.Eq(send)).Return(nil)

	_, err = s.api(c).UpdateLatestRevisions(context.Background())
	c.Assert(err, jc.ErrorIsNil)
}

func (s *updaterSuite) TestEmptyModelNoMetrics(c *gc.C) {
	ctrl := s.setupMocksNoResources(c)
	defer ctrl.Finish()

	cfg, err := config.New(config.UseDefaults, map[string]interface{}{
		"name":                     "model",
		"type":                     "type",
		"uuid":                     testing.ModelTag.Id(),
		config.DisableTelemetryKey: true,
	})
	c.Assert(err, jc.ErrorIsNil)
	s.modelConfigService.EXPECT().ModelConfig(gomock.Any()).Return(cfg, nil)
	s.state.EXPECT().AllApplications().Return([]charmrevisionupdater.Application{}, nil)

	_, err = s.api(c).UpdateLatestRevisions(context.Background())
	c.Assert(err, jc.ErrorIsNil)
}

func (s *updaterSuite) TestCharmhubUpdateWithResources(c *gc.C) {
	ctrl := s.setupMocks(c)
	defer ctrl.Finish()
	s.expectCharmHubModel(c)

	expectedResources := []resource.Resource{
		makeResource(c, "reza", 7, 5, "59e1748777448c69de6b800d7a33bbfb9ff1b463e44354c3553bcdb9c666fa90125a3c79f90397bdf5f6a13de828684f"),
		makeResource(c, "rezb", 1, 6, "03130092073c5ac523ecb21f548b9ad6e1387d1cb05f3cb892fcc26029d01428afbe74025b6c567b6564a3168a47179a"),
	}
	s.resources.EXPECT().SetCharmStoreResources("app-1", expectedResources, s.clock.Now().UTC()).Return(nil)

	matcher := charmhubConfigMatcher{expected: []charmhubConfigExpected{
		{id: "charm-3", revision: 1},
	}}
	s.client.EXPECT().RefreshWithRequestMetrics(gomock.Any(), matcher, gomock.Any()).Return([]transport.RefreshResponse{
		{
			Entity: transport.RefreshEntity{
				Resources: []transport.ResourceRevision{
					{
						Download: transport.Download{
							HashSHA384: "59e1748777448c69de6b800d7a33bbfb9ff1b463e44354c3553bcdb9c666fa90125a3c79f90397bdf5f6a13de828684f",
							Size:       5,
						},
						Name:     "reza",
						Revision: 7,
						Type:     "file",
					},
					{
						Download: transport.Download{
							HashSHA384: "03130092073c5ac523ecb21f548b9ad6e1387d1cb05f3cb892fcc26029d01428afbe74025b6c567b6564a3168a47179a",
							Size:       6,
						},
						Name:     "rezb",
						Revision: 1,
						Type:     "file",
					},
				},
				Revision: 1,
			},
			ID:   "charm-3",
			Name: "resourcey",
		},
	}, nil)

	s.state.EXPECT().Resources(gomock.Any()).Return(s.resources).AnyTimes()
	s.state.EXPECT().AllApplications().Return([]charmrevisionupdater.Application{
		makeApplication(ctrl, "ch", "resourcey", "charm-3", "app-1", 1),
	}, nil).AnyTimes()

	result, err := s.api(c).UpdateLatestRevisions(context.Background())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(result.Error, gc.IsNil)
}

func (s *updaterSuite) TestCharmhubNoUpdate(c *gc.C) {
	ctrl := s.setupMocks(c)
	defer ctrl.Finish()
	s.expectCharmHubModel(c)

	matcher := charmhubConfigMatcher{expected: []charmhubConfigExpected{
		{id: "charm-2", revision: 42},
	}}
	s.client.EXPECT().RefreshWithRequestMetrics(gomock.Any(), matcher, gomock.Any()).Return([]transport.RefreshResponse{
		{
			Entity: transport.RefreshEntity{Revision: 42},
			ID:     "charm-2",
			Name:   "postgresql",
		},
	}, nil)

	s.state.EXPECT().AllApplications().Return([]charmrevisionupdater.Application{
		makeApplication(ctrl, "ch", "postgresql", "charm-2", "app-2", 42),
	}, nil).AnyTimes()

	result, err := s.api(c).UpdateLatestRevisions(context.Background())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(result.Error, gc.IsNil)
}

func (s *updaterSuite) TestCharmNotInStore(c *gc.C) {
	ctrl := s.setupMocks(c)
	defer ctrl.Finish()
	s.expectCharmHubModel(c)

	s.client.EXPECT().RefreshWithRequestMetrics(gomock.Any(), gomock.Any(), gomock.Any()).Return([]transport.RefreshResponse{}, nil)

	s.state.EXPECT().AllApplications().Return([]charmrevisionupdater.Application{
		makeApplication(ctrl, "ch", "varnish", "charm-5", "app-1", 1),
	}, nil).AnyTimes()

	result, err := s.api(c).UpdateLatestRevisions(context.Background())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(result.Error, gc.IsNil)
}

func (s *updaterSuite) setupMocks(c *gc.C) *gomock.Controller {
	ctrl := s.setupMocksNoResources(c)

	s.resources.EXPECT().SetCharmStoreResources(gomock.Any(), gomock.Len(0), s.clock.Now().UTC()).Return(nil).AnyTimes()

	return ctrl
}

func (s *updaterSuite) setupMocksNoResources(c *gc.C) *gomock.Controller {
	ctrl := gomock.NewController(c)
	s.model = mocks.NewMockModel(ctrl)
	s.resources = mocks.NewMockResources(ctrl)
	s.cloudService = commonmocks.NewMockCloudService(ctrl)
	s.cloudService.EXPECT().Cloud(gomock.Any(), "dummy").Return(&cloud.Cloud{Type: "cloud"}, nil).AnyTimes()

	s.state = mocks.NewMockState(ctrl)

	s.state.EXPECT().ControllerUUID().Return("controller-1").AnyTimes()
	s.state.EXPECT().Model().Return(s.model, nil).AnyTimes()
	s.state.EXPECT().Resources(gomock.Any()).Return(s.resources).AnyTimes()

	s.objectStore = mocks.NewMockObjectStore(ctrl)
	s.modelConfigService = mocks.NewMockModelConfigService(ctrl)
	s.client = mocks.NewMockCharmhubRefreshClient(ctrl)
	return ctrl
}

func (s *updaterSuite) expectCharmHubModel(c *gc.C) {
	mExp := s.model.EXPECT()
	uuid := testing.ModelTag.Id()

	cfg, err := config.New(config.UseDefaults, map[string]interface{}{
		"name": "model",
		"type": "type",
		"uuid": uuid,
	})
	c.Assert(err, jc.ErrorIsNil)
	s.modelConfigService.EXPECT().ModelConfig(gomock.Any()).Return(cfg, nil).AnyTimes()

	mExp.Metrics().Return(state.ModelMetrics{
		UUID:           uuid,
		ControllerUUID: "controller-1",
		CloudName:      "cloud",
	}, nil).AnyTimes()
	mExp.ModelTag().Return(testing.ModelTag).AnyTimes()
	s.state.EXPECT().AliveRelationKeys().Return(nil)
}

func (s *updaterSuite) api(c *gc.C) *charmrevisionupdater.CharmRevisionUpdaterAPI {
	clientFunc := func(_ context.Context) (charmrevisionupdater.CharmhubRefreshClient, error) {
		return s.client, nil
	}
	api, err := charmrevisionupdater.NewCharmRevisionUpdaterAPIState(
		s.state,
		s.objectStore,
		s.clock,
		s.modelConfigService,
		clientFunc,
		loggertesting.WrapCheckLog(c))
	c.Assert(err, jc.ErrorIsNil)
	return api
}
