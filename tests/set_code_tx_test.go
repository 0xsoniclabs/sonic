package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/batch"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetCodeTransaction(t *testing.T) {

	net, err := StartIntegrationTestNet(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to start the fake network: %v", err)
	}
	defer net.Stop()

	t.Run("Transaction Sponsoring", func(t *testing.T) {
		testSponsoring(t, net)
	})

	t.Run("Transaction Batching", func(t *testing.T) {
		testBatching(t, net)
	})
}

func testSponsoring(t *testing.T, net *IntegrationTestNet) {
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// sponsor issues the SetCode transaction and pays for it
	sponsor := makeAccountWithBalance(t, net, 1e18)
	// sponsored is used as context for the call, its state will be modified
	// without paying for the transaction
	sponsored := makeAccountWithBalance(t, net, 0)

	// Deploy the a contract to use as delegate
	counter, receipt, err := DeployContract(net, counter.DeployCounter)
	require.NoError(t, err)
	delegate := receipt.ContractAddress

	// Extract the call data of a normal call to the delegate contract
	// to know the ABI encoding of the callData
	callData := getCallData(t, net, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return counter.IncrementCounter(opts)
	})

	// Create a setCode transaction calling the incrementCounter function
	// in the context of the sponsored account.
	setCodeTx := makeEip7702Transaction(t, client, sponsor, sponsored, delegate, callData)
	client.Close()
	receipt, err = net.Run(setCodeTx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Read code at sponsored address, must contain the delegate address
	code, err := client.CodeAt(context.Background(), sponsored.Address(), nil)
	require.NoError(t, err)
	expectedCode := append([]byte{0xef, 0x01, 0x00}, delegate[:]...)
	require.Equal(t, expectedCode, code, "code in account is expected to be delegation designation")

	// Read storage at sponsored address (instead of contract address as in a normal tx)
	// counter must exist and be 1
	data, err := client.StorageAt(context.Background(), sponsored.Address(), common.Hash{}, nil)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(1), new(big.Int).SetBytes(data), "unexpected storage value")
}

func testBatching(t *testing.T, net *IntegrationTestNet) {
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// sender account batches multiple transfers of funds in a single transaction
	// receivers will receive the funds
	sender := makeAccountWithBalance(t, net, 1e18)
	receiver1 := makeAccountWithBalance(t, net, 0)
	receiver2 := makeAccountWithBalance(t, net, 0)

	batchContract, deployReceipt, err := DeployContract(net, batch.DeployBatch)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, deployReceipt.Status)
	batchContractAddress := deployReceipt.ContractAddress

	// Extract the call data of a normal call to the delegate contract
	// to know the ABI encoding of the callData.
	// This code creates the Batch of calls, which the batch contract will execute
	callData := getCallData(t, net, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return batchContract.Execute(opts, []batch.BatchCallDelegationCall{
			{
				To:    receiver1.Address(),
				Value: big.NewInt(1234),
			},
			{
				To:    receiver2.Address(),
				Value: big.NewInt(4321),
			},
		})
	})

	// Send a SetCode transaction to the batch contract
	tx := makeEip7702Transaction(t, client, sender, sender, batchContractAddress, callData)
	batchReceipt, err := net.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, batchReceipt.Status)

	// Check that the sender has paid for the transaction
	effectiveCost := new(big.Int)
	effectiveCost = effectiveCost.Mul(
		batchReceipt.EffectiveGasPrice,
		big.NewInt(int64(batchReceipt.GasUsed)))
	effectiveCost = effectiveCost.Add(effectiveCost, big.NewInt(1234+4321))

	balance, err := client.BalanceAt(context.Background(), sender.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t,
		new(big.Int).Sub(
			big.NewInt(1e18), effectiveCost), balance)

	// Check that the receivers have received the funds
	balance1, err := client.BalanceAt(context.Background(), receiver1.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(1234), balance1)

	balance2, err := client.BalanceAt(context.Background(), receiver2.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(4321), balance2)
}

// makeLegacyTx creates a legacy transaction from a CallMsg, filling in the nonce
// and gas limit.
func makeEip7702Transaction(t *testing.T,
	client *ethclient.Client,
	sponsor *Account, // signs and pays for the tx
	sponsored *Account, // the account where the delegator will be written in
	delegate common.Address, // the address of the delegate contract
	callData []byte,
) *types.Transaction {
	t.Helper()

	chainId, err := client.ChainID(context.Background())
	require.NoError(t, err, "failed to get chain ID")

	sponsoredNonce, err := client.NonceAt(context.Background(), sponsored.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", sponsored.Address())

	sponsorNonce, err := client.NonceAt(context.Background(), sponsor.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", sponsor.Address())

	// If self sponsored, there are two nonces values to take care of, the transaction
	// nonce and the authorization nonce. The authorization nonce is checked after
	// the transaction has incremented nonce. Therefore, the authorization nonce
	// needs to be 1 higher than the transaction nonce.
	nonceIncrement := uint64(0)
	if sponsor == sponsored {
		nonceIncrement = 1
	}

	authorization, err := types.SignSetCode(sponsored.PrivateKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(chainId),
		Address: delegate,
		Nonce:   sponsoredNonce + nonceIncrement,
	})
	require.NoError(t, err, "failed to sign SetCode authorization")

	tx := types.NewTx(&types.SetCodeTx{
		ChainID:   uint256.MustFromBig(chainId),
		Nonce:     sponsorNonce,
		To:        sponsored.Address(),
		Gas:       150_000,
		GasFeeCap: uint256.NewInt(10e10),
		AuthList: []types.SetCodeAuthorization{
			authorization,
		},
		Data: callData,
	})

	signer := types.NewPragueSigner(chainId)
	tx, err = types.SignTx(tx, signer, sponsor.PrivateKey)
	require.NoError(t, err, "failed to sign transaction")
	return tx
}

// getCallData creates a transaction and returns the data field of the transaction.
// This function can be used to retrieve the ABI encoding of a the call data, and
// use such encoding to create a SetCode transaction.
func getCallData(t *testing.T, net *IntegrationTestNet,
	transactionConstructor func(*bind.TransactOpts) (*types.Transaction, error)) []byte {
	txOpts, err := net.GetTransactOptions(&net.validator)
	require.NoError(t, err)
	txOpts.NoSend = true // <- create the transaction to read callData, but do not send it.
	tx, err := transactionConstructor(txOpts)
	require.NoError(t, err)
	return tx.Data()
}
