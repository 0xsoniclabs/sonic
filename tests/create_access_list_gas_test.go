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

package tests

import (
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// TestCreateAccessList_GivesTheSameAnswerWithAndWithoutGas checks that
// eth_createAccessList answers a call that omits "gas" just like the same
// call with an explicit gas limit. Without a gas limit, every round of the
// access list discovery estimates its own gas, and that estimate must account
// for the intrinsic cost of the access list used in that round.
func TestCreateAccessList_GivesTheSameAnswerWithAndWithoutGas(t *testing.T) {
	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	t.Parallel()

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	sender := MakeAccountWithBalance(t, session, big.NewInt(1e18))

	untouched := common.Address{0x42}
	external := common.Address{0x43}
	slot0 := common.Hash{}
	slot1 := common.BigToHash(big.NewInt(1))

	// readsSlot0 reads its own storage slot 0.
	readsSlot0 := []byte{
		byte(vm.PUSH0), byte(vm.SLOAD), byte(vm.POP),
		byte(vm.STOP),
	}
	// readsSlots0And1 reads its own storage slots 0 and 1.
	readsSlots0And1 := []byte{
		byte(vm.PUSH0), byte(vm.SLOAD), byte(vm.POP),
		byte(vm.PUSH1), 1, byte(vm.SLOAD), byte(vm.POP),
		byte(vm.STOP),
	}
	// readsBalanceOfExternal reads the balance of an external account.
	readsBalanceOfExternal := append(append(
		[]byte{byte(vm.PUSH20)}, external.Bytes()...),
		byte(vm.BALANCE), byte(vm.POP),
		byte(vm.STOP),
	)
	// readsSlot0AndBalanceOfExternal combines the two above.
	readsSlot0AndBalanceOfExternal := append(append(
		[]byte{byte(vm.PUSH0), byte(vm.SLOAD), byte(vm.POP), byte(vm.PUSH20)}, external.Bytes()...),
		byte(vm.BALANCE), byte(vm.POP),
		byte(vm.STOP),
	)

	slot0Reader := deployRuntimeCode(t, session, readsSlot0)
	slots0And1Reader := deployRuntimeCode(t, session, readsSlots0And1)
	externalReader := deployRuntimeCode(t, session, readsBalanceOfExternal)
	combinedReader := deployRuntimeCode(t, session, readsSlot0AndBalanceOfExternal)

	// A contract creation from a fresh sender is placed at the address derived
	// from the sender's nonce, which does not change as nothing is executed.
	nonce, err := client.PendingNonceAt(t.Context(), sender.Address())
	require.NoError(t, err)
	created := crypto.CreateAddress(sender.Address(), nonce)

	tests := map[string]struct {
		to         *common.Address
		data       []byte
		accessList *types.AccessList
		want       types.AccessList
	}{
		"transfer to an account without code": {
			to:   &untouched,
			want: types.AccessList{},
		},
		"transfer with an unused caller list keeps the list": {
			to:         &untouched,
			accessList: &types.AccessList{{Address: external, StorageKeys: []common.Hash{}}},
			want:       types.AccessList{{Address: external, StorageKeys: []common.Hash{}}},
		},
		"call reading one own slot": {
			to:   &slot0Reader,
			want: types.AccessList{{Address: slot0Reader, StorageKeys: []common.Hash{slot0}}},
		},
		"call reading two own slots": {
			to:   &slots0And1Reader,
			want: types.AccessList{{Address: slots0And1Reader, StorageKeys: []common.Hash{slot0, slot1}}},
		},
		"call reading an external account": {
			to:   &externalReader,
			want: types.AccessList{{Address: external, StorageKeys: []common.Hash{}}},
		},
		"call reading an own slot and an external account": {
			to: &combinedReader,
			want: types.AccessList{
				{Address: combinedReader, StorageKeys: []common.Hash{slot0}},
				{Address: external, StorageKeys: []common.Hash{}},
			},
		},
		"call with the complete list provided by the caller": {
			to:         &slot0Reader,
			accessList: &types.AccessList{{Address: slot0Reader, StorageKeys: []common.Hash{slot0}}},
			want:       types.AccessList{{Address: slot0Reader, StorageKeys: []common.Hash{slot0}}},
		},
		"call with a partial list provided by the caller": {
			to:         &slots0And1Reader,
			accessList: &types.AccessList{{Address: slots0And1Reader, StorageKeys: []common.Hash{slot1}}},
			want:       types.AccessList{{Address: slots0And1Reader, StorageKeys: []common.Hash{slot0, slot1}}},
		},
		"call with an empty list provided by the caller": {
			to:         &slot0Reader,
			accessList: &types.AccessList{},
			want:       types.AccessList{{Address: slot0Reader, StorageKeys: []common.Hash{slot0}}},
		},
		"contract creation reading an own slot": {
			data: readsSlot0,
			want: types.AccessList{{Address: created, StorageKeys: []common.Hash{slot0}}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			createAccessList := func(t *testing.T, gas *hexutil.Uint64) createAccessListResult {
				t.Helper()
				args := map[string]any{"from": sender.Address()}
				if test.to != nil {
					args["to"] = *test.to
				}
				if test.data != nil {
					args["data"] = hexutil.Bytes(test.data)
				}
				if test.accessList != nil {
					args["accessList"] = *test.accessList
				}
				if gas != nil {
					args["gas"] = *gas
				}
				var res createAccessListResult
				err := client.Client().Call(&res, "eth_createAccessList", args, "latest")
				require.NoError(t, err, "eth_createAccessList failed")
				require.Empty(t, res.Error, "call reported an execution error")
				return res
			}

			withGas := createAccessList(t, AsPointer(hexutil.Uint64(1_000_000)))
			withoutGas := createAccessList(t, nil)

			// The method lists addresses in ascending order.
			want := slices.Clone(test.want)
			slices.SortFunc(want, func(a, b types.AccessTuple) int { return a.Address.Cmp(b.Address) })

			require.Equal(t, want, withGas.AccessList)
			require.Equal(t, want, withoutGas.AccessList)
			// Sonic charges a share of the unused gas, so a call with a
			// generous gas limit uses more gas than one with an estimated one.
			require.LessOrEqual(t, withoutGas.GasUsed, withGas.GasUsed)
		})
	}
}

// createAccessListResult mirrors the unexported result of eth_createAccessList.
type createAccessListResult struct {
	AccessList types.AccessList `json:"accessList"`
	Error      string           `json:"error,omitempty"`
	GasUsed    hexutil.Uint64   `json:"gasUsed"`
}

// deployRuntimeCode deploys a contract with the given runtime code and returns
// its address.
func deployRuntimeCode(t *testing.T, session IntegrationTestNetSession, runtime []byte) common.Address {
	t.Helper()
	require.Less(t, len(runtime), 256, "runtime code too long for this constructor")
	size := byte(len(runtime))

	// The constructor copies the runtime code, which follows it, to memory
	// and returns it.
	constructor := []byte{
		byte(vm.PUSH1), size, byte(vm.PUSH1), 10, byte(vm.PUSH0), byte(vm.CODECOPY),
		byte(vm.PUSH1), size, byte(vm.PUSH0), byte(vm.RETURN),
	}
	require.Len(t, constructor, 10, "constructor size must match the copy offset")

	tx := CreateTransaction(t, session, &types.LegacyTx{
		Gas:  200_000,
		Data: append(constructor, runtime...),
	}, session.GetSessionSponsor())
	receipt, err := session.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	return receipt.ContractAddress
}
