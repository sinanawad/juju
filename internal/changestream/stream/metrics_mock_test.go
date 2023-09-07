// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/internal/changestream/stream (interfaces: MetricsCollector)

// Package stream is a generated GoMock package.
package stream

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockMetricsCollector is a mock of MetricsCollector interface.
type MockMetricsCollector struct {
	ctrl     *gomock.Controller
	recorder *MockMetricsCollectorMockRecorder
}

// MockMetricsCollectorMockRecorder is the mock recorder for MockMetricsCollector.
type MockMetricsCollectorMockRecorder struct {
	mock *MockMetricsCollector
}

// NewMockMetricsCollector creates a new mock instance.
func NewMockMetricsCollector(ctrl *gomock.Controller) *MockMetricsCollector {
	mock := &MockMetricsCollector{ctrl: ctrl}
	mock.recorder = &MockMetricsCollectorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMetricsCollector) EXPECT() *MockMetricsCollectorMockRecorder {
	return m.recorder
}

// ChangesCountObserve mocks base method.
func (m *MockMetricsCollector) ChangesCountObserve(arg0 int) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "ChangesCountObserve", arg0)
}

// ChangesCountObserve indicates an expected call of ChangesCountObserve.
func (mr *MockMetricsCollectorMockRecorder) ChangesCountObserve(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChangesCountObserve", reflect.TypeOf((*MockMetricsCollector)(nil).ChangesCountObserve), arg0)
}

// ChangesRequestDurationObserve mocks base method.
func (m *MockMetricsCollector) ChangesRequestDurationObserve(arg0 float64) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "ChangesRequestDurationObserve", arg0)
}

// ChangesRequestDurationObserve indicates an expected call of ChangesRequestDurationObserve.
func (mr *MockMetricsCollectorMockRecorder) ChangesRequestDurationObserve(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChangesRequestDurationObserve", reflect.TypeOf((*MockMetricsCollector)(nil).ChangesRequestDurationObserve), arg0)
}

// WatermarkInsertsInc mocks base method.
func (m *MockMetricsCollector) WatermarkInsertsInc() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "WatermarkInsertsInc")
}

// WatermarkInsertsInc indicates an expected call of WatermarkInsertsInc.
func (mr *MockMetricsCollectorMockRecorder) WatermarkInsertsInc() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatermarkInsertsInc", reflect.TypeOf((*MockMetricsCollector)(nil).WatermarkInsertsInc))
}

// WatermarkRetriesInc mocks base method.
func (m *MockMetricsCollector) WatermarkRetriesInc() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "WatermarkRetriesInc")
}

// WatermarkRetriesInc indicates an expected call of WatermarkRetriesInc.
func (mr *MockMetricsCollectorMockRecorder) WatermarkRetriesInc() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatermarkRetriesInc", reflect.TypeOf((*MockMetricsCollector)(nil).WatermarkRetriesInc))
}