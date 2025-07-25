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
// Source: c_block_callbacks.go
//
// Generated by this command:
//
//	mockgen -source=c_block_callbacks.go -package=gossip -destination=c_block_callbacks_mock.go
//

// Package gossip is a generated GoMock package.
package gossip

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockmetricCounter is a mock of metricCounter interface.
type MockmetricCounter struct {
	ctrl     *gomock.Controller
	recorder *MockmetricCounterMockRecorder
	isgomock struct{}
}

// MockmetricCounterMockRecorder is the mock recorder for MockmetricCounter.
type MockmetricCounterMockRecorder struct {
	mock *MockmetricCounter
}

// NewMockmetricCounter creates a new mock instance.
func NewMockmetricCounter(ctrl *gomock.Controller) *MockmetricCounter {
	mock := &MockmetricCounter{ctrl: ctrl}
	mock.recorder = &MockmetricCounterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockmetricCounter) EXPECT() *MockmetricCounterMockRecorder {
	return m.recorder
}

// Mark mocks base method.
func (m *MockmetricCounter) Mark(arg0 int64) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Mark", arg0)
}

// Mark indicates an expected call of Mark.
func (mr *MockmetricCounterMockRecorder) Mark(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Mark", reflect.TypeOf((*MockmetricCounter)(nil).Mark), arg0)
}
