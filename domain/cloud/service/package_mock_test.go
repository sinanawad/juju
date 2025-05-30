// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/domain/cloud/service (interfaces: State,WatcherFactory)
//
// Generated by this command:
//
//	mockgen -typed -package service -destination package_mock_test.go github.com/juju/juju/domain/cloud/service State,WatcherFactory
//

// Package service is a generated GoMock package.
package service

import (
	context "context"
	reflect "reflect"

	cloud "github.com/juju/juju/cloud"
	user "github.com/juju/juju/core/user"
	watcher "github.com/juju/juju/core/watcher"
	eventsource "github.com/juju/juju/core/watcher/eventsource"
	gomock "go.uber.org/mock/gomock"
)

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

// Cloud mocks base method.
func (m *MockState) Cloud(arg0 context.Context, arg1 string) (*cloud.Cloud, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Cloud", arg0, arg1)
	ret0, _ := ret[0].(*cloud.Cloud)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Cloud indicates an expected call of Cloud.
func (mr *MockStateMockRecorder) Cloud(arg0, arg1 any) *MockStateCloudCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Cloud", reflect.TypeOf((*MockState)(nil).Cloud), arg0, arg1)
	return &MockStateCloudCall{Call: call}
}

// MockStateCloudCall wrap *gomock.Call
type MockStateCloudCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateCloudCall) Return(arg0 *cloud.Cloud, arg1 error) *MockStateCloudCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateCloudCall) Do(f func(context.Context, string) (*cloud.Cloud, error)) *MockStateCloudCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateCloudCall) DoAndReturn(f func(context.Context, string) (*cloud.Cloud, error)) *MockStateCloudCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// CreateCloud mocks base method.
func (m *MockState) CreateCloud(arg0 context.Context, arg1 user.Name, arg2 string, arg3 cloud.Cloud) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateCloud", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateCloud indicates an expected call of CreateCloud.
func (mr *MockStateMockRecorder) CreateCloud(arg0, arg1, arg2, arg3 any) *MockStateCreateCloudCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateCloud", reflect.TypeOf((*MockState)(nil).CreateCloud), arg0, arg1, arg2, arg3)
	return &MockStateCreateCloudCall{Call: call}
}

// MockStateCreateCloudCall wrap *gomock.Call
type MockStateCreateCloudCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateCreateCloudCall) Return(arg0 error) *MockStateCreateCloudCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateCreateCloudCall) Do(f func(context.Context, user.Name, string, cloud.Cloud) error) *MockStateCreateCloudCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateCreateCloudCall) DoAndReturn(f func(context.Context, user.Name, string, cloud.Cloud) error) *MockStateCreateCloudCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// DeleteCloud mocks base method.
func (m *MockState) DeleteCloud(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteCloud", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteCloud indicates an expected call of DeleteCloud.
func (mr *MockStateMockRecorder) DeleteCloud(arg0, arg1 any) *MockStateDeleteCloudCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteCloud", reflect.TypeOf((*MockState)(nil).DeleteCloud), arg0, arg1)
	return &MockStateDeleteCloudCall{Call: call}
}

// MockStateDeleteCloudCall wrap *gomock.Call
type MockStateDeleteCloudCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateDeleteCloudCall) Return(arg0 error) *MockStateDeleteCloudCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateDeleteCloudCall) Do(f func(context.Context, string) error) *MockStateDeleteCloudCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateDeleteCloudCall) DoAndReturn(f func(context.Context, string) error) *MockStateDeleteCloudCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// ListClouds mocks base method.
func (m *MockState) ListClouds(arg0 context.Context) ([]cloud.Cloud, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListClouds", arg0)
	ret0, _ := ret[0].([]cloud.Cloud)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListClouds indicates an expected call of ListClouds.
func (mr *MockStateMockRecorder) ListClouds(arg0 any) *MockStateListCloudsCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListClouds", reflect.TypeOf((*MockState)(nil).ListClouds), arg0)
	return &MockStateListCloudsCall{Call: call}
}

// MockStateListCloudsCall wrap *gomock.Call
type MockStateListCloudsCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateListCloudsCall) Return(arg0 []cloud.Cloud, arg1 error) *MockStateListCloudsCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateListCloudsCall) Do(f func(context.Context) ([]cloud.Cloud, error)) *MockStateListCloudsCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateListCloudsCall) DoAndReturn(f func(context.Context) ([]cloud.Cloud, error)) *MockStateListCloudsCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// UpdateCloud mocks base method.
func (m *MockState) UpdateCloud(arg0 context.Context, arg1 cloud.Cloud) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateCloud", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateCloud indicates an expected call of UpdateCloud.
func (mr *MockStateMockRecorder) UpdateCloud(arg0, arg1 any) *MockStateUpdateCloudCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateCloud", reflect.TypeOf((*MockState)(nil).UpdateCloud), arg0, arg1)
	return &MockStateUpdateCloudCall{Call: call}
}

