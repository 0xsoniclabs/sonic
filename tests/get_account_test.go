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

package tests

import (
	"testing"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestGetAccount(t *testing.T) {
	net := StartIntegrationTestNet(t)

	// Deploy the transient storage contract
	_, deployReceipt, err := DeployContract(net, counter.DeployCounter)
	require.NoError(t, err, "failed to deploy contract")

	addr := deployReceipt.ContractAddress

	c, err := net.GetClient()
	require.NoError(t, err, "failed to get client")
	defer c.Close()

	rpcClient := c.Client()
	defer rpcClient.Close()

	var res ethapi.GetAccountResult
	err = rpcClient.Call(&res, "eth_getAccount", addr, rpc.LatestBlockNumber)
	require.NoError(t, err, "failed to call get account")

	// Extract proof to find actual StorageHash(Root), Nonce, Balance and CodeHash
	var proofRes struct {
		StorageHash common.Hash
		Nonce       hexutil.Uint64
		Balance     *hexutil.U256
		CodeHash    common.Hash
	}
	err = rpcClient.Call(
		&proofRes,
		"eth_getProof",
		addr,
		nil,
		rpc.LatestBlockNumber,
	)
	require.NoError(t, err, "failed call to get proof")

	require.Equal(t, proofRes.CodeHash, res.CodeHash)
	require.Equal(t, proofRes.StorageHash, res.StorageRoot)
	require.Equal(t, proofRes.Balance, res.Balance)
	require.Equal(t, proofRes.Nonce, res.Nonce)
}
