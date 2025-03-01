// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/domain/application/service (interfaces: CharmStore,WatcherFactory)
//
// Generated by this command:
//
//	mockgen -typed -package service -destination charm_mock_test.go github.com/juju/juju/domain/application/service CharmStore,WatcherFactory
//

// Package service is a generated GoMock package.
package service

import (
	context "context"
	io "io"
	reflect "reflect"

	changestream "github.com/juju/juju/core/changestream"
	watcher "github.com/juju/juju/core/watcher"
	eventsource "github.com/juju/juju/core/watcher/eventsource"
	store "github.com/juju/juju/domain/application/charm/store"
	gomock "go.uber.org/mock/gomock"
)

// MockCharmStore is a mock of CharmStore interface.
type MockCharmStore struct {
	ctrl     *gomock.Controller
	recorder *MockCharmStoreMockRecorder
}

// MockCharmStoreMockRecorder is the mock recorder for MockCharmStore.
type MockCharmStoreMockRecorder struct {
	mock *MockCharmStore
}

// NewMockCharmStore creates a new mock instance.
func NewMockCharmStore(ctrl *gomock.Controller) *MockCharmStore {
	mock := &MockCharmStore{ctrl: ctrl}
	mock.recorder = &MockCharmStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCharmStore) EXPECT() *MockCharmStoreMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockCharmStore) Get(arg0 context.Context, arg1 string) (io.ReadCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(io.ReadCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockCharmStoreMockRecorder) Get(arg0, arg1 any) *MockCharmStoreGetCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockCharmStore)(nil).Get), arg0, arg1)
	return &MockCharmStoreGetCall{Call: call}
}

// MockCharmStoreGetCall wrap *gomock.Call
type MockCharmStoreGetCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockCharmStoreGetCall) Return(arg0 io.ReadCloser, arg1 error) *MockCharmStoreGetCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockCharmStoreGetCall) Do(f func(context.Context, string) (io.ReadCloser, error)) *MockCharmStoreGetCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockCharmStoreGetCall) DoAndReturn(f func(context.Context, string) (io.ReadCloser, error)) *MockCharmStoreGetCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// GetBySHA256Prefix mocks base method.
func (m *MockCharmStore) GetBySHA256Prefix(arg0 context.Context, arg1 string) (io.ReadCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBySHA256Prefix", arg0, arg1)
	ret0, _ := ret[0].(io.ReadCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBySHA256Prefix indicates an expected call of GetBySHA256Prefix.
func (mr *MockCharmStoreMockRecorder) GetBySHA256Prefix(arg0, arg1 any) *MockCharmStoreGetBySHA256PrefixCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBySHA256Prefix", reflect.TypeOf((*MockCharmStore)(nil).GetBySHA256Prefix), arg0, arg1)
	return &MockCharmStoreGetBySHA256PrefixCall{Call: call}
}

// MockCharmStoreGetBySHA256PrefixCall wrap *gomock.Call
type MockCharmStoreGetBySHA256PrefixCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockCharmStoreGetBySHA256PrefixCall) Return(arg0 io.ReadCloser, arg1 error) *MockCharmStoreGetBySHA256PrefixCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockCharmStoreGetBySHA256PrefixCall) Do(f func(context.Context, string) (io.ReadCloser, error)) *MockCharmStoreGetBySHA256PrefixCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockCharmStoreGetBySHA256PrefixCall) DoAndReturn(f func(context.Context, string) (io.ReadCloser, error)) *MockCharmStoreGetBySHA256PrefixCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Store mocks base method.
func (m *MockCharmStore) Store(arg0 context.Context, arg1 string, arg2 int64, arg3 string) (store.StoreResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Store", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(store.StoreResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Store indicates an expected call of Store.
func (mr *MockCharmStoreMockRecorder) Store(arg0, arg1, arg2, arg3 any) *MockCharmStoreStoreCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Store", reflect.TypeOf((*MockCharmStore)(nil).Store), arg0, arg1, arg2, arg3)
	return &MockCharmStoreStoreCall{Call: call}
}

