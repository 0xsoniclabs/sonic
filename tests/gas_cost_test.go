package tests

import (
	"context"
	"iter"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type txModification func(tx *types.AccessListTx)

type CostDefinition struct {
	name         string
	modification txModification
	cost         uint64

	variableCost func(tx *types.AccessListTx) uint64
}

type TestCase struct {
	names     []string
	txPayload *types.AccessListTx
}

func (tc *TestCase) String() string {
	return strings.Join(tc.names, "/")
}

func TestGasCostTest_Sonic(t *testing.T) {

	net := StartIntegrationTestNet(t,
		IntegrationTestNetOptions{
			FeatureSet: opera.SonicFeatures,
		})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId, err := client.ChainID(context.Background())
	require.NoError(t, err)

	// From https://eips.ethereum.org/EIPS/eip-7623
	// Gas used before Prague update:
	// > tx.gasUsed = (
	// >     21000
	// >     + STANDARD_TOKEN_COST * tokens_in_calldata
	// >     + execution_gas_used
	// >     + isContractCreation * (32000 + INITCODE_WORD_COST * words(calldata))
	// > )

	t.Run("reject transactions with insufficient gas", func(t *testing.T) {
		t.Parallel()
		session := net.SpawnSession(t)
		for test := range makeGasCostTestInputs(t, session) {
			t.Run(test.String(), func(t *testing.T) {
				test.txPayload.Gas = test.txPayload.Gas - 1
				test.txPayload = setTransactionDefaults(t, session, test.txPayload, session.GetSessionSponsor())

				tx := signTransaction(t, chainId, test.txPayload, session.GetSessionSponsor())
				require.NoError(t, err)

				err := client.SendTransaction(context.Background(), tx)
				require.Error(t, err)
				require.Condition(t, func() bool {
					return strings.Contains(err.Error(), "intrinsic gas too low")
				}, err.Error())
			})
		}
	})

	t.Run("transactions with exact gas succeed", func(t *testing.T) {
		t.Parallel()
		session := net.SpawnSession(t)
		for test := range makeGasCostTestInputs(t, session) {
			t.Run(test.String(), func(t *testing.T) {
				test.txPayload = setTransactionDefaults(t, session, test.txPayload, session.GetSessionSponsor())
				tx := signTransaction(t, chainId, test.txPayload, session.GetSessionSponsor())
				require.NoError(t, err)

				expectedCost, err := core.IntrinsicGas(tx.Data(), tx.AccessList(), tx.SetCodeAuthorizations(), tx.To() == nil, true, true, true)
				require.NoError(t, err)
				require.Equal(t, expectedCost, tx.Gas())

				receipt, err := session.Run(tx)
				require.NoError(t, err)
				assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
				assert.Equal(t, expectedCost, receipt.GasUsed)
			})
		}
	})

	t.Run("Sonic processor charges 10% of unused gas", func(t *testing.T) {
		t.Parallel()
		session := net.SpawnSession(t)
		for test := range makeGasCostTestInputs(t, session) {
			t.Run(test.String(), func(t *testing.T) {

				// Increase gas by 20% to make sure we have some unused gas
				test.txPayload.Gas = uint64(float32(test.txPayload.Gas) * 1.2)
				test.txPayload = setTransactionDefaults(t, session, test.txPayload, session.GetSessionSponsor())

				tx := signTransaction(t, chainId, test.txPayload, session.GetSessionSponsor())
				require.NoError(t, err)

				expectedCost, err := core.IntrinsicGas(tx.Data(), tx.AccessList(), tx.SetCodeAuthorizations(), tx.To() == nil, true, true, true)
				require.NoError(t, err)
				unused := tx.Gas() - expectedCost
				expectedCost += unused / 10

				receipt, err := session.Run(tx)
				require.NoError(t, err)
				assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
				assert.Equal(t, expectedCost, receipt.GasUsed)
			})
		}
	})
}

func TestGasCostTest_Allegro(t *testing.T) {

	net := StartIntegrationTestNet(t,
		IntegrationTestNetOptions{
			FeatureSet: opera.AllegroFeatures,
		})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId, err := client.ChainID(context.Background())
	require.NoError(t, err)

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

	computeEIP7623GasCost := func(t *testing.T, tx *types.AccessListTx) uint64 {
		intrinsicGas, err := core.IntrinsicGas(tx.Data, tx.AccessList, nil, tx.To == nil, true, true, true)
		require.NoError(t, err)
		require.Equal(t, intrinsicGas, tx.Gas)

		floorDataGas, err := core.FloorDataGas(tx.Data)
		require.NoError(t, err)

		return max(intrinsicGas, floorDataGas)
	}

	t.Run("reject transactions with insufficient gas", func(t *testing.T) {
		t.Parallel()
		session := net.SpawnSession(t)
		for test := range makeGasCostTestInputs(t, session) {
			t.Run(test.String(), func(t *testing.T) {

				test.txPayload.Gas = computeEIP7623GasCost(t, test.txPayload) - 1
				test.txPayload = setTransactionDefaults(t, session, test.txPayload, session.GetSessionSponsor())

				tx := signTransaction(t, chainId, test.txPayload, session.GetSessionSponsor())
				require.NoError(t, err)

				err := client.SendTransaction(context.Background(), tx)
				require.Error(t, err)
				require.Condition(t, func() bool {
					return strings.Contains(err.Error(), "intrinsic gas too low") ||
						strings.Contains(err.Error(), "insufficient gas for floor data gas cost")
				}, "unexpected error, got:", err.Error())
			})
		}
	})

	t.Run("transactions with exact gas succeed", func(t *testing.T) {
		t.Parallel()
		session := net.SpawnSession(t)

		var corrections int
		for test := range makeGasCostTestInputs(t, session) {
			t.Run(test.String(), func(t *testing.T) {

				correctedGasCost := computeEIP7623GasCost(t, test.txPayload)
				if correctedGasCost != test.txPayload.Gas {
					corrections++
				}
				test.txPayload.Gas = correctedGasCost
				test.txPayload = setTransactionDefaults(t, session, test.txPayload, session.GetSessionSponsor())

				tx := signTransaction(t, chainId, test.txPayload, session.GetSessionSponsor())
				require.NoError(t, err)

				receipt, err := session.Run(tx)
				require.NoError(t, err)
				assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
				assert.Equal(t, correctedGasCost, receipt.GasUsed)
			})
		}
		// If the test case generation is modified, please change the expected number of corrections
		// It is important for this test that this value is never 0
		require.Equal(t, 16, corrections, "expected 16 floor data gas corrections in the generated inputs, got %d", corrections)
	})

	t.Run("Sonic processor charges 10% of unused gas", func(t *testing.T) {
		t.Parallel()
		session := net.SpawnSession(t)

		var floorGreaterThan20Percent int
		var expectedSmallerThanFloor int
		for test := range makeGasCostTestInputs(t, session) {
			t.Run(test.String(), func(t *testing.T) {

				floorDataGas, err := core.FloorDataGas(test.txPayload.Data)
				require.NoError(t, err)

				// Increase gas by 20% to make sure we have some unused gas
				incremented := uint64(float32(test.txPayload.Gas) * 1.2)
				// If increased gas is still less than floor data gas, increase it until it is
				// more than necessary
				if floorDataGas > incremented {
					incremented = uint64(float32(floorDataGas) * 1.2)
					floorGreaterThan20Percent++
				}

				test.txPayload.Gas = incremented
				test.txPayload = setTransactionDefaults(t, session, test.txPayload, session.GetSessionSponsor())

				tx := signTransaction(t, chainId, test.txPayload, session.GetSessionSponsor())
				require.NoError(t, err)

				expectedCost, err := core.IntrinsicGas(tx.Data(), tx.AccessList(), tx.SetCodeAuthorizations(), tx.To() == nil, true, true, true)
				require.NoError(t, err)
				unused := tx.Gas() - expectedCost
				expectedCost += unused / 10

				if floorDataGas > expectedCost {
					expectedCost = floorDataGas
					expectedSmallerThanFloor++
				}

				receipt, err := session.Run(tx)
				require.NoError(t, err)
				assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
				assert.Equal(t, expectedCost, receipt.GasUsed)
			})
		}

		// If the test case generation is modified, please change the expected number of out of bound cases
		// It is important for this test that these values are never 0
		require.Equal(t, 12, floorGreaterThan20Percent, "expected 12 cases where floor data gas is greater than 20% of the gas, got %d", floorGreaterThan20Percent)
		require.Equal(t, 12, expectedSmallerThanFloor, "expected 12 cases where the expected cost is smaller than the floor data gas, got %d", expectedSmallerThanFloor)
	})
}

func makeGasCostTestInputs(
	t *testing.T,
	session IntegrationTestNetSession,
) iter.Seq[TestCase] {
	t.Helper()

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	gasPrice, err := client.SuggestGasPrice(context.Background())
	require.NoError(t, err)

	existingAccountAddress := session.GetSessionSponsor().Address()
	nonExistingAccountAddress := NewAccount().Address()

	withCallData := func(size, zeroes int) txModification {
		return func(tx *types.AccessListTx) {
			if zeroes > size {
				panic("zeroes cannot be greater than size")
			}

			if size != 0 && zeroes == 0 {
				panic("please use at least one zero to start calldata, otherwise create contract code does not start with STOP")
			}

			tx.Data = make([]byte, size)
			for i := zeroes; i < size; i++ {
				tx.Data[i] = 1
			}
		}
	}

	withTo := func(addr *common.Address) txModification {
		return func(tx *types.AccessListTx) {
			tx.To = addr
		}
	}

	withAccessList := func(accounts, storageKeysPerAccount int) txModification {
		return func(tx *types.AccessListTx) {
			tx.AccessList = make([]types.AccessTuple, accounts)
			for i := 0; i < accounts; i++ {
				tx.AccessList[i].Address = NewAccount().Address()
				tx.AccessList[i].StorageKeys = make([]common.Hash, storageKeysPerAccount)
				for j := 0; j < storageKeysPerAccount; j++ {
					tx.AccessList[i].StorageKeys[j] = common.Hash{0x42}
				}
			}
		}
	}

	return generateTestDataBasedOnModificationCombinations(
		func() TestCase {
			t.Helper()

			tc := TestCase{
				txPayload: &types.AccessListTx{
					GasPrice: gasPrice,
					Gas:      21000,
				}}

			return tc
		},
		[][]CostDefinition{

			// Call data domain
			// STANDARD_TOKEN_COST * tokens_in_calldata
			// STANDARD_TOKEN_COST = 4
			// tokens_in_calldata = zero_bytes_in_calldata + nonzero_bytes_in_calldata * 4
			{
				{
					name:         "no calldata",
					modification: withCallData(0, 0),
					cost:         0,
				},
				{
					name:         "ones calldata",
					modification: withCallData(10, 1),
					cost:         (9*4 + 1) * 4,
				},
				{
					name:         "mix calldata",
					modification: withCallData(10, 5),
					cost:         (5*4 + 5) * 4,
				},
				{
					name:         "max code size calldata",
					modification: withCallData(params.MaxCodeSize, 500),
					cost:         (((params.MaxCodeSize - 500) * 4) + 500) * 4,
				},
			},
			// To domain
			// isContractCreation * (32000 + INITCODE_WORD_COST * words(calldata))
			// INITCODE_WORD_COST = 2
			{
				{
					name:         "existing account",
					modification: withTo(&existingAccountAddress),
				},
				{
					name:         "non existing account",
					modification: withTo(&nonExistingAccountAddress),
				},
				{
					name:         "contract creation",
					modification: withTo(nil),
					cost:         32000,
					variableCost: func(tx *types.AccessListTx) uint64 {
						lenWords := (uint64(len(tx.Data)) + 31) / 32
						return lenWords * params.InitCodeWordGas
					},
				},
			},
			// Access list domain
			// Represents the execution_gas_used part of the gas calculation,
			// this allows to exercise the formula without having to execute real
			// contracts.
			{
				{
					name:         "no access list",
					modification: withAccessList(0, 0),
					cost:         0,
				},
				{
					name:         "access list one account",
					modification: withAccessList(1, 0),
					cost:         2400,
				},
				{
					name:         "access list one account one key",
					modification: withAccessList(1, 1),
					cost:         2400 + 1900,
				},
				{
					name:         "heavy access list",
					modification: withAccessList(8, 4),
					cost:         8*2400 + 8*4*1900,
				},
			},
		},
		func(tc TestCase, pieces []CostDefinition) TestCase {
			for _, piece := range pieces {
				if piece.modification != nil {
					piece.modification(tc.txPayload)
				}
				tc.txPayload.Gas += piece.cost
				if piece.variableCost != nil {
					tc.txPayload.Gas += piece.variableCost(tc.txPayload)
				}
				tc.names = append(tc.names, piece.name)
			}
			return tc
		},
	)

}
