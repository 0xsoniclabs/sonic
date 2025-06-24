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

package evmcore

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// getTestPoolOptions returns a set of options to adjust the validation of transactions
// considering them as local transactions with a min tip of 1.
func getTestPoolOptions() poolOptions {
	return poolOptions{
		minTip:       big.NewInt(1),
		isLocal:      true,
		currentState: nil,
		locals:       nil,
	}
}

// getTestBlockState returns a test set of base feee and max gas that can be used for
// validating transactions in the context of a block. It initializes the state
// with a base fee of 1 and a max gas of 100_000
func getTestBlockState() blockState {
	return blockState{
		baseFee: big.NewInt(1),
		maxGas:  100_000,
	}
}

// getTestNetworkRules returns a set of network rules to adjust the validation of transactions
// so that it accepts all types of transactions. It also sets the signer to a new Prague signer with chain ID 1.
func getTestNetworkRules() NetworkRules {
	return NetworkRules{
		istanbul: true,
		eip2718:  true,
		eip1559:  true,
		shanghai: true,
		eip4844:  true,
		eip7623:  true,
		eip7702:  true,
		signer:   types.NewPragueSigner(big.NewInt(1)),
	}
}

////////////////////////////////////////////////////////////////////////////////
// Static Validation

func TestValidateTxStatic_Value_RejectsTxWithNegativeValue(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(name, func(t *testing.T) {
			if isBlobOrSetCode(tx) {
				t.Skip("blob and setCode transactions cannot have negative value because they use uint256 Value")
			}
			setValueToNegative(t, tx)
			err := ValidateTxStatic(types.NewTx(tx))
			require.ErrorIs(t, err, ErrNegativeValue)
		})
	}
}

func TestValidateTxStatic_GasPriceAndTip_RejectsTxWith(t *testing.T) {
	extremelyLargeN := new(big.Int).Lsh(big.NewInt(1), 256)

	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("gas fee longer than 256 bits/%s", name), func(t *testing.T) {
			if isBlobOrSetCode(tx) {
				t.Skip("blob and setCode transactions cannot have gas price larger than uint256")
			}
			setGasPriceOrFeeCap(t, tx, extremelyLargeN)
			err := ValidateTxStatic(types.NewTx(tx))
			require.ErrorIs(t, err, ErrFeeCapVeryHigh)
		})
	}

	// GasTipCap test
	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("gas tip longer than 256 bits/%s", name), func(t *testing.T) {
			// Blob and setCode transactions can never have a Tip with bit length
			// larger than 256 because of the type they use for this field.
			if isBlobOrSetCode(tx) {
				t.Skip("blob and setCode transactions cannot have gas tip larger than u256")
			}

			// set gas tip cap too large
			setEffectiveTip(t, tx, extremelyLargeN)

			err := ValidateTxStatic(types.NewTx(tx))

			if _, ok := tx.(*types.DynamicFeeTx); ok {
				require.ErrorIs(t, err, ErrTipVeryHigh)
			} else if isLegacyOrAccessList(tx) {
				// because tip is the same as gas fee cap for legacy and access list
				// transactions, we need to check if the error is the same as for
				// gas fee cap instead
				require.ErrorIs(t, err, ErrFeeCapVeryHigh)
			} else {
				t.Fatal("unknown transaction type")
			}
		})
	}

	// GasFeeCap and GasTipCap test
	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("gas fee lower than gas tip/%v", name), func(t *testing.T) {

			setGasPriceOrFeeCap(t, tx, big.NewInt(1))
			setEffectiveTip(t, tx, big.NewInt(2))

			err := ValidateTxStatic(types.NewTx(tx))
			if isLegacyOrAccessList(tx) {
				// legacy and access list transactions use the same field for
				// gas fee and gas tip, so no error will be produced.
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ErrTipAboveFeeCap)
			}
		})
	}
}

func TestValidateTxStatic_AuthorizationList_RejectsTxWithEmptyAuthorization(t *testing.T) {
	tx := types.NewTx(&types.SetCodeTx{})
	err := ValidateTxStatic(tx)
	require.ErrorIs(t, err, ErrEmptyAuthorizations)
}

