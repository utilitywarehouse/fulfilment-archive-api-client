// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/utilitywarehouse/finance-fulfilment-archive-api-cli/internal/ffaac (interfaces: FilesFinder)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockFilesFinder is a mock of FilesFinder interface
type MockFilesFinder struct {
	ctrl     *gomock.Controller
	recorder *MockFilesFinderMockRecorder
}

// MockFilesFinderMockRecorder is the mock recorder for MockFilesFinder
type MockFilesFinderMockRecorder struct {
	mock *MockFilesFinder
}

// NewMockFilesFinder creates a new mock instance
func NewMockFilesFinder(ctrl *gomock.Controller) *MockFilesFinder {
	mock := &MockFilesFinder{ctrl: ctrl}
	mock.recorder = &MockFilesFinderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockFilesFinder) EXPECT() *MockFilesFinderMockRecorder {
	return m.recorder
}

// Run mocks base method
func (m *MockFilesFinder) Run(arg0 context.Context, arg1 chan<- string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Run indicates an expected call of Run
func (mr *MockFilesFinderMockRecorder) Run(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockFilesFinder)(nil).Run), arg0, arg1)
}
