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

package txtrace

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
)

var (
	addr1 = common.Address{1}
	addr2 = common.Address{2}
)

func TestOnBalanceChange_NilInputsDoNotPanic(t *testing.T) {
	tests := []struct {
		name       string
		prev       *big.Int
		newBal     *big.Int
		wantBefore int64
		wantAfter  int64
	}{
		{"nil prev treated as zero", nil, big.NewInt(100), 0, 100},
		{"nil newBal treated as zero", big.NewInt(50), nil, 50, 0},
		{"both nil treated as zero", nil, nil, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStateDiffLogger()
			require.NotPanics(t, func() {
				s.onBalanceChange(addr1, tt.prev, tt.newBal, tracing.BalanceChangeUnspecified)
			})
			require.Equal(t, tt.wantBefore, s.balances[addr1].before.Int64())
			require.Equal(t, tt.wantAfter, s.balances[addr1].after.Int64())
		})
	}
}

func TestOnBalanceChange_OnlyFirstBeforeIsKept(t *testing.T) {
	// The "before" value must reflect the state before the transaction,
	// not after an intermediate change. Only the first call sets "before".
	s := NewStateDiffLogger()
	s.onBalanceChange(addr1, big.NewInt(10), big.NewInt(20), tracing.BalanceChangeUnspecified)
	s.onBalanceChange(addr1, big.NewInt(20), big.NewInt(30), tracing.BalanceChangeUnspecified)

	require.Equal(t, int64(10), s.balances[addr1].before.Int64(), "before must stay at original value")
	require.Equal(t, int64(30), s.balances[addr1].after.Int64(), "after must reflect latest value")
}

func TestOnBalanceChange_InputsAreCopied(t *testing.T) {
	// Mutating the original big.Int after the call must not affect stored values.
	prev := big.NewInt(5)
	newBal := big.NewInt(15)
	s := NewStateDiffLogger()
	s.onBalanceChange(addr1, prev, newBal, tracing.BalanceChangeUnspecified)

	prev.SetInt64(999)
	newBal.SetInt64(999)

	require.Equal(t, int64(5), s.balances[addr1].before.Int64())
	require.Equal(t, int64(15), s.balances[addr1].after.Int64())
}

func TestOnBalanceChange_SelfDestructReasonMarksAccount(t *testing.T) {
	reasons := []tracing.BalanceChangeReason{
		tracing.BalanceDecreaseSelfdestruct,
		tracing.BalanceDecreaseSelfdestructBurn,
	}
	for _, reason := range reasons {
		s := NewStateDiffLogger()
		s.onBalanceChange(addr1, big.NewInt(100), big.NewInt(0), reason)
		_, marked := s.selfDestructed[addr1]
		require.True(t, marked, "reason %v must mark account as self-destructed", reason)
	}
}

func TestOnBalanceChange_NonSelfDestructReasonDoesNotMark(t *testing.T) {
	s := NewStateDiffLogger()
	s.onBalanceChange(addr1, big.NewInt(0), big.NewInt(100), tracing.BalanceIncreaseGenesisBalance)
	_, marked := s.selfDestructed[addr1]
	require.False(t, marked)
}

func TestOnNonceChange_OnlyFirstBeforeIsKept(t *testing.T) {
	s := NewStateDiffLogger()
	s.onNonceChange(addr1, 0, 1)
	s.onNonceChange(addr1, 1, 2)

	require.Equal(t, uint64(0), s.nonces[addr1].before, "before must stay at original value")
	require.Equal(t, uint64(2), s.nonces[addr1].after, "after must reflect latest value")
}

func TestOnCodeChange_NilCodeDoesNotPanic(t *testing.T) {
	s := NewStateDiffLogger()
	require.NotPanics(t, func() {
		s.onCodeChange(addr1, common.Hash{}, nil, common.Hash{}, nil)
	})
	require.Empty(t, s.codes[addr1].before)
	require.Empty(t, s.codes[addr1].after)
}