func TestValidateTxStatic_AcceptsValidTransactions(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(name, func(t *testing.T) {
			// set acceptable values
			setGasPriceOrFeeCap(t, tx, big.NewInt(2))
			setEffectiveTip(t, tx, big.NewInt(1))
			setValue(t, tx, big.NewInt(1))
			setData(t, tx, []byte("some data"))

			err := ValidateTxStatic(types.NewTx(tx))
			require.NoError(t, err)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////
// Network Validation

func TestValidateTxForNetwork_BeforeEip2718_RejectsNonLegacyTransactions(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		if _, ok := tx.(*types.LegacyTx); ok {
			continue // Skip legacy transactions because they are always supported
		}
		t.Run(name, func(t *testing.T) {
			err := ValidateTxForNetwork(types.NewTx(tx),
				NetworkRules{eip2718: false})
			require.ErrorIs(t, ErrTxTypeNotSupported, err)
		})
	}
}

func TestValidateTxForNetwork_RejectsTxBasedOnTypeAndActiveRevision(t *testing.T) {
	tests := map[string]struct {
		tx        *types.Transaction
		configure func(NetworkRules) NetworkRules
	}{
		"accessList tx before eip2718": {
			tx: types.NewTx(&types.AccessListTx{}),
			configure: func(opts NetworkRules) NetworkRules {
				opts.eip2718 = false
				return opts
			},
		},
		"dynamic fee tx before eip1559": {
			tx: types.NewTx(&types.DynamicFeeTx{}),
			configure: func(opts NetworkRules) NetworkRules {
				opts.eip2718 = true
				opts.eip1559 = false
				return opts
			},
		},
		"blob tx before eip4844": {
			tx: types.NewTx(makeBlobTx(nil, nil)),
			configure: func(opts NetworkRules) NetworkRules {
				opts.eip2718 = true
				opts.eip4844 = false
				return opts
			},
		},
		"setCode tx before eip7702": {
			tx: types.NewTx(&types.SetCodeTx{}),
			configure: func(opts NetworkRules) NetworkRules {
				opts.eip2718 = true
				opts.eip7702 = false
				return opts
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := ValidateTxForNetwork(test.tx, test.configure(getTestNetworkRules()))
			require.Equal(t, ErrTxTypeNotSupported, err)
		})
	}
}

func TestValidateTxForNetwork_Blobs_RejectsTxWith(t *testing.T) {
	//  only Blob Transactions with empty blob has and no sidecar are accepted in sonic.

	t.Run("blob tx with non-empty blob hashes", func(t *testing.T) {
		tx := types.NewTx(makeBlobTx([]common.Hash{{0x01}}, nil))
		err := ValidateTxForNetwork(tx, getTestNetworkRules())
		require.ErrorIs(t, err, ErrNonEmptyBlobTx)
	})

	t.Run("blob tx with non-empty sidecar", func(t *testing.T) {
		tx := types.NewTx(makeBlobTx(nil,
			&types.BlobTxSidecar{Commitments: []kzg4844.Commitment{{0x01}}}))
		err := ValidateTxForNetwork(tx, getTestNetworkRules())
		require.ErrorIs(t, err, ErrNonEmptyBlobTx)
	})
}

func TestValidateTxForNetwork_Gas_RejectsTxWith(t *testing.T) {

	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("gas lower than intrinsic gas/%v", name), func(t *testing.T) {
			netRules := getTestNetworkRules()

			setGas(t, tx, getIntrinsicGasForTest(t, tx, netRules)-1)

			err := ValidateTxForNetwork(types.NewTx(tx), netRules)
			require.ErrorIs(t, err, ErrIntrinsicGas)
		})
	}

	// EIP-7623
	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("gas lower than floor data gas/%v", name), func(t *testing.T) {
			netRules := getTestNetworkRules()

			someData := make([]byte, txSlotSize)
			setData(t, tx, someData)
			floorDataGas, err := core.FloorDataGas(someData)
			require.NoError(t, err)
			setGas(t, tx, floorDataGas-1)

			err = ValidateTxForNetwork(types.NewTx(tx), netRules)
			require.ErrorIs(t, err, ErrFloorDataGas)
		})
	}

	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("floor data gas not checked before eip7623/%v", name), func(t *testing.T) {
			netRules := getTestNetworkRules()
			netRules.eip7623 = false

			someData := make([]byte, txSlotSize)
			setData(t, tx, someData)
			floorDataGas, err := core.FloorDataGas(someData)
			require.NoError(t, err)
			setGas(t, tx, floorDataGas-1)
			_, signedTx := signTxForTest(t, tx, netRules.signer)

			err = ValidateTxForNetwork(signedTx, netRules)
			require.NoError(t, err)

		})
	}
}

