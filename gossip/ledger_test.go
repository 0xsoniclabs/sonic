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

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
)

// TestNet146Block8054923GasLimit_IsCanonicalHeaderValue pins the hard-coded gas
// limit to the canonical header value of net-146 block 8054923. This value is
// what protects historical byte-equivalence; it must never silently change.
func TestNet146Block8054923GasLimit_IsCanonicalHeaderValue(t *testing.T) {
	require.Equal(t, uint64(0x12a05f200), uint64(net146Block8054923GasLimit))
}

// TestBlockLedger_Finalize_AppliesNet146GasLimitException verifies that
// Finalize stamps the hard-coded net-146 gas limit for block 8054923 on network
// 146 regardless of the rules' MaxBlockGas, and leaves the announced gas limit
// untouched for any other block.
func TestBlockLedger_Finalize_AppliesNet146GasLimitException(t *testing.T) {
	const announcedGasLimit = uint64(123)

	tests := map[string]struct {
		networkID uint64
		number    uint64
		want      uint64
	}{
		"net-146 block 8054923 uses the hard-coded exception": {
			networkID: 146,
			number:    8054923,
			want:      net146Block8054923GasLimit,
		},
		"different block on net 146 keeps the announced limit": {
			networkID: 146,
			number:    8054924,
			want:      announcedGasLimit,
		},
		"same block number on a different network keeps the announced limit": {
			networkID: 250,
			number:    8054923,
			want:      announcedGasLimit,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)

			evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
			evmProcessor.EXPECT().Finalize().Return(&evmcore.EvmBlock{
				EvmHeader: evmcore.EvmHeader{BaseFee: big.NewInt(0)},
			}, 0, types.Receipts{})

			processor := &blockProcessor{
				evmProcessor: evmProcessor,
				blockBuilder: inter.NewBlockBuilder().
					WithNumber(test.number).
					WithGasLimit(announcedGasLimit),
				params: BlockParams{
					Number: test.number,
					// A MaxBlockGas distinct from the hard-coded constant proves
					// the stamped value comes from the constant, not the rules.
					Rules: opera.Rules{
						NetworkID: test.networkID,
						Blocks:    opera.BlocksRules{MaxBlockGas: net146Block8054923GasLimit + 1},
					},
				},
			}

			candidate := processor.Finalize()
			require.Equal(test.want, candidate.Block.GasLimit)
		})
	}
}