// MockCharmStoreStoreCall wrap *gomock.Call
type MockCharmStoreStoreCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockCharmStoreStoreCall) Return(arg0 store.StoreResult, arg1 error) *MockCharmStoreStoreCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockCharmStoreStoreCall) Do(f func(context.Context, string, int64, string) (store.StoreResult, error)) *MockCharmStoreStoreCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockCharmStoreStoreCall) DoAndReturn(f func(context.Context, string, int64, string) (store.StoreResult, error)) *MockCharmStoreStoreCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// StoreFromReader mocks base method.
func (m *MockCharmStore) StoreFromReader(arg0 context.Context, arg1 io.Reader, arg2 string) (store.StoreFromReaderResult, store.Digest, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreFromReader", arg0, arg1, arg2)
	ret0, _ := ret[0].(store.StoreFromReaderResult)
	ret1, _ := ret[1].(store.Digest)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// StoreFromReader indicates an expected call of StoreFromReader.
func (mr *MockCharmStoreMockRecorder) StoreFromReader(arg0, arg1, arg2 any) *MockCharmStoreStoreFromReaderCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreFromReader", reflect.TypeOf((*MockCharmStore)(nil).StoreFromReader), arg0, arg1, arg2)
	return &MockCharmStoreStoreFromReaderCall{Call: call}
}

// MockCharmStoreStoreFromReaderCall wrap *gomock.Call
type MockCharmStoreStoreFromReaderCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockCharmStoreStoreFromReaderCall) Return(arg0 store.StoreFromReaderResult, arg1 store.Digest, arg2 error) *MockCharmStoreStoreFromReaderCall {
	c.Call = c.Call.Return(arg0, arg1, arg2)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockCharmStoreStoreFromReaderCall) Do(f func(context.Context, io.Reader, string) (store.StoreFromReaderResult, store.Digest, error)) *MockCharmStoreStoreFromReaderCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockCharmStoreStoreFromReaderCall) DoAndReturn(f func(context.Context, io.Reader, string) (store.StoreFromReaderResult, store.Digest, error)) *MockCharmStoreStoreFromReaderCall {
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

// NewNamespaceMapperWatcher mocks base method.
func (m *MockWatcherFactory) NewNamespaceMapperWatcher(arg0 string, arg1 changestream.ChangeType, arg2 eventsource.NamespaceQuery, arg3 eventsource.Mapper) (watcher.Watcher[[]string], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewNamespaceMapperWatcher", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(watcher.Watcher[[]string])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewNamespaceMapperWatcher indicates an expected call of NewNamespaceMapperWatcher.
func (mr *MockWatcherFactoryMockRecorder) NewNamespaceMapperWatcher(arg0, arg1, arg2, arg3 any) *MockWatcherFactoryNewNamespaceMapperWatcherCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewNamespaceMapperWatcher", reflect.TypeOf((*MockWatcherFactory)(nil).NewNamespaceMapperWatcher), arg0, arg1, arg2, arg3)
	return &MockWatcherFactoryNewNamespaceMapperWatcherCall{Call: call}
}

// MockWatcherFactoryNewNamespaceMapperWatcherCall wrap *gomock.Call
type MockWatcherFactoryNewNamespaceMapperWatcherCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockWatcherFactoryNewNamespaceMapperWatcherCall) Return(arg0 watcher.Watcher[[]string], arg1 error) *MockWatcherFactoryNewNamespaceMapperWatcherCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockWatcherFactoryNewNamespaceMapperWatcherCall) Do(f func(string, changestream.ChangeType, eventsource.NamespaceQuery, eventsource.Mapper) (watcher.Watcher[[]string], error)) *MockWatcherFactoryNewNamespaceMapperWatcherCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockWatcherFactoryNewNamespaceMapperWatcherCall) DoAndReturn(f func(string, changestream.ChangeType, eventsource.NamespaceQuery, eventsource.Mapper) (watcher.Watcher[[]string], error)) *MockWatcherFactoryNewNamespaceMapperWatcherCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// NewUUIDsWatcher mocks base method.
func (m *MockWatcherFactory) NewUUIDsWatcher(arg0 string, arg1 changestream.ChangeType) (watcher.Watcher[[]string], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewUUIDsWatcher", arg0, arg1)
	ret0, _ := ret[0].(watcher.Watcher[[]string])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewUUIDsWatcher indicates an expected call of NewUUIDsWatcher.
func (mr *MockWatcherFactoryMockRecorder) NewUUIDsWatcher(arg0, arg1 any) *MockWatcherFactoryNewUUIDsWatcherCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewUUIDsWatcher", reflect.TypeOf((*MockWatcherFactory)(nil).NewUUIDsWatcher), arg0, arg1)
	return &MockWatcherFactoryNewUUIDsWatcherCall{Call: call}
}

