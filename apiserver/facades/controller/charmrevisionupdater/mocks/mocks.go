// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/apiserver/facades/controller/charmrevisionupdater (interfaces: Application,CharmhubRefreshClient,Model,State,ModelConfigService,Resources)
//
// Generated by this command:
//
//	mockgen -typed -package mocks -destination mocks/mocks.go github.com/juju/juju/apiserver/facades/controller/charmrevisionupdater Application,CharmhubRefreshClient,Model,State,ModelConfigService,Resources
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"
	time "time"

	charmrevisionupdater "github.com/juju/juju/apiserver/facades/controller/charmrevisionupdater"
	metrics "github.com/juju/juju/core/charm/metrics"
	objectstore "github.com/juju/juju/core/objectstore"
	config "github.com/juju/juju/environs/config"
	resource "github.com/juju/juju/internal/charm/resource"
	charmhub "github.com/juju/juju/internal/charmhub"
	transport "github.com/juju/juju/internal/charmhub/transport"
	state "github.com/juju/juju/state"
	names "github.com/juju/names/v5"
	gomock "go.uber.org/mock/gomock"
)

// MockApplication is a mock of Application interface.
type MockApplication struct {
	ctrl     *gomock.Controller
	recorder *MockApplicationMockRecorder
}

// MockApplicationMockRecorder is the mock recorder for MockApplication.
type MockApplicationMockRecorder struct {
	mock *MockApplication
}

// NewMockApplication creates a new mock instance.
func NewMockApplication(ctrl *gomock.Controller) *MockApplication {
	mock := &MockApplication{ctrl: ctrl}
	mock.recorder = &MockApplicationMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockApplication) EXPECT() *MockApplicationMockRecorder {
	return m.recorder
}

// ApplicationTag mocks base method.
func (m *MockApplication) ApplicationTag() names.ApplicationTag {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ApplicationTag")
	ret0, _ := ret[0].(names.ApplicationTag)
	return ret0
}

// ApplicationTag indicates an expected call of ApplicationTag.
func (mr *MockApplicationMockRecorder) ApplicationTag() *MockApplicationApplicationTagCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ApplicationTag", reflect.TypeOf((*MockApplication)(nil).ApplicationTag))
	return &MockApplicationApplicationTagCall{Call: call}
}

// MockApplicationApplicationTagCall wrap *gomock.Call
type MockApplicationApplicationTagCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockApplicationApplicationTagCall) Return(arg0 names.ApplicationTag) *MockApplicationApplicationTagCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockApplicationApplicationTagCall) Do(f func() names.ApplicationTag) *MockApplicationApplicationTagCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockApplicationApplicationTagCall) DoAndReturn(f func() names.ApplicationTag) *MockApplicationApplicationTagCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// CharmOrigin mocks base method.
func (m *MockApplication) CharmOrigin() *state.CharmOrigin {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CharmOrigin")
	ret0, _ := ret[0].(*state.CharmOrigin)
	return ret0
}

// CharmOrigin indicates an expected call of CharmOrigin.
func (mr *MockApplicationMockRecorder) CharmOrigin() *MockApplicationCharmOriginCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CharmOrigin", reflect.TypeOf((*MockApplication)(nil).CharmOrigin))
	return &MockApplicationCharmOriginCall{Call: call}
}

// MockApplicationCharmOriginCall wrap *gomock.Call
type MockApplicationCharmOriginCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockApplicationCharmOriginCall) Return(arg0 *state.CharmOrigin) *MockApplicationCharmOriginCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockApplicationCharmOriginCall) Do(f func() *state.CharmOrigin) *MockApplicationCharmOriginCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockApplicationCharmOriginCall) DoAndReturn(f func() *state.CharmOrigin) *MockApplicationCharmOriginCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// CharmURL mocks base method.
func (m *MockApplication) CharmURL() (*string, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CharmURL")
	ret0, _ := ret[0].(*string)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// CharmURL indicates an expected call of CharmURL.
func (mr *MockApplicationMockRecorder) CharmURL() *MockApplicationCharmURLCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CharmURL", reflect.TypeOf((*MockApplication)(nil).CharmURL))
	return &MockApplicationCharmURLCall{Call: call}
}