func TestValidateTxForNetwork_Data_RejectsTxWith(t *testing.T) {
	// EIP-3860
	maxInitCode := make([]byte, params.MaxInitCodeSize+1)
	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("init code too large/%v", name), func(t *testing.T) {
			if isBlobOrSetCode(tx) {
				t.Skip("blob and setCode transactions cannot be used as create")
			}

			setData(t, tx, maxInitCode)
			setReceiverToNil(t, tx)

			err := ValidateTxForNetwork(types.NewTx(tx), getTestNetworkRules())
			require.ErrorIs(t, err, ErrMaxInitCodeSizeExceeded)
		})
	}

	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("init code size not checked before shanghai/%v", name), func(t *testing.T) {
			if isBlobOrSetCode(tx) {
				t.Skip("blob and setCode transactions cannot be used to initialize a contract")
			}
			netRules := getTestNetworkRules()
			netRules.shanghai = false
			netRules.eip4844 = false
			netRules.eip7623 = false

			setData(t, tx, maxInitCode)
			setReceiverToNil(t, tx)
			setGas(t, tx, getIntrinsicGasForTest(t, tx, netRules)+1)
			_, signedTx := signTxForTest(t, tx, netRules.signer)

			err := ValidateTxForNetwork(signedTx, netRules)
			require.NoError(t, err)
		})
	}
}

func TestValidateTxForNetwork_Signer_RejectsTxWithInvalidSigner(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(name, func(t *testing.T) {

			netRules := getTestNetworkRules()
			netRules.signer = types.NewPragueSigner(big.NewInt(2))

			key, err := crypto.GenerateKey()
			require.NoError(t, err)
			signer1 := types.NewPragueSigner(big.NewInt(1))
			setGas(t, tx, getIntrinsicGasForTest(t, tx, netRules)+1)
			signedTx, err := types.SignTx(types.NewTx(tx),
				signer1, key)
			require.NoError(t, err)

			err = ValidateTxForNetwork(signedTx, netRules)
			require.ErrorIs(t, err, ErrInvalidSender)
		})
	}
}