// MockWatcherFactoryNewUUIDsWatcherCall wrap *gomock.Call
type MockWatcherFactoryNewUUIDsWatcherCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockWatcherFactoryNewUUIDsWatcherCall) Return(arg0 watcher.Watcher[[]string], arg1 error) *MockWatcherFactoryNewUUIDsWatcherCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockWatcherFactoryNewUUIDsWatcherCall) Do(f func(string, changestream.ChangeType) (watcher.Watcher[[]string], error)) *MockWatcherFactoryNewUUIDsWatcherCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockWatcherFactoryNewUUIDsWatcherCall) DoAndReturn(f func(string, changestream.ChangeType) (watcher.Watcher[[]string], error)) *MockWatcherFactoryNewUUIDsWatcherCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// NewValueMapperWatcher mocks base method.
func (m *MockWatcherFactory) NewValueMapperWatcher(arg0, arg1 string, arg2 changestream.ChangeType, arg3 eventsource.Mapper) (watcher.Watcher[struct{}], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewValueMapperWatcher", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(watcher.Watcher[struct{}])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewValueMapperWatcher indicates an expected call of NewValueMapperWatcher.
func (mr *MockWatcherFactoryMockRecorder) NewValueMapperWatcher(arg0, arg1, arg2, arg3 any) *MockWatcherFactoryNewValueMapperWatcherCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewValueMapperWatcher", reflect.TypeOf((*MockWatcherFactory)(nil).NewValueMapperWatcher), arg0, arg1, arg2, arg3)
	return &MockWatcherFactoryNewValueMapperWatcherCall{Call: call}
}

// MockWatcherFactoryNewValueMapperWatcherCall wrap *gomock.Call
type MockWatcherFactoryNewValueMapperWatcherCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockWatcherFactoryNewValueMapperWatcherCall) Return(arg0 watcher.Watcher[struct{}], arg1 error) *MockWatcherFactoryNewValueMapperWatcherCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockWatcherFactoryNewValueMapperWatcherCall) Do(f func(string, string, changestream.ChangeType, eventsource.Mapper) (watcher.Watcher[struct{}], error)) *MockWatcherFactoryNewValueMapperWatcherCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockWatcherFactoryNewValueMapperWatcherCall) DoAndReturn(f func(string, string, changestream.ChangeType, eventsource.Mapper) (watcher.Watcher[struct{}], error)) *MockWatcherFactoryNewValueMapperWatcherCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// NewValueWatcher mocks base method.
func (m *MockWatcherFactory) NewValueWatcher(arg0, arg1 string, arg2 changestream.ChangeType) (watcher.Watcher[struct{}], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewValueWatcher", arg0, arg1, arg2)
	ret0, _ := ret[0].(watcher.Watcher[struct{}])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewValueWatcher indicates an expected call of NewValueWatcher.
func (mr *MockWatcherFactoryMockRecorder) NewValueWatcher(arg0, arg1, arg2 any) *MockWatcherFactoryNewValueWatcherCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewValueWatcher", reflect.TypeOf((*MockWatcherFactory)(nil).NewValueWatcher), arg0, arg1, arg2)
	return &MockWatcherFactoryNewValueWatcherCall{Call: call}
}

// MockWatcherFactoryNewValueWatcherCall wrap *gomock.Call
type MockWatcherFactoryNewValueWatcherCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockWatcherFactoryNewValueWatcherCall) Return(arg0 watcher.Watcher[struct{}], arg1 error) *MockWatcherFactoryNewValueWatcherCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockWatcherFactoryNewValueWatcherCall) Do(f func(string, string, changestream.ChangeType) (watcher.Watcher[struct{}], error)) *MockWatcherFactoryNewValueWatcherCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockWatcherFactoryNewValueWatcherCall) DoAndReturn(f func(string, string, changestream.ChangeType) (watcher.Watcher[struct{}], error)) *MockWatcherFactoryNewValueWatcherCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
