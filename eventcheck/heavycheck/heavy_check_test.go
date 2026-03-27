// Copyright 2026 Sonic Operations Ltd
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

package heavycheck

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
)

func TestNew_DefaultThreads(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)

	c := New(Config{MaxQueuedTasks: 128, Threads: 0}, reader, nil)
	if c == nil {
		t.Fatal("New returned nil")
	}
	if c.config.Threads < 1 {
		t.Errorf("expected at least 1 thread, got %d", c.config.Threads)
	}
}

func TestNew_ExplicitThreads(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)

	c := New(Config{MaxQueuedTasks: 128, Threads: 4}, reader, nil)
	if c.config.Threads != 4 {
		t.Errorf("expected 4 threads, got %d", c.config.Threads)
	}
}

func TestOverloaded(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)

	c := New(Config{MaxQueuedTasks: 4, Threads: 1}, reader, nil)
	if c.Overloaded() {
		t.Error("should not be overloaded when empty")
	}
}

func TestStartStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)

	c := New(Config{MaxQueuedTasks: 128, Threads: 2}, reader, nil)
	c.Start()
	c.Stop()
}

func TestValidateEvent_WrongEpoch(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)

	pubkeys := map[idx.ValidatorID]validatorpk.PubKey{
		1: {Type: validatorpk.Types.Secp256k1, Raw: make([]byte, 33)},
	}
	reader.EXPECT().GetEpochPubKeys().Return(pubkeys, idx.Epoch(5)).AnyTimes()

	c := New(Config{MaxQueuedTasks: 128, Threads: 1}, reader, nil)

	// Build an event with epoch 10 (doesn't match reader's epoch 5).
	me := &inter.MutableEventPayload{}
	me.SetEpoch(10)
	me.SetCreator(1)
	e := me.Build()

	err := c.ValidateEvent(e)
	if err == nil {
		t.Fatal("expected error for wrong epoch")
	}
}

func TestValidateEvent_UnknownCreator(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)

	pubkeys := map[idx.ValidatorID]validatorpk.PubKey{
		1: {Type: validatorpk.Types.Secp256k1, Raw: make([]byte, 33)},
	}
	reader.EXPECT().GetEpochPubKeys().Return(pubkeys, idx.Epoch(1)).AnyTimes()

	c := New(Config{MaxQueuedTasks: 128, Threads: 1}, reader, nil)

	me := &inter.MutableEventPayload{}
	me.SetEpoch(1)
	me.SetCreator(99) // not in pubkeys
	e := me.Build()

	err := c.ValidateEvent(e)
	if err == nil {
		t.Fatal("expected error for unknown creator")
	}
}

func TestEnqueueEvent_AndProcess(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)

	pubkeys := map[idx.ValidatorID]validatorpk.PubKey{
		1: {Type: validatorpk.Types.Secp256k1, Raw: make([]byte, 33)},
	}
	reader.EXPECT().GetEpochPubKeys().Return(pubkeys, idx.Epoch(1)).AnyTimes()

	c := New(Config{MaxQueuedTasks: 128, Threads: 2}, reader, nil)
	c.Start()
	defer c.Stop()

	me := &inter.MutableEventPayload{}
	me.SetEpoch(1)
	me.SetCreator(99) // unknown creator - will get ErrAuth
	e := me.Build()

	var gotErr atomic.Value
	done := make(chan struct{})
	err := c.EnqueueEvent(e, func(err error) {
		gotErr.Store(err)
		close(done)
	})
	if err != nil {
		t.Fatalf("EnqueueEvent failed: %v", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for validation callback")
	}

	if gotErr.Load() == nil {
		t.Fatal("expected validation error")
	}
}

func TestEnqueueEvent_Terminated(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)

	// Use MaxQueuedTasks=0 so the channel has no buffer, forcing the select
	// to choose the quit case.
	c := New(Config{MaxQueuedTasks: 0, Threads: 1}, reader, nil)
	close(c.quit)

	me := &inter.MutableEventPayload{}
	me.SetEpoch(1)
	e := me.Build()

	err := c.EnqueueEvent(e, func(err error) {})
	if err == nil {
		t.Fatal("expected errTerminated")
	}
}

func TestEventsOnly_Enqueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)

	pubkeys := map[idx.ValidatorID]validatorpk.PubKey{
		1: {Type: validatorpk.Types.Secp256k1, Raw: make([]byte, 33)},
	}
	reader.EXPECT().GetEpochPubKeys().Return(pubkeys, idx.Epoch(1)).AnyTimes()

	c := New(Config{MaxQueuedTasks: 128, Threads: 2}, reader, nil)
	c.Start()
	defer c.Stop()

	adapter := &EventsOnly{Checker: c}

	me := &inter.MutableEventPayload{}
	me.SetEpoch(1)
	me.SetCreator(99)
	e := me.Build()

	done := make(chan struct{})
	err := adapter.Enqueue(e, func(err error) {
		close(done)
	})
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for callback")
	}
}

func TestErrors_NotNil(t *testing.T) {
	errs := []error{
		ErrWrongEventSig,
		ErrMalformedTxSig,
		ErrWrongPayloadHash,
		ErrPubkeyChanged,
	}
	for _, e := range errs {
		if e == nil {
			t.Error("error should not be nil")
		}
		if e.Error() == "" {
			t.Error("error message should not be empty")
		}
	}
}
