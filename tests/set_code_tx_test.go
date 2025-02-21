package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestSetCodeTransaction(t *testing.T) {
	net, err := StartIntegrationTestNet(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to start the fake network: %v", err)
	}
	defer net.Stop()
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
	txOpts, err := net.GetTransactOptions(&net.validator)
	require.NoError(t, err)
	tx, err := counter.IncrementCounter(txOpts)
	require.NoError(t, err)
	callData := tx.Data()

	// Create a setCode transaction calling the incrementCounter function
	// in the context of the sponsored account.
	setCodeTx := makeEip7702Transaction(t, client, sponsor, sponsored, delegate, callData)
	client.Close()
	receipt, err = net.Run(setCodeTx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// read code at sponsored address, must contain the delegate address
	code, err := client.CodeAt(context.Background(), sponsored.Address(), nil)
	require.NoError(t, err)
	expectedCode := append([]byte{0xef, 0x01, 0x00}, delegate[:]...)
	require.Equal(t, expectedCode, code, "code in account is expected to be delegation designation")

	// read storage at sponsored address (instead of contract address as in a normal tx)
	// counter must exist and be 1
	data, err := client.StorageAt(context.Background(), sponsored.Address(), common.Hash{}, nil)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(1), new(big.Int).SetBytes(data), "unexpected storage value")
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

	authorization, err := types.SignSetCode(sponsored.PrivateKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(chainId),
		Address: delegate,
		Nonce:   sponsoredNonce,
	})
	require.NoError(t, err, "failed to sign SetCode authorization")

	tx := types.NewTx(&types.SetCodeTx{
		ChainID: uint256.MustFromBig(chainId),
		Nonce:   sponsorNonce,
		To:      sponsored.Address(),
		Gas: 25_000 + // One entry in auth list
			21_000 + // Base for not creating a contract
			2400*2 + // Two addresses in access list
			22_100 + // store cold data
			5000, // some extra gas gas for other opcodes

		GasFeeCap: uint256.NewInt(10e10),
		AccessList: types.AccessList{
			{Address: sponsored.Address()},
			{Address: delegate},
		},
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
