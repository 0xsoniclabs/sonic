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
// Source: event.go
//
// Generated by this command:
//
//	mockgen -source=event.go -destination=event_mock.go -package=inter
//

// Package inter is a generated GoMock package.
package inter

import (
	reflect "reflect"

	hash "github.com/Fantom-foundation/lachesis-base/hash"
	idx "github.com/Fantom-foundation/lachesis-base/inter/idx"
	types "github.com/ethereum/go-ethereum/core/types"
	gomock "go.uber.org/mock/gomock"
)

// MockEventI is a mock of EventI interface.
type MockEventI struct {
	ctrl     *gomock.Controller
	recorder *MockEventIMockRecorder
	isgomock struct{}
}

// MockEventIMockRecorder is the mock recorder for MockEventI.
type MockEventIMockRecorder struct {
	mock *MockEventI
}

// NewMockEventI creates a new mock instance.
func NewMockEventI(ctrl *gomock.Controller) *MockEventI {
	mock := &MockEventI{ctrl: ctrl}
	mock.recorder = &MockEventIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEventI) EXPECT() *MockEventIMockRecorder {
	return m.recorder
}

// AnyBlockVotes mocks base method.
func (m *MockEventI) AnyBlockVotes() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AnyBlockVotes")
	ret0, _ := ret[0].(bool)
	return ret0
}

// AnyBlockVotes indicates an expected call of AnyBlockVotes.
func (mr *MockEventIMockRecorder) AnyBlockVotes() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AnyBlockVotes", reflect.TypeOf((*MockEventI)(nil).AnyBlockVotes))
}

// AnyEpochVote mocks base method.
func (m *MockEventI) AnyEpochVote() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AnyEpochVote")
	ret0, _ := ret[0].(bool)
	return ret0
}

// AnyEpochVote indicates an expected call of AnyEpochVote.
func (mr *MockEventIMockRecorder) AnyEpochVote() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AnyEpochVote", reflect.TypeOf((*MockEventI)(nil).AnyEpochVote))
}

// AnyMisbehaviourProofs mocks base method.
func (m *MockEventI) AnyMisbehaviourProofs() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AnyMisbehaviourProofs")
	ret0, _ := ret[0].(bool)
	return ret0
}

// AnyMisbehaviourProofs indicates an expected call of AnyMisbehaviourProofs.
func (mr *MockEventIMockRecorder) AnyMisbehaviourProofs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AnyMisbehaviourProofs", reflect.TypeOf((*MockEventI)(nil).AnyMisbehaviourProofs))
}

// AnyTxs mocks base method.
func (m *MockEventI) AnyTxs() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AnyTxs")
	ret0, _ := ret[0].(bool)
	return ret0
}

// AnyTxs indicates an expected call of AnyTxs.
func (mr *MockEventIMockRecorder) AnyTxs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AnyTxs", reflect.TypeOf((*MockEventI)(nil).AnyTxs))
}

// CreationTime mocks base method.
func (m *MockEventI) CreationTime() Timestamp {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreationTime")
	ret0, _ := ret[0].(Timestamp)
	return ret0
}

// CreationTime indicates an expected call of CreationTime.
func (mr *MockEventIMockRecorder) CreationTime() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreationTime", reflect.TypeOf((*MockEventI)(nil).CreationTime))
}

// Creator mocks base method.
func (m *MockEventI) Creator() idx.ValidatorID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Creator")
	ret0, _ := ret[0].(idx.ValidatorID)
	return ret0
}

// Creator indicates an expected call of Creator.
func (mr *MockEventIMockRecorder) Creator() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Creator", reflect.TypeOf((*MockEventI)(nil).Creator))
}

// Epoch mocks base method.
func (m *MockEventI) Epoch() idx.Epoch {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Epoch")
	ret0, _ := ret[0].(idx.Epoch)
	return ret0
}

// Epoch indicates an expected call of Epoch.
func (mr *MockEventIMockRecorder) Epoch() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Epoch", reflect.TypeOf((*MockEventI)(nil).Epoch))
}

// Extra mocks base method.
func (m *MockEventI) Extra() []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Extra")
	ret0, _ := ret[0].([]byte)
	return ret0
}

// Extra indicates an expected call of Extra.
func (mr *MockEventIMockRecorder) Extra() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Extra", reflect.TypeOf((*MockEventI)(nil).Extra))
}

