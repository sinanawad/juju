// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/domain/modelagent/service (interfaces: State)
//
// Generated by this command:
//
//	mockgen -typed -package service -destination service_mock_test.go github.com/juju/juju/domain/modelagent/service State
//

// Package service is a generated GoMock package.
package service

import (
	context "context"
	reflect "reflect"

	model "github.com/juju/juju/core/model"
	version "github.com/juju/version/v2"
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

// GetModelAgentVersion mocks base method.
func (m *MockState) GetModelAgentVersion(arg0 context.Context, arg1 model.UUID) (version.Number, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModelAgentVersion", arg0, arg1)
	ret0, _ := ret[0].(version.Number)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModelAgentVersion indicates an expected call of GetModelAgentVersion.
func (mr *MockStateMockRecorder) GetModelAgentVersion(arg0, arg1 any) *MockStateGetModelAgentVersionCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModelAgentVersion", reflect.TypeOf((*MockState)(nil).GetModelAgentVersion), arg0, arg1)
	return &MockStateGetModelAgentVersionCall{Call: call}
}

// MockStateGetModelAgentVersionCall wrap *gomock.Call
type MockStateGetModelAgentVersionCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockStateGetModelAgentVersionCall) Return(arg0 version.Number, arg1 error) *MockStateGetModelAgentVersionCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockStateGetModelAgentVersionCall) Do(f func(context.Context, model.UUID) (version.Number, error)) *MockStateGetModelAgentVersionCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockStateGetModelAgentVersionCall) DoAndReturn(f func(context.Context, model.UUID) (version.Number, error)) *MockStateGetModelAgentVersionCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}