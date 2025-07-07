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

package gossip

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/gossip/contract/ballot"
	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/utils"
)

func BenchmarkBallotTxsProcessing(b *testing.B) {
	logger.SetLevel("warn")
	logger.SetTestMode(b)
	require := require.New(b)

	env := newTestEnv(2, 3, b)
	b.Cleanup(func() {
		err := env.Close()
		require.NoError(err)
	})

	for bi := 0; bi < b.N; bi++ {
		count := idx.ValidatorID(10)

		proposals := [][32]byte{
			ballotOption("Option 1"),
			ballotOption("Option 2"),
			ballotOption("Option 3"),
		}

		// contract deploy
		addr, tx, cBallot, err := ballot.DeployBallot(env.Pay(1), env, proposals)
		require.NoError(err)
		require.NotNil(cBallot)
		r, err := env.ApplyTxs(nextEpoch, tx)
		require.NoError(err)

		require.Equal(addr, r[0].ContractAddress)

		admin, err := cBallot.Chairperson(env.ReadOnly())
		require.NoError(err)
		require.Equal(env.Address(1), admin)

		txs := make([]*types.Transaction, 0, count-1)
		flushTxs := func() {
			if len(txs) != 0 {
				_, err := env.ApplyTxs(nextEpoch, txs...)
				require.NoError(err, "failed to apply txs")
			}
			txs = txs[:0]
		}

		// Init accounts
		for vid := idx.ValidatorID(2); vid <= count; vid++ {
			tx := env.Transfer(1, vid, utils.ToFtm(10))
			txs = append(txs, tx)
			if len(txs) > 2 {
				flushTxs()
			}
		}
		flushTxs()

		// GiveRightToVote
		for vid := idx.ValidatorID(1); vid <= count; vid++ {
			tx, err := cBallot.GiveRightToVote(env.Pay(1), env.Address(vid))
			require.NoError(err)
			txs = append(txs, tx)
			if len(txs) > 2 {
				flushTxs()
			}
		}
		flushTxs()

		// Vote
		for vid := idx.ValidatorID(1); vid <= count; vid++ {
			proposal := big.NewInt(int64(vid) % int64(len(proposals)))
			tx, err := cBallot.Vote(env.Pay(vid), proposal)
			require.NoError(err)
			txs = append(txs, tx)
			if len(txs) > 2 {
				flushTxs()
			}
		}
		flushTxs()

		// Winner
		_, err = cBallot.WinnerName(env.ReadOnly())
		require.NoError(err)
	}
}

func ballotOption(str string) (res [32]byte) {
	buf := []byte(str)
	if len(buf) > 32 {
		panic("string too long")
	}
	copy(res[:], buf)
	return
}
