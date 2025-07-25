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
// Source: proposals.go
//
// Generated by this command:
//
//	mockgen -source=proposals.go -destination=proposals_mock.go -package=emitter
//

// Package emitter is a generated GoMock package.
package emitter

import (
	context "context"
	reflect "reflect"
	time "time"

	scheduler "github.com/0xsoniclabs/sonic/gossip/emitter/scheduler"
	inter "github.com/0xsoniclabs/sonic/inter"
	opera "github.com/0xsoniclabs/sonic/opera"
	hash "github.com/Fantom-foundation/lachesis-base/hash"
	idx "github.com/Fantom-foundation/lachesis-base/inter/idx"
	txpool "github.com/ethereum/go-ethereum/core/txpool"
	types "github.com/ethereum/go-ethereum/core/types"
	uint256 "github.com/holiman/uint256"
	gomock "go.uber.org/mock/gomock"
)

// MockproposalTracker is a mock of proposalTracker interface.
type MockproposalTracker struct {
	ctrl     *gomock.Controller
	recorder *MockproposalTrackerMockRecorder
	isgomock struct{}
}

// MockproposalTrackerMockRecorder is the mock recorder for MockproposalTracker.
type MockproposalTrackerMockRecorder struct {
	mock *MockproposalTracker
}

// NewMockproposalTracker creates a new mock instance.
func NewMockproposalTracker(ctrl *gomock.Controller) *MockproposalTracker {
	mock := &MockproposalTracker{ctrl: ctrl}
	mock.recorder = &MockproposalTrackerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockproposalTracker) EXPECT() *MockproposalTrackerMockRecorder {
	return m.recorder
}

// IsPending mocks base method.
func (m *MockproposalTracker) IsPending(frame idx.Frame, block idx.Block) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsPending", frame, block)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsPending indicates an expected call of IsPending.
func (mr *MockproposalTrackerMockRecorder) IsPending(frame, block any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsPending", reflect.TypeOf((*MockproposalTracker)(nil).IsPending), frame, block)
}

// MockworldReader is a mock of worldReader interface.
type MockworldReader struct {
	ctrl     *gomock.Controller
	recorder *MockworldReaderMockRecorder
	isgomock struct{}
}

// MockworldReaderMockRecorder is the mock recorder for MockworldReader.
type MockworldReaderMockRecorder struct {
	mock *MockworldReader
}

// NewMockworldReader creates a new mock instance.
func NewMockworldReader(ctrl *gomock.Controller) *MockworldReader {
	mock := &MockworldReader{ctrl: ctrl}
	mock.recorder = &MockworldReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockworldReader) EXPECT() *MockworldReaderMockRecorder {
	return m.recorder
}

// GetEventPayload mocks base method.
func (m *MockworldReader) GetEventPayload(arg0 hash.Event) inter.Payload {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEventPayload", arg0)
	ret0, _ := ret[0].(inter.Payload)
	return ret0
}

// GetEventPayload indicates an expected call of GetEventPayload.
func (mr *MockworldReaderMockRecorder) GetEventPayload(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEventPayload", reflect.TypeOf((*MockworldReader)(nil).GetEventPayload), arg0)
}

// GetLatestBlock mocks base method.
func (m *MockworldReader) GetLatestBlock() *inter.Block {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLatestBlock")
	ret0, _ := ret[0].(*inter.Block)
	return ret0
}

// GetLatestBlock indicates an expected call of GetLatestBlock.
func (mr *MockworldReaderMockRecorder) GetLatestBlock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLatestBlock", reflect.TypeOf((*MockworldReader)(nil).GetLatestBlock))
}

// GetRules mocks base method.
func (m *MockworldReader) GetRules() opera.Rules {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRules")
	ret0, _ := ret[0].(opera.Rules)
	return ret0
}

// GetRules indicates an expected call of GetRules.
func (mr *MockworldReaderMockRecorder) GetRules() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRules", reflect.TypeOf((*MockworldReader)(nil).GetRules))
}

// MocktxScheduler is a mock of txScheduler interface.
type MocktxScheduler struct {
	ctrl     *gomock.Controller
	recorder *MocktxSchedulerMockRecorder
	isgomock struct{}
}

// MocktxSchedulerMockRecorder is the mock recorder for MocktxScheduler.
type MocktxSchedulerMockRecorder struct {
	mock *MocktxScheduler
}

// NewMocktxScheduler creates a new mock instance.
func NewMocktxScheduler(ctrl *gomock.Controller) *MocktxScheduler {
	mock := &MocktxScheduler{ctrl: ctrl}
	mock.recorder = &MocktxSchedulerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MocktxScheduler) EXPECT() *MocktxSchedulerMockRecorder {
	return m.recorder
}

