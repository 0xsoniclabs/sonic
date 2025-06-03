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

	rpcClient := client.Client()
	defer rpcClient.Close()

	t.Run("Debug 7702 transaction with callTracer", func(t *testing.T) {
		debugTraceSponsoredTransaction(t, rpcClient, setCodeTx.Hash(), expectedAddress)
	})

	t.Run("Trace 7702 transaction", func(t *testing.T) {
		traceSponsoredTransaction(t, rpcClient, setCodeTx.Hash(), expectedAddress)
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

	require.Len(res.Calls, 1)
	require.Equal(expected.Sponsor, res.From)
	require.Equal(expected.Sponsored, res.To)
	require.Equal(expected.Sponsored, res.Calls[0].From)
	require.Equal(expected.CalledContract, res.Calls[0].To)
}

func traceSponsoredTransaction(t *testing.T, rpcClient *rpc.Client, txHash common.Hash, expected calledAddresses) {
	require := require.New(t)

	type traceResult []struct {
		Action struct {
			From common.Address `json:"from"`
			To   common.Address `json:"to"`
		} `json:"action"`
		TraceAddress []int `json:"traceAddress"`
		Subtraces    int   `json:"subtraces"`
	}

	var res traceResult
	err := rpcClient.Call(&res, "trace_transaction", txHash)
	require.NoError(err, "failed to call trace_transaction; %v", err)

	// There should be two inner contract calls
	require.Len(res, 2)

	// Check the first call
	require.Equal(res[0].Subtraces, 1)
	require.Len(res[0].TraceAddress, 0)
	require.Equal(expected.Sponsor, res[0].Action.From)
	require.Equal(expected.Sponsored, res[0].Action.To)

	// Check the second call
	require.Equal(res[1].Subtraces, 0)
	require.Len(res[1].TraceAddress, 1)
	require.Equal(res[1].TraceAddress[0], 0)
	require.Equal(expected.Sponsored, res[1].Action.From)
	require.Equal(expected.CalledContract, res[1].Action.To)
}