// Frame mocks base method.
func (m *MockEventI) Frame() idx.Frame {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Frame")
	ret0, _ := ret[0].(idx.Frame)
	return ret0
}

// Frame indicates an expected call of Frame.
func (mr *MockEventIMockRecorder) Frame() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Frame", reflect.TypeOf((*MockEventI)(nil).Frame))
}

// GasPowerLeft mocks base method.
func (m *MockEventI) GasPowerLeft() GasPowerLeft {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GasPowerLeft")
	ret0, _ := ret[0].(GasPowerLeft)
	return ret0
}

// GasPowerLeft indicates an expected call of GasPowerLeft.
func (mr *MockEventIMockRecorder) GasPowerLeft() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GasPowerLeft", reflect.TypeOf((*MockEventI)(nil).GasPowerLeft))
}

// GasPowerUsed mocks base method.
func (m *MockEventI) GasPowerUsed() uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GasPowerUsed")
	ret0, _ := ret[0].(uint64)
	return ret0
}

// GasPowerUsed indicates an expected call of GasPowerUsed.
func (mr *MockEventIMockRecorder) GasPowerUsed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GasPowerUsed", reflect.TypeOf((*MockEventI)(nil).GasPowerUsed))
}

// HasProposal mocks base method.
func (m *MockEventI) HasProposal() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasProposal")
	ret0, _ := ret[0].(bool)
	return ret0
}

// HasProposal indicates an expected call of HasProposal.
func (mr *MockEventIMockRecorder) HasProposal() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasProposal", reflect.TypeOf((*MockEventI)(nil).HasProposal))
}

// HashToSign mocks base method.
func (m *MockEventI) HashToSign() hash.Hash {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HashToSign")
	ret0, _ := ret[0].(hash.Hash)
	return ret0
}

// HashToSign indicates an expected call of HashToSign.
func (mr *MockEventIMockRecorder) HashToSign() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HashToSign", reflect.TypeOf((*MockEventI)(nil).HashToSign))
}

// ID mocks base method.
func (m *MockEventI) ID() hash.Event {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ID")
	ret0, _ := ret[0].(hash.Event)
	return ret0
}

// ID indicates an expected call of ID.
func (mr *MockEventIMockRecorder) ID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ID", reflect.TypeOf((*MockEventI)(nil).ID))
}

// IsSelfParent mocks base method.
func (m *MockEventI) IsSelfParent(hash hash.Event) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsSelfParent", hash)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsSelfParent indicates an expected call of IsSelfParent.
func (mr *MockEventIMockRecorder) IsSelfParent(hash any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsSelfParent", reflect.TypeOf((*MockEventI)(nil).IsSelfParent), hash)
}

// Lamport mocks base method.
func (m *MockEventI) Lamport() idx.Lamport {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Lamport")
	ret0, _ := ret[0].(idx.Lamport)
	return ret0
}

// Lamport indicates an expected call of Lamport.
func (mr *MockEventIMockRecorder) Lamport() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Lamport", reflect.TypeOf((*MockEventI)(nil).Lamport))
}

// Locator mocks base method.
func (m *MockEventI) Locator() EventLocator {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Locator")
	ret0, _ := ret[0].(EventLocator)
	return ret0
}

// Locator indicates an expected call of Locator.
func (mr *MockEventIMockRecorder) Locator() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Locator", reflect.TypeOf((*MockEventI)(nil).Locator))
}

// MedianTime mocks base method.
func (m *MockEventI) MedianTime() Timestamp {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MedianTime")
	ret0, _ := ret[0].(Timestamp)
	return ret0
}

// MedianTime indicates an expected call of MedianTime.
func (mr *MockEventIMockRecorder) MedianTime() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MedianTime", reflect.TypeOf((*MockEventI)(nil).MedianTime))
}

// NetForkID mocks base method.
func (m *MockEventI) NetForkID() uint16 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NetForkID")
	ret0, _ := ret[0].(uint16)
	return ret0
}

// NetForkID indicates an expected call of NetForkID.
func (mr *MockEventIMockRecorder) NetForkID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NetForkID", reflect.TypeOf((*MockEventI)(nil).NetForkID))
}

// Parents mocks base method.
func (m *MockEventI) Parents() hash.Events {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Parents")
	ret0, _ := ret[0].(hash.Events)
	return ret0
}