// Schedule mocks base method.
func (m *MocktxScheduler) Schedule(arg0 context.Context, arg1 *scheduler.BlockInfo, arg2 scheduler.PrioritizedTransactions, arg3 scheduler.Limits) []*types.Transaction {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Schedule", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([]*types.Transaction)
	return ret0
}

// Schedule indicates an expected call of Schedule.
func (mr *MocktxSchedulerMockRecorder) Schedule(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Schedule", reflect.TypeOf((*MocktxScheduler)(nil).Schedule), arg0, arg1, arg2, arg3)
}

// MocktimerMetric is a mock of timerMetric interface.
type MocktimerMetric struct {
	ctrl     *gomock.Controller
	recorder *MocktimerMetricMockRecorder
	isgomock struct{}
}

// MocktimerMetricMockRecorder is the mock recorder for MocktimerMetric.
type MocktimerMetricMockRecorder struct {
	mock *MocktimerMetric
}

// NewMocktimerMetric creates a new mock instance.
func NewMocktimerMetric(ctrl *gomock.Controller) *MocktimerMetric {
	mock := &MocktimerMetric{ctrl: ctrl}
	mock.recorder = &MocktimerMetricMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MocktimerMetric) EXPECT() *MocktimerMetricMockRecorder {
	return m.recorder
}

// Update mocks base method.
func (m *MocktimerMetric) Update(arg0 time.Duration) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Update", arg0)
}

// Update indicates an expected call of Update.
func (mr *MocktimerMetricMockRecorder) Update(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MocktimerMetric)(nil).Update), arg0)
}

// MockcounterMetric is a mock of counterMetric interface.
type MockcounterMetric struct {
	ctrl     *gomock.Controller
	recorder *MockcounterMetricMockRecorder
	isgomock struct{}
}

// MockcounterMetricMockRecorder is the mock recorder for MockcounterMetric.
type MockcounterMetricMockRecorder struct {
	mock *MockcounterMetric
}

// NewMockcounterMetric creates a new mock instance.
func NewMockcounterMetric(ctrl *gomock.Controller) *MockcounterMetric {
	mock := &MockcounterMetric{ctrl: ctrl}
	mock.recorder = &MockcounterMetricMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockcounterMetric) EXPECT() *MockcounterMetricMockRecorder {
	return m.recorder
}

// Inc mocks base method.
func (m *MockcounterMetric) Inc(arg0 int64) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Inc", arg0)
}

// Inc indicates an expected call of Inc.
func (mr *MockcounterMetricMockRecorder) Inc(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Inc", reflect.TypeOf((*MockcounterMetric)(nil).Inc), arg0)
}

// MocktransactionIndex is a mock of transactionIndex interface.
type MocktransactionIndex struct {
	ctrl     *gomock.Controller
	recorder *MocktransactionIndexMockRecorder
	isgomock struct{}
}

// MocktransactionIndexMockRecorder is the mock recorder for MocktransactionIndex.
type MocktransactionIndexMockRecorder struct {
	mock *MocktransactionIndex
}

// NewMocktransactionIndex creates a new mock instance.
func NewMocktransactionIndex(ctrl *gomock.Controller) *MocktransactionIndex {
	mock := &MocktransactionIndex{ctrl: ctrl}
	mock.recorder = &MocktransactionIndexMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MocktransactionIndex) EXPECT() *MocktransactionIndexMockRecorder {
	return m.recorder
}

// Peek mocks base method.
func (m *MocktransactionIndex) Peek() (*txpool.LazyTransaction, *uint256.Int) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Peek")
	ret0, _ := ret[0].(*txpool.LazyTransaction)
	ret1, _ := ret[1].(*uint256.Int)
	return ret0, ret1
}

// Peek indicates an expected call of Peek.
func (mr *MocktransactionIndexMockRecorder) Peek() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Peek", reflect.TypeOf((*MocktransactionIndex)(nil).Peek))
}

// Pop mocks base method.
func (m *MocktransactionIndex) Pop() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Pop")
}

// Pop indicates an expected call of Pop.
func (mr *MocktransactionIndexMockRecorder) Pop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Pop", reflect.TypeOf((*MocktransactionIndex)(nil).Pop))
}

// Shift mocks base method.
func (m *MocktransactionIndex) Shift() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Shift")
}

// Shift indicates an expected call of Shift.
func (mr *MocktransactionIndexMockRecorder) Shift() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Shift", reflect.TypeOf((*MocktransactionIndex)(nil).Shift))
}