func TestOnCodeChange_BytesAreCopied(t *testing.T) {
	// Mutating the original code slices after the call must not affect stored values.
	prevCode := []byte{0x60, 0x00}
	newCode := []byte{0x60, 0x01}
	s := NewStateDiffLogger()
	s.onCodeChange(addr1, common.Hash{}, prevCode, common.Hash{}, newCode)

	prevCode[0] = 0xFF
	newCode[0] = 0xFF

	require.Equal(t, byte(0x60), s.codes[addr1].before[0], "stored prevCode must be independent of original slice")
	require.Equal(t, byte(0x60), s.codes[addr1].after[0], "stored newCode must be independent of original slice")
}

func TestOnCodeChange_OnlyFirstBeforeIsKept(t *testing.T) {
	s := NewStateDiffLogger()
	s.onCodeChange(addr1, common.Hash{}, []byte{0x01}, common.Hash{}, []byte{0x02})
	s.onCodeChange(addr1, common.Hash{}, []byte{0x02}, common.Hash{}, []byte{0x03})

	require.Equal(t, []byte{0x01}, s.codes[addr1].before, "before must stay at original value")
	require.Equal(t, []byte{0x03}, s.codes[addr1].after)
}

func TestOnStorageChange_OnlyFirstBeforeIsKept(t *testing.T) {
	s := NewStateDiffLogger()
	slot := common.Hash{0xAA}
	v1 := common.Hash{0x01}
	v2 := common.Hash{0x02}
	v3 := common.Hash{0x03}
	s.onStorageChange(addr1, slot, v1, v2)
	s.onStorageChange(addr1, slot, v2, v3)

	require.Equal(t, v1, s.storage[addr1][slot].before, "before must stay at original value")
	require.Equal(t, v3, s.storage[addr1][slot].after)
}

func TestOnEnter_SelfDestructMarksFromAddress(t *testing.T) {
	s := NewStateDiffLogger()
	s.OnEnter(0, byte(vm.SELFDESTRUCT), addr1, addr2, nil, 0, nil)
	_, marked := s.selfDestructed[addr1]
	require.True(t, marked, "SELFDESTRUCT must mark the from address")
	_, markedTo := s.selfDestructed[addr2]
	require.False(t, markedTo, "SELFDESTRUCT must not mark the to address")
}

func TestOnEnter_NonSelfDestructOpcodeDoesNotMark(t *testing.T) {
	s := NewStateDiffLogger()
	nonDestructOpcodes := []vm.OpCode{vm.CALL, vm.CREATE, vm.DELEGATECALL, vm.STATICCALL}
	for _, op := range nonDestructOpcodes {
		s.OnEnter(0, byte(op), addr1, addr2, nil, 0, nil)
	}
	require.Empty(t, s.selfDestructed, "non-SELFDESTRUCT opcodes must not mark any account")
}

func TestOnEnter_WithNilValueDoesNotPanic(t *testing.T) {
	s := NewStateDiffLogger()
	require.NotPanics(t, func() {
		s.OnEnter(0, byte(vm.SELFDESTRUCT), addr1, addr2, nil, 0, nil)
	})
}

func TestGetResult_EmptyLoggerReturnsEmptyMap(t *testing.T) {
	s := NewStateDiffLogger()
	result := s.GetResult()
	require.NotNil(t, result, "GetResult must return a non-nil map")
	require.Empty(t, result)
}

