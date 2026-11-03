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
	"testing"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/carmen/go/state/gostate"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

var (
	addr1 = common.Address{1}
	addr2 = common.Address{2}
	slot1 = common.Hash{1}
)

func newTestState(t *testing.T) state.StateDB {
	st, err := carmen.NewState(carmen.Parameters{
		Variant:   gostate.VariantGoMemory,
		Schema:    carmen.Schema(5),
		Archive:   carmen.NoArchive,
		Directory: t.TempDir(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, st.Close()) })
	return evmstore.CreateNonCommittableCarmenStateDb(carmen.CreateNonCommittableStateDBUsing(st), nil)
}

func TestStateDiffLogger_NoModificationsGiveEmptyDiff(t *testing.T) {
	s := NewStateDiffLogger(newTestState(t))
	s.EndTransaction()

	result := s.GetResult()
	require.NotNil(t, result)
	require.Empty(t, result)
}

func TestStateDiffLogger_GetResult(t *testing.T) {
	tests := map[string]struct {
		prepare func(state.StateDB)
		modify  func(state.StateDB)
		want    string
	}{
		"changed fields of an existing account use *": {
			prepare: func(db state.StateDB) {
				db.SetBalance(addr1, uint256.NewInt(10))
				db.SetNonce(addr1, 1, tracing.NonceChangeUnspecified)
				db.SetCode(addr1, []byte{0x01}, tracing.CodeChangeUnspecified)
				db.SetState(addr1, slot1, common.Hash{31: 1})
			},
			modify: func(db state.StateDB) {
				db.AddBalance(addr1, uint256.NewInt(5), tracing.BalanceChangeUnspecified)
				db.AddBalance(addr1, uint256.NewInt(5), tracing.BalanceChangeUnspecified)
				db.SetNonce(addr1, 2, tracing.NonceChangeUnspecified)
				db.SetCode(addr1, []byte{0x02}, tracing.CodeChangeUnspecified)
				db.SetState(addr1, slot1, common.Hash{31: 2})
				db.SetState(addr1, slot1, common.Hash{31: 3})
			},
			want: `{"0x0100000000000000000000000000000000000000": {
				"balance": {"*": {"from": "0xa", "to": "0x14"}},
				"code": {"*": {"from": "0x01", "to": "0x02"}},
				"nonce": {"*": {"from": "0x1", "to": "0x2"}},
				"storage": {"0x0100000000000000000000000000000000000000000000000000000000000000": {"*": {
					"from": "0x0000000000000000000000000000000000000000000000000000000000000001",
					"to": "0x0000000000000000000000000000000000000000000000000000000000000003"
				}}}
			}}`,
		},
		"unchanged fields of an existing account use =": {
			prepare: func(db state.StateDB) {
				db.SetBalance(addr1, uint256.NewInt(10))
				db.SetCode(addr1, []byte{0x01}, tracing.CodeChangeUnspecified)
			},
			modify: func(db state.StateDB) {
				db.SetNonce(addr1, 1, tracing.NonceChangeUnspecified)
			},
			want: `{"0x0100000000000000000000000000000000000000": {
				"balance": "=", "code": "=", "nonce": {"*": {"from": "0x0", "to": "0x1"}}, "storage": {}
			}}`,
		},
		"existing account with zero balance is not born": {
			prepare: func(db state.StateDB) {
				db.SetNonce(addr1, 1, tracing.NonceChangeUnspecified)
			},
			modify: func(db state.StateDB) {
				db.AddBalance(addr1, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
			},
			want: `{"0x0100000000000000000000000000000000000000": {
				"balance": {"*": {"from": "0x0", "to": "0x1"}}, "code": "=", "nonce": "=", "storage": {}
			}}`,
		},
		"existing account without net change is left out": {
			prepare: func(db state.StateDB) {
				db.SetNonce(addr1, 1, tracing.NonceChangeUnspecified)
				db.SetBalance(addr1, uint256.NewInt(10))
				db.SetState(addr1, slot1, common.Hash{31: 1})
			},
			modify: func(db state.StateDB) {
				db.AddBalance(addr1, uint256.NewInt(5), tracing.BalanceChangeUnspecified)
				db.SubBalance(addr1, uint256.NewInt(5), tracing.BalanceChangeUnspecified)
				db.SetState(addr1, slot1, common.Hash{31: 2})
				db.SetState(addr1, slot1, common.Hash{31: 1})
			},
			want: `{}`,
		},
		"new account is born": {
			modify: func(db state.StateDB) {
				db.CreateContract(addr2)
				db.AddBalance(addr2, uint256.NewInt(5), tracing.BalanceChangeUnspecified)
				db.SetNonce(addr2, 1, tracing.NonceChangeUnspecified)
				db.SetCode(addr2, []byte{0x01}, tracing.CodeChangeUnspecified)
				db.SetState(addr2, slot1, common.Hash{31: 1})
			},
			want: `{"0x0200000000000000000000000000000000000000": {
				"balance": {"+": "0x5"}, "code": {"+": "0x01"}, "nonce": {"+": "0x1"},
				"storage": {"0x0100000000000000000000000000000000000000000000000000000000000000": {
					"+": "0x0000000000000000000000000000000000000000000000000000000000000001"
				}}
			}}`,
		},
		"touched account that stays empty is left out": {
			modify: func(db state.StateDB) {
				db.AddBalance(addr2, uint256.NewInt(0), tracing.BalanceChangeUnspecified)
			},
			want: `{}`,
		},
		"new account destroyed in the same transaction is left out": {
			modify: func(db state.StateDB) {
				db.CreateContract(addr2)
				db.AddBalance(addr2, uint256.NewInt(5), tracing.BalanceChangeUnspecified)
				db.SetNonce(addr2, 1, tracing.NonceChangeUnspecified)
				db.SetState(addr2, slot1, common.Hash{31: 1})
				db.SelfDestruct(addr2)
			},
			want: `{}`,
		},
		"existing account destroyed in the transaction dies": {
			prepare: func(db state.StateDB) {
				db.SetBalance(addr1, uint256.NewInt(5))
			},
			modify: func(db state.StateDB) {
				db.CreateContract(addr1)
				db.SetNonce(addr1, 1, tracing.NonceChangeUnspecified)
				db.SetCode(addr1, []byte{0x01}, tracing.CodeChangeUnspecified)
				db.SelfDestruct(addr1)
			},
			want: `{"0x0100000000000000000000000000000000000000": {
				"balance": {"-": "0x5"}, "code": {"-": "0x"}, "nonce": {"-": "0x0"}, "storage": {}
			}}`,
		},
		"reverted modifications are left out": {
			prepare: func(db state.StateDB) {
				db.SetBalance(addr1, uint256.NewInt(10))
			},
			modify: func(db state.StateDB) {
				snapshot := db.Snapshot()
				db.SubBalance(addr1, uint256.NewInt(5), tracing.BalanceChangeUnspecified)
				db.SetState(addr1, slot1, common.Hash{31: 1})
				db.CreateContract(addr2)
				db.AddBalance(addr2, uint256.NewInt(5), tracing.BalanceChangeUnspecified)
				db.SetNonce(addr2, 1, tracing.NonceChangeUnspecified)
				db.RevertToSnapshot(snapshot)
			},
			want: `{}`,
		},
		"reverted self-destruct of a new account keeps it": {
			modify: func(db state.StateDB) {
				db.CreateContract(addr2)
				db.SetNonce(addr2, 1, tracing.NonceChangeUnspecified)
				db.SetCode(addr2, []byte{0x01}, tracing.CodeChangeUnspecified)
				snapshot := db.Snapshot()
				db.SelfDestruct(addr2)
				db.RevertToSnapshot(snapshot)
			},
			want: `{"0x0200000000000000000000000000000000000000": {
				"balance": {"+": "0x0"}, "code": {"+": "0x01"}, "nonce": {"+": "0x1"}, "storage": {}
			}}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db := newTestState(t)
			if tt.prepare != nil {
				tt.prepare(db)
			}
			db.EndTransaction()

			s := NewStateDiffLogger(db)
			tt.modify(s)
			s.EndTransaction()

			got, err := json.Marshal(s.GetResult())
			require.NoError(t, err)
			require.JSONEq(t, tt.want, string(got))
		})
	}
}
