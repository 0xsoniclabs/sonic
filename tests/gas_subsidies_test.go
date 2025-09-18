package tests

import (
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
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
	donation := big.NewInt(1e16)
	sponsorship, err := registry.UserSponsorships(nil, sponsee.Address(), receiver.Address())
	require.NoError(err)
	require.Equal(donation, sponsorship.Funds)

	// --- try to submit a sponsored transaction ---

	burnedBefore, err := client.BalanceAt(t.Context(), common.Address{}, nil)
	require.NoError(err)

	// Now it should be possible to submit the transaction from the sponsee.
	require.NoError(client.SendTransaction(t.Context(), tx))

	// Wait for the sponsored transaction to be executed.
	receipt, err = net.GetReceipt(tx.Hash())
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(err)
	require.True(slices.ContainsFunc(
		block.Transactions(),
		func(cur *types.Transaction) bool {
			return cur.Hash() == tx.Hash()
		},
	))

	// Check that the payment transaction is included right after the sponsored
	// transaction and that it was successful and has a non-zero value.
	found := false
	for i, tx := range block.Transactions() {
		if tx.Hash() == receipt.TxHash {
			require.Less(i, len(block.Transactions()))
			payment := block.Transactions()[i+1]
			receipt, err := net.GetReceipt(payment.Hash())
			require.NoError(err)
			require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
			found = true
			break
		}
	}
	require.True(found, "sponsored transaction not found in the block")

	// check that the sponsorship funds got deducted
	ops := &bind.CallOpts{
		BlockNumber: receipt.BlockNumber,
	}
	sponsorship, err = registry.UserSponsorships(ops, sponsee.Address(), receiver.Address())
	require.NoError(err)
	require.Less(sponsorship.Funds.Uint64(), donation.Uint64())

	// the different in the sponsorship funds should have been burned
	burnedAfter, err := client.BalanceAt(t.Context(), common.Address{}, nil)
	require.NoError(err)
	require.Greater(burnedAfter.Uint64(), burnedBefore.Uint64())

	// the sponsorship difference and the increase in burned funds should be equal
	diff := new(big.Int).Sub(burnedAfter, burnedBefore)
	reduced := new(big.Int).Sub(donation, sponsorship.Funds)
	require.Equal(0, diff.Cmp(reduced),
		"the burned amount should equal the reduction of the sponsorship funds",
	)
}

// TODO: test the following properties
//  - sponsorship requests work with all types of transactions (legacy, dynamic fee, etc.)
//  - check the enforcement of the GasSponsorship flag in the network rules
//  - check that the sponsorship funds are correctly deducted after a sponsored tx
//  - check that the sponsorship request is rejected if there are insufficient funds
//  - check that the sponsorship request is rejected if the registry contract is not deployed
//  - test that fee charging transactions and sealing transactions use proper nonces (incrementally, no gaps)
