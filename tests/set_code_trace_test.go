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

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/0xsoniclabs/sonic/tests/contracts/sponsoring"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// TestTrace7702Transaction tests the transaction trace and debug callTracer
// using a sponsoring delegate calling a simple counter contract
// which act as a dApp, so it can verify, that FeeM will be able to
// assign fees for a dApp also in this delegate scenario and dApp
// address will be visible in the trace
func TestTrace7702Transaction(t *testing.T) {
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		Upgrades: AsPointer(opera.GetAllegroUpgrades()),
	})

	sponsor := makeAccountWithBalance(t, net, big.NewInt(1e18))
	sponsored := makeAccountWithBalance(t, net, big.NewInt(10))

	// Deploy the contract to forward the call
	sponsoringDelegate, receipt, err := DeployContract(net, sponsoring.DeploySponsoring)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	delegateAddress := receipt.ContractAddress

	// Deploy simple contract to increment the counter
	counterContract, receipt, err := DeployContract(net, counter.DeployCounter)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	counterAddress := receipt.ContractAddress

	// Prepare calldata for incrementing the counter
	counterCallData := getCallData(t, net, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return counterContract.IncrementCounter(opts)
	})

	// Prepare calldata for the sponsoring transaction
	sponsoringCallData := getCallData(t, net, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		// Increment the counter in the context of the sponsored account
		return sponsoringDelegate.Execute(opts, counterAddress, big.NewInt(0), counterCallData)
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Create a setCode transaction calling the counter contract
	setCodeTx := makeEip7702Transaction(t, client, sponsor, sponsored, delegateAddress, sponsoringCallData)
	receipt, err = net.Run(setCodeTx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	expectedAddress := calledAddresses{
		Sponsor:        sponsor.Address(),
		Sponsored:      sponsored.Address(),
		CalledContract: counterAddress,
	}

	t.Run("Debug 7702 transaction with callTracer", func(t *testing.T) {
		debugTraceSponsoredTransaction(t, client.Client(), setCodeTx.Hash(), expectedAddress)
	})

	t.Run("Trace 7702 transaction", func(t *testing.T) {
		traceSponsoredTransaction(t, client.Client(), setCodeTx.Hash(), expectedAddress)
	})
}

type calledAddresses struct {
	Sponsor        common.Address
	Sponsored      common.Address
	CalledContract common.Address
}

func debugTraceSponsoredTransaction(t *testing.T, rpcClient *rpc.Client, txHash common.Hash, expected calledAddresses) {
	require := require.New(t)

	tracer := "callTracer"
	traceConfig := &ethapi.TraceCallConfig{
		TraceConfig: tracers.TraceConfig{
			Tracer: &tracer,
		},
	}
	type Calls struct {
		From  common.Address `json:"from"`
		To    common.Address `json:"to"`
		Calls []Calls        `json:"calls"`
	}

	var res Calls
	err := rpcClient.Call(&res, "debug_traceTransaction", txHash, traceConfig)
	require.NoError(err, "failed to call debug_traceTransaction; %v", err)

	// Debug callTracer preserves hierarchical structure of the calls.
	// Root call is the sponsoring transaction from the sponsor targeting the sponsored contract
	// in code of sponsored EOA. Then the sponsoring contract in code of sponsored EOA calls
	// the increment function of counter contract, which acts as a dApp contract.
	require.Len(res.Calls, 1)
	// Root call
	require.Equal(expected.Sponsor, res.From)
	require.Equal(expected.Sponsored, res.To)
	// Inner call
	require.Equal(expected.Sponsored, res.Calls[0].From)
	require.Equal(expected.CalledContract, res.Calls[0].To)
}

func traceSponsoredTransaction(t *testing.T, rpcClient *rpc.Client, txHash common.Hash, expected calledAddresses) {
	require := require.New(t)

	type traceAction struct {
		From common.Address `json:"from"`
		To   common.Address `json:"to"`
	}
	type trace struct {
		Action       traceAction `json:"action"`
		TraceAddress []int       `json:"traceAddress"`
		Subtraces    int         `json:"subtraces"`
	}

	var traces []trace
	err := rpcClient.Call(&traces, "trace_transaction", txHash)
	require.NoError(err, "failed to call trace_transaction; %v", err)

	// Transaction tracing is not preserving hierarchical structure of the calls.
	// Each call has a traceAddress, which contains the index of the call
	// and subtraces count of the nested contract calls.

	// There should be two contract calls for this transaction
	// and they don't need to be in order
	require.Len(traces, 2)

	// First call is the sponsoring transaction targeting the sponsored contract
	// in code of sponsored EOA. It is a root trace so the traceAddress is empty
	// and has 1 subtrace
	require.Contains(traces, trace{
		Action: traceAction{
			From: expected.Sponsor,
			To:   expected.Sponsored,
		},
		TraceAddress: []int{},
		Subtraces:    1,
	})

	// Second call is the sponsoring contract in code of sponsored EOA
	// calling the increment function of counter contract,
	// which acts as a dApp contract. This trace is a first child of a root trace,
	// so the traceAddress is [0] and has 0 subtraces as there are no other nested calls
	require.Contains(traces, trace{
		Action: traceAction{
			From: expected.Sponsored,
			To:   expected.CalledContract,
		},
		TraceAddress: []int{0},
		Subtraces:    0,
	})
}
