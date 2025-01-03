// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/apiserver (interfaces: ApplicationServiceGetter,ApplicationService)
//
// Generated by this command:
//
//	mockgen -typed -package apiserver -destination application_service_mock_test.go github.com/juju/juju/apiserver ApplicationServiceGetter,ApplicationService
//

// Package apiserver is a generated GoMock package.
package apiserver

import (
	context "context"
	io "io"
	reflect "reflect"

	charm "github.com/juju/juju/core/charm"
	charm0 "github.com/juju/juju/domain/application/charm"
	charm1 "github.com/juju/juju/internal/charm"
	gomock "go.uber.org/mock/gomock"
)

// MockApplicationServiceGetter is a mock of ApplicationServiceGetter interface.
type MockApplicationServiceGetter struct {
	ctrl     *gomock.Controller
	recorder *MockApplicationServiceGetterMockRecorder
}

// MockApplicationServiceGetterMockRecorder is the mock recorder for MockApplicationServiceGetter.
type MockApplicationServiceGetterMockRecorder struct {
	mock *MockApplicationServiceGetter
}

// NewMockApplicationServiceGetter creates a new mock instance.
func NewMockApplicationServiceGetter(ctrl *gomock.Controller) *MockApplicationServiceGetter {
	mock := &MockApplicationServiceGetter{ctrl: ctrl}
	mock.recorder = &MockApplicationServiceGetterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockApplicationServiceGetter) EXPECT() *MockApplicationServiceGetterMockRecorder {
	return m.recorder
}

// Application mocks base method.
func (m *MockApplicationServiceGetter) Application(arg0 context.Context) (ApplicationService, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Application", arg0)
	ret0, _ := ret[0].(ApplicationService)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Application indicates an expected call of Application.
func (mr *MockApplicationServiceGetterMockRecorder) Application(arg0 any) *MockApplicationServiceGetterApplicationCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Application", reflect.TypeOf((*MockApplicationServiceGetter)(nil).Application), arg0)
	return &MockApplicationServiceGetterApplicationCall{Call: call}
}

// MockApplicationServiceGetterApplicationCall wrap *gomock.Call
type MockApplicationServiceGetterApplicationCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockApplicationServiceGetterApplicationCall) Return(arg0 ApplicationService, arg1 error) *MockApplicationServiceGetterApplicationCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockApplicationServiceGetterApplicationCall) Do(f func(context.Context) (ApplicationService, error)) *MockApplicationServiceGetterApplicationCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockApplicationServiceGetterApplicationCall) DoAndReturn(f func(context.Context) (ApplicationService, error)) *MockApplicationServiceGetterApplicationCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// MockApplicationService is a mock of ApplicationService interface.
type MockApplicationService struct {
	ctrl     *gomock.Controller
	recorder *MockApplicationServiceMockRecorder
}

// MockApplicationServiceMockRecorder is the mock recorder for MockApplicationService.
type MockApplicationServiceMockRecorder struct {
	mock *MockApplicationService
}

// NewMockApplicationService creates a new mock instance.
func NewMockApplicationService(ctrl *gomock.Controller) *MockApplicationService {
	mock := &MockApplicationService{ctrl: ctrl}
	mock.recorder = &MockApplicationServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockApplicationService) EXPECT() *MockApplicationServiceMockRecorder {
	return m.recorder
}

// GetCharm mocks base method.
func (m *MockApplicationService) GetCharm(arg0 context.Context, arg1 charm.ID) (charm1.Charm, charm0.CharmLocator, bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCharm", arg0, arg1)
	ret0, _ := ret[0].(charm1.Charm)
	ret1, _ := ret[1].(charm0.CharmLocator)
	ret2, _ := ret[2].(bool)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// GetCharm indicates an expected call of GetCharm.
func (mr *MockApplicationServiceMockRecorder) GetCharm(arg0, arg1 any) *MockApplicationServiceGetCharmCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCharm", reflect.TypeOf((*MockApplicationService)(nil).GetCharm), arg0, arg1)
	return &MockApplicationServiceGetCharmCall{Call: call}
}

// MockApplicationServiceGetCharmCall wrap *gomock.Call
type MockApplicationServiceGetCharmCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockApplicationServiceGetCharmCall) Return(arg0 charm1.Charm, arg1 charm0.CharmLocator, arg2 bool, arg3 error) *MockApplicationServiceGetCharmCall {
	c.Call = c.Call.Return(arg0, arg1, arg2, arg3)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockApplicationServiceGetCharmCall) Do(f func(context.Context, charm.ID) (charm1.Charm, charm0.CharmLocator, bool, error)) *MockApplicationServiceGetCharmCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockApplicationServiceGetCharmCall) DoAndReturn(f func(context.Context, charm.ID) (charm1.Charm, charm0.CharmLocator, bool, error)) *MockApplicationServiceGetCharmCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// GetCharmArchiveBySHA256Prefix mocks base method.
