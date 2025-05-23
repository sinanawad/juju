// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/core/lease (interfaces: Manager)
//
// Generated by this command:
//
//	mockgen -typed -package modelworkermanager_test -destination lease_mock_test.go github.com/juju/juju/core/lease Manager
//

// Package modelworkermanager_test is a generated GoMock package.
package modelworkermanager_test

import (
	reflect "reflect"

	lease "github.com/juju/juju/core/lease"
	gomock "go.uber.org/mock/gomock"
)

// MockManager is a mock of Manager interface.
type MockManager struct {
	ctrl     *gomock.Controller
	recorder *MockManagerMockRecorder
}

// MockManagerMockRecorder is the mock recorder for MockManager.
type MockManagerMockRecorder struct {
	mock *MockManager
}

// NewMockManager creates a new mock instance.
func NewMockManager(ctrl *gomock.Controller) *MockManager {
	mock := &MockManager{ctrl: ctrl}
	mock.recorder = &MockManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockManager) EXPECT() *MockManagerMockRecorder {
	return m.recorder
}

// Checker mocks base method.
func (m *MockManager) Checker(arg0, arg1 string) (lease.Checker, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Checker", arg0, arg1)
	ret0, _ := ret[0].(lease.Checker)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Checker indicates an expected call of Checker.
func (mr *MockManagerMockRecorder) Checker(arg0, arg1 any) *MockManagerCheckerCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Checker", reflect.TypeOf((*MockManager)(nil).Checker), arg0, arg1)
	return &MockManagerCheckerCall{Call: call}
}

// MockManagerCheckerCall wrap *gomock.Call
type MockManagerCheckerCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockManagerCheckerCall) Return(arg0 lease.Checker, arg1 error) *MockManagerCheckerCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockManagerCheckerCall) Do(f func(string, string) (lease.Checker, error)) *MockManagerCheckerCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockManagerCheckerCall) DoAndReturn(f func(string, string) (lease.Checker, error)) *MockManagerCheckerCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Claimer mocks base method.
func (m *MockManager) Claimer(arg0, arg1 string) (lease.Claimer, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Claimer", arg0, arg1)
	ret0, _ := ret[0].(lease.Claimer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Claimer indicates an expected call of Claimer.
func (mr *MockManagerMockRecorder) Claimer(arg0, arg1 any) *MockManagerClaimerCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Claimer", reflect.TypeOf((*MockManager)(nil).Claimer), arg0, arg1)
	return &MockManagerClaimerCall{Call: call}
}

// MockManagerClaimerCall wrap *gomock.Call
type MockManagerClaimerCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockManagerClaimerCall) Return(arg0 lease.Claimer, arg1 error) *MockManagerClaimerCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockManagerClaimerCall) Do(f func(string, string) (lease.Claimer, error)) *MockManagerClaimerCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockManagerClaimerCall) DoAndReturn(f func(string, string) (lease.Claimer, error)) *MockManagerClaimerCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Pinner mocks base method.
func (m *MockManager) Pinner(arg0, arg1 string) (lease.Pinner, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Pinner", arg0, arg1)
	ret0, _ := ret[0].(lease.Pinner)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Pinner indicates an expected call of Pinner.
func (mr *MockManagerMockRecorder) Pinner(arg0, arg1 any) *MockManagerPinnerCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Pinner", reflect.TypeOf((*MockManager)(nil).Pinner), arg0, arg1)
	return &MockManagerPinnerCall{Call: call}
}

// MockManagerPinnerCall wrap *gomock.Call
type MockManagerPinnerCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockManagerPinnerCall) Return(arg0 lease.Pinner, arg1 error) *MockManagerPinnerCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockManagerPinnerCall) Do(f func(string, string) (lease.Pinner, error)) *MockManagerPinnerCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockManagerPinnerCall) DoAndReturn(f func(string, string) (lease.Pinner, error)) *MockManagerPinnerCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Reader mocks base method.
func (m *MockManager) Reader(arg0, arg1 string) (lease.Reader, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Reader", arg0, arg1)
	ret0, _ := ret[0].(lease.Reader)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Reader indicates an expected call of Reader.
func (mr *MockManagerMockRecorder) Reader(arg0, arg1 any) *MockManagerReaderCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Reader", reflect.TypeOf((*MockManager)(nil).Reader), arg0, arg1)
	return &MockManagerReaderCall{Call: call}
}

// MockManagerReaderCall wrap *gomock.Call
type MockManagerReaderCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockManagerReaderCall) Return(arg0 lease.Reader, arg1 error) *MockManagerReaderCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockManagerReaderCall) Do(f func(string, string) (lease.Reader, error)) *MockManagerReaderCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockManagerReaderCall) DoAndReturn(f func(string, string) (lease.Reader, error)) *MockManagerReaderCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Revoker mocks base method.
func (m *MockManager) Revoker(arg0, arg1 string) (lease.Revoker, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Revoker", arg0, arg1)
	ret0, _ := ret[0].(lease.Revoker)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Revoker indicates an expected call of Revoker.
func (mr *MockManagerMockRecorder) Revoker(arg0, arg1 any) *MockManagerRevokerCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Revoker", reflect.TypeOf((*MockManager)(nil).Revoker), arg0, arg1)
	return &MockManagerRevokerCall{Call: call}
}

// MockManagerRevokerCall wrap *gomock.Call
type MockManagerRevokerCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockManagerRevokerCall) Return(arg0 lease.Revoker, arg1 error) *MockManagerRevokerCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockManagerRevokerCall) Do(f func(string, string) (lease.Revoker, error)) *MockManagerRevokerCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockManagerRevokerCall) DoAndReturn(f func(string, string) (lease.Revoker, error)) *MockManagerRevokerCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
