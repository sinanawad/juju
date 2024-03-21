// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/internal/servicefactory (interfaces: ServiceFactoryGetter,ServiceFactory)
//
// Generated by this command:
//
//	mockgen -package migration_test -destination servicefactory_mock_test.go github.com/juju/juju/internal/servicefactory ServiceFactoryGetter,ServiceFactory
//

// Package migration_test is a generated GoMock package.
package migration_test

import (
	reflect "reflect"

	service "github.com/juju/juju/domain/annotation/service"
	service0 "github.com/juju/juju/domain/application/service"
	service1 "github.com/juju/juju/domain/autocert/service"
	service2 "github.com/juju/juju/domain/blockdevice/service"
	service3 "github.com/juju/juju/domain/cloud/service"
	service4 "github.com/juju/juju/domain/controllerconfig/service"
	service5 "github.com/juju/juju/domain/controllernode/service"
	service6 "github.com/juju/juju/domain/credential/service"
	service7 "github.com/juju/juju/domain/externalcontroller/service"
	service8 "github.com/juju/juju/domain/flag/service"
	service9 "github.com/juju/juju/domain/machine/service"
	service10 "github.com/juju/juju/domain/model/service"
	service11 "github.com/juju/juju/domain/modelconfig/service"
	service12 "github.com/juju/juju/domain/modeldefaults/service"
	service13 "github.com/juju/juju/domain/network/service"
	service14 "github.com/juju/juju/domain/objectstore/service"
	service15 "github.com/juju/juju/domain/storage/service"
	service16 "github.com/juju/juju/domain/unit/service"
	service17 "github.com/juju/juju/domain/upgrade/service"
	service18 "github.com/juju/juju/domain/user/service"
	servicefactory "github.com/juju/juju/internal/servicefactory"
	storage "github.com/juju/juju/internal/storage"
	gomock "go.uber.org/mock/gomock"
)

// MockServiceFactoryGetter is a mock of ServiceFactoryGetter interface.
type MockServiceFactoryGetter struct {
	ctrl     *gomock.Controller
	recorder *MockServiceFactoryGetterMockRecorder
}

// MockServiceFactoryGetterMockRecorder is the mock recorder for MockServiceFactoryGetter.
type MockServiceFactoryGetterMockRecorder struct {
	mock *MockServiceFactoryGetter
}

// NewMockServiceFactoryGetter creates a new mock instance.
func NewMockServiceFactoryGetter(ctrl *gomock.Controller) *MockServiceFactoryGetter {
	mock := &MockServiceFactoryGetter{ctrl: ctrl}
	mock.recorder = &MockServiceFactoryGetterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockServiceFactoryGetter) EXPECT() *MockServiceFactoryGetterMockRecorder {
	return m.recorder
}

// FactoryForModel mocks base method.
func (m *MockServiceFactoryGetter) FactoryForModel(arg0 string) servicefactory.ServiceFactory {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FactoryForModel", arg0)
	ret0, _ := ret[0].(servicefactory.ServiceFactory)
	return ret0
}

// FactoryForModel indicates an expected call of FactoryForModel.
func (mr *MockServiceFactoryGetterMockRecorder) FactoryForModel(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FactoryForModel", reflect.TypeOf((*MockServiceFactoryGetter)(nil).FactoryForModel), arg0)
}

// MockServiceFactory is a mock of ServiceFactory interface.
type MockServiceFactory struct {
	ctrl     *gomock.Controller
	recorder *MockServiceFactoryMockRecorder
}

// MockServiceFactoryMockRecorder is the mock recorder for MockServiceFactory.
type MockServiceFactoryMockRecorder struct {
	mock *MockServiceFactory
}

// NewMockServiceFactory creates a new mock instance.
func NewMockServiceFactory(ctrl *gomock.Controller) *MockServiceFactory {
	mock := &MockServiceFactory{ctrl: ctrl}
	mock.recorder = &MockServiceFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockServiceFactory) EXPECT() *MockServiceFactoryMockRecorder {
	return m.recorder
}

// AgentObjectStore mocks base method.
func (m *MockServiceFactory) AgentObjectStore() *service14.WatchableService {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AgentObjectStore")
	ret0, _ := ret[0].(*service14.WatchableService)
	return ret0
}

