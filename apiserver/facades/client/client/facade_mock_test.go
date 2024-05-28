// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/apiserver/facade (interfaces: Authorizer)
//
// Generated by this command:
//
//	mockgen -typed -package client_test -destination facade_mock_test.go github.com/juju/juju/apiserver/facade Authorizer
//

// Package client_test is a generated GoMock package.
package client_test

import (
	reflect "reflect"

	permission "github.com/juju/juju/core/permission"
	names "github.com/juju/names/v5"
	gomock "go.uber.org/mock/gomock"
)

// MockAuthorizer is a mock of Authorizer interface.
type MockAuthorizer struct {
	ctrl     *gomock.Controller
	recorder *MockAuthorizerMockRecorder
}

// MockAuthorizerMockRecorder is the mock recorder for MockAuthorizer.
type MockAuthorizerMockRecorder struct {
	mock *MockAuthorizer
}

// NewMockAuthorizer creates a new mock instance.
func NewMockAuthorizer(ctrl *gomock.Controller) *MockAuthorizer {
	mock := &MockAuthorizer{ctrl: ctrl}
	mock.recorder = &MockAuthorizerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAuthorizer) EXPECT() *MockAuthorizerMockRecorder {
	return m.recorder
}

// AuthApplicationAgent mocks base method.
func (m *MockAuthorizer) AuthApplicationAgent() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AuthApplicationAgent")
	ret0, _ := ret[0].(bool)
	return ret0
}

// AuthApplicationAgent indicates an expected call of AuthApplicationAgent.
func (mr *MockAuthorizerMockRecorder) AuthApplicationAgent() *MockAuthorizerAuthApplicationAgentCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AuthApplicationAgent", reflect.TypeOf((*MockAuthorizer)(nil).AuthApplicationAgent))
	return &MockAuthorizerAuthApplicationAgentCall{Call: call}
}

// MockAuthorizerAuthApplicationAgentCall wrap *gomock.Call
type MockAuthorizerAuthApplicationAgentCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockAuthorizerAuthApplicationAgentCall) Return(arg0 bool) *MockAuthorizerAuthApplicationAgentCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockAuthorizerAuthApplicationAgentCall) Do(f func() bool) *MockAuthorizerAuthApplicationAgentCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockAuthorizerAuthApplicationAgentCall) DoAndReturn(f func() bool) *MockAuthorizerAuthApplicationAgentCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// AuthClient mocks base method.
func (m *MockAuthorizer) AuthClient() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AuthClient")
	ret0, _ := ret[0].(bool)
	return ret0
}

// AuthClient indicates an expected call of AuthClient.
func (mr *MockAuthorizerMockRecorder) AuthClient() *MockAuthorizerAuthClientCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AuthClient", reflect.TypeOf((*MockAuthorizer)(nil).AuthClient))
	return &MockAuthorizerAuthClientCall{Call: call}
}

// MockAuthorizerAuthClientCall wrap *gomock.Call
type MockAuthorizerAuthClientCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockAuthorizerAuthClientCall) Return(arg0 bool) *MockAuthorizerAuthClientCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockAuthorizerAuthClientCall) Do(f func() bool) *MockAuthorizerAuthClientCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockAuthorizerAuthClientCall) DoAndReturn(f func() bool) *MockAuthorizerAuthClientCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// AuthController mocks base method.
func (m *MockAuthorizer) AuthController() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AuthController")
	ret0, _ := ret[0].(bool)
	return ret0
}

// AuthController indicates an expected call of AuthController.
func (mr *MockAuthorizerMockRecorder) AuthController() *MockAuthorizerAuthControllerCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AuthController", reflect.TypeOf((*MockAuthorizer)(nil).AuthController))
	return &MockAuthorizerAuthControllerCall{Call: call}
}

// MockAuthorizerAuthControllerCall wrap *gomock.Call
type MockAuthorizerAuthControllerCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockAuthorizerAuthControllerCall) Return(arg0 bool) *MockAuthorizerAuthControllerCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockAuthorizerAuthControllerCall) Do(f func() bool) *MockAuthorizerAuthControllerCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockAuthorizerAuthControllerCall) DoAndReturn(f func() bool) *MockAuthorizerAuthControllerCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// AuthMachineAgent mocks base method.
func (m *MockAuthorizer) AuthMachineAgent() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AuthMachineAgent")
	ret0, _ := ret[0].(bool)
	return ret0
}