func TestGetResult_PersistingAccountFields(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*StateDiffLogger)
		getField  func(*StateDiffAccount) any
		wantKey   string // if non-empty: expect map[string]any containing this key
		wantEqual string // if non-empty: expect this string value directly
	}{
		{
			name: "changed balance uses *",
			setup: func(s *StateDiffLogger) {
				s.onBalanceChange(addr1, big.NewInt(100), big.NewInt(200), tracing.BalanceChangeUnspecified)
			},
			getField: func(a *StateDiffAccount) any { return a.Balance },
			wantKey:  "*",
		},
		{
			name: "unchanged balance uses =",
			setup: func(s *StateDiffLogger) {
				s.onBalanceChange(addr1, big.NewInt(100), big.NewInt(100), tracing.BalanceChangeUnspecified)
			},
			getField:  func(a *StateDiffAccount) any { return a.Balance },
			wantEqual: "=",
		},
		{
			name: "changed nonce uses *",
			setup: func(s *StateDiffLogger) {
				s.onBalanceChange(addr1, big.NewInt(50), big.NewInt(50), tracing.BalanceChangeUnspecified)
				s.onNonceChange(addr1, 3, 4)
			},
			getField: func(a *StateDiffAccount) any { return a.Nonce },
			wantKey:  "*",
		},
		{
			name: "unchanged nonce uses =",
			setup: func(s *StateDiffLogger) {
				s.onBalanceChange(addr1, big.NewInt(50), big.NewInt(50), tracing.BalanceChangeUnspecified)
				s.onNonceChange(addr1, 5, 5)
			},
			getField:  func(a *StateDiffAccount) any { return a.Nonce },
			wantEqual: "=",
		},
		{
			name: "changed code uses *",
			setup: func(s *StateDiffLogger) {
				s.onBalanceChange(addr1, big.NewInt(1), big.NewInt(1), tracing.BalanceChangeUnspecified)
				s.onCodeChange(addr1, common.Hash{}, []byte{0x01}, common.Hash{}, []byte{0x02})
			},
			getField: func(a *StateDiffAccount) any { return a.Code },
			wantKey:  "*",
		},
		{
			name: "unchanged code uses =",
			setup: func(s *StateDiffLogger) {
				s.onBalanceChange(addr1, big.NewInt(1), big.NewInt(1), tracing.BalanceChangeUnspecified)
				s.onCodeChange(addr1, common.Hash{}, []byte{0x01}, common.Hash{}, []byte{0x01})
			},
			getField:  func(a *StateDiffAccount) any { return a.Code },
			wantEqual: "=",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStateDiffLogger()
			tt.setup(s)
			result := s.GetResult()
			require.Contains(t, result, addr1)
			field := tt.getField(result[addr1])
			if tt.wantEqual != "" {
				require.Equal(t, tt.wantEqual, field)
			} else {
				m, ok := field.(map[string]any)
				require.True(t, ok, "expected map[string]any, got %T", field)
				require.Contains(t, m, tt.wantKey)
			}
		})
	}
}

func TestGetResult_PersistingAccountStorageChange(t *testing.T) {
	s := NewStateDiffLogger()
	s.onBalanceChange(addr1, big.NewInt(1), big.NewInt(1), tracing.BalanceChangeUnspecified)
	slot := common.Hash{0x01}
	s.onStorageChange(addr1, slot, common.Hash{0x00}, common.Hash{0xFF})

	result := s.GetResult()
	storageEntry, ok := result[addr1].Storage[slot]
	require.True(t, ok)
	m, ok := storageEntry.(map[string]any)
	require.True(t, ok, "persisting account storage must use '*' operator")
	require.Contains(t, m, "*")
}

func TestGetResult_BornAccountUsesPlus(t *testing.T) {
	s := NewStateDiffLogger()
	// All "before" values are zero/empty → account is born.
	s.onBalanceChange(addr1, big.NewInt(0), big.NewInt(500), tracing.BalanceChangeUnspecified)
	s.onNonceChange(addr1, 0, 1)
	s.onCodeChange(addr1, common.Hash{}, nil, common.Hash{}, []byte{0x60})

	result := s.GetResult()
	require.Contains(t, result, addr1)
	entry := result[addr1]

	balM, ok := entry.Balance.(map[string]any)
	require.True(t, ok)
	require.Contains(t, balM, "+", "born account balance must use '+' operator")

	nonceM, ok := entry.Nonce.(map[string]any)
	require.True(t, ok)
	require.Contains(t, nonceM, "+", "born account nonce must use '+' operator")

	codeM, ok := entry.Code.(map[string]any)
	require.True(t, ok)
	require.Contains(t, codeM, "+", "born account code must use '+' operator")
}