// Parents indicates an expected call of Parents.
func (mr *MockEventIMockRecorder) Parents() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Parents", reflect.TypeOf((*MockEventI)(nil).Parents))
}

// PayloadHash mocks base method.
func (m *MockEventI) PayloadHash() hash.Hash {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PayloadHash")
	ret0, _ := ret[0].(hash.Hash)
	return ret0
}

// PayloadHash indicates an expected call of PayloadHash.
func (mr *MockEventIMockRecorder) PayloadHash() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PayloadHash", reflect.TypeOf((*MockEventI)(nil).PayloadHash))
}

// PrevEpochHash mocks base method.
func (m *MockEventI) PrevEpochHash() *hash.Hash {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PrevEpochHash")
	ret0, _ := ret[0].(*hash.Hash)
	return ret0
}

// PrevEpochHash indicates an expected call of PrevEpochHash.
func (mr *MockEventIMockRecorder) PrevEpochHash() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PrevEpochHash", reflect.TypeOf((*MockEventI)(nil).PrevEpochHash))
}

// SelfParent mocks base method.
func (m *MockEventI) SelfParent() *hash.Event {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SelfParent")
	ret0, _ := ret[0].(*hash.Event)
	return ret0
}

// SelfParent indicates an expected call of SelfParent.
func (mr *MockEventIMockRecorder) SelfParent() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SelfParent", reflect.TypeOf((*MockEventI)(nil).SelfParent))
}

// Seq mocks base method.
func (m *MockEventI) Seq() idx.Event {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Seq")
	ret0, _ := ret[0].(idx.Event)
	return ret0
}

// Seq indicates an expected call of Seq.
func (mr *MockEventIMockRecorder) Seq() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Seq", reflect.TypeOf((*MockEventI)(nil).Seq))
}

// Size mocks base method.
func (m *MockEventI) Size() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Size")
	ret0, _ := ret[0].(int)
	return ret0
}

// Size indicates an expected call of Size.
func (mr *MockEventIMockRecorder) Size() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Size", reflect.TypeOf((*MockEventI)(nil).Size))
}

// String mocks base method.
func (m *MockEventI) String() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "String")
	ret0, _ := ret[0].(string)
	return ret0
}

// String indicates an expected call of String.
func (mr *MockEventIMockRecorder) String() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "String", reflect.TypeOf((*MockEventI)(nil).String))
}

// Version mocks base method.
func (m *MockEventI) Version() uint8 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Version")
	ret0, _ := ret[0].(uint8)
	return ret0
}

// Version indicates an expected call of Version.
func (mr *MockEventIMockRecorder) Version() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Version", reflect.TypeOf((*MockEventI)(nil).Version))
}

// MockEventPayloadI is a mock of EventPayloadI interface.
type MockEventPayloadI struct {
	ctrl     *gomock.Controller
	recorder *MockEventPayloadIMockRecorder
	isgomock struct{}
}

// MockEventPayloadIMockRecorder is the mock recorder for MockEventPayloadI.
type MockEventPayloadIMockRecorder struct {
	mock *MockEventPayloadI
}

// NewMockEventPayloadI creates a new mock instance.
func NewMockEventPayloadI(ctrl *gomock.Controller) *MockEventPayloadI {
	mock := &MockEventPayloadI{ctrl: ctrl}
	mock.recorder = &MockEventPayloadIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEventPayloadI) EXPECT() *MockEventPayloadIMockRecorder {
	return m.recorder
}

// AnyBlockVotes mocks base method.
func (m *MockEventPayloadI) AnyBlockVotes() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AnyBlockVotes")
	ret0, _ := ret[0].(bool)
	return ret0
}

// AnyBlockVotes indicates an expected call of AnyBlockVotes.
func (mr *MockEventPayloadIMockRecorder) AnyBlockVotes() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AnyBlockVotes", reflect.TypeOf((*MockEventPayloadI)(nil).AnyBlockVotes))
}

// AnyEpochVote mocks base method.
func (m *MockEventPayloadI) AnyEpochVote() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AnyEpochVote")
	ret0, _ := ret[0].(bool)
	return ret0
}

// AnyEpochVote indicates an expected call of AnyEpochVote.
func (mr *MockEventPayloadIMockRecorder) AnyEpochVote() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AnyEpochVote", reflect.TypeOf((*MockEventPayloadI)(nil).AnyEpochVote))
}