// AuthMachineAgent indicates an expected call of AuthMachineAgent.
func (mr *MockAuthorizerMockRecorder) AuthMachineAgent() *MockAuthorizerAuthMachineAgentCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AuthMachineAgent", reflect.TypeOf((*MockAuthorizer)(nil).AuthMachineAgent))
	return &MockAuthorizerAuthMachineAgentCall{Call: call}
}

// MockAuthorizerAuthMachineAgentCall wrap *gomock.Call
type MockAuthorizerAuthMachineAgentCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockAuthorizerAuthMachineAgentCall) Return(arg0 bool) *MockAuthorizerAuthMachineAgentCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockAuthorizerAuthMachineAgentCall) Do(f func() bool) *MockAuthorizerAuthMachineAgentCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockAuthorizerAuthMachineAgentCall) DoAndReturn(f func() bool) *MockAuthorizerAuthMachineAgentCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// AuthModelAgent mocks base method.
func (m *MockAuthorizer) AuthModelAgent() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AuthModelAgent")
	ret0, _ := ret[0].(bool)
	return ret0
}

// AuthModelAgent indicates an expected call of AuthModelAgent.
func (mr *MockAuthorizerMockRecorder) AuthModelAgent() *MockAuthorizerAuthModelAgentCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AuthModelAgent", reflect.TypeOf((*MockAuthorizer)(nil).AuthModelAgent))
	return &MockAuthorizerAuthModelAgentCall{Call: call}
}

// MockAuthorizerAuthModelAgentCall wrap *gomock.Call
type MockAuthorizerAuthModelAgentCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockAuthorizerAuthModelAgentCall) Return(arg0 bool) *MockAuthorizerAuthModelAgentCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockAuthorizerAuthModelAgentCall) Do(f func() bool) *MockAuthorizerAuthModelAgentCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockAuthorizerAuthModelAgentCall) DoAndReturn(f func() bool) *MockAuthorizerAuthModelAgentCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// AuthOwner mocks base method.
func (m *MockAuthorizer) AuthOwner(arg0 names.Tag) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AuthOwner", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// AuthOwner indicates an expected call of AuthOwner.
func (mr *MockAuthorizerMockRecorder) AuthOwner(arg0 any) *MockAuthorizerAuthOwnerCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AuthOwner", reflect.TypeOf((*MockAuthorizer)(nil).AuthOwner), arg0)
	return &MockAuthorizerAuthOwnerCall{Call: call}
}

// MockAuthorizerAuthOwnerCall wrap *gomock.Call
type MockAuthorizerAuthOwnerCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockAuthorizerAuthOwnerCall) Return(arg0 bool) *MockAuthorizerAuthOwnerCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockAuthorizerAuthOwnerCall) Do(f func(names.Tag) bool) *MockAuthorizerAuthOwnerCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockAuthorizerAuthOwnerCall) DoAndReturn(f func(names.Tag) bool) *MockAuthorizerAuthOwnerCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// AuthUnitAgent mocks base method.
func (m *MockAuthorizer) AuthUnitAgent() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AuthUnitAgent")
	ret0, _ := ret[0].(bool)
	return ret0
}

// AuthUnitAgent indicates an expected call of AuthUnitAgent.
func (mr *MockAuthorizerMockRecorder) AuthUnitAgent() *MockAuthorizerAuthUnitAgentCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AuthUnitAgent", reflect.TypeOf((*MockAuthorizer)(nil).AuthUnitAgent))
	return &MockAuthorizerAuthUnitAgentCall{Call: call}
}

// MockAuthorizerAuthUnitAgentCall wrap *gomock.Call
type MockAuthorizerAuthUnitAgentCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockAuthorizerAuthUnitAgentCall) Return(arg0 bool) *MockAuthorizerAuthUnitAgentCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockAuthorizerAuthUnitAgentCall) Do(f func() bool) *MockAuthorizerAuthUnitAgentCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockAuthorizerAuthUnitAgentCall) DoAndReturn(f func() bool) *MockAuthorizerAuthUnitAgentCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// ConnectedModel mocks base method.
func (m *MockAuthorizer) ConnectedModel() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConnectedModel")
	ret0, _ := ret[0].(string)
	return ret0
}

