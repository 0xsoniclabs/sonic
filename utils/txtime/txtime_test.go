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

package txtime

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestSaw_Disabled(t *testing.T) {
	Enabled.Store(false)
	txid := common.HexToHash("0x01")
	Saw(txid, time.Now())
	// Should not store when disabled
	result := Get(txid)
	if !result.IsZero() {
		t.Fatal("expected zero time when disabled")
	}
}

func TestSaw_Enabled(t *testing.T) {
	Enabled.Store(true)
	defer Enabled.Store(false)

	txid := common.HexToHash("0x02")
	now := time.Now()
	Saw(txid, now)

	result := Get(txid)
	if result.IsZero() {
		t.Fatal("expected non-zero time after Saw")
	}
}

func TestValidated_Disabled(t *testing.T) {
	Enabled.Store(false)
	txid := common.HexToHash("0x03")
	Validated(txid, time.Now())
	result := Get(txid)
	if !result.IsZero() {
		t.Fatal("expected zero time when disabled")
	}
}

func TestValidated_Enabled_WithPriorSaw(t *testing.T) {
	Enabled.Store(true)
	defer Enabled.Store(false)

	txid := common.HexToHash("0x04")
	sawTime := time.Now().Add(-time.Second)
	Saw(txid, sawTime)

	validatedTime := time.Now()
	Validated(txid, validatedTime)

	result := Get(txid)
	if result.IsZero() {
		t.Fatal("expected non-zero time")
	}
	// Should use the Saw time, not validated time
	if result.Sub(sawTime) > time.Millisecond {
		t.Fatal("expected Saw time to be preserved")
	}
}

func TestValidated_Enabled_WithoutPriorSaw(t *testing.T) {
	Enabled.Store(true)
	defer Enabled.Store(false)

	txid := common.HexToHash("0x05")
	now := time.Now()
	Validated(txid, now)

	result := Get(txid)
	if result.IsZero() {
		t.Fatal("expected non-zero time")
	}
}

func TestOf_Disabled(t *testing.T) {
	Enabled.Store(false)
	txid := common.HexToHash("0x06")
	result := Of(txid)
	if !result.IsZero() {
		t.Fatal("expected zero time when disabled")
	}
}

func TestOf_UnknownTx(t *testing.T) {
	Enabled.Store(true)
	defer Enabled.Store(false)

	txid := common.HexToHash("0x07")
	before := time.Now()
	result := Of(txid)
	after := time.Now()

	// Of should return approximately "now" for unknown txs
	if result.Before(before) || result.After(after) {
		t.Fatal("expected current time for unknown tx")
	}
}

func TestOf_Finalized(t *testing.T) {
	Enabled.Store(true)
	defer Enabled.Store(false)

	txid := common.HexToHash("0x08")
	sawTime := time.Now().Add(-2 * time.Second)
	Saw(txid, sawTime)
	Validated(txid, sawTime)

	result := Of(txid)
	if result.Sub(sawTime) > time.Millisecond {
		t.Fatal("expected finalized time")
	}
}

func TestGet_Disabled(t *testing.T) {
	Enabled.Store(false)
	txid := common.HexToHash("0x09")
	result := Get(txid)
	if !result.IsZero() {
		t.Fatal("expected zero time when disabled")
	}
}

func TestGet_Unknown(t *testing.T) {
	Enabled.Store(true)
	defer Enabled.Store(false)

	txid := common.HexToHash("0x0a")
	result := Get(txid)
	if !result.IsZero() {
		t.Fatal("expected zero time for unknown tx from Get")
	}
}