// AnyMisbehaviourProofs mocks base method.
func (m *MockEventPayloadI) AnyMisbehaviourProofs() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AnyMisbehaviourProofs")
	ret0, _ := ret[0].(bool)
	return ret0
}

// AnyMisbehaviourProofs indicates an expected call of AnyMisbehaviourProofs.
func (mr *MockEventPayloadIMockRecorder) AnyMisbehaviourProofs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AnyMisbehaviourProofs", reflect.TypeOf((*MockEventPayloadI)(nil).AnyMisbehaviourProofs))
}

// AnyTxs mocks base method.
func (m *MockEventPayloadI) AnyTxs() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AnyTxs")
	ret0, _ := ret[0].(bool)
	return ret0
}

// AnyTxs indicates an expected call of AnyTxs.
func (mr *MockEventPayloadIMockRecorder) AnyTxs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AnyTxs", reflect.TypeOf((*MockEventPayloadI)(nil).AnyTxs))
}

// BlockVotes mocks base method.
func (m *MockEventPayloadI) BlockVotes() LlrBlockVotes {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BlockVotes")
	ret0, _ := ret[0].(LlrBlockVotes)
	return ret0
}

// BlockVotes indicates an expected call of BlockVotes.
func (mr *MockEventPayloadIMockRecorder) BlockVotes() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BlockVotes", reflect.TypeOf((*MockEventPayloadI)(nil).BlockVotes))
}

// CreationTime mocks base method.
func (m *MockEventPayloadI) CreationTime() Timestamp {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreationTime")
	ret0, _ := ret[0].(Timestamp)
	return ret0
}

// CreationTime indicates an expected call of CreationTime.
func (mr *MockEventPayloadIMockRecorder) CreationTime() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreationTime", reflect.TypeOf((*MockEventPayloadI)(nil).CreationTime))
}

// Creator mocks base method.
func (m *MockEventPayloadI) Creator() idx.ValidatorID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Creator")
	ret0, _ := ret[0].(idx.ValidatorID)
	return ret0
}

// Creator indicates an expected call of Creator.
func (mr *MockEventPayloadIMockRecorder) Creator() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Creator", reflect.TypeOf((*MockEventPayloadI)(nil).Creator))
}

// Epoch mocks base method.
func (m *MockEventPayloadI) Epoch() idx.Epoch {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Epoch")
	ret0, _ := ret[0].(idx.Epoch)
	return ret0
}

// Epoch indicates an expected call of Epoch.
func (mr *MockEventPayloadIMockRecorder) Epoch() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Epoch", reflect.TypeOf((*MockEventPayloadI)(nil).Epoch))
}

// EpochVote mocks base method.
func (m *MockEventPayloadI) EpochVote() LlrEpochVote {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EpochVote")
	ret0, _ := ret[0].(LlrEpochVote)
	return ret0
}

// EpochVote indicates an expected call of EpochVote.
func (mr *MockEventPayloadIMockRecorder) EpochVote() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EpochVote", reflect.TypeOf((*MockEventPayloadI)(nil).EpochVote))
}

// Extra mocks base method.
func (m *MockEventPayloadI) Extra() []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Extra")
	ret0, _ := ret[0].([]byte)
	return ret0
}

// Extra indicates an expected call of Extra.
func (mr *MockEventPayloadIMockRecorder) Extra() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Extra", reflect.TypeOf((*MockEventPayloadI)(nil).Extra))
}

// Frame mocks base method.
func (m *MockEventPayloadI) Frame() idx.Frame {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Frame")
	ret0, _ := ret[0].(idx.Frame)
	return ret0
}

// Frame indicates an expected call of Frame.
func (mr *MockEventPayloadIMockRecorder) Frame() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Frame", reflect.TypeOf((*MockEventPayloadI)(nil).Frame))
}

// GasPowerLeft mocks base method.
func (m *MockEventPayloadI) GasPowerLeft() GasPowerLeft {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GasPowerLeft")
	ret0, _ := ret[0].(GasPowerLeft)
	return ret0
}

// GasPowerLeft indicates an expected call of GasPowerLeft.
func (mr *MockEventPayloadIMockRecorder) GasPowerLeft() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GasPowerLeft", reflect.TypeOf((*MockEventPayloadI)(nil).GasPowerLeft))
}

// GasPowerUsed mocks base method.
func (m *MockEventPayloadI) GasPowerUsed() uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GasPowerUsed")
	ret0, _ := ret[0].(uint64)
	return ret0
}

