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
	"context"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/contracts/basefee"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"
)

func TestBaseFee_CanReadBaseFeeFromHeadAndBlockAndHistory(t *testing.T) {
	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	t.Parallel()

	// Deploy the base fee contract.
	contract, deployReceipt, err := DeployContract(session, basefee.DeployBasefee)
	require.NoError(t, err)

	// Collect the current base fee from the head state.
	receipt, err := session.Apply(contract.LogCurrentBaseFee)
	require.NoError(t, err)
	require.Len(t, receipt.Logs, 1, "expected exactly one log entry for the base fee")

	entry, err := contract.ParseCurrentFee(*receipt.Logs[0])
	require.NoError(t, err)
	fromLog := entry.Fee

	// Collect the base fee from the block header.
	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(t, err)
	fromBlock := block.BaseFee()

	// Collect the base fee from the archive. A call that names no gas price
	// runs with a base fee of zero, as in go-ethereum, so the call is priced
	// to observe the block's base fee.
	priced, err := basefee.NewBasefeeCaller(
		deployReceipt.ContractAddress,
		pricedCaller{ContractCaller: client, gasPrice: fromBlock},
	)
	require.NoError(t, err)
	fromArchive, err := priced.GetBaseFee(&bind.CallOpts{
		BlockNumber: receipt.BlockNumber,
		From:        session.GetSessionSponsor().Address(),
	})
	require.NoError(t, err)

	// An unpriced call observes a base fee of zero.
	fromUnpricedCall, err := contract.GetBaseFee(&bind.CallOpts{BlockNumber: receipt.BlockNumber})
	require.NoError(t, err)

	require.Positive(t, fromLog.Int64(),
		"base fee should be non-negative",
	)
	require.Equal(t, fromLog, fromBlock,
		"base fee from log should match base fee from block header",
	)
	require.Equal(t, fromLog, fromArchive,
		"base fee from log should match base fee from archive",
	)
	require.Zero(t, fromUnpricedCall.Sign(),
		"base fee observed by an unpriced call should be zero",
	)
}

// pricedCaller is a bind.ContractCaller that runs every call with the given
// gas price. eth_call runs a call that names no gas price with a base fee of
// zero, so a call observing the block's base fee must be priced. The sender
// named in the call options must be able to pay for the gas limit.
type pricedCaller struct {
	bind.ContractCaller
	gasPrice *big.Int
}

func (c pricedCaller) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	call.GasPrice = c.gasPrice
	call.Gas = 1_000_000
	return c.ContractCaller.CallContract(ctx, call, blockNumber)
}