// AgentObjectStore indicates an expected call of AgentObjectStore.
func (mr *MockServiceFactoryMockRecorder) AgentObjectStore() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AgentObjectStore", reflect.TypeOf((*MockServiceFactory)(nil).AgentObjectStore))
}

// Annotation mocks base method.
func (m *MockServiceFactory) Annotation() *service.Service {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Annotation")
	ret0, _ := ret[0].(*service.Service)
	return ret0
}

// Annotation indicates an expected call of Annotation.
func (mr *MockServiceFactoryMockRecorder) Annotation() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Annotation", reflect.TypeOf((*MockServiceFactory)(nil).Annotation))
}

// Application mocks base method.
func (m *MockServiceFactory) Application(arg0 storage.ProviderRegistry) *service0.Service {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Application", arg0)
	ret0, _ := ret[0].(*service0.Service)
	return ret0
}

// Application indicates an expected call of Application.
func (mr *MockServiceFactoryMockRecorder) Application(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Application", reflect.TypeOf((*MockServiceFactory)(nil).Application), arg0)
}

// AutocertCache mocks base method.
func (m *MockServiceFactory) AutocertCache() *service1.Service {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AutocertCache")
	ret0, _ := ret[0].(*service1.Service)
	return ret0
}

// AutocertCache indicates an expected call of AutocertCache.
func (mr *MockServiceFactoryMockRecorder) AutocertCache() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AutocertCache", reflect.TypeOf((*MockServiceFactory)(nil).AutocertCache))
}

// BlockDevice mocks base method.
func (m *MockServiceFactory) BlockDevice() *service2.WatchableService {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BlockDevice")
	ret0, _ := ret[0].(*service2.WatchableService)
	return ret0
}

// BlockDevice indicates an expected call of BlockDevice.
func (mr *MockServiceFactoryMockRecorder) BlockDevice() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BlockDevice", reflect.TypeOf((*MockServiceFactory)(nil).BlockDevice))
}

// Cloud mocks base method.
func (m *MockServiceFactory) Cloud() *service3.WatchableService {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Cloud")
	ret0, _ := ret[0].(*service3.WatchableService)
	return ret0
}

// Cloud indicates an expected call of Cloud.
func (mr *MockServiceFactoryMockRecorder) Cloud() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Cloud", reflect.TypeOf((*MockServiceFactory)(nil).Cloud))
}

// Config mocks base method.
func (m *MockServiceFactory) Config(arg0 service11.ModelDefaultsProvider) *service11.WatchableService {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Config", arg0)
	ret0, _ := ret[0].(*service11.WatchableService)
	return ret0
}

// Config indicates an expected call of Config.
func (mr *MockServiceFactoryMockRecorder) Config(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Config", reflect.TypeOf((*MockServiceFactory)(nil).Config), arg0)
}

// ControllerConfig mocks base method.
func (m *MockServiceFactory) ControllerConfig() *service4.WatchableService {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ControllerConfig")
	ret0, _ := ret[0].(*service4.WatchableService)
	return ret0
}

// ControllerConfig indicates an expected call of ControllerConfig.
func (mr *MockServiceFactoryMockRecorder) ControllerConfig() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ControllerConfig", reflect.TypeOf((*MockServiceFactory)(nil).ControllerConfig))
}

// ControllerNode mocks base method.
func (m *MockServiceFactory) ControllerNode() *service5.Service {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ControllerNode")
	ret0, _ := ret[0].(*service5.Service)
	return ret0
}

// ControllerNode indicates an expected call of ControllerNode.
func (mr *MockServiceFactoryMockRecorder) ControllerNode() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ControllerNode", reflect.TypeOf((*MockServiceFactory)(nil).ControllerNode))
}

// Credential mocks base method.
func (m *MockServiceFactory) Credential() *service6.WatchableService {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Credential")
	ret0, _ := ret[0].(*service6.WatchableService)
	return ret0
}

// Credential indicates an expected call of Credential.
func (mr *MockServiceFactoryMockRecorder) Credential() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Credential", reflect.TypeOf((*MockServiceFactory)(nil).Credential))
}

// ExternalController mocks base method.
func (m *MockServiceFactory) ExternalController() *service7.WatchableService {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExternalController")
	ret0, _ := ret[0].(*service7.WatchableService)
	return ret0
}