// GasPowerUsed indicates an expected call of GasPowerUsed.
func (mr *MockEventPayloadIMockRecorder) GasPowerUsed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GasPowerUsed", reflect.TypeOf((*MockEventPayloadI)(nil).GasPowerUsed))
}

// HasProposal mocks base method.
func (m *MockEventPayloadI) HasProposal() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasProposal")
	ret0, _ := ret[0].(bool)
	return ret0
}

// HasProposal indicates an expected call of HasProposal.
func (mr *MockEventPayloadIMockRecorder) HasProposal() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasProposal", reflect.TypeOf((*MockEventPayloadI)(nil).HasProposal))
}

// HashToSign mocks base method.
func (m *MockEventPayloadI) HashToSign() hash.Hash {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HashToSign")
	ret0, _ := ret[0].(hash.Hash)
	return ret0
}

// HashToSign indicates an expected call of HashToSign.
func (mr *MockEventPayloadIMockRecorder) HashToSign() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HashToSign", reflect.TypeOf((*MockEventPayloadI)(nil).HashToSign))
}

// ID mocks base method.
func (m *MockEventPayloadI) ID() hash.Event {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ID")
	ret0, _ := ret[0].(hash.Event)
	return ret0
}

// ID indicates an expected call of ID.
func (mr *MockEventPayloadIMockRecorder) ID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ID", reflect.TypeOf((*MockEventPayloadI)(nil).ID))
}

// IsSelfParent mocks base method.
func (m *MockEventPayloadI) IsSelfParent(hash hash.Event) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsSelfParent", hash)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsSelfParent indicates an expected call of IsSelfParent.
func (mr *MockEventPayloadIMockRecorder) IsSelfParent(hash any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsSelfParent", reflect.TypeOf((*MockEventPayloadI)(nil).IsSelfParent), hash)
}

// Lamport mocks base method.
func (m *MockEventPayloadI) Lamport() idx.Lamport {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Lamport")
	ret0, _ := ret[0].(idx.Lamport)
	return ret0
}

// Lamport indicates an expected call of Lamport.
func (mr *MockEventPayloadIMockRecorder) Lamport() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Lamport", reflect.TypeOf((*MockEventPayloadI)(nil).Lamport))
}

// Locator mocks base method.
func (m *MockEventPayloadI) Locator() EventLocator {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Locator")
	ret0, _ := ret[0].(EventLocator)
	return ret0
}

// Locator indicates an expected call of Locator.
func (mr *MockEventPayloadIMockRecorder) Locator() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Locator", reflect.TypeOf((*MockEventPayloadI)(nil).Locator))
}

// MedianTime mocks base method.
func (m *MockEventPayloadI) MedianTime() Timestamp {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MedianTime")
	ret0, _ := ret[0].(Timestamp)
	return ret0
}

// MedianTime indicates an expected call of MedianTime.
func (mr *MockEventPayloadIMockRecorder) MedianTime() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MedianTime", reflect.TypeOf((*MockEventPayloadI)(nil).MedianTime))
}

// MisbehaviourProofs mocks base method.
func (m *MockEventPayloadI) MisbehaviourProofs() []MisbehaviourProof {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MisbehaviourProofs")
	ret0, _ := ret[0].([]MisbehaviourProof)
	return ret0
}

// MisbehaviourProofs indicates an expected call of MisbehaviourProofs.
func (mr *MockEventPayloadIMockRecorder) MisbehaviourProofs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MisbehaviourProofs", reflect.TypeOf((*MockEventPayloadI)(nil).MisbehaviourProofs))
}

// NetForkID mocks base method.
func (m *MockEventPayloadI) NetForkID() uint16 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NetForkID")
	ret0, _ := ret[0].(uint16)
	return ret0
}

// NetForkID indicates an expected call of NetForkID.
func (mr *MockEventPayloadIMockRecorder) NetForkID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NetForkID", reflect.TypeOf((*MockEventPayloadI)(nil).NetForkID))
}

// Parents mocks base method.
func (m *MockEventPayloadI) Parents() hash.Events {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Parents")
	ret0, _ := ret[0].(hash.Events)
	return ret0
}

// Parents indicates an expected call of Parents.
func (mr *MockEventPayloadIMockRecorder) Parents() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Parents", reflect.TypeOf((*MockEventPayloadI)(nil).Parents))
}

