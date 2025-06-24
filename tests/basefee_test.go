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

	"github.com/0xsoniclabs/sonic/tests/contracts/basefee"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

func TestBaseFee_CanReadBaseFeeFromHeadAndBlockAndHistory(t *testing.T) {
	net := StartIntegrationTestNet(t)

	// Deploy the base fee contract.
	contract, _, err := DeployContract(net, basefee.DeployBasefee)
	if err != nil {
		t.Fatalf("failed to deploy contract; %v", err)
	}

	// Collect the current base fee from the head state.
	receipt, err := net.Apply(contract.LogCurrentBaseFee)
	if err != nil {
		t.Fatalf("failed to log current base fee; %v", err)
	}

	if len(receipt.Logs) != 1 {
		t.Fatalf("unexpected number of logs; expected 1, got %d", len(receipt.Logs))
	}

	entry, err := contract.ParseCurrentFee(*receipt.Logs[0])
	if err != nil {
		t.Fatalf("failed to parse log; %v", err)
	}
	fromLog := entry.Fee

	// Collect the base fee from the block header.
	client, err := net.GetClient()
	if err != nil {
		t.Fatalf("failed to get client; %v", err)
	}
	defer client.Close()

	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	if err != nil {
		t.Fatalf("failed to get block header; %v", err)
	}
	fromBlock := block.BaseFee()

	// Collect the base fee from the archive.
	fromArchive, err := contract.GetBaseFee(&bind.CallOpts{BlockNumber: receipt.BlockNumber})
	if err != nil {
		t.Fatalf("failed to get base fee from archive; %v", err)
	}

	if fromLog.Sign() < 1 {
		t.Fatalf("invalid base fee from log; %v", fromLog)
	}

	if fromLog.Cmp(fromBlock) != 0 {
		t.Fatalf("base fee mismatch; from log %v, from block %v", fromLog, fromBlock)
	}
	if fromLog.Cmp(fromArchive) != 0 {
		t.Fatalf("base fee mismatch; from log %v, from archive %v", fromLog, fromArchive)
	}
}