// ExternalController indicates an expected call of ExternalController.
func (mr *MockServiceFactoryMockRecorder) ExternalController() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExternalController", reflect.TypeOf((*MockServiceFactory)(nil).ExternalController))
}

// Flag mocks base method.
func (m *MockServiceFactory) Flag() *service8.Service {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Flag")
	ret0, _ := ret[0].(*service8.Service)
	return ret0
}

// Flag indicates an expected call of Flag.
func (mr *MockServiceFactoryMockRecorder) Flag() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Flag", reflect.TypeOf((*MockServiceFactory)(nil).Flag))
}

// Machine mocks base method.
func (m *MockServiceFactory) Machine() *service9.Service {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Machine")
	ret0, _ := ret[0].(*service9.Service)
	return ret0
}

// Machine indicates an expected call of Machine.
func (mr *MockServiceFactoryMockRecorder) Machine() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Machine", reflect.TypeOf((*MockServiceFactory)(nil).Machine))
}

// Model mocks base method.
func (m *MockServiceFactory) Model() *service10.Service {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Model")
	ret0, _ := ret[0].(*service10.Service)
	return ret0
}

// Model indicates an expected call of Model.
func (mr *MockServiceFactoryMockRecorder) Model() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Model", reflect.TypeOf((*MockServiceFactory)(nil).Model))
}

// ModelDefaults mocks base method.
func (m *MockServiceFactory) ModelDefaults() *service12.Service {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelDefaults")
	ret0, _ := ret[0].(*service12.Service)
	return ret0
}

// ModelDefaults indicates an expected call of ModelDefaults.
func (mr *MockServiceFactoryMockRecorder) ModelDefaults() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModelDefaults", reflect.TypeOf((*MockServiceFactory)(nil).ModelDefaults))
}

// ObjectStore mocks base method.
func (m *MockServiceFactory) ObjectStore() *service14.WatchableService {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ObjectStore")
	ret0, _ := ret[0].(*service14.WatchableService)
	return ret0
}

// ObjectStore indicates an expected call of ObjectStore.
func (mr *MockServiceFactoryMockRecorder) ObjectStore() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ObjectStore", reflect.TypeOf((*MockServiceFactory)(nil).ObjectStore))
}

// Space mocks base method.
func (m *MockServiceFactory) Space() *service13.SpaceService {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Space")
	ret0, _ := ret[0].(*service13.SpaceService)
	return ret0
}

// Space indicates an expected call of Space.
func (mr *MockServiceFactoryMockRecorder) Space() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Space", reflect.TypeOf((*MockServiceFactory)(nil).Space))
}

// Storage mocks base method.
func (m *MockServiceFactory) Storage(arg0 storage.ProviderRegistry) *service15.Service {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Storage", arg0)
	ret0, _ := ret[0].(*service15.Service)
	return ret0
}

// Storage indicates an expected call of Storage.
func (mr *MockServiceFactoryMockRecorder) Storage(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Storage", reflect.TypeOf((*MockServiceFactory)(nil).Storage), arg0)
}

// Unit mocks base method.
func (m *MockServiceFactory) Unit() *service16.Service {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Unit")
	ret0, _ := ret[0].(*service16.Service)
	return ret0
}

// Unit indicates an expected call of Unit.
func (mr *MockServiceFactoryMockRecorder) Unit() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Unit", reflect.TypeOf((*MockServiceFactory)(nil).Unit))
}

// Upgrade mocks base method.
func (m *MockServiceFactory) Upgrade() *service17.WatchableService {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Upgrade")
	ret0, _ := ret[0].(*service17.WatchableService)
	return ret0
}

// Upgrade indicates an expected call of Upgrade.
func (mr *MockServiceFactoryMockRecorder) Upgrade() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Upgrade", reflect.TypeOf((*MockServiceFactory)(nil).Upgrade))
}

// User mocks base method.
func (m *MockServiceFactory) User() *service18.Service {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "User")
	ret0, _ := ret[0].(*service18.Service)
	return ret0
}

// User indicates an expected call of User.
func (mr *MockServiceFactoryMockRecorder) User() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "User", reflect.TypeOf((*MockServiceFactory)(nil).User))
}