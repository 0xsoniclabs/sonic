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

package rpctest

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func Test_TraceSimpleTransfer(t *testing.T) {
	acc1 := NewWallet(t)
	acc2 := NewWallet(t)
	transferBalance := big.NewInt(1e17)

	be := NewBackendBuilder(t).
		WithAccount(*acc1.Address(), AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(*acc2.Address(), AccountState{}).
		WithBlockHistory(
			[]Block{
				{
					Number: 1,
					Hash:   common.HexToHash("0x1"),
				},
				{
					Number:     2,
					Hash:       common.HexToHash("0x2"),
					ParentHash: common.HexToHash("0x1"),
				},
			},
		).
		Build()

	api := ethapi.NewPublicTxTraceAPI(be, 100_000, 1000)

	txRequest1 := ethapi.TransactionArgs{
		From:     acc1.Address(),
		To:       acc2.Address(),
		Nonce:    ToHexUint64(0),
		Gas:      ToHexUint64(8_000_000),
		GasPrice: ToHexBigInt(big.NewInt(1)),
		Value:    ToHexBigInt(transferBalance),
	}

	res, err := api.Call(
		t.Context(),
		txRequest1,
		[]string{"stateDiff"},
		rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
		&ethapi.TraceCallConfig{},
	)
	require.NoError(t, err)
	require.NotNil(t, res)

	// check state diff for correct balance changes
	require.NotNil(t, res.StateDiff)
	require.NotNil(t, res.StateDiff[*acc1.Address()])
	require.NotNil(t, res.StateDiff[*acc2.Address()])

	balanceDiff1, ok1 := res.StateDiff[*acc1.Address()].Balance.(map[string]any)
	require.True(t, ok1)
	balanceDiff2, ok2 := res.StateDiff[*acc2.Address()].Balance.(map[string]any)
	require.True(t, ok2)

	require.NotNil(t, balanceDiff1["*"])
	require.NotNil(t, balanceDiff2["+"])

	balanceAmount2, ok := balanceDiff2["+"].(*hexutil.Big)
	require.True(t, ok)
	require.Equal(t, transferBalance, (*big.Int)(balanceAmount2))

	// real state should not be mutated after trace call
	state, _, err := be.StateAndBlockByNumberOrHash(t.Context(), rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber))
	require.NoError(t, err)
	require.NotNil(t, state)
	require.Equal(t, uint256.NewInt(0), state.GetBalance(*acc2.Address()))
}

func Test_TraceTransaction_MustProcessWithBaseFeeSet(t *testing.T) {

	var chainId uint64 = 123
	signer := types.LatestSignerForChainID(new(big.Int).SetUint64(chainId))

	toAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")

	txData := &types.DynamicFeeTx{
		Nonce:     0,
		Gas:       8_000_000,
		GasFeeCap: big.NewInt(0), // no fee
		GasTipCap: big.NewInt(0), // no tip
		Value:     big.NewInt(1e17),
		To:        &toAddr,
	}

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	sender := crypto.PubkeyToAddress(key.PublicKey)

	tx1 := types.MustSignNewTx(key, signer, txData)

	receipt := &types.Receipt{
		Type:              types.LegacyTxType,
		Status:            types.ReceiptStatusSuccessful,
		CumulativeGasUsed: 21000,
		GasUsed:           21000,
	}
	txs := map[common.Hash]*Transaction{
		tx1.Hash(): {
			tx:          tx1,
			blockNumber: 2,
			txIndex:     0,
			receipt:     receipt,
		},
	}

	be := NewBackendBuilder(t).
		WithChainID(chainId).
		WithAccount(sender, AccountState{Balance: big.NewInt(1e18)}).
		WithBlockHistory(
			[]Block{
				{
					Number: 1,
					Hash:   common.HexToHash("0x1"),
				},
				{
					Number:       2,
					Hash:         common.HexToHash("0x2"),
					ParentHash:   common.HexToHash("0x1"),
					Transactions: txs,
					BaseFee:      big.NewInt(200), // base fee for block is set
				},
			},
		).
		Build()

	api := ethapi.NewPublicTxTraceAPI(be, 100_000, 1000)

	// trace call should succeed and not error out due to base fee set in block
	res, err := api.Transaction(t.Context(), tx1.Hash())
	require.NoError(t, err)
	require.NotNil(t, res)

}