func TestValidateTxForNetwork_AcceptsTxWith(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(name, func(t *testing.T) {
			netRules := getTestNetworkRules()

			setGas(t, tx, getIntrinsicGasForTest(t, tx, netRules)+1)
			_, signedTx := signTxForTest(t, tx, netRules.signer)

			err := ValidateTxForNetwork(signedTx, netRules)
			require.NoError(t, err)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////
// Block Validation

func TestValidateTxForBlock_MaxGas_RejectsTxWithGasOverMaxGas(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(name, func(t *testing.T) {
			blockState := getTestBlockState()
			blockState.maxGas = 1

			setGas(t, tx, 2)
			// --- needed for execution up to relevant check ---
			setGasPriceOrFeeCap(t, tx, big.NewInt(blockState.baseFee.Int64()))
			// ---

			err := ValidateTxForBlock(types.NewTx(tx), blockState)
			require.ErrorIs(t, err, ErrGasLimit)
		})
	}
}

func TestValidateTxForBlock_BaseFee_RejectsTxWithGasPriceLowerThanBaseFee(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(name, func(t *testing.T) {
			blockState := getTestBlockState()
			blockState.baseFee = big.NewInt(2)

			// gas fee cap should be higher than current gas price
			setGasPriceOrFeeCap(t, tx, big.NewInt(1))

			err := ValidateTxForBlock(types.NewTx(tx), blockState)
			require.ErrorIs(t, err, ErrUnderpriced)
		})
	}
}

func TestValidateTxForBlock_AcceptsTxWith(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(name, func(t *testing.T) {
			blockState := getTestBlockState()

			// setup transaction enough gas and fee cap to reach base fee check
			setGasPriceOrFeeCap(t, tx, blockState.baseFee)
			setGas(t, tx, blockState.maxGas-1)

			err := ValidateTxForBlock(types.NewTx(tx), blockState)
			require.NoError(t, err)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////
// State Validation

func TestValidateTxForState_Signer_RejectsTxWithInvalidSigner(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(name, func(t *testing.T) {
			// sign txs with sender
			key, err := crypto.GenerateKey()
			require.NoError(t, err)
			signer1 := types.NewPragueSigner(big.NewInt(1))
			signer2 := types.NewPragueSigner(big.NewInt(2))
			signedTx, err := types.SignTx(types.NewTx(tx),
				signer1, key)
			require.NoError(t, err)

			err = ValidateTxForState(signedTx, newTestTxPoolStateDb(), signer2)
			require.ErrorIs(t, err, ErrInvalidSender)
		})
	}
}

func TestValidateTxForState_Nonce_RejectsTxWithOlderNonce(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(name, func(t *testing.T) {

			signer := types.NewPragueSigner(big.NewInt(1))
			address, signedTx := signTxForTest(t, tx, signer)
			testDb := newTestTxPoolStateDb()
			testDb.nonces[address] = signedTx.Nonce() + 1

			err := ValidateTxForState(signedTx, testDb, signer)
			require.ErrorIs(t, err, ErrNonceTooLow)
		})
	}
}

func TestValidateTxForState_Balance_RejectsTxWhenInsufficientBalance(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(name, func(t *testing.T) {

			setValue(t, tx, big.NewInt(42))

			// setup transaction enough gas and fee cap to reach balance check
			someGas := uint64(100_000)
			someGasFee := big.NewInt(1)
			setGasPriceOrFeeCap(t, tx, someGasFee)
			setGas(t, tx, someGas)
			signer := types.NewPragueSigner(big.NewInt(1))
			address, signedTx := signTxForTest(t, tx, signer)

			// setup low balance
			testDb := newTestTxPoolStateDb()

			// balance = gas * fee cap + value
			blank := uint256.NewInt(0)
			txCost := blank.Mul(
				uint256.NewInt(someGas),
				uint256.MustFromBig(someGasFee),
			)
			txCost = blank.Add(txCost, uint256.MustFromBig(signedTx.Value()))
			// just not enough than needed
			testDb.balances[address] = blank.Sub(txCost, uint256.NewInt(1))

			err := ValidateTxForState(signedTx, testDb, signer)
			require.ErrorIs(t, err, ErrInsufficientFunds)
		})
	}
}

func TestValidateTxForState_AcceptsTxWith(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(name, func(t *testing.T) {

			setNonce(t, tx, 42)
			signer := types.NewPragueSigner(big.NewInt(1))
			address, signedTx := signTxForTest(t, tx, signer)
			testDb := newTestTxPoolStateDb()
			testDb.balances[address] = uint256.NewInt(math.MaxUint64)
			testDb.nonces[address] = 42

			err := ValidateTxForState(signedTx, testDb, signer)
			require.NoError(t, err)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////
// TxPool Policies Validation

func TestValidateTxForPool_Data_RejectsTxWithOversizedData(t *testing.T) {
	oversizedData := make([]byte, txMaxSize+1)
	for name, tx := range getTxsOfAllTypes() {
		t.Run(name, func(t *testing.T) {

			setData(t, tx, oversizedData)

			signer := types.NewPragueSigner(big.NewInt(1))
			_, signedTx := signTxForTest(t, tx, signer)
			err := validateTxForPool(signedTx, getTestPoolOptions(), signer)
			require.ErrorIs(t, err, ErrOversizedData)
		})
	}
}

func TestValidateTxForPool_Signer_RejectsTxWithInvalidSigner(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(name, func(t *testing.T) {
			key, err := crypto.GenerateKey()
			require.NoError(t, err)
			signer1 := types.NewPragueSigner(big.NewInt(1))
			signedTx, err := types.SignTx(types.NewTx(tx),
				signer1, key)
			require.NoError(t, err)

			signer2 := types.NewPragueSigner(big.NewInt(2))
			err = validateTxForPool(signedTx, getTestPoolOptions(), signer2)
			require.ErrorIs(t, err, ErrInvalidSender)
		})
	}
}

func TestValidateTxForPool_RejectsNonLocalTxWithTipLowerThanMinPool(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(name, func(t *testing.T) {

			opt := getTestPoolOptions()
			opt.isLocal = false
			opt.minTip = big.NewInt(2)

			// setup low tip cap
			lowTipCap := new(big.Int).Sub(opt.minTip, big.NewInt(1))
			// fee cap needs to be greater than or equal to tip cap
			setEffectiveTip(t, tx, lowTipCap)

			signer := types.NewPragueSigner(big.NewInt(1))
			opt.locals = newAccountSet(signer)
			_, signedTx := signTxForTest(t, tx, signer)

			err := validateTxForPool(signedTx, opt, signer)
			require.ErrorIs(t, err, ErrUnderpriced)
		})
	}
}

func TestValidateTxForPool_AcceptsNonLocalTxWithBigTip(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(name, func(t *testing.T) {

			opt := getTestPoolOptions()
			opt.isLocal = false
			opt.minTip = big.NewInt(2)

			// setup low tip cap
			bigTip := new(big.Int).Add(opt.minTip, big.NewInt(1))
			// fee cap needs to be greater than or equal to tip cap
			setEffectiveTip(t, tx, bigTip)

			signer := types.NewPragueSigner(big.NewInt(1))
			opt.locals = newAccountSet(signer)
			_, signedTx := signTxForTest(t, tx, signer)

			err := validateTxForPool(signedTx, opt, signer)
			require.NoError(t, err)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////
// ValidateTx

func TestValidateTx_RejectsTxWhen(t *testing.T) {

	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("fails network validation/%v", name), func(t *testing.T) {
			netRules := getTestNetworkRules()
			// setup tx to fail intrinsic gas calculation
			setGas(t, tx, getIntrinsicGasForTest(t, tx, netRules)-1)
			_, signedTx := signTxForTest(t, tx, netRules.signer)

			err := validateTx(signedTx, getTestPoolOptions(),
				getTestBlockState(), netRules)
			require.ErrorIs(t, err, ErrIntrinsicGas)
		})
	}

	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("fails static validation/%v", name), func(t *testing.T) {
			var expectedErr error
			if !isBlobOrSetCode(tx) {
				// for legacy and access list transactions, gas price is the same
				// as tip, so negative value is used to reject the transaction.
				setValueToNegative(t, tx)
				expectedErr = ErrNegativeValue
			} else {
				// for blob and setCode transactions, value cannot be negative
				// because they use uint256, so a tip bigger than gas price
				// is used to reject the transaction.
				setEffectiveTip(t, tx, big.NewInt(2))
				setGasPriceOrFeeCap(t, tx, big.NewInt(1))
				expectedErr = ErrTipAboveFeeCap
			}

			netRules := getTestNetworkRules()
			setGas(t, tx, getIntrinsicGasForTest(t, tx, netRules)+1)
			_, signedTx := signTxForTest(t, tx, netRules.signer)

			err := validateTx(signedTx, getTestPoolOptions(),
				getTestBlockState(), netRules)
			require.ErrorIs(t, err, expectedErr)
		})
	}

	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("fails block state/%v", name), func(t *testing.T) {
			blockState := getTestBlockState()
			blockState.baseFee = big.NewInt(2)

			// --- needed for execution up to relevant check ---
			netRules := getTestNetworkRules()
			setGas(t, tx, getIntrinsicGasForTest(t, tx, netRules)+1)
			setGasPriceOrFeeCap(t, tx, big.NewInt(1))
			_, signedTx := signTxForTest(t, tx, netRules.signer)
			// ---

			err := validateTx(signedTx, getTestPoolOptions(),
				blockState, netRules)
			require.ErrorIs(t, err, ErrUnderpriced)
		})
	}

	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("fails pool policies/%v", name), func(t *testing.T) {
			opt := getTestPoolOptions()
			opt.isLocal = false
			opt.minTip = big.NewInt(2)

			// setup low tip cap
			lowTipCap := new(big.Int).Sub(opt.minTip, big.NewInt(1))
			// fee cap needs to be greater than or equal to tip cap
			setEffectiveTip(t, tx, lowTipCap)

			// --- needed for execution up to relevant check ---
			blockState := getTestBlockState()
			setGasPriceOrFeeCap(t, tx, blockState.baseFee)
			netRules := getTestNetworkRules()
			intrinsicGas := getIntrinsicGasForTest(t, tx, netRules)
			setGas(t, tx, intrinsicGas+1) // enough gas
			// ---
			_, signedTx := signTxForTest(t, tx, netRules.signer)
			opt.locals = newAccountSet(netRules.signer)

			err := validateTx(signedTx, opt, blockState, netRules)
			require.ErrorIs(t, err, ErrUnderpriced)
		})
	}

	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("fails state validation/%v", name), func(t *testing.T) {
			// set nonce lower than the current account nonce
			currentNonce := uint64(2)
			setNonce(t, tx, currentNonce-1)

			// --- needed for execution up to relevant check ---
			netRules := getTestNetworkRules()
			blockState := getTestBlockState()
			setGasPriceOrFeeCap(t, tx, blockState.baseFee)
			intrinsicGas := getIntrinsicGasForTest(t, tx, netRules)
			setGas(t, tx, intrinsicGas+1) // enough gas
			// ---

			// sign txs with sender and set current balance for account
			address, signedTx := signTxForTest(t, tx, netRules.signer)
			testDb := newTestTxPoolStateDb()
			testDb.nonces[address] = currentNonce
			opt := getTestPoolOptions()
			opt.currentState = testDb

			err := validateTx(signedTx, opt, blockState, netRules)
			require.ErrorIs(t, err, ErrNonceTooLow)
		})
	}
}

