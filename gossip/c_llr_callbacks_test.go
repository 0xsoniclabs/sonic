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

package gossip

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/ibr"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func TestWriteFullBlockRecord_GasUsedOverridesSurviveGenesisExportAndImport(t *testing.T) {
	tests := map[string]struct {
		gasUsed           []uint64
		cumulativeGasUsed []uint64
		overrides         []ibr.GasUsedOverride
		staleOverrides    []ibr.GasUsedOverride // present in the importing store
	}{
		"no surplus": {
			gasUsed:           []uint64{10, 20},
			cumulativeGasUsed: []uint64{10, 30},
		},
		"surplus before first receipt": {
			gasUsed:           []uint64{10, 20},
			cumulativeGasUsed: []uint64{110, 130},
			overrides:         []ibr.GasUsedOverride{{Index: 0, GasUsed: 10}},
		},
		"surplus between receipts": {
			gasUsed:           []uint64{10, 20},
			cumulativeGasUsed: []uint64{10, 130},
			overrides:         []ibr.GasUsedOverride{{Index: 1, GasUsed: 20}},
		},
		"multiple surpluses": {
			gasUsed:           []uint64{10, 20, 30, 40},
			cumulativeGasUsed: []uint64{10, 80, 110, 190},
			overrides: []ibr.GasUsedOverride{
				{Index: 1, GasUsed: 20},
				{Index: 3, GasUsed: 40},
			},
		},
		"stale overrides of the importing store are dropped": {
			gasUsed:           []uint64{10, 20},
			cumulativeGasUsed: []uint64{10, 30},
			staleOverrides:    []ibr.GasUsedOverride{{Index: 1, GasUsed: 1}},
		},
		"stale overrides of the importing store are replaced": {
			gasUsed:           []uint64{10, 20},
			cumulativeGasUsed: []uint64{10, 130},
			overrides:         []ibr.GasUsedOverride{{Index: 1, GasUsed: 20}},
			staleOverrides:    []ibr.GasUsedOverride{{Index: 0, GasUsed: 1}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			const blockIdx = idx.Block(7)
			epochState := iblockproc.EpochState{
				Rules: opera.FakeNetRules(opera.GetSonicUpgrades()),
			}
			to := common.Address{1}
			receipts := make(types.Receipts, len(test.gasUsed))
			txs := make(types.Transactions, len(test.gasUsed))
			builder := inter.NewBlockBuilder().
				WithNumber(uint64(blockIdx)).
				WithTime(inter.FromUnix(1_000)).
				WithGasLimit(1_000_000).
				WithGasUsed(test.cumulativeGasUsed[len(test.cumulativeGasUsed)-1]).
				WithBaseFee(big.NewInt(1))
			for i := range txs {
				txs[i] = types.NewTx(&types.LegacyTx{To: &to, Nonce: uint64(i)})
				receipts[i] = &types.Receipt{
					Type:              types.LegacyTxType,
					GasUsed:           test.gasUsed[i],
					CumulativeGasUsed: test.cumulativeGasUsed[i],
				}
				builder.AddTransaction(txs[i], receipts[i])
			}
			block := builder.Build()

			// Source store, as it results from processing the block.
			source, err := NewMemStore(t)
			require.NoError(t, err)
			source.SetBlockEpochState(iblockproc.BlockState{}, epochState)
			for _, tx := range txs {
				source.EvmStore().SetTx(tx.Hash(), tx)
			}
			source.EvmStore().SetReceipts(blockIdx, receipts)
			source.SetBlock(blockIdx, block)

			// Export the block and pass it through its genesis serialization.
			record := source.GetFullBlockRecord(blockIdx)
			require.NotNil(t, record)
			require.Equal(t, test.overrides, record.GasUsedOverrides)
			encoded, err := rlp.EncodeToBytes(ibr.LlrIdxFullBlockRecord{
				LlrFullBlockRecord: *record,
				Idx:                blockIdx,
			})
			require.NoError(t, err)
			var imported ibr.LlrIdxFullBlockRecord
			require.NoError(t, rlp.DecodeBytes(encoded, &imported))

			// Import into a fresh store.
			target, err := NewMemStore(t)
			require.NoError(t, err)
			target.SetBlockEpochState(iblockproc.BlockState{}, epochState)
			if test.staleOverrides != nil {
				target.EvmStore().SetGasUsedOverrides(blockIdx, test.staleOverrides)
			}
			require.NoError(t, target.WriteFullBlockRecord(imported))

			got := target.EvmStore().GetReceipts(
				blockIdx, target.GetEvmChainConfig(blockIdx),
				block.Hash(), uint64(block.Time.Unix()), block.BaseFee, nil, txs,
			)
			require.Len(t, got, len(receipts))
			for i := range receipts {
				require.Equal(t, test.gasUsed[i], got[i].GasUsed, "receipt %d", i)
				require.Equal(t, test.cumulativeGasUsed[i], got[i].CumulativeGasUsed, "receipt %d", i)
			}
			require.Equal(t, test.overrides, target.EvmStore().GetGasUsedOverrides(blockIdx))
		})
	}
}