// MockApplicationCharmURLCall wrap *gomock.Call
type MockApplicationCharmURLCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockApplicationCharmURLCall) Return(arg0 *string, arg1 bool) *MockApplicationCharmURLCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockApplicationCharmURLCall) Do(f func() (*string, bool)) *MockApplicationCharmURLCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockApplicationCharmURLCall) DoAndReturn(f func() (*string, bool)) *MockApplicationCharmURLCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// UnitCount mocks base method.
func (m *MockApplication) UnitCount() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnitCount")
	ret0, _ := ret[0].(int)
	return ret0
}

// UnitCount indicates an expected call of UnitCount.
func (mr *MockApplicationMockRecorder) UnitCount() *MockApplicationUnitCountCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnitCount", reflect.TypeOf((*MockApplication)(nil).UnitCount))
	return &MockApplicationUnitCountCall{Call: call}
}

// MockApplicationUnitCountCall wrap *gomock.Call
type MockApplicationUnitCountCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockApplicationUnitCountCall) Return(arg0 int) *MockApplicationUnitCountCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockApplicationUnitCountCall) Do(f func() int) *MockApplicationUnitCountCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockApplicationUnitCountCall) DoAndReturn(f func() int) *MockApplicationUnitCountCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// MockCharmhubRefreshClient is a mock of CharmhubRefreshClient interface.
type MockCharmhubRefreshClient struct {
	ctrl     *gomock.Controller
	recorder *MockCharmhubRefreshClientMockRecorder
}

// MockCharmhubRefreshClientMockRecorder is the mock recorder for MockCharmhubRefreshClient.
type MockCharmhubRefreshClientMockRecorder struct {
	mock *MockCharmhubRefreshClient
}

// NewMockCharmhubRefreshClient creates a new mock instance.
func NewMockCharmhubRefreshClient(ctrl *gomock.Controller) *MockCharmhubRefreshClient {
	mock := &MockCharmhubRefreshClient{ctrl: ctrl}
	mock.recorder = &MockCharmhubRefreshClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCharmhubRefreshClient) EXPECT() *MockCharmhubRefreshClientMockRecorder {
	return m.recorder
}

// RefreshWithMetricsOnly mocks base method.
func (m *MockCharmhubRefreshClient) RefreshWithMetricsOnly(arg0 context.Context, arg1 map[metrics.MetricKey]map[metrics.MetricKey]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RefreshWithMetricsOnly", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// RefreshWithMetricsOnly indicates an expected call of RefreshWithMetricsOnly.
func (mr *MockCharmhubRefreshClientMockRecorder) RefreshWithMetricsOnly(arg0, arg1 any) *MockCharmhubRefreshClientRefreshWithMetricsOnlyCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RefreshWithMetricsOnly", reflect.TypeOf((*MockCharmhubRefreshClient)(nil).RefreshWithMetricsOnly), arg0, arg1)
	return &MockCharmhubRefreshClientRefreshWithMetricsOnlyCall{Call: call}
}