// blockCounter counts the blocks a trace_filter reads from the backend.
type blockCounter struct {
	*fakeBackend
	reads atomic.Int64
}

func (b *blockCounter) BlockByNumber(ctx context.Context, n rpc.BlockNumber) (*evmcore.EvmBlock, error) {
	b.reads.Add(1)
	return b.fakeBackend.BlockByNumber(ctx, n)
}

// Test_TraceFilter_RefusesARangeWiderThanTheLimit checks that a span wider
// than the limit is refused before any block is read.
func Test_TraceFilter_RefusesARangeWiderThanTheLimit(t *testing.T) {
	const head, limit = 2_000, 1000
	history := make([]Block, 0, head+1)
	for n := uint64(0); n <= head; n++ {
		history = append(history, Block{Number: n, Hash: common.BigToHash(big.NewInt(int64(n + 1)))})
	}
	be := &blockCounter{fakeBackend: NewBackendBuilder(t).WithBlockHistory(history).Build()}
	api := ethapi.NewPublicTxTraceAPI(be, 100_000, limit)

	block := func(n int64) *rpc.BlockNumberOrHash {
		b := rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(n))
		return &b
	}
	for _, args := range []ethapi.FilterArgs{
		{},
		{FromBlock: block(0), ToBlock: block(limit + 1)},
		{FromBlock: block(0), ToBlock: block(head), After: 1 << 40, Count: 1},
	} {
		_, err := api.Filter(t.Context(), args)
		require.ErrorContains(t, err, fmt.Sprintf("too wide blocks range, the limit is %d", limit), "%+v", args)
	}
	require.Zero(t, be.reads.Load(), "a refused filter reads no block")

	res, err := api.Filter(t.Context(), ethapi.FilterArgs{FromBlock: block(0), ToBlock: block(limit)})
	require.NoError(t, err)
	require.JSONEq(t, "[]", string(res))
	require.EqualValues(t, limit+1, be.reads.Load())
}

func Test_TraceFilter_ZeroLimitMeansUnlimited(t *testing.T) {
	const head = 2_000
	history := make([]Block, 0, head+1)
	for n := uint64(0); n <= head; n++ {
		history = append(history, Block{Number: n, Hash: common.BigToHash(big.NewInt(int64(n + 1)))})
	}
	be := &blockCounter{fakeBackend: NewBackendBuilder(t).WithBlockHistory(history).Build()}
	api := ethapi.NewPublicTxTraceAPI(be, 100_000, 0)

	res, err := api.Filter(t.Context(), ethapi.FilterArgs{})
	require.NoError(t, err)
	require.JSONEq(t, "[]", string(res))
	require.EqualValues(t, head+1, be.reads.Load())
}