// ConnectedModel indicates an expected call of ConnectedModel.
func (mr *MockAuthorizerMockRecorder) ConnectedModel() *MockAuthorizerConnectedModelCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectedModel", reflect.TypeOf((*MockAuthorizer)(nil).ConnectedModel))
	return &MockAuthorizerConnectedModelCall{Call: call}
}

// MockAuthorizerConnectedModelCall wrap *gomock.Call
type MockAuthorizerConnectedModelCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockAuthorizerConnectedModelCall) Return(arg0 string) *MockAuthorizerConnectedModelCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockAuthorizerConnectedModelCall) Do(f func() string) *MockAuthorizerConnectedModelCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockAuthorizerConnectedModelCall) DoAndReturn(f func() string) *MockAuthorizerConnectedModelCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// EntityHasPermission mocks base method.
func (m *MockAuthorizer) EntityHasPermission(arg0 names.Tag, arg1 permission.Access, arg2 names.Tag) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EntityHasPermission", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// EntityHasPermission indicates an expected call of EntityHasPermission.
func (mr *MockAuthorizerMockRecorder) EntityHasPermission(arg0, arg1, arg2 any) *MockAuthorizerEntityHasPermissionCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EntityHasPermission", reflect.TypeOf((*MockAuthorizer)(nil).EntityHasPermission), arg0, arg1, arg2)
	return &MockAuthorizerEntityHasPermissionCall{Call: call}
}

// MockAuthorizerEntityHasPermissionCall wrap *gomock.Call
type MockAuthorizerEntityHasPermissionCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockAuthorizerEntityHasPermissionCall) Return(arg0 error) *MockAuthorizerEntityHasPermissionCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockAuthorizerEntityHasPermissionCall) Do(f func(names.Tag, permission.Access, names.Tag) error) *MockAuthorizerEntityHasPermissionCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockAuthorizerEntityHasPermissionCall) DoAndReturn(f func(names.Tag, permission.Access, names.Tag) error) *MockAuthorizerEntityHasPermissionCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// GetAuthTag mocks base method.
func (m *MockAuthorizer) GetAuthTag() names.Tag {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAuthTag")
	ret0, _ := ret[0].(names.Tag)
	return ret0
}

// GetAuthTag indicates an expected call of GetAuthTag.
func (mr *MockAuthorizerMockRecorder) GetAuthTag() *MockAuthorizerGetAuthTagCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAuthTag", reflect.TypeOf((*MockAuthorizer)(nil).GetAuthTag))
	return &MockAuthorizerGetAuthTagCall{Call: call}
}

// MockAuthorizerGetAuthTagCall wrap *gomock.Call
type MockAuthorizerGetAuthTagCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockAuthorizerGetAuthTagCall) Return(arg0 names.Tag) *MockAuthorizerGetAuthTagCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockAuthorizerGetAuthTagCall) Do(f func() names.Tag) *MockAuthorizerGetAuthTagCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockAuthorizerGetAuthTagCall) DoAndReturn(f func() names.Tag) *MockAuthorizerGetAuthTagCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// HasPermission mocks base method.
func (m *MockAuthorizer) HasPermission(arg0 permission.Access, arg1 names.Tag) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasPermission", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// HasPermission indicates an expected call of HasPermission.
func (mr *MockAuthorizerMockRecorder) HasPermission(arg0, arg1 any) *MockAuthorizerHasPermissionCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasPermission", reflect.TypeOf((*MockAuthorizer)(nil).HasPermission), arg0, arg1)
	return &MockAuthorizerHasPermissionCall{Call: call}
}

// MockAuthorizerHasPermissionCall wrap *gomock.Call
type MockAuthorizerHasPermissionCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockAuthorizerHasPermissionCall) Return(arg0 error) *MockAuthorizerHasPermissionCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockAuthorizerHasPermissionCall) Do(f func(permission.Access, names.Tag) error) *MockAuthorizerHasPermissionCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockAuthorizerHasPermissionCall) DoAndReturn(f func(permission.Access, names.Tag) error) *MockAuthorizerHasPermissionCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}