// MockCharmhubRefreshClientRefreshWithMetricsOnlyCall wrap *gomock.Call
type MockCharmhubRefreshClientRefreshWithMetricsOnlyCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockCharmhubRefreshClientRefreshWithMetricsOnlyCall) Return(arg0 error) *MockCharmhubRefreshClientRefreshWithMetricsOnlyCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockCharmhubRefreshClientRefreshWithMetricsOnlyCall) Do(f func(context.Context, map[metrics.MetricKey]map[metrics.MetricKey]string) error) *MockCharmhubRefreshClientRefreshWithMetricsOnlyCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockCharmhubRefreshClientRefreshWithMetricsOnlyCall) DoAndReturn(f func(context.Context, map[metrics.MetricKey]map[metrics.MetricKey]string) error) *MockCharmhubRefreshClientRefreshWithMetricsOnlyCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// RefreshWithRequestMetrics mocks base method.
func (m *MockCharmhubRefreshClient) RefreshWithRequestMetrics(arg0 context.Context, arg1 charmhub.RefreshConfig, arg2 map[metrics.MetricKey]map[metrics.MetricKey]string) ([]transport.RefreshResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RefreshWithRequestMetrics", arg0, arg1, arg2)
	ret0, _ := ret[0].([]transport.RefreshResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RefreshWithRequestMetrics indicates an expected call of RefreshWithRequestMetrics.
func (mr *MockCharmhubRefreshClientMockRecorder) RefreshWithRequestMetrics(arg0, arg1, arg2 any) *MockCharmhubRefreshClientRefreshWithRequestMetricsCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RefreshWithRequestMetrics", reflect.TypeOf((*MockCharmhubRefreshClient)(nil).RefreshWithRequestMetrics), arg0, arg1, arg2)
	return &MockCharmhubRefreshClientRefreshWithRequestMetricsCall{Call: call}
}

// MockCharmhubRefreshClientRefreshWithRequestMetricsCall wrap *gomock.Call
type MockCharmhubRefreshClientRefreshWithRequestMetricsCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockCharmhubRefreshClientRefreshWithRequestMetricsCall) Return(arg0 []transport.RefreshResponse, arg1 error) *MockCharmhubRefreshClientRefreshWithRequestMetricsCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockCharmhubRefreshClientRefreshWithRequestMetricsCall) Do(f func(context.Context, charmhub.RefreshConfig, map[metrics.MetricKey]map[metrics.MetricKey]string) ([]transport.RefreshResponse, error)) *MockCharmhubRefreshClientRefreshWithRequestMetricsCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockCharmhubRefreshClientRefreshWithRequestMetricsCall) DoAndReturn(f func(context.Context, charmhub.RefreshConfig, map[metrics.MetricKey]map[metrics.MetricKey]string) ([]transport.RefreshResponse, error)) *MockCharmhubRefreshClientRefreshWithRequestMetricsCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// MockModel is a mock of Model interface.
type MockModel struct {
	ctrl     *gomock.Controller
	recorder *MockModelMockRecorder
}

// MockModelMockRecorder is the mock recorder for MockModel.
type MockModelMockRecorder struct {
	mock *MockModel
}

// NewMockModel creates a new mock instance.
func NewMockModel(ctrl *gomock.Controller) *MockModel {
	mock := &MockModel{ctrl: ctrl}
	mock.recorder = &MockModelMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModel) EXPECT() *MockModelMockRecorder {
	return m.recorder
}

// CloudName mocks base method.
func (m *MockModel) CloudName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloudName")
	ret0, _ := ret[0].(string)
	return ret0
}

// CloudName indicates an expected call of CloudName.
func (mr *MockModelMockRecorder) CloudName() *MockModelCloudNameCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloudName", reflect.TypeOf((*MockModel)(nil).CloudName))
	return &MockModelCloudNameCall{Call: call}
}

// MockModelCloudNameCall wrap *gomock.Call
type MockModelCloudNameCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockModelCloudNameCall) Return(arg0 string) *MockModelCloudNameCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockModelCloudNameCall) Do(f func() string) *MockModelCloudNameCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockModelCloudNameCall) DoAndReturn(f func() string) *MockModelCloudNameCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// CloudRegion mocks base method.
func (m *MockModel) CloudRegion() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloudRegion")
	ret0, _ := ret[0].(string)
	return ret0
}

// CloudRegion indicates an expected call of CloudRegion.
func (mr *MockModelMockRecorder) CloudRegion() *MockModelCloudRegionCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloudRegion", reflect.TypeOf((*MockModel)(nil).CloudRegion))
	return &MockModelCloudRegionCall{Call: call}
}

