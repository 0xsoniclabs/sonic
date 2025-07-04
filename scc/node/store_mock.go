// Copyright 2025 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

// Code generated by MockGen. DO NOT EDIT.
// Source: store.go
//
// Generated by this command:
//
//	mockgen -source=store.go -destination=store_mock.go -package=node
//

// Package node is a generated GoMock package.
package node

import (
	reflect "reflect"

	scc "github.com/0xsoniclabs/sonic/scc"
	cert "github.com/0xsoniclabs/sonic/scc/cert"
	idx "github.com/Fantom-foundation/lachesis-base/inter/idx"
	gomock "go.uber.org/mock/gomock"
)

// MockStore is a mock of Store interface.
type MockStore struct {
	ctrl     *gomock.Controller
	recorder *MockStoreMockRecorder
	isgomock struct{}
}

// MockStoreMockRecorder is the mock recorder for MockStore.
type MockStoreMockRecorder struct {
	mock *MockStore
}

// NewMockStore creates a new mock instance.
func NewMockStore(ctrl *gomock.Controller) *MockStore {
	mock := &MockStore{ctrl: ctrl}
	mock.recorder = &MockStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStore) EXPECT() *MockStoreMockRecorder {
	return m.recorder
}

// GetBlockCertificate mocks base method.
func (m *MockStore) GetBlockCertificate(arg0 idx.Block) (cert.BlockCertificate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockCertificate", arg0)
	ret0, _ := ret[0].(cert.BlockCertificate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlockCertificate indicates an expected call of GetBlockCertificate.
func (mr *MockStoreMockRecorder) GetBlockCertificate(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockCertificate", reflect.TypeOf((*MockStore)(nil).GetBlockCertificate), arg0)
}

// GetCommitteeCertificate mocks base method.
func (m *MockStore) GetCommitteeCertificate(arg0 scc.Period) (cert.CommitteeCertificate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCommitteeCertificate", arg0)
	ret0, _ := ret[0].(cert.CommitteeCertificate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCommitteeCertificate indicates an expected call of GetCommitteeCertificate.
func (mr *MockStoreMockRecorder) GetCommitteeCertificate(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCommitteeCertificate", reflect.TypeOf((*MockStore)(nil).GetCommitteeCertificate), arg0)
}

// UpdateBlockCertificate mocks base method.
func (m *MockStore) UpdateBlockCertificate(arg0 cert.BlockCertificate) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateBlockCertificate", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateBlockCertificate indicates an expected call of UpdateBlockCertificate.
func (mr *MockStoreMockRecorder) UpdateBlockCertificate(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateBlockCertificate", reflect.TypeOf((*MockStore)(nil).UpdateBlockCertificate), arg0)
}

// UpdateCommitteeCertificate mocks base method.
func (m *MockStore) UpdateCommitteeCertificate(arg0 cert.CommitteeCertificate) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateCommitteeCertificate", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateCommitteeCertificate indicates an expected call of UpdateCommitteeCertificate.
func (mr *MockStoreMockRecorder) UpdateCommitteeCertificate(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateCommitteeCertificate", reflect.TypeOf((*MockStore)(nil).UpdateCommitteeCertificate), arg0)
}
