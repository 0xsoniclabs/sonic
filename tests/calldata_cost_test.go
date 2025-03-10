package tests

import (
	"context"
	"iter"
	"math/big"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestCallData_Sonic_GasUsed(t *testing.T) {

	// From https://eips.ethereum.org/EIPS/eip-7623
	// Gas used before Prague update:
	// > tx.gasUsed = (
	// >     21000
	// >     + STANDARD_TOKEN_COST * tokens_in_calldata
	// >     + execution_gas_used
	// >     + isContractCreation * (32000 + INITCODE_WORD_COST * words(calldata))
	// > )
	testUsedGas(t,
		opera.SonicFeatures,
		func(t *testing.T, data []byte) uint64 {
			// transaction does not create an account
			// transaction does not execute code
			// remaining gas is the intrinsic gas
			intrinsicGas, err := core.IntrinsicGas(data, nil, nil, false, true, true, true)
			require.NoError(t, err, "Failed to get the intrinsic gas: ", err)
			return intrinsicGas
		})
}

func TestCallData_Allegro_GasUsed(t *testing.T) {

	// From https://eips.ethereum.org/EIPS/eip-7623
	// Gas used after Prague update:
	// > tx.gasUsed = (
	// >     21000
	// >     +
	// >     max(
	// >         STANDARD_TOKEN_COST * tokens_in_calldata
	// >         + execution_gas_used
	// >         + isContractCreation * (32000 + INITCODE_WORD_COST * words(calldata)),
	// >         TOTAL_COST_FLOOR_PER_TOKEN * tokens_in_calldata
	// >     )
	// > )
	testUsedGas(t,
		opera.AllegroFeatures,
		func(t *testing.T, data []byte) uint64 {
			// transaction does not create an account
			// transaction does not execute code
			// remaining gas is the intrinsic gas
			intrinsicGas, err := core.IntrinsicGas(data, nil, nil, false, true, true, true)
			require.NoError(t, err, "Failed to get the intrinsic gas: ", err)

			floorGas, err := core.FloorDataGas(data)
			require.NoError(t, err, "Failed to get the floor gas: ", err)

			return max(intrinsicGas, floorGas)
		})
}

func testUsedGas(t *testing.T,
	featureSet opera.FeatureSet,
	estimateGasUsage func(t *testing.T, data []byte) uint64,
) {
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		FeatureSet: featureSet,
	})

	client, err := net.GetClient()
	require.NoError(t, err, "Failed to get the client: ", err)
	defer client.Close()

	receiver := makeAccountWithBalance(t, net, big.NewInt(1))
	receiverAddress := receiver.Address()

	chainId, err := client.ChainID(context.Background())
	require.NoError(t, err, "Failed to get the chain ID: ", err)

	// This checks that the estimation is the minimum required gas
	t.Run("test rejection", func(t *testing.T) {
		nonce, err := client.NonceAt(context.Background(), net.GetSessionSponsor().Address(), nil)
		require.NoError(t, err, "failed to get nonce for sponsor account")

		for data := range generateTestData() {
			price, err := client.SuggestGasPrice(context.Background())
			require.NoError(t, err, "Failed to get the gas price: ", err)
			estimatedGasUse := estimateGasUsage(t, data)
			estimatedGasUse--
			tx := signTransaction(t, chainId, &types.LegacyTx{
				Nonce:    nonce,
				To:       &receiverAddress,
				GasPrice: price,
				Gas:      estimatedGasUse,
				Data:     data,
			}, net.GetSessionSponsor())

			// transactions must fail, because they have exactly one gas less than the required amount
			err = client.SendTransaction(context.Background(), tx)
			require.Error(t, err, "Transaction cannot be submitted with estimation-1 gas")

			// Error has been serialized over the client and errors.Is cannot be
			// used to compare the error type
			require.Condition(t, func() bool {
				return strings.Contains(err.Error(), "intrinsic gas too low") ||
					strings.Contains(err.Error(), "insufficient gas for floor data gas cost")
			})
		}
	})

	// This test checks that the transaction consumes the expected amount of gas
	t.Run("test gasUsed after execution", func(t *testing.T) {
		nonce, err := client.NonceAt(context.Background(), net.GetSessionSponsor().Address(), nil)
		require.NoError(t, err, "failed to get nonce for sponsor account")

		expectedResults := make(map[common.Hash]uint64)
		for data := range generateTestData() {

			price, err := client.SuggestGasPrice(context.Background())
			require.NoError(t, err, "Failed to get the gas price: ", err)

			estimatedGasUse := estimateGasUsage(t, data)

			tx := signTransaction(t, chainId, &types.LegacyTx{
				Nonce:    nonce,
				To:       &receiverAddress,
				GasPrice: price,
				Gas:      estimatedGasUse,
				Data:     data,
			}, net.GetSessionSponsor())
			nonce++

			err = client.SendTransaction(context.Background(), tx)
			require.NoError(t, err, "Failed to submit transaction: ", err)

			expectedResults[tx.Hash()] = estimatedGasUse
		}
		// Check the gas used for each transaction
		for txHash, expectedCost := range expectedResults {
			receipt, err := net.GetReceipt(txHash)
			require.NoError(t, err, "Failed to get the receipt: ", err)
			require.EqualValues(t, expectedCost, receipt.GasUsed, "unexpected gas used")
		}
	})

	// This test checks that the transaction consumes the expected amount of gas when more than
	// the estimated gas is provided..
	// This tests a sonic feature where 10% of the unused gas is consumed.
	// https://github.com/0xsoniclabs/go-ethereum/blob/86eca3554809383eb0068b6321221f3bfe9dbd6c/core/state_transition.go#L522
	// https://github.com/0xsoniclabs/go-ethereum/commit/f13c5cf345fdecfc75dd949cf3c6f956a06bb86e
	t.Run("10% of unused gas is consumed", func(t *testing.T) {
		nonce, err := client.NonceAt(context.Background(), net.GetSessionSponsor().Address(), nil)
		require.NoError(t, err, "failed to get nonce for sponsor account")

		expectedResults := make(map[common.Hash]uint64)
		txs := make(map[common.Hash]*types.Transaction)
		for data := range generateTestData() {

			price, err := client.SuggestGasPrice(context.Background())
			require.NoError(t, err, "Failed to get the gas price: ", err)

			estimatedGasUse := estimateGasUsage(t, data)

			tx := signTransaction(t, chainId, &types.LegacyTx{
				Nonce:    nonce,
				To:       &receiverAddress,
				GasPrice: price,
				// add an extra 20% to the estimated gas use
				Gas:  uint64(float64(estimatedGasUse) * 1.2),
				Data: data,
			}, net.GetSessionSponsor())
			nonce++

			err = client.SendTransaction(context.Background(), tx)
			require.NoError(t, err, "Failed to submit transaction: ", err)

			intrinsicGas, err := core.IntrinsicGas(tx.Data(), nil, nil, false, true, true, true)
			require.NoError(t, err, "Failed to get the intrinsic gas: ", err)

			// The Sonic gas cost model adds 10% of the gas used as an additional cost
			expectedGas := intrinsicGas + (tx.Gas()-intrinsicGas)/10

			if featureSet == opera.AllegroFeatures {
				// If the cost is still under the floor data gas value, consume it all.
				floorDataGas, err := core.FloorDataGas(tx.Data())
				require.NoError(t, err, "Failed to get the floor data gas: ", err)
				if expectedGas < floorDataGas {
					expectedGas = floorDataGas
				}
			}

			expectedResults[tx.Hash()] = expectedGas
			txs[tx.Hash()] = tx
		}
		// Check the gas used for each transaction
		for txHash, expectedCost := range expectedResults {
			receipt, err := net.GetReceipt(txHash)
			require.NoError(t, err, "Failed to get the receipt: ", err)
			require.EqualValues(t, expectedCost, receipt.GasUsed, "unexpected gas used")
		}
	})
}

func makeCallData(size int, zeroesPercentage float32) []byte {
	zeroes := int(float32(size) * zeroesPercentage)
	data := make([]byte, size)
	for i := zeroes; i < size; i++ {
		data[i] = 1
	}
	return data
}

func generateTestData() iter.Seq[[]byte] {
	return func(yield func(data []byte) bool) {
		for _, size := range []int{0, 1, 10, 100, 1000} {
			for _, zerosPercentage := range []float32{.0, .25, .50, .75, 1} {
				if !yield(makeCallData(size, zerosPercentage)) {
					return
				}
			}
		}
	}
}
