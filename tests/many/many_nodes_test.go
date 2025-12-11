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

package many

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/gossip/contract/sfc100"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/opera/contracts/sfc"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestManyNodes(t *testing.T) {

	validatorStake := []uint64{
		750, 750, // 75% of stake
		125, 125, 125, 125, // 25% of stake
	}

	net := tests.StartIntegrationTestNetWithJsonGenesis(t, tests.IntegrationTestNetOptions{
		ValidatorsStake: validatorStake,
		ClientExtraArguments: []string{
			"--emitter.throttle-events",
			"--emitter.throttle-skip-in-same-frame=10000",
			"--emitter.throttle-heartbeat-frames=10000",
		},
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	sfcContract, err := sfc100.NewContract(sfc.ContractAddress, client)
	require.NoError(t, err)

	epoch, err := sfcContract.CurrentEpoch(nil)
	require.NoError(t, err)

	ids, err := sfcContract.GetEpochValidatorIDs(nil, epoch)
	require.NoError(t, err)
	validators := make(map[idx.ValidatorID]*big.Int)
	for _, bigId := range ids {

		id := idx.ValidatorID(bigId.Uint64())

		key := makefakegenesis.FakeKey(id)
		delegator := crypto.PubkeyToAddress(key.PublicKey)
		stake, err := sfcContract.GetStake(nil, delegator, bigId)
		require.NoError(t, err)

		validators[id] = stake
		fmt.Println("validator", id, "stake", stake)
	}

	// First Epoch: initial staking, two validators dominate the stake

	time.Sleep(30 * time.Second)

	// Second Epoch: the suppressed validators increase their stake, all validators have equal stake

	for id := range validators {
		if id <= 2 {
			// skip the two dominating validators
			continue
		}

		receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
			opts.Value = utils.ToFtm(625)
			return sfcContract.Delegate(opts, big.NewInt(int64(id)))
		})
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	}
	net.AdvanceEpoch(t, 1)

	time.Sleep(30 * time.Second)

	//  Third Epoch: Increase state of one validators to dominate the stake again

	// id := idx.ValidatorID(3)
	// receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
	// 	opts.Value = utils.ToFtm(1_500)
	// 	return sfcContract.Delegate(opts, big.NewInt(int64(id)))
	// })
	// require.NoError(t, err)
	// require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	// net.AdvanceEpoch(t, 1)

	// time.Sleep(30 * time.Second)
}
