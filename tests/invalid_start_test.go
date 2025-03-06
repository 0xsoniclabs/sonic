package tests

import (
	"context"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/invalidstart"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestInvalidStart_IdentifiesInvalidStartContract(t *testing.T) {
	r := require.New(t)
	// byteccode source: https://eips.ethereum.org/EIPS/eip-3541#test-cases
	invalidCode := []byte{0x60, 0xef, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0xf3}
	validCode := []byte{0x60, 0xfe, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0xf3}

	net := StartIntegrationTestNet(t)

	// Deploy the invalid start contract.
	contract, _, err := DeployContract(net, invalidstart.DeployInvalidstart)
	r.NoError(err)

	// -- invalid codes

	// attempt to create a contract with code starting with 0xEF using CREATE
	receipt, err := net.Apply(contract.CreateContractWithInvalidCode)
	r.NoError(err)
	r.Equal(types.ReceiptStatusFailed, receipt.Status, "unexpected succeeded on invalid code with CREATE")

	// attempt to create a contract with code starting with 0xEF using CREATE2
	receipt, err = net.Apply(contract.Create2ContractWithInvalidCode)
	r.NoError(err)
	r.Equal(types.ReceiptStatusFailed, receipt.Status, "unexpected succeeded on invalid code with CREATE2")

	// attempt to run a transaction without receiver, with an invalid code.
	invalidTransaction, err := getTransactionWithCodeAndNoReceiver(r, invalidCode, net)
	r.NoError(err)
	receipt, err = net.Run(invalidTransaction)
	r.NoError(err)
	r.Equal(types.ReceiptStatusFailed, receipt.Status, "unexpected succeeded on transfer to empty receiver with invalid code")

	// -- valid codes

	// create a contract with valid code using CREATE
	receipt, err = net.Apply(contract.CreateContractWithValidCode)
	r.NoError(err)
	r.Equal(types.ReceiptStatusSuccessful, receipt.Status, "failed on valid code with CREATE")

	// create a contract with valid code using CREATE2
	receipt, err = net.Apply(contract.Create2ContractWithValidCode)
	r.NoError(err)
	r.Equal(types.ReceiptStatusSuccessful, receipt.Status, "failed on valid code with CREATE2")

	// run a transaction without receiver, with a valid code.
	validTransaction, err := getTransactionWithCodeAndNoReceiver(r, validCode, net)
	r.NoError(err)
	receipt, err = net.Run(validTransaction)
	r.NoError(err)
	r.Equal(types.ReceiptStatusSuccessful, receipt.Status, "failed on transfer to empty receiver with valid code")
}

func getTransactionWithCodeAndNoReceiver(r *require.Assertions, code []byte, net *IntegrationTestNet) (*types.Transaction, error) {
	// these values are needed for the transaction but are irrelevant for the test
	client, err := net.GetClient()
	r.NoError(err, "failed to connect to the network:")

	defer client.Close()
	chainId, err := client.ChainID(context.Background())
	r.NoError(err, "failed to get chain ID::")

	nonce, err := client.NonceAt(context.Background(), net.GetSessionSponsor().Address(), nil)
	r.NoError(err, "failed to get nonce:")

	price, err := client.SuggestGasPrice(context.Background())
	r.NoError(err, "failed to get gas price:")
	// ---------

	transaction, err := types.SignTx(types.NewTx(&types.AccessListTx{
		ChainID:  chainId,
		Gas:      500_000, // some gas that is big enough to run the code
		GasPrice: price,
		To:       nil,
		Nonce:    nonce,
		Data:     code,
	}), types.NewLondonSigner(chainId), net.GetSessionSponsor().PrivateKey)
	r.NoError(err, "failed to sign transaction:")

	return transaction, nil
}
