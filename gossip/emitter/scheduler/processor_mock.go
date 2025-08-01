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
// Source: processor.go
//
// Generated by this command:
//
//	mockgen -source=processor.go -destination=processor_mock.go -package=scheduler
//

// Package scheduler is a generated GoMock package.
package scheduler

import (
	reflect "reflect"

	evmcore "github.com/0xsoniclabs/sonic/evmcore"
	state "github.com/0xsoniclabs/sonic/inter/state"
	opera "github.com/0xsoniclabs/sonic/opera"
	idx "github.com/Fantom-foundation/lachesis-base/inter/idx"
	common "github.com/ethereum/go-ethereum/common"
	types "github.com/ethereum/go-ethereum/core/types"
	params "github.com/ethereum/go-ethereum/params"
	gomock "go.uber.org/mock/gomock"
)

// MockprocessorFactory is a mock of processorFactory interface.
type MockprocessorFactory struct {
	ctrl     *gomock.Controller
	recorder *MockprocessorFactoryMockRecorder
	isgomock struct{}
}

// MockprocessorFactoryMockRecorder is the mock recorder for MockprocessorFactory.
type MockprocessorFactoryMockRecorder struct {
	mock *MockprocessorFactory
}

// NewMockprocessorFactory creates a new mock instance.
func NewMockprocessorFactory(ctrl *gomock.Controller) *MockprocessorFactory {
	mock := &MockprocessorFactory{ctrl: ctrl}
	mock.recorder = &MockprocessorFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockprocessorFactory) EXPECT() *MockprocessorFactoryMockRecorder {
	return m.recorder
}

// beginBlock mocks base method.
func (m *MockprocessorFactory) beginBlock(arg0 *evmcore.EvmBlock) processor {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "beginBlock", arg0)
	ret0, _ := ret[0].(processor)
	return ret0
}

// beginBlock indicates an expected call of beginBlock.
func (mr *MockprocessorFactoryMockRecorder) beginBlock(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "beginBlock", reflect.TypeOf((*MockprocessorFactory)(nil).beginBlock), arg0)
}

// Mockprocessor is a mock of processor interface.
type Mockprocessor struct {
	ctrl     *gomock.Controller
	recorder *MockprocessorMockRecorder
	isgomock struct{}
}

// MockprocessorMockRecorder is the mock recorder for Mockprocessor.
type MockprocessorMockRecorder struct {
	mock *Mockprocessor
}

// NewMockprocessor creates a new mock instance.
func NewMockprocessor(ctrl *gomock.Controller) *Mockprocessor {
	mock := &Mockprocessor{ctrl: ctrl}
	mock.recorder = &MockprocessorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mockprocessor) EXPECT() *MockprocessorMockRecorder {
	return m.recorder
}

// release mocks base method.
func (m *Mockprocessor) release() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "release")
}

// release indicates an expected call of release.
func (mr *MockprocessorMockRecorder) release() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "release", reflect.TypeOf((*Mockprocessor)(nil).release))
}

// run mocks base method.
func (m *Mockprocessor) run(tx *types.Transaction) (bool, uint64) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "run", tx)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(uint64)
	return ret0, ret1
}

// run indicates an expected call of run.
func (mr *MockprocessorMockRecorder) run(tx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "run", reflect.TypeOf((*Mockprocessor)(nil).run), tx)
}

// MockChain is a mock of Chain interface.
type MockChain struct {
	ctrl     *gomock.Controller
	recorder *MockChainMockRecorder
	isgomock struct{}
}

// MockChainMockRecorder is the mock recorder for MockChain.
type MockChainMockRecorder struct {
	mock *MockChain
}

// NewMockChain creates a new mock instance.
func NewMockChain(ctrl *gomock.Controller) *MockChain {
	mock := &MockChain{ctrl: ctrl}
	mock.recorder = &MockChainMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChain) EXPECT() *MockChainMockRecorder {
	return m.recorder
}

// GetCurrentNetworkRules mocks base method.
func (m *MockChain) GetCurrentNetworkRules() opera.Rules {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCurrentNetworkRules")
	ret0, _ := ret[0].(opera.Rules)
	return ret0
}

// GetCurrentNetworkRules indicates an expected call of GetCurrentNetworkRules.
func (mr *MockChainMockRecorder) GetCurrentNetworkRules() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCurrentNetworkRules", reflect.TypeOf((*MockChain)(nil).GetCurrentNetworkRules))
}

// GetEvmChainConfig mocks base method.
func (m *MockChain) GetEvmChainConfig(blockHeight idx.Block) *params.ChainConfig {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEvmChainConfig", blockHeight)
	ret0, _ := ret[0].(*params.ChainConfig)
	return ret0
}

// GetEvmChainConfig indicates an expected call of GetEvmChainConfig.
func (mr *MockChainMockRecorder) GetEvmChainConfig(blockHeight any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEvmChainConfig", reflect.TypeOf((*MockChain)(nil).GetEvmChainConfig), blockHeight)
}

// GetHeader mocks base method.
func (m *MockChain) GetHeader(arg0 common.Hash, arg1 uint64) *evmcore.EvmHeader {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHeader", arg0, arg1)
	ret0, _ := ret[0].(*evmcore.EvmHeader)
	return ret0
}

// GetHeader indicates an expected call of GetHeader.
func (mr *MockChainMockRecorder) GetHeader(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHeader", reflect.TypeOf((*MockChain)(nil).GetHeader), arg0, arg1)
}

// StateDB mocks base method.
func (m *MockChain) StateDB() state.StateDB {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StateDB")
	ret0, _ := ret[0].(state.StateDB)
	return ret0
}

// StateDB indicates an expected call of StateDB.
func (mr *MockChainMockRecorder) StateDB() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StateDB", reflect.TypeOf((*MockChain)(nil).StateDB))
}

// MockevmProcessorRunner is a mock of evmProcessorRunner interface.
type MockevmProcessorRunner struct {
	ctrl     *gomock.Controller
	recorder *MockevmProcessorRunnerMockRecorder
	isgomock struct{}
}

// MockevmProcessorRunnerMockRecorder is the mock recorder for MockevmProcessorRunner.
type MockevmProcessorRunnerMockRecorder struct {
	mock *MockevmProcessorRunner
}

// NewMockevmProcessorRunner creates a new mock instance.
func NewMockevmProcessorRunner(ctrl *gomock.Controller) *MockevmProcessorRunner {
	mock := &MockevmProcessorRunner{ctrl: ctrl}
	mock.recorder = &MockevmProcessorRunnerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockevmProcessorRunner) EXPECT() *MockevmProcessorRunnerMockRecorder {
	return m.recorder
}

// Run mocks base method.
func (m *MockevmProcessorRunner) Run(index int, tx *types.Transaction) (*types.Receipt, bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", index, tx)
	ret0, _ := ret[0].(*types.Receipt)
	ret1, _ := ret[1].(bool)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Run indicates an expected call of Run.
func (mr *MockevmProcessorRunnerMockRecorder) Run(index, tx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockevmProcessorRunner)(nil).Run), index, tx)
}