func Test_TraceCall_StateDiff(t *testing.T) {
	sender := common.Address{0x5e}
	contract := common.Address{0xc0}
	beneficiary := common.Address{0xbe}
	reverter := common.Address{0xee}
	drained := common.Address{0xd0}
	factory := common.Address{0xfa}
	child := crypto.CreateAddress(factory, 1)

	selfDestructTo := func(addr common.Address) []byte {
		return slices.Concat([]byte{0x73}, addr[:], []byte{0xff})
	}
	// Calls the address given as first calldata word, then reverts.
	reverterCode := []byte{
		0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00,
		0x60, 0x00, 0x35, 0x5a, 0xf1, 0x50,
		0x60, 0x00, 0x60, 0x00, 0xfd,
	}
	// The factory creates a child with runtime code CALLER SELFDESTRUCT, then
	// calls the reverter with the child's address.
	childInitCode := []byte{0x61, 0x33, 0xff, 0x60, 0x00, 0x52, 0x60, 0x02, 0x60, 0x1e, 0xf3}
	factoryCode := slices.Concat(
		[]byte{0x6a}, childInitCode,
		[]byte{0x60, 0x00, 0x52, 0x60, 0x0b, 0x60, 0x15, 0x60, 0x00, 0xf0, 0x60, 0x00, 0x52},
		[]byte{0x60, 0x00, 0x60, 0x00, 0x60, 0x20, 0x60, 0x00, 0x60, 0x00, 0x73}, reverter[:],
		[]byte{0x5a, 0xf1, 0x50, 0x00},
	)

	const senderNonceOnly = `"0x5e00000000000000000000000000000000000000": {
		"balance": "=", "code": "=", "nonce": {"*": {"from": "0x0", "to": "0x1"}}, "storage": {}
	}`

	tests := map[string]struct {
		accounts map[common.Address]AccountState
		to       *common.Address
		value    int64
		data     []byte
		want     string
	}{
		"value to a new account": {
			to:    &beneficiary,
			value: 1,
			want: `{
				"0x5e00000000000000000000000000000000000000": {
					"balance": {"*": {"from": "0x64", "to": "0x63"}}, "code": "=", "nonce": {"*": {"from": "0x0", "to": "0x1"}}, "storage": {}
				},
				"0xbe00000000000000000000000000000000000000": {
					"balance": {"+": "0x1"}, "code": {"+": "0x"}, "nonce": {"+": "0x0"}, "storage": {}
				}
			}`,
		},
		"deployment": {
			data: []byte{0x60, 0x01, 0x60, 0x00, 0x55, 0x60, 0x01, 0x60, 0x00, 0xf3},
			want: `{` + senderNonceOnly + `,
				"` + strings.ToLower(crypto.CreateAddress(sender, 0).Hex()) + `": {
					"balance": {"+": "0x0"}, "code": {"+": "0x00"}, "nonce": {"+": "0x1"}, "storage": {
						"0x0000000000000000000000000000000000000000000000000000000000000000": {
							"+": "0x0000000000000000000000000000000000000000000000000000000000000001"
						}
					}
				}
			}`,
		},
		"contract created and destroyed in the same transaction": {
			value: 3,
			data:  selfDestructTo(beneficiary),
			want: `{
				"0x5e00000000000000000000000000000000000000": {
					"balance": {"*": {"from": "0x64", "to": "0x61"}}, "code": "=", "nonce": {"*": {"from": "0x0", "to": "0x1"}}, "storage": {}
				},
				"0xbe00000000000000000000000000000000000000": {
					"balance": {"+": "0x3"}, "code": {"+": "0x"}, "nonce": {"+": "0x0"}, "storage": {}
				}
			}`,
		},
		"value to an existing contract with zero balance": {
			accounts: map[common.Address]AccountState{
				contract: {Nonce: 1, Code: []byte{0x00}},
			},
			to:    &contract,
			value: 1,
			want: `{
				"0x5e00000000000000000000000000000000000000": {
					"balance": {"*": {"from": "0x64", "to": "0x63"}}, "code": "=", "nonce": {"*": {"from": "0x0", "to": "0x1"}}, "storage": {}
				},
				"0xc000000000000000000000000000000000000000": {
					"balance": {"*": {"from": "0x0", "to": "0x1"}}, "code": "=", "nonce": "=", "storage": {}
				}
			}`,
		},
		"value to a drained account": {
			accounts: map[common.Address]AccountState{
				drained: {Nonce: 1},
			},
			to:    &drained,
			value: 1,
			want: `{
				"0x5e00000000000000000000000000000000000000": {
					"balance": {"*": {"from": "0x64", "to": "0x63"}}, "code": "=", "nonce": {"*": {"from": "0x0", "to": "0x1"}}, "storage": {}
				},
				"0xd000000000000000000000000000000000000000": {
					"balance": {"*": {"from": "0x0", "to": "0x1"}}, "code": "=", "nonce": "=", "storage": {}
				}
			}`,
		},
		"selfdestruct of an existing contract": {
			accounts: map[common.Address]AccountState{
				contract: {Nonce: 1, Balance: big.NewInt(5), Code: selfDestructTo(beneficiary)},
			},
			to: &contract,
			want: `{` + senderNonceOnly + `,
				"0xc000000000000000000000000000000000000000": {
					"balance": {"*": {"from": "0x5", "to": "0x0"}}, "code": "=", "nonce": "=", "storage": {}
				},
				"0xbe00000000000000000000000000000000000000": {
					"balance": {"+": "0x5"}, "code": {"+": "0x"}, "nonce": {"+": "0x0"}, "storage": {}
				}
			}`,
		},
		"selfdestruct of an existing contract to itself after a storage write": {
			accounts: map[common.Address]AccountState{
				contract: {Nonce: 1, Balance: big.NewInt(5), Code: []byte{0x60, 0x01, 0x60, 0x00, 0x55, 0x30, 0xff}},
			},
			to: &contract,
			want: `{` + senderNonceOnly + `,
				"0xc000000000000000000000000000000000000000": {
					"balance": "=", "code": "=", "nonce": "=", "storage": {
						"0x0000000000000000000000000000000000000000000000000000000000000000": {"*": {
							"from": "0x0000000000000000000000000000000000000000000000000000000000000000",
							"to": "0x0000000000000000000000000000000000000000000000000000000000000001"
						}}
					}
				}
			}`,
		},
		"storage write and value transfer in a reverted call": {
			accounts: map[common.Address]AccountState{
				contract: {Nonce: 1, Code: []byte{0x60, 0x01, 0x60, 0x00, 0x55, 0x60, 0x00, 0x60, 0x00, 0xfd}},
			},
			to:    &contract,
			value: 7,
			want:  `{` + senderNonceOnly + `}`,
		},
		"creation with a reverting constructor": {
			value: 3,
			data:  []byte{0x60, 0x00, 0x60, 0x00, 0xfd},
			want:  `{` + senderNonceOnly + `}`,
		},
		"selfdestruct of an existing contract in a reverted call": {
			accounts: map[common.Address]AccountState{
				contract: {Nonce: 1, Balance: big.NewInt(5), Code: selfDestructTo(beneficiary)},
				reverter: {Nonce: 1, Code: reverterCode},
			},
			to:   &reverter,
			data: common.LeftPadBytes(contract[:], 32),
			want: `{` + senderNonceOnly + `}`,
		},
		"selfdestruct of a new contract in a reverted call": {
			accounts: map[common.Address]AccountState{
				factory:  {Nonce: 1, Code: factoryCode},
				reverter: {Nonce: 1, Code: reverterCode},
			},
			to: &factory,
			want: `{` + senderNonceOnly + `,
				"0xfa00000000000000000000000000000000000000": {
					"balance": "=", "code": "=", "nonce": {"*": {"from": "0x1", "to": "0x2"}}, "storage": {}
				},
				"` + strings.ToLower(child.Hex()) + `": {
					"balance": {"+": "0x0"}, "code": {"+": "0x33ff"}, "nonce": {"+": "0x1"}, "storage": {}
				}
			}`,
		},
		"storage slot written back to its original value": {
			accounts: map[common.Address]AccountState{
				contract: {Nonce: 1, Code: []byte{0x60, 0x01, 0x60, 0x00, 0x55, 0x60, 0x00, 0x60, 0x00, 0x55, 0x00}},
			},
			to:   &contract,
			want: `{` + senderNonceOnly + `}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			builder := NewBackendBuilder(t).
				WithAccount(sender, AccountState{Balance: big.NewInt(100)}).
				WithBlockHistory([]Block{
					{Number: 1, Hash: common.HexToHash("0x1")},
					{Number: 2, Hash: common.HexToHash("0x2"), ParentHash: common.HexToHash("0x1"), PrevRandao: common.HexToHash("0x42")},
				})
			for addr, account := range tt.accounts {
				builder = builder.WithAccount(addr, account)
			}
			api := ethapi.NewPublicTxTraceAPI(builder.Build(), 100_000, 1000)

			for _, traceTypes := range [][]string{{"stateDiff"}, {"trace", "stateDiff"}} {
				res, err := api.Call(
					t.Context(),
					ethapi.TransactionArgs{
						From:  &sender,
						To:    tt.to,
						Gas:   ToHexUint64(1_000_000),
						Value: ToHexBigInt(big.NewInt(tt.value)),
						Data:  ToHexBytes(tt.data),
					},
					traceTypes,
					rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
					nil,
				)
				require.NoError(t, err)

				got, err := json.Marshal(res.StateDiff)
				require.NoError(t, err)
				require.JSONEq(t, tt.want, string(got), "trace types %v", traceTypes)
			}
		})
	}
}