// MockStateUpdateCloudCall wrap *gomock.Call
type MockStateUpdateCloudCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateUpdateCloudCall) Return(arg0 error) *MockStateUpdateCloudCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateUpdateCloudCall) Do(f func(context.Context, cloud.Cloud) error) *MockStateUpdateCloudCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateUpdateCloudCall) DoAndReturn(f func(context.Context, cloud.Cloud) error) *MockStateUpdateCloudCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// WatchCloud mocks base method.
func (m *MockState) WatchCloud(arg0 context.Context, arg1 func(eventsource.FilterOption, ...eventsource.FilterOption) (watcher.Watcher[struct{}], error), arg2 string) (watcher.Watcher[struct{}], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchCloud", arg0, arg1, arg2)
	ret0, _ := ret[0].(watcher.Watcher[struct{}])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchCloud indicates an expected call of WatchCloud.
func (mr *MockStateMockRecorder) WatchCloud(arg0, arg1, arg2 any) *MockStateWatchCloudCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchCloud", reflect.TypeOf((*MockState)(nil).WatchCloud), arg0, arg1, arg2)
	return &MockStateWatchCloudCall{Call: call}
}

// MockStateWatchCloudCall wrap *gomock.Call
type MockStateWatchCloudCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateWatchCloudCall) Return(arg0 watcher.Watcher[struct{}], arg1 error) *MockStateWatchCloudCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateWatchCloudCall) Do(f func(context.Context, func(eventsource.FilterOption, ...eventsource.FilterOption) (watcher.Watcher[struct{}], error), string) (watcher.Watcher[struct{}], error)) *MockStateWatchCloudCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateWatchCloudCall) DoAndReturn(f func(context.Context, func(eventsource.FilterOption, ...eventsource.FilterOption) (watcher.Watcher[struct{}], error), string) (watcher.Watcher[struct{}], error)) *MockStateWatchCloudCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// MockWatcherFactory is a mock of WatcherFactory interface.
type MockWatcherFactory struct {
	ctrl     *gomock.Controller
	recorder *MockWatcherFactoryMockRecorder
}

// MockWatcherFactoryMockRecorder is the mock recorder for MockWatcherFactory.
type MockWatcherFactoryMockRecorder struct {
	mock *MockWatcherFactory
}

// NewMockWatcherFactory creates a new mock instance.
func NewMockWatcherFactory(ctrl *gomock.Controller) *MockWatcherFactory {
	mock := &MockWatcherFactory{ctrl: ctrl}
	mock.recorder = &MockWatcherFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockWatcherFactory) EXPECT() *MockWatcherFactoryMockRecorder {
	return m.recorder
}

// NewNotifyWatcher mocks base method.
func (m *MockWatcherFactory) NewNotifyWatcher(arg0 eventsource.FilterOption, arg1 ...eventsource.FilterOption) (watcher.Watcher[struct{}], error) {
	m.ctrl.T.Helper()
	varargs := []any{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "NewNotifyWatcher", varargs...)
	ret0, _ := ret[0].(watcher.Watcher[struct{}])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewNotifyWatcher indicates an expected call of NewNotifyWatcher.
func (mr *MockWatcherFactoryMockRecorder) NewNotifyWatcher(arg0 any, arg1 ...any) *MockWatcherFactoryNewNotifyWatcherCall {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0}, arg1...)
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewNotifyWatcher", reflect.TypeOf((*MockWatcherFactory)(nil).NewNotifyWatcher), varargs...)
	return &MockWatcherFactoryNewNotifyWatcherCall{Call: call}
}

// MockWatcherFactoryNewNotifyWatcherCall wrap *gomock.Call
type MockWatcherFactoryNewNotifyWatcherCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockWatcherFactoryNewNotifyWatcherCall) Return(arg0 watcher.Watcher[struct{}], arg1 error) *MockWatcherFactoryNewNotifyWatcherCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockWatcherFactoryNewNotifyWatcherCall) Do(f func(eventsource.FilterOption, ...eventsource.FilterOption) (watcher.Watcher[struct{}], error)) *MockWatcherFactoryNewNotifyWatcherCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockWatcherFactoryNewNotifyWatcherCall) DoAndReturn(f func(eventsource.FilterOption, ...eventsource.FilterOption) (watcher.Watcher[struct{}], error)) *MockWatcherFactoryNewNotifyWatcherCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