func TestGetResult_BornAccountStorageUsesPlus(t *testing.T) {
	s := NewStateDiffLogger()
	s.onBalanceChange(addr1, big.NewInt(0), big.NewInt(1), tracing.BalanceChangeUnspecified)
	slot := common.Hash{0x01}
	s.onStorageChange(addr1, slot, common.Hash{0x00}, common.Hash{0xAB})

	result := s.GetResult()
	storageEntry, ok := result[addr1].Storage[slot]
	require.True(t, ok)
	m, ok := storageEntry.(map[string]any)
	require.True(t, ok)
	require.Contains(t, m, "+", "born account storage must use '+' operator (not '*')")
	require.NotContains(t, m, "*")
}

func TestGetResult_DiedAccountUsesMinus(t *testing.T) {
	s := NewStateDiffLogger()
	// Account with prior balance (existed before) self-destructs.
	s.onBalanceChange(addr1, big.NewInt(300), big.NewInt(0), tracing.BalanceDecreaseSelfdestruct)

	result := s.GetResult()
	require.Contains(t, result, addr1)
	entry := result[addr1]

	balM, ok := entry.Balance.(map[string]any)
	require.True(t, ok)
	require.Contains(t, balM, "-", "died account balance must use '-' operator")

	nonceM, ok := entry.Nonce.(map[string]any)
	require.True(t, ok)
	require.Contains(t, nonceM, "-", "died account nonce must use '-' operator")

	codeM, ok := entry.Code.(map[string]any)
	require.True(t, ok)
	require.Contains(t, codeM, "-", "died account code must use '-' operator")
}

func TestGetResult_BornAndDiedInSameTxIsExcluded(t *testing.T) {
	// An account that was created AND self-destructed in the same transaction
	// must not appear in the StateDiff at all.
	s := NewStateDiffLogger()
	// Nonce changed from 0 → account has no prior existence.
	s.onNonceChange(addr1, 0, 1)
	// Mark as self-destructed via OnEnter (EVM-level detection).
	s.OnEnter(0, byte(vm.SELFDESTRUCT), addr1, addr2, nil, 0, nil)

	result := s.GetResult()
	require.NotContains(t, result, addr1, "born+died account must be excluded from StateDiff")
}

func TestGetResult_ExistingAccountSelfDestructNotExcluded(t *testing.T) {
	// An account that existed before and self-destructs must appear as "died".
	s := NewStateDiffLogger()
	s.onBalanceChange(addr1, big.NewInt(100), big.NewInt(0), tracing.BalanceDecreaseSelfdestruct)

	result := s.GetResult()
	require.Contains(t, result, addr1, "pre-existing self-destructed account must appear in StateDiff")
}

func TestGetResult_StorageOnlyImpliesPreExistingContract(t *testing.T) {
	// A storage change without any balance/nonce/code change implies the contract
	// already existed — it must not be classified as born.
	s := NewStateDiffLogger()
	slot := common.Hash{0x01}
	s.onStorageChange(addr1, slot, common.Hash{0x00}, common.Hash{0xFF})

	result := s.GetResult()
	require.Contains(t, result, addr1)
	m, ok := result[addr1].Storage[slot].(map[string]any)
	require.True(t, ok)
	require.Contains(t, m, "*", "storage-only change must use '*' operator (persisting account)")
	require.NotContains(t, m, "+", "storage-only change must not use '+' operator (not born)")
}

func TestGetResult_SelfDestructWithNoOtherChangesNotInOutput(t *testing.T) {
	// A self-destructed account with zero balance that never triggered any
	// balance/nonce/code/storage hooks won't appear in the output.
	// This is a known limitation: the diff is built only from observed hooks.
	// Documenting and asserting the current behavior prevents regressions.
	s := NewStateDiffLogger()
	s.OnEnter(0, byte(vm.SELFDESTRUCT), addr1, addr2, nil, 0, nil)

	result := s.GetResult()
	require.NotContains(t, result, addr1,
		"self-destructed account with no observed state changes is absent from output (known limitation)")
}

func TestGetResult_MultipleAddressesIndependent(t *testing.T) {
	s := NewStateDiffLogger()
	// addr1: existing account (born=false)
	s.onBalanceChange(addr1, big.NewInt(10), big.NewInt(20), tracing.BalanceChangeUnspecified)
	// addr2: new account (born=true)
	s.onNonceChange(addr2, 0, 1)

	result := s.GetResult()
	require.Len(t, result, 2)

	// addr1 must use "*" for changed balance
	m1, ok := result[addr1].Balance.(map[string]any)
	require.True(t, ok)
	require.Contains(t, m1, "*")

	// addr2 must use "+" (born)
	m2, ok := result[addr2].Nonce.(map[string]any)
	require.True(t, ok)
	require.Contains(t, m2, "+")
}