// MockModelCloudRegionCall wrap *gomock.Call
type MockModelCloudRegionCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockModelCloudRegionCall) Return(arg0 string) *MockModelCloudRegionCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockModelCloudRegionCall) Do(f func() string) *MockModelCloudRegionCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockModelCloudRegionCall) DoAndReturn(f func() string) *MockModelCloudRegionCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Metrics mocks base method.
func (m *MockModel) Metrics() (state.ModelMetrics, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Metrics")
	ret0, _ := ret[0].(state.ModelMetrics)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Metrics indicates an expected call of Metrics.
func (mr *MockModelMockRecorder) Metrics() *MockModelMetricsCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Metrics", reflect.TypeOf((*MockModel)(nil).Metrics))
	return &MockModelMetricsCall{Call: call}
}

// MockModelMetricsCall wrap *gomock.Call
type MockModelMetricsCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockModelMetricsCall) Return(arg0 state.ModelMetrics, arg1 error) *MockModelMetricsCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockModelMetricsCall) Do(f func() (state.ModelMetrics, error)) *MockModelMetricsCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockModelMetricsCall) DoAndReturn(f func() (state.ModelMetrics, error)) *MockModelMetricsCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// ModelTag mocks base method.
func (m *MockModel) ModelTag() names.ModelTag {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelTag")
	ret0, _ := ret[0].(names.ModelTag)
	return ret0
}

// ModelTag indicates an expected call of ModelTag.
func (mr *MockModelMockRecorder) ModelTag() *MockModelModelTagCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModelTag", reflect.TypeOf((*MockModel)(nil).ModelTag))
	return &MockModelModelTagCall{Call: call}
}

// MockModelModelTagCall wrap *gomock.Call
type MockModelModelTagCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockModelModelTagCall) Return(arg0 names.ModelTag) *MockModelModelTagCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockModelModelTagCall) Do(f func() names.ModelTag) *MockModelModelTagCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockModelModelTagCall) DoAndReturn(f func() names.ModelTag) *MockModelModelTagCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// UUID mocks base method.
func (m *MockModel) UUID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UUID")
	ret0, _ := ret[0].(string)
	return ret0
}

// UUID indicates an expected call of UUID.
func (mr *MockModelMockRecorder) UUID() *MockModelUUIDCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UUID", reflect.TypeOf((*MockModel)(nil).UUID))
	return &MockModelUUIDCall{Call: call}
}

// MockModelUUIDCall wrap *gomock.Call
type MockModelUUIDCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockModelUUIDCall) Return(arg0 string) *MockModelUUIDCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockModelUUIDCall) Do(f func() string) *MockModelUUIDCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockModelUUIDCall) DoAndReturn(f func() string) *MockModelUUIDCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// MockState is a mock of State interface.
type MockState struct {
	ctrl     *gomock.Controller
	recorder *MockStateMockRecorder
}

// MockStateMockRecorder is the mock recorder for MockState.
type MockStateMockRecorder struct {
	mock *MockState
}

// NewMockState creates a new mock instance.
func NewMockState(ctrl *gomock.Controller) *MockState {
	mock := &MockState{ctrl: ctrl}
	mock.recorder = &MockStateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockState) EXPECT() *MockStateMockRecorder {
	return m.recorder
}

// AliveRelationKeys mocks base method.
func (m *MockState) AliveRelationKeys() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AliveRelationKeys")
	ret0, _ := ret[0].([]string)
	return ret0
}

// AliveRelationKeys indicates an expected call of AliveRelationKeys.
func (mr *MockStateMockRecorder) AliveRelationKeys() *MockStateAliveRelationKeysCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AliveRelationKeys", reflect.TypeOf((*MockState)(nil).AliveRelationKeys))
	return &MockStateAliveRelationKeysCall{Call: call}
}

// MockStateAliveRelationKeysCall wrap *gomock.Call
type MockStateAliveRelationKeysCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateAliveRelationKeysCall) Return(arg0 []string) *MockStateAliveRelationKeysCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateAliveRelationKeysCall) Do(f func() []string) *MockStateAliveRelationKeysCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateAliveRelationKeysCall) DoAndReturn(f func() []string) *MockStateAliveRelationKeysCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// AllApplications mocks base method.
func (m *MockState) AllApplications() ([]charmrevisionupdater.Application, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AllApplications")
	ret0, _ := ret[0].([]charmrevisionupdater.Application)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AllApplications indicates an expected call of AllApplications.
func (mr *MockStateMockRecorder) AllApplications() *MockStateAllApplicationsCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AllApplications", reflect.TypeOf((*MockState)(nil).AllApplications))
	return &MockStateAllApplicationsCall{Call: call}
}

