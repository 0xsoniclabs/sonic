package tests

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGasSubsidies_CanRunSubsidizedTransactions(t *testing.T) {
	require := require.New(t)

	upgrades := opera.GetSonicUpgrades()
	upgrades.GasSubsidies = true

	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		ClientExtraArguments: []string{
			"--allow-zero-chainid-txs",
		},
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	// --- deploy the subsidies registry contract ---

	// check that the contract is not deployed yet
	registryAddress := registry.GetAddress()
	nonce, err := client.NonceAt(t.Context(), registryAddress, nil)
	require.NoError(err)
	require.Equal(uint64(0), nonce)

	// Deploy the subsidies registry contract.
	tx, creator := registry.GetDeploymentTransaction()

	receipt, err := net.EndowAccount(creator, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(100)))
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	require.NoError(client.SendTransaction(t.Context(), tx))

	receipt, err = net.GetReceipt(tx.Hash())
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// check that the contract is deployed
	nonce, err = client.NonceAt(t.Context(), registryAddress, nil)
	require.NoError(err)
	require.Equal(uint64(1), nonce)
	code, err := client.CodeAt(t.Context(), registryAddress, nil)
	require.NoError(err)
	require.NotEmpty(code)
	require.Equal(registry.GetCode(), code)

	// -------------------------------------------------------------------------

	sponsor := NewAccount()
	sponsee := NewAccount()
	receiver := NewAccount()

	// Before the sponsorship is set up, a transaction from the sponsee
	// to the receiver should fail due to lack of funds.
	chainId := net.GetChainId()
	receiverAddress := receiver.Address()
	signer := types.LatestSignerForChainID(chainId)
	tx, err = types.SignNewTx(sponsee.PrivateKey, signer, &types.LegacyTx{
		To:       &receiverAddress,
		Gas:      21000,
		GasPrice: big.NewInt(0),
	})
	require.NoError(err)
	require.Error(
		client.SendTransaction(t.Context(), tx),
		"should be rejected due to lack of funds and no sponsorship",
	)

	// --- deposit sponsorship funds ---

	registry, err := registry.NewRegistry(registry.GetAddress(), client)
	require.NoError(err)

	receipt, err = net.EndowAccount(sponsor.Address(), big.NewInt(1e18))
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	receipt, err = net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		opts.Value = big.NewInt(1e16)
		return registry.SponsorUser(opts, sponsee.Address(), receiver.Address())
	})
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// check that the sponsorship funds got deposited
	sponsorship, err := registry.UserSponsorships(nil, sponsee.Address(), receiver.Address())
	require.NoError(err)
	require.Equal(big.NewInt(1e16), sponsorship.Funds)

	// --- try to submit a sponsored transaction ---

	// Now it should be possible to submit the transaction from the sponsee.
	require.NoError(client.SendTransaction(t.Context(), tx))

	// Wait for the sponsored transaction to be executed.
	receipt, err = net.GetReceipt(tx.Hash())
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// TODO:
	// - check the effects of the sponsored transaction
}

// TODO: test the following properties
//  - sponsorship requests work with all types of transactions (legacy, dynamic fee, etc.)
//  - check the enforcement of the GasSponsorship flag in the network rules
//  - check that the sponsorship funds are correctly deducted after a sponsored tx
//  - check that the sponsorship request is rejected if there are insufficient funds
//  - check that the sponsorship request is rejected if the registry contract is not deployed