func TestAccountExistedBefore(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*StateDiffLogger)
		want  bool
	}{
		{
			name: "nonzero balance returns true",
			setup: func(s *StateDiffLogger) {
				s.onBalanceChange(addr1, big.NewInt(1), big.NewInt(0), tracing.BalanceChangeUnspecified)
			},
			want: true,
		},
		{
			name: "zero balance nonzero nonce returns true",
			setup: func(s *StateDiffLogger) {
				s.onBalanceChange(addr1, big.NewInt(0), big.NewInt(5), tracing.BalanceChangeUnspecified)
				s.onNonceChange(addr1, 2, 3)
			},
			want: true,
		},
		{
			name: "non-empty code returns true",
			setup: func(s *StateDiffLogger) {
				s.onCodeChange(addr1, common.Hash{}, []byte{0x60}, common.Hash{}, []byte{0x61})
			},
			want: true,
		},
		{
			name: "storage only returns true",
			setup: func(s *StateDiffLogger) {
				s.onStorageChange(addr1, common.Hash{0x01}, common.Hash{0x00}, common.Hash{0xFF})
			},
			want: true,
		},
		{
			name: "all zero fields returns false",
			setup: func(s *StateDiffLogger) {
				s.onBalanceChange(addr1, big.NewInt(0), big.NewInt(100), tracing.BalanceChangeUnspecified)
				s.onNonceChange(addr1, 0, 1)
				s.onCodeChange(addr1, common.Hash{}, nil, common.Hash{}, []byte{0x60})
			},
			want: false,
		},
		{
			name:  "unknown address returns false",
			setup: func(s *StateDiffLogger) {},
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStateDiffLogger()
			tt.setup(s)
			require.Equal(t, tt.want, s.accountExistedBefore(addr1))
		})
	}
}

func TestGetResult_BalanceSerializesCorrectly(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*StateDiffLogger)
		wantKey string // expected JSON key in parsed map ("*", "+", "-"); empty means raw string match
		wantRaw string // expected raw JSON (used when wantKey is empty)
	}{
		{
			name: "persisting changed balance uses *",
			setup: func(s *StateDiffLogger) {
				s.onBalanceChange(addr1, big.NewInt(100), big.NewInt(200), tracing.BalanceChangeUnspecified)
			},
			wantKey: "*",
		},
		{
			name: "born balance uses +",
			setup: func(s *StateDiffLogger) {
				s.onBalanceChange(addr1, big.NewInt(0), big.NewInt(500), tracing.BalanceChangeUnspecified)
			},
			wantKey: "+",
		},
		{
			name: "died balance uses -",
			setup: func(s *StateDiffLogger) {
				s.onBalanceChange(addr1, big.NewInt(300), big.NewInt(0), tracing.BalanceDecreaseSelfdestruct)
			},
			wantKey: "-",
		},
		{
			name: "unchanged balance serializes as = string",
			setup: func(s *StateDiffLogger) {
				s.onBalanceChange(addr1, big.NewInt(100), big.NewInt(100), tracing.BalanceChangeUnspecified)
			},
			wantRaw: `"="`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStateDiffLogger()
			tt.setup(s)
			result := s.GetResult()
			data, err := json.Marshal(result[addr1].Balance)
			require.NoError(t, err)
			if tt.wantRaw != "" {
				require.Equal(t, tt.wantRaw, string(data))
				return
			}
			var parsed map[string]any
			require.NoError(t, json.Unmarshal(data, &parsed))
			require.Contains(t, parsed, tt.wantKey)
		})
	}
}