func (m *MockApplicationService) GetCharmArchiveBySHA256Prefix(arg0 context.Context, arg1 string) (io.ReadCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCharmArchiveBySHA256Prefix", arg0, arg1)
	ret0, _ := ret[0].(io.ReadCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCharmArchiveBySHA256Prefix indicates an expected call of GetCharmArchiveBySHA256Prefix.
func (mr *MockApplicationServiceMockRecorder) GetCharmArchiveBySHA256Prefix(arg0, arg1 any) *MockApplicationServiceGetCharmArchiveBySHA256PrefixCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCharmArchiveBySHA256Prefix", reflect.TypeOf((*MockApplicationService)(nil).GetCharmArchiveBySHA256Prefix), arg0, arg1)
	return &MockApplicationServiceGetCharmArchiveBySHA256PrefixCall{Call: call}
}

// MockApplicationServiceGetCharmArchiveBySHA256PrefixCall wrap *gomock.Call
type MockApplicationServiceGetCharmArchiveBySHA256PrefixCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockApplicationServiceGetCharmArchiveBySHA256PrefixCall) Return(arg0 io.ReadCloser, arg1 error) *MockApplicationServiceGetCharmArchiveBySHA256PrefixCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockApplicationServiceGetCharmArchiveBySHA256PrefixCall) Do(f func(context.Context, string) (io.ReadCloser, error)) *MockApplicationServiceGetCharmArchiveBySHA256PrefixCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockApplicationServiceGetCharmArchiveBySHA256PrefixCall) DoAndReturn(f func(context.Context, string) (io.ReadCloser, error)) *MockApplicationServiceGetCharmArchiveBySHA256PrefixCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// GetCharmID mocks base method.
func (m *MockApplicationService) GetCharmID(arg0 context.Context, arg1 charm0.GetCharmArgs) (charm.ID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCharmID", arg0, arg1)
	ret0, _ := ret[0].(charm.ID)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCharmID indicates an expected call of GetCharmID.
func (mr *MockApplicationServiceMockRecorder) GetCharmID(arg0, arg1 any) *MockApplicationServiceGetCharmIDCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCharmID", reflect.TypeOf((*MockApplicationService)(nil).GetCharmID), arg0, arg1)
	return &MockApplicationServiceGetCharmIDCall{Call: call}
}

// MockApplicationServiceGetCharmIDCall wrap *gomock.Call
type MockApplicationServiceGetCharmIDCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockApplicationServiceGetCharmIDCall) Return(arg0 charm.ID, arg1 error) *MockApplicationServiceGetCharmIDCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockApplicationServiceGetCharmIDCall) Do(f func(context.Context, charm0.GetCharmArgs) (charm.ID, error)) *MockApplicationServiceGetCharmIDCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockApplicationServiceGetCharmIDCall) DoAndReturn(f func(context.Context, charm0.GetCharmArgs) (charm.ID, error)) *MockApplicationServiceGetCharmIDCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// SetCharm mocks base method.
func (m *MockApplicationService) SetCharm(arg0 context.Context, arg1 charm0.SetCharmArgs) (charm.ID, []string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetCharm", arg0, arg1)
	ret0, _ := ret[0].(charm.ID)
	ret1, _ := ret[1].([]string)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// SetCharm indicates an expected call of SetCharm.
func (mr *MockApplicationServiceMockRecorder) SetCharm(arg0, arg1 any) *MockApplicationServiceSetCharmCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetCharm", reflect.TypeOf((*MockApplicationService)(nil).SetCharm), arg0, arg1)
	return &MockApplicationServiceSetCharmCall{Call: call}
}

// MockApplicationServiceSetCharmCall wrap *gomock.Call
type MockApplicationServiceSetCharmCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockApplicationServiceSetCharmCall) Return(arg0 charm.ID, arg1 []string, arg2 error) *MockApplicationServiceSetCharmCall {
	c.Call = c.Call.Return(arg0, arg1, arg2)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockApplicationServiceSetCharmCall) Do(f func(context.Context, charm0.SetCharmArgs) (charm.ID, []string, error)) *MockApplicationServiceSetCharmCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockApplicationServiceSetCharmCall) DoAndReturn(f func(context.Context, charm0.SetCharmArgs) (charm.ID, []string, error)) *MockApplicationServiceSetCharmCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