func TestValidateTx_Success(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(name, func(t *testing.T) {

			netRules := getTestNetworkRules()

			setNonce(t, tx, 0)
			setGasPriceOrFeeCap(t, tx, big.NewInt(2))
			setEffectiveTip(t, tx, big.NewInt(1))
			setData(t, tx, []byte("some data"))

			intrinsicGas := getIntrinsicGasForTest(t, tx, netRules)
			setValue(t, tx, big.NewInt(1))
			floorDataGas, err := core.FloorDataGas(types.NewTx(tx).Data())
			require.NoError(t, err)
			minGas := max(intrinsicGas, floorDataGas) + 1
			setGas(t, tx, minGas)

			// Sign the transaction
			address, signedTx := signTxForTest(t, tx, netRules.signer)

			// Set up sufficient balance and nonce
			testDb := newTestTxPoolStateDb()
			testDb.balances[address] = uint256.NewInt(math.MaxUint64)
			testDb.nonces[address] = 0

			opts := getTestPoolOptions()
			opts.currentState = testDb

			err = validateTx(signedTx, opts, getTestBlockState(), netRules)
			require.NoError(t, err)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////
// Helper functions for testing.

// getTxsOfAllTypes returns a list of all transaction types for testing.
func getTxsOfAllTypes() map[string]types.TxData {
	return map[string]types.TxData{
		"Legacy":     &types.LegacyTx{},
		"AccessList": &types.AccessListTx{},
		"DynamicFee": &types.DynamicFeeTx{},
		"Blob":       makeBlobTx(nil, nil),
		"SetCode":    &types.SetCodeTx{AuthList: []types.SetCodeAuthorization{{}}},
	}
}

// signTxForTest generates a new key, signs the transaction with it, and returns
// the signer, address, and signed transaction.
func signTxForTest(t *testing.T, tx types.TxData, signer types.Signer) (common.Address, *types.Transaction) {
	key, err := crypto.GenerateKey()
	address := crypto.PubkeyToAddress(key.PublicKey)
	require.NoError(t, err)
	signedTx, err := types.SignTx(types.NewTx(tx), signer, key)
	require.NoError(t, err)
	return address, signedTx
}

// setNonce sets the nonce for a transaction.
func setNonce(t *testing.T, tx types.TxData, nonce uint64) {
	switch tx := tx.(type) {
	case *types.LegacyTx:
		tx.Nonce = nonce
	case *types.AccessListTx:
		tx.Nonce = nonce
	case *types.DynamicFeeTx:
		tx.Nonce = nonce
	case *types.BlobTx:
		tx.Nonce = nonce
	case *types.SetCodeTx:
		tx.Nonce = nonce
	default:
		t.Fatalf("unexpected transaction type: %T", tx)
	}
}

// setEffectiveTip sets the gas fee cap for a transaction. For legacy and access list
// transactions, it sets the gas price instead since those types do not have a tip.
func setEffectiveTip(t *testing.T, tx types.TxData, gasTipCap *big.Int) {
	switch tx := tx.(type) {
	case *types.LegacyTx:
		tx.GasPrice = gasTipCap
	case *types.AccessListTx:
		tx.GasPrice = gasTipCap
	case *types.DynamicFeeTx:
		tx.GasTipCap = gasTipCap
	case *types.BlobTx:
		tx.GasTipCap = uint256.MustFromBig(gasTipCap)
	case *types.SetCodeTx:
		tx.GasTipCap = uint256.MustFromBig(gasTipCap)
	default:
		t.Fatalf("unexpected transaction type: %T", tx)
	}
}

// setGasFeeCap sets the gas fee cap for a transaction. For legacy and access list
// transactions, it sets the gas price.
func setGasPriceOrFeeCap(t *testing.T, tx types.TxData, gasFeeCap *big.Int) {
	// for all transaction types, the methods GasPrice and GasFeeCap return
	// always the same field. Either gasPrice or gasFeeCap, depending on the
	// transaction type.
	switch tx := tx.(type) {
	case *types.LegacyTx:
		tx.GasPrice = gasFeeCap
	case *types.AccessListTx:
		tx.GasPrice = gasFeeCap
	case *types.DynamicFeeTx:
		tx.GasFeeCap = gasFeeCap
	case *types.BlobTx:
		tx.GasFeeCap = uint256.MustFromBig(gasFeeCap)
	case *types.SetCodeTx:
		tx.GasFeeCap = uint256.MustFromBig(gasFeeCap)
	default:
		t.Fatalf("unexpected transaction type: %T", tx)
	}
}

// setGas sets the gas limit for a transaction.
func setGas(t *testing.T, tx types.TxData, gas uint64) {
	switch tx := tx.(type) {
	case *types.LegacyTx:
		tx.Gas = gas
	case *types.AccessListTx:
		tx.Gas = gas
	case *types.DynamicFeeTx:
		tx.Gas = gas
	case *types.BlobTx:
		tx.Gas = gas
	case *types.SetCodeTx:
		tx.Gas = gas
	default:
		t.Fatalf("unexpected transaction type: %T", tx)
	}
}

// setData is a helper function to add oversized data to a transaction.
func setData(t *testing.T, tx types.TxData, data []byte) {
	switch tx := tx.(type) {
	case *types.LegacyTx:
		tx.Data = data
	case *types.AccessListTx:
		tx.Data = data
	case *types.DynamicFeeTx:
		tx.Data = data
	case *types.BlobTx:
		tx.Data = data
	case *types.SetCodeTx:
		tx.Data = data
	default:
		t.Fatalf("unexpected transaction type: %T", tx)
	}
}

// setReceiverToNil is a helper function to set the "To" field of a transaction to nil.
func setReceiverToNil(t *testing.T, tx types.TxData) {
	switch tx := tx.(type) {
	case *types.LegacyTx:
		tx.To = nil
	case *types.AccessListTx:
		tx.To = nil
	case *types.DynamicFeeTx:
		tx.To = nil
	case *types.BlobTx:
		t.Fatal("blob transaction cannot have nil To field")
	case *types.SetCodeTx:
		t.Fatal("setCode transaction cannot have nil To field")
	default:
		t.Fatalf("unexpected transaction type: %T", tx)
	}
}

func setValue(t *testing.T, tx types.TxData, value *big.Int) {
	switch tx := tx.(type) {
	case *types.LegacyTx:
		tx.Value = value
	case *types.AccessListTx:
		tx.Value = value
	case *types.DynamicFeeTx:
		tx.Value = value
	case *types.BlobTx:
		tx.Value = uint256.MustFromBig(value)
	case *types.SetCodeTx:
		tx.Value = uint256.MustFromBig(value)
	default:
		t.Fatalf("unexpected transaction type: %T", tx)
	}
}

// setValueToNegative is a helper function to set the "Value" field of a transaction to a negative value.
// for blob and setCode transactions, it sets the value to zero since they use uint256.
func setValueToNegative(t *testing.T, tx types.TxData) {
	switch tx := tx.(type) {
	case *types.LegacyTx:
		tx.Value = big.NewInt(-1)
	case *types.AccessListTx:
		tx.Value = big.NewInt(-1)
	case *types.DynamicFeeTx:
		tx.Value = big.NewInt(-1)
	case *types.BlobTx:
		t.Fatal("blob transactions cannot have negative value")
	case *types.SetCodeTx:
		t.Fatal("setCode transactions cannot have negative value")
	default:
		t.Fatalf("unexpected transaction type: %T", tx)
	}
}

func getIntrinsicGasForTest(t *testing.T, tx types.TxData, opt NetworkRules) uint64 {
	transaction := types.NewTx(tx)
	intrGas, err := core.IntrinsicGas(
		transaction.Data(),
		transaction.AccessList(),
		transaction.SetCodeAuthorizations(),
		transaction.To() == nil, // is contract creation
		true,                    // is homestead
		opt.istanbul,            // is eip-2028 (transactional data gas cost reduction)
		opt.shanghai,            // is eip-3860 (limit and meter init-code )
	)
	require.NoError(t, err)
	return intrGas
}

func isLegacyOrAccessList(tx types.TxData) bool {
	_, okLegacy := tx.(*types.LegacyTx)
	_, okAccessList := tx.(*types.AccessListTx)
	return okLegacy || okAccessList
}

func isBlobOrSetCode(tx types.TxData) bool {
	_, okBlob := tx.(*types.BlobTx)
	_, okSetCode := tx.(*types.SetCodeTx)
	return okBlob || okSetCode
}

// blobTx
func makeBlobTx(hashes []common.Hash, sidecar *types.BlobTxSidecar) types.TxData {
	return &types.BlobTx{
		BlobHashes: hashes,
		Sidecar:    sidecar,
	}
}

///////////////////////////////////////////////////////////////////////
// Benchmarks

func BenchmarkValidateTx(b *testing.B) {
	key, err := crypto.GenerateKey()
	require.NoError(b, err)
	address := crypto.PubkeyToAddress(key.PublicKey)

	netRules := getTestNetworkRules()
	blockState := getTestBlockState()
	opts := getTestPoolOptions()
	testDB := newTestTxPoolStateDb()
	testDB.balances[address] = uint256.NewInt(math.MaxUint64)
	testDB.nonces[address] = 1
	opts.currentState = testDB

	// make a good transaction
	tx, err := types.SignTx(types.NewTx(&types.SetCodeTx{
		Nonce:     1,
		Gas:       50_000,
		GasFeeCap: uint256.MustFromBig(blockState.baseFee),
		Value:     uint256.NewInt(1),
		Data:      []byte("some data"),
		AuthList:  []types.SetCodeAuthorization{{}},
	}), netRules.signer, key)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = validateTx(tx, opts, blockState, netRules)
		require.NoError(b, err)
	}
}