// MockStateAllApplicationsCall wrap *gomock.Call
type MockStateAllApplicationsCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateAllApplicationsCall) Return(arg0 []charmrevisionupdater.Application, arg1 error) *MockStateAllApplicationsCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateAllApplicationsCall) Do(f func() ([]charmrevisionupdater.Application, error)) *MockStateAllApplicationsCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateAllApplicationsCall) DoAndReturn(f func() ([]charmrevisionupdater.Application, error)) *MockStateAllApplicationsCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Charm mocks base method.
func (m *MockState) Charm(arg0 string) (state.CharmRefFull, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Charm", arg0)
	ret0, _ := ret[0].(state.CharmRefFull)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Charm indicates an expected call of Charm.
func (mr *MockStateMockRecorder) Charm(arg0 any) *MockStateCharmCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Charm", reflect.TypeOf((*MockState)(nil).Charm), arg0)
	return &MockStateCharmCall{Call: call}
}

// MockStateCharmCall wrap *gomock.Call
type MockStateCharmCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateCharmCall) Return(arg0 state.CharmRefFull, arg1 error) *MockStateCharmCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateCharmCall) Do(f func(string) (state.CharmRefFull, error)) *MockStateCharmCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateCharmCall) DoAndReturn(f func(string) (state.CharmRefFull, error)) *MockStateCharmCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// ControllerUUID mocks base method.
func (m *MockState) ControllerUUID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ControllerUUID")
	ret0, _ := ret[0].(string)
	return ret0
}

// ControllerUUID indicates an expected call of ControllerUUID.
func (mr *MockStateMockRecorder) ControllerUUID() *MockStateControllerUUIDCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ControllerUUID", reflect.TypeOf((*MockState)(nil).ControllerUUID))
	return &MockStateControllerUUIDCall{Call: call}
}

// MockStateControllerUUIDCall wrap *gomock.Call
type MockStateControllerUUIDCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateControllerUUIDCall) Return(arg0 string) *MockStateControllerUUIDCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateControllerUUIDCall) Do(f func() string) *MockStateControllerUUIDCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateControllerUUIDCall) DoAndReturn(f func() string) *MockStateControllerUUIDCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Model mocks base method.
func (m *MockState) Model() (charmrevisionupdater.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Model")
	ret0, _ := ret[0].(charmrevisionupdater.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Model indicates an expected call of Model.
func (mr *MockStateMockRecorder) Model() *MockStateModelCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Model", reflect.TypeOf((*MockState)(nil).Model))
	return &MockStateModelCall{Call: call}
}

// MockStateModelCall wrap *gomock.Call
type MockStateModelCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateModelCall) Return(arg0 charmrevisionupdater.Model, arg1 error) *MockStateModelCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateModelCall) Do(f func() (charmrevisionupdater.Model, error)) *MockStateModelCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateModelCall) DoAndReturn(f func() (charmrevisionupdater.Model, error)) *MockStateModelCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Resources mocks base method.
func (m *MockState) Resources(arg0 objectstore.ObjectStore) charmrevisionupdater.Resources {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Resources", arg0)
	ret0, _ := ret[0].(charmrevisionupdater.Resources)
	return ret0
}

// Resources indicates an expected call of Resources.
func (mr *MockStateMockRecorder) Resources(arg0 any) *MockStateResourcesCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Resources", reflect.TypeOf((*MockState)(nil).Resources), arg0)
	return &MockStateResourcesCall{Call: call}
}

