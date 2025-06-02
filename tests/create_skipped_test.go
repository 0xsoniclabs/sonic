package tests

import (
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestAccountCreation_CreateCallsWithInitCodesTooLargeDoNotAlterBalance(t *testing.T) {
	versions := map[string]opera.Upgrades{
		"sonic":   opera.GetSonicUpgrades(),
		"allegro": opera.GetAllegroUpgrades(),
	}

	for name, version := range versions {
		t.Run(name, func(t *testing.T) {
			net := StartIntegrationTestNetWithJsonGenesis(t, IntegrationTestNetOptions{
				Upgrades: &version,
				ClientExtraArguments: []string{
					"--disable-txPool-validation",
				},
			})

			client, err := net.GetClient()
			require.NoError(t, err)
			defer client.Close()

			sender := makeAccountWithBalance(t, net, big.NewInt(1e18))

			gasPrice, err := client.SuggestGasPrice(t.Context())
			require.NoError(t, err)

			chainId, err := client.ChainID(t.Context())
			require.NoError(t, err)

			initCode := make([]byte, 50000)
			txData := &types.LegacyTx{
				Nonce:    0,
				Gas:      10000000,
				GasPrice: gasPrice,
				To:       nil, // address 0x00 for contract creation
				Value:    big.NewInt(0),
				Data:     initCode,
			}
			tx := signTransaction(t, chainId, txData, sender)

			// Check balance before sending the transaction
			preBalance, err := client.BalanceAt(t.Context(), sender.Address(), nil)
			require.NoError(t, err)

			// Send the transaction
			err = client.SendTransaction(t.Context(), tx)
			require.NoError(t, err)

			// Send another simple transaction to ensure a block is created
			receipt, err := net.EndowAccount(common.Address{0x42}, big.NewInt(42))
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

			// Check that the simple transaction was included in the block
			blockTransaction, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
			require.NoError(t, err)
			contains := slices.ContainsFunc(blockTransaction.Transactions(), func(tx *types.Transaction) bool {
				return tx.Hash() == receipt.TxHash
			})
			require.True(t, contains, "transaction should be included in the block")

			// Check that no other transactions were included in the block
			require.Len(t, blockTransaction.Transactions(), 1, "block should contain exactly one transaction")

			// Check balance after sending the transaction
			postBalance, err := client.BalanceAt(t.Context(), sender.Address(), receipt.BlockNumber)
			require.NoError(t, err)

			if version == opera.GetSonicUpgrades() {
				require.Less(t, postBalance.Uint64(), preBalance.Uint64(), "balance should decrease after failed contract creation")
			}
			if version == opera.GetAllegroUpgrades() {
				require.Equal(t, preBalance, postBalance, "balance should not change after failed contract creation")
			}
		})
	}
}

func TestAccountCreation_CreateCallsProducingCodesTooLargeProduceAUnsuccessfulReceipt(t *testing.T) {
	codeSize := uint256.NewInt(25000).Bytes32()
	initCode := []byte{byte(vm.PUSH32)}
	initCode = append(initCode, codeSize[:]...)
	initCode = append(initCode, []byte{
		byte(vm.PUSH1), byte(0),
		byte(vm.RETURN),
	}...)

	upgrade := opera.GetSonicUpgrades()
	net := StartIntegrationTestNetWithJsonGenesis(t, IntegrationTestNetOptions{
		Upgrades: &upgrade,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	sender := makeAccountWithBalance(t, net, big.NewInt(1e18))

	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err)

	chainId, err := client.ChainID(t.Context())
	require.NoError(t, err)

	txData := &types.LegacyTx{
		Nonce:    0,
		Gas:      100000,
		GasPrice: gasPrice,
		To:       nil, // address 0x00 for contract creation
		Value:    big.NewInt(0),
		Data:     initCode,
	}
	tx := signTransaction(t, chainId, txData, sender)

	receipt, err := net.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status)
}