func TestGetResult_DiedAccountWithNoBalanceRecorded(t *testing.T) {
	// A died account with no balance hook (only nonce change + OnEnter SELFDESTRUCT).
	// diedBalanceDiff must return a zero-valued diff without panicking.
	s := NewStateDiffLogger()
	s.onNonceChange(addr1, 5, 0) // nonce before=5 → existed before
	s.OnEnter(0, byte(vm.SELFDESTRUCT), addr1, addr2, nil, 0, nil)

	require.NotPanics(t, func() {
		result := s.GetResult()
		require.Contains(t, result, addr1)
		balM, ok := result[addr1].Balance.(map[string]any)
		require.True(t, ok)
		require.Contains(t, balM, "-")
		// Balance defaults to zero when no balance hook fired.
		bal := balM["-"].(*hexutil.Big)
		require.Equal(t, int64(0), bal.ToInt().Int64())
	})
}

func TestGetResult_DiedAccountWithNoCodeRecorded(t *testing.T) {
	// diedCodeDiff must return empty bytes without panicking.
	s := NewStateDiffLogger()
	s.onBalanceChange(addr1, big.NewInt(100), big.NewInt(0), tracing.BalanceDecreaseSelfdestruct)

	require.NotPanics(t, func() {
		result := s.GetResult()
		codeM, ok := result[addr1].Code.(map[string]any)
		require.True(t, ok)
		require.Contains(t, codeM, "-")
		code := codeM["-"].(hexutil.Bytes)
		require.Empty(t, code)
	})
}

func TestGetResult_DiedAccountStorageUsesStar(t *testing.T) {
	// A self-destructed account that also had storage changes must show those
	// storage slots with the "*" operator (from/to pair), not "+" or "-".
	s := NewStateDiffLogger()
	// Existing account (nonzero balance before) self-destructs.
	s.onBalanceChange(addr1, big.NewInt(100), big.NewInt(0), tracing.BalanceDecreaseSelfdestruct)
	slot := common.Hash{0x01}
	s.onStorageChange(addr1, slot, common.Hash{0xAA}, common.Hash{0x00})

	result := s.GetResult()
	require.Contains(t, result, addr1)
	storageEntry, ok := result[addr1].Storage[slot]
	require.True(t, ok, "died account storage slot must be present")
	m, ok := storageEntry.(map[string]any)
	require.True(t, ok)
	require.Contains(t, m, "*", "died account storage must use '*' operator")
	require.NotContains(t, m, "+")
	require.NotContains(t, m, "-")
	sd, ok := m["*"].(*StateDiffStorage)
	require.True(t, ok)
	require.Equal(t, common.Hash{0xAA}, sd.From)
	require.Equal(t, common.Hash{0x00}, sd.To)
}

func TestGetResult_DiedAccountWithCodeRecorded(t *testing.T) {
	// diedCodeDiff must return the before-code with "-" operator when a code
	// hook was observed for the account (exercises the `if cc, ok` branch).
	s := NewStateDiffLogger()
	s.onBalanceChange(addr1, big.NewInt(100), big.NewInt(0), tracing.BalanceDecreaseSelfdestruct)
	s.onCodeChange(addr1, common.Hash{}, []byte{0x60, 0x00}, common.Hash{}, []byte{})

	require.NotPanics(t, func() {
		result := s.GetResult()
		require.Contains(t, result, addr1)
		codeM, ok := result[addr1].Code.(map[string]any)
		require.True(t, ok)
		require.Contains(t, codeM, "-")
		code := codeM["-"].(hexutil.Bytes)
		require.Equal(t, hexutil.Bytes{0x60, 0x00}, code, "died account code must contain before-value")
	})
}

func TestGetResult_BornAccountWithNoBalanceRecorded(t *testing.T) {
	// bornBalanceDiff must return a zero-valued diff without panicking.
	s := NewStateDiffLogger()
	s.onNonceChange(addr1, 0, 1) // all-zero before → born

	require.NotPanics(t, func() {
		result := s.GetResult()
		require.Contains(t, result, addr1)
		balM, ok := result[addr1].Balance.(map[string]any)
		require.True(t, ok)
		require.Contains(t, balM, "+")
		bal := balM["+"].(*hexutil.Big)
		require.Equal(t, int64(0), bal.ToInt().Int64())
	})
}