// Payload mocks base method.
func (m *MockEventPayloadI) Payload() *Payload {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Payload")
	ret0, _ := ret[0].(*Payload)
	return ret0
}

// Payload indicates an expected call of Payload.
func (mr *MockEventPayloadIMockRecorder) Payload() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Payload", reflect.TypeOf((*MockEventPayloadI)(nil).Payload))
}

// PayloadHash mocks base method.
func (m *MockEventPayloadI) PayloadHash() hash.Hash {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PayloadHash")
	ret0, _ := ret[0].(hash.Hash)
	return ret0
}

// PayloadHash indicates an expected call of PayloadHash.
func (mr *MockEventPayloadIMockRecorder) PayloadHash() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PayloadHash", reflect.TypeOf((*MockEventPayloadI)(nil).PayloadHash))
}

// PrevEpochHash mocks base method.
func (m *MockEventPayloadI) PrevEpochHash() *hash.Hash {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PrevEpochHash")
	ret0, _ := ret[0].(*hash.Hash)
	return ret0
}

// PrevEpochHash indicates an expected call of PrevEpochHash.
func (mr *MockEventPayloadIMockRecorder) PrevEpochHash() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PrevEpochHash", reflect.TypeOf((*MockEventPayloadI)(nil).PrevEpochHash))
}

// SelfParent mocks base method.
func (m *MockEventPayloadI) SelfParent() *hash.Event {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SelfParent")
	ret0, _ := ret[0].(*hash.Event)
	return ret0
}

// SelfParent indicates an expected call of SelfParent.
func (mr *MockEventPayloadIMockRecorder) SelfParent() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SelfParent", reflect.TypeOf((*MockEventPayloadI)(nil).SelfParent))
}

// Seq mocks base method.
func (m *MockEventPayloadI) Seq() idx.Event {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Seq")
	ret0, _ := ret[0].(idx.Event)
	return ret0
}

// Seq indicates an expected call of Seq.
func (mr *MockEventPayloadIMockRecorder) Seq() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Seq", reflect.TypeOf((*MockEventPayloadI)(nil).Seq))
}

// Sig mocks base method.
func (m *MockEventPayloadI) Sig() Signature {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Sig")
	ret0, _ := ret[0].(Signature)
	return ret0
}

// Sig indicates an expected call of Sig.
func (mr *MockEventPayloadIMockRecorder) Sig() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Sig", reflect.TypeOf((*MockEventPayloadI)(nil).Sig))
}

// Size mocks base method.
func (m *MockEventPayloadI) Size() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Size")
	ret0, _ := ret[0].(int)
	return ret0
}

// Size indicates an expected call of Size.
func (mr *MockEventPayloadIMockRecorder) Size() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Size", reflect.TypeOf((*MockEventPayloadI)(nil).Size))
}

// String mocks base method.
func (m *MockEventPayloadI) String() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "String")
	ret0, _ := ret[0].(string)
	return ret0
}

// String indicates an expected call of String.
func (mr *MockEventPayloadIMockRecorder) String() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "String", reflect.TypeOf((*MockEventPayloadI)(nil).String))
}

// Transactions mocks base method.
func (m *MockEventPayloadI) Transactions() types.Transactions {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Transactions")
	ret0, _ := ret[0].(types.Transactions)
	return ret0
}

// Transactions indicates an expected call of Transactions.
func (mr *MockEventPayloadIMockRecorder) Transactions() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Transactions", reflect.TypeOf((*MockEventPayloadI)(nil).Transactions))
}

// TransactionsToMeter mocks base method.
func (m *MockEventPayloadI) TransactionsToMeter() types.Transactions {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TransactionsToMeter")
	ret0, _ := ret[0].(types.Transactions)
	return ret0
}

// TransactionsToMeter indicates an expected call of TransactionsToMeter.
func (mr *MockEventPayloadIMockRecorder) TransactionsToMeter() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TransactionsToMeter", reflect.TypeOf((*MockEventPayloadI)(nil).TransactionsToMeter))
}

// Version mocks base method.
func (m *MockEventPayloadI) Version() uint8 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Version")
	ret0, _ := ret[0].(uint8)
	return ret0
}

// Version indicates an expected call of Version.
func (mr *MockEventPayloadIMockRecorder) Version() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Version", reflect.TypeOf((*MockEventPayloadI)(nil).Version))
}