// MockStateResourcesCall wrap *gomock.Call
type MockStateResourcesCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateResourcesCall) Return(arg0 charmrevisionupdater.Resources) *MockStateResourcesCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateResourcesCall) Do(f func(objectstore.ObjectStore) charmrevisionupdater.Resources) *MockStateResourcesCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateResourcesCall) DoAndReturn(f func(objectstore.ObjectStore) charmrevisionupdater.Resources) *MockStateResourcesCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// MockModelConfigService is a mock of ModelConfigService interface.
type MockModelConfigService struct {
	ctrl     *gomock.Controller
	recorder *MockModelConfigServiceMockRecorder
}

// MockModelConfigServiceMockRecorder is the mock recorder for MockModelConfigService.
type MockModelConfigServiceMockRecorder struct {
	mock *MockModelConfigService
}

// NewMockModelConfigService creates a new mock instance.
func NewMockModelConfigService(ctrl *gomock.Controller) *MockModelConfigService {
	mock := &MockModelConfigService{ctrl: ctrl}
	mock.recorder = &MockModelConfigServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModelConfigService) EXPECT() *MockModelConfigServiceMockRecorder {
	return m.recorder
}

// ModelConfig mocks base method.
func (m *MockModelConfigService) ModelConfig(arg0 context.Context) (*config.Config, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelConfig", arg0)
	ret0, _ := ret[0].(*config.Config)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ModelConfig indicates an expected call of ModelConfig.
func (mr *MockModelConfigServiceMockRecorder) ModelConfig(arg0 any) *MockModelConfigServiceModelConfigCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModelConfig", reflect.TypeOf((*MockModelConfigService)(nil).ModelConfig), arg0)
	return &MockModelConfigServiceModelConfigCall{Call: call}
}

// MockModelConfigServiceModelConfigCall wrap *gomock.Call
type MockModelConfigServiceModelConfigCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockModelConfigServiceModelConfigCall) Return(arg0 *config.Config, arg1 error) *MockModelConfigServiceModelConfigCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockModelConfigServiceModelConfigCall) Do(f func(context.Context) (*config.Config, error)) *MockModelConfigServiceModelConfigCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockModelConfigServiceModelConfigCall) DoAndReturn(f func(context.Context) (*config.Config, error)) *MockModelConfigServiceModelConfigCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// MockResources is a mock of Resources interface.
type MockResources struct {
	ctrl     *gomock.Controller
	recorder *MockResourcesMockRecorder
}

// MockResourcesMockRecorder is the mock recorder for MockResources.
type MockResourcesMockRecorder struct {
	mock *MockResources
}

// NewMockResources creates a new mock instance.
func NewMockResources(ctrl *gomock.Controller) *MockResources {
	mock := &MockResources{ctrl: ctrl}
	mock.recorder = &MockResourcesMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockResources) EXPECT() *MockResourcesMockRecorder {
	return m.recorder
}

// SetCharmStoreResources mocks base method.
func (m *MockResources) SetCharmStoreResources(arg0 string, arg1 []resource.Resource, arg2 time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetCharmStoreResources", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetCharmStoreResources indicates an expected call of SetCharmStoreResources.
func (mr *MockResourcesMockRecorder) SetCharmStoreResources(arg0, arg1, arg2 any) *MockResourcesSetCharmStoreResourcesCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetCharmStoreResources", reflect.TypeOf((*MockResources)(nil).SetCharmStoreResources), arg0, arg1, arg2)
	return &MockResourcesSetCharmStoreResourcesCall{Call: call}
}

// MockResourcesSetCharmStoreResourcesCall wrap *gomock.Call
type MockResourcesSetCharmStoreResourcesCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockResourcesSetCharmStoreResourcesCall) Return(arg0 error) *MockResourcesSetCharmStoreResourcesCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockResourcesSetCharmStoreResourcesCall) Do(f func(string, []resource.Resource, time.Time) error) *MockResourcesSetCharmStoreResourcesCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockResourcesSetCharmStoreResourcesCall) DoAndReturn(f func(string, []resource.Resource, time.Time) error) *MockResourcesSetCharmStoreResourcesCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
