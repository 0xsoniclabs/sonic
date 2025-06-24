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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/prevrandao"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

func TestPrevRandao(t *testing.T) {
	net := StartIntegrationTestNet(t)

	// Deploy the contract.
	contract, _, err := DeployContract(net, prevrandao.DeployPrevrandao)
	if err != nil {
		t.Fatalf("failed to deploy contract; %v", err)
	}
	// Collect the current PrevRandao fee from the head state.
	receipt, err := net.Apply(contract.LogCurrentPrevRandao)
	if err != nil {
		t.Fatalf("failed to log current prevrandao; %v", err)
	}
	if len(receipt.Logs) != 1 {
		t.Fatalf("unexpected number of logs; expected 1, got %d", len(receipt.Logs))
	}
	entry, err := contract.ParseCurrentPrevRandao(*receipt.Logs[0])
	if err != nil {
		t.Fatalf("failed to parse log; %v", err)
	}
	fromLog := entry.Prevrandao

	client, err := net.GetClient()
	if err != nil {
		t.Fatalf("failed to get client; %v", err)
	}
	defer client.Close()
	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	if err != nil {
		t.Fatalf("failed to get block header; %v", err)
	}
	fromLatestBlock := block.MixDigest().Big() // MixDigest == MixHash == PrevRandao
	if block.Difficulty().Uint64() != 0 {
		t.Errorf("incorrect header difficulty got: %d, want: %d", block.Difficulty().Uint64(), 0)
	}
	// Collect the prevrandao from the archive.
	fromArchive, err := contract.GetPrevRandao(&bind.CallOpts{BlockNumber: receipt.BlockNumber})
	if err != nil {
		t.Fatalf("failed to get prevrandao from archive; %v", err)
	}
	if fromLog.Sign() < 1 {
		t.Fatalf("invalid prevrandao from log; %v", fromLog)
	}

	if fromLog.Cmp(fromLatestBlock) != 0 {
		t.Errorf("prevrandao mismatch; from log %v, from block %v", fromLog, fromLatestBlock)
	}
	if fromLog.Cmp(fromArchive) != 0 {
		t.Errorf("prevrandao mismatch; from log %v, from archive %v", fromLog, fromArchive)
	}

	fromSecondLastBlock, err := contract.GetPrevRandao(&bind.CallOpts{BlockNumber: big.NewInt(receipt.BlockNumber.Int64() - 1)})
	if err != nil {
		t.Fatalf("failed to get prevrandao from archive; %v", err)
	}

	if fromSecondLastBlock.Cmp(fromLatestBlock) == 0 {
		t.Errorf("prevrandao must be different for each block, found same: %s, %s", fromSecondLastBlock, fromLatestBlock)
	}
}
