// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package evmcore

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
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

// TestValidation_validateTx_RejectsWhen tests various scenarios where
// the validateTx function should reject a transaction because of the different
// eip flags.
func TestValidation_validateTx_RejectsWhen(t *testing.T) {
	tests := map[string]struct {
		tx   *types.Transaction // Transaction data to validate.
		opts validationOptions  // Validation options (e.g., EIP flags).
		err  error              // Expected error.
	}{
		"tx is nil": {
			tx:  nil,
			err: ErrNilTransaction,
		},
		"non legacy tx before eip2718": {
			tx: types.NewTx(makeDynamicFeeTx()),
			opts: validationOptions{
				eip2718: false,
			},
			err: ErrTxTypeNotSupported,
		},
		"dynamic fee tx before eip1559": {
			tx: types.NewTx(makeDynamicFeeTx()),
			opts: validationOptions{
				eip2718: true,
				eip1559: false,
			},
			err: ErrTxTypeNotSupported,
		},
		"blob tx before eip4844": {
			tx: types.NewTx(makeBlobTx(nil, nil)),
			opts: validationOptions{
				eip2718: true,
				eip4844: false,
			},
			err: ErrTxTypeNotSupported,
		},
		"blob tx without sidecar": {
			tx: types.NewTx(makeBlobTx([]common.Hash{{0x01}}, nil)),
			opts: validationOptions{
				eip2718: true,
				eip4844: true,
			},
			err: ErrEmptyBlobTx,
		},
		"blob tx without hash": {
			tx: types.NewTx(makeBlobTx(nil,
				&types.BlobTxSidecar{Commitments: []kzg4844.Commitment{{0x01}}})),
			opts: validationOptions{
				eip2718: true,
				eip4844: true,
			},
			err: ErrEmptyBlobTx,
		},
		"setCode tx before 7702": {
			tx: types.NewTx(makeSetCodeTx(nil)),
			opts: validationOptions{
				eip2718: true,
				eip7702: false,
			},
			err: ErrTxTypeNotSupported,
		},
		"setCode tx empty auth list": {
			tx: types.NewTx(makeSetCodeTx(nil)),
			opts: validationOptions{
				eip2718: true,
				eip7702: true,
			},
			err: ErrEmptyAuthorizations,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateTx(test.tx, types.NewPragueSigner(big.NewInt(1)), test.opts)
			require.Equal(t, test.err, err)
		})
	}
}

// testTransactionsOption is a set of options to adjust the validation of transactions
func testTransactionsOption() validationOptions {
	return validationOptions{
		eip1559:         true,
		eip2718:         true,
		eip4844:         true,
		eip7702:         true,
		shanghai:        true,
		currentMaxGas:   1,
		currentGasPrice: big.NewInt(1),
		isLocal:         true,
	}
}

func TestValidation_validateTx_Rejects(t *testing.T) {
	oversizedData := make([]byte, txMaxSize+1) // Create oversized data.
	maxInitCode := make([]byte, params.MaxInitCodeSize+1)
	extremelyLargeN := new(big.Int).Lsh(big.NewInt(1), 256)

	tests := []func(types.TxData){
		func(tx types.TxData) { test_OversizedData(t, tx, oversizedData) },
		func(tx types.TxData) { test_MaxInitCodeSize(t, tx, maxInitCode) },
		func(tx types.TxData) { test_NegativeValue(t, tx) },
		func(tx types.TxData) { test_MaxGas(t, tx) },
		func(tx types.TxData) { test_TooLargeGas(t, tx, extremelyLargeN) },
		func(tx types.TxData) { test_TooLargeGasTip(t, tx, extremelyLargeN) },
		func(tx types.TxData) { test_GasFeeLessThanTip(t, tx) },
		func(tx types.TxData) { test_TxWithInvalidSender(t, tx) },
		func(tx types.TxData) { test_UnderpricedLocal(t, tx) },
		func(tx types.TxData) { test_BaseFeeLowerThanChainLimit(t, tx) },
		func(tx types.TxData) { test_NonceOutOfOrder(t, tx) },
		func(tx types.TxData) { test_InsufficientFunds(t, tx) },
		func(tx types.TxData) { test_CannotAffordIntrinsicGas(t, tx) },
		func(tx types.TxData) { test_CannotAffordFloorDataGas(t, tx, oversizedData) },
	}

	for _, test := range tests {
		for _, tx := range allTxTypes() {
			test(tx)
		}
	}
}

// TestValidation_validateTx_RejectsOversizedData tests that validateTx
// rejects transactions with oversized data fields.
func test_OversizedData(t *testing.T, tx types.TxData, oversizedData []byte) {
	t.Run(fmt.Sprintf("OversizedData_%v", txTypeName(tx)), func(t *testing.T) {
		setData(t, types.TxData(tx), oversizedData)
		err := validateTx(types.NewTx(tx), types.NewPragueSigner(big.NewInt(1)),
			testTransactionsOption())
		require.Equal(t, ErrOversizedData, err)
	})
}

func test_MaxInitCodeSize(t *testing.T, tx types.TxData, maxInitCode []byte) {
	t.Run(fmt.Sprintf("MaxInitCodeSize_%v", txTypeName(tx)), func(t *testing.T) {
		if isBlobOrSetCode(tx) {
			t.Skip("blob and setCode transactions cannot be used as create")
		}
		setData(t, types.TxData(tx), maxInitCode)
		setToNil(t, tx)
		err := validateTx(types.NewTx(tx), types.NewPragueSigner(big.NewInt(1)),
			testTransactionsOption())
		require.ErrorContains(t, err, ErrMaxInitCodeSizeExceeded.Error())
	})
}

func test_NegativeValue(t *testing.T, tx types.TxData) {
	t.Run(fmt.Sprintf("NegativeValue_%v", txTypeName(tx)), func(t *testing.T) {
		if isBlobOrSetCode(tx) {
			t.Skip("blob and setCode transactions cannot have negative value because they use uint256 Value")
		}
		negativeValue(t, tx)
		err := validateTx(types.NewTx(tx), types.NewPragueSigner(big.NewInt(1)),
			testTransactionsOption())
		require.ErrorIs(t, err, ErrNegativeValue)
	})
}

func test_MaxGas(t *testing.T, tx types.TxData) {
	t.Run(fmt.Sprintf("MaxGas_%v", txTypeName(tx)), func(t *testing.T) {
		setGas(t, tx, 2)
		err := validateTx(types.NewTx(tx), types.NewPragueSigner(big.NewInt(1)),
			testTransactionsOption())
		require.ErrorIs(t, err, ErrGasLimit)
	})
}

func test_TooLargeGas(t *testing.T, tx types.TxData, n *big.Int) {
	t.Run(fmt.Sprintf("TooLargeGas_%v", txTypeName(tx)), func(t *testing.T) {
		if isBlobOrSetCode(tx) {
			t.Skip("blob and setCode transactions cannot have gas larger than u256")
		}
		setGasFeeCap(t, tx, n)
		err := validateTx(types.NewTx(tx), types.NewPragueSigner(big.NewInt(1)),
			testTransactionsOption())
		require.ErrorIs(t, err, ErrFeeCapVeryHigh)
	})
}

func test_TooLargeGasTip(t *testing.T, tx types.TxData, n *big.Int) {
	t.Run(fmt.Sprintf("TooLargeGasTip_%v", txTypeName(tx)), func(t *testing.T) {
		if isBlobOrSetCode(tx) {
			t.Skip("blob and setCode transactions cannot have gas larger than u256")
		}
		// set gas fee cap too large
		setGasTipCap(t, tx, n)

		err := validateTx(types.NewTx(tx), types.NewPragueSigner(big.NewInt(1)),
			testTransactionsOption())

		// Since for legacy and access list transactions, the gas price is used as the gas tip cap,
		// it would be rejected as well in the gas fee cap check.
		// For blob and setCode transactions, the gas tip cap is of type uint256
		// so it can never have a bit length larger than 256.
		if isLegacyOrAccessList(tx) {
			require.ErrorIs(t, err, ErrFeeCapVeryHigh)
		}
		if _, ok := tx.(*types.DynamicFeeTx); ok {
			require.ErrorIs(t, err, ErrTipVeryHigh)
		}
	})
}

func test_GasFeeLessThanTip(t *testing.T, tx types.TxData) {
	t.Run(fmt.Sprintf("GasFeeLessThanTip_%v", txTypeName(tx)), func(t *testing.T) {
		if isLegacyOrAccessList(tx) {
			t.Skip("legacy and access list transactions use the same field for gas fee and tip")
		}
		setGasFeeCap(t, tx, big.NewInt(1))
		setGasTipCap(t, tx, big.NewInt(2))
		err := validateTx(types.NewTx(tx), types.NewPragueSigner(big.NewInt(1)),
			testTransactionsOption())
		require.ErrorIs(t, err, ErrTipAboveFeeCap)
	})
}

func test_TxWithInvalidSender(t *testing.T, tx types.TxData) {
	t.Run(fmt.Sprintf("TxWithInvalidSender_%v", txTypeName(tx)), func(t *testing.T) {
		signer := types.HomesteadSigner{}
		err := validateTx(types.NewTx(tx), signer, testTransactionsOption())
		require.ErrorIs(t, err, ErrInvalidSender)
	})
}

func test_UnderpricedLocal(t *testing.T, tx types.TxData) {
	t.Run(fmt.Sprintf("UnderpricedLocal_%v", txTypeName(tx)), func(t *testing.T) {
		// setup validation context
		opt := testTransactionsOption()
		opt.isLocal = false

		// setup low tip cap
		lowTipCap := new(big.Int).Sub(opt.currentGasPrice, big.NewInt(1))
		setGasTipCap(t, tx, lowTipCap)
		// fee cap needs to be greater than or equal to tip cap
		setGasFeeCap(t, tx, lowTipCap)

		// sign txs with sender
		signer, _, signedTx := testSignTx(t, tx)
		opt.locals = newAccountSet(signer)
		opt.currentGasPrice = big.NewInt(2)

		// validate transaction
		err := validateTx(signedTx, signer, opt)
		require.ErrorIs(t, err, ErrUnderpriced)
	})
}

func test_BaseFeeLowerThanChainLimit(t *testing.T, tx types.TxData) {
	t.Run(fmt.Sprintf("BaseFeeLowerThanChainLimit_%v", txTypeName(tx)),
		func(t *testing.T) {
			// setup validation context
			opt := testTransactionsOption()
			opt.currentGasPrice = big.NewInt(2)

			// gas fee cap should be higher than current gas price
			setGasFeeCap(t, tx, big.NewInt(1))

			// sign txs with sender
			signer, _, signedTx := testSignTx(t, tx)

			// validate transaction
			err := validateTx(signedTx, signer, opt)
			require.ErrorIs(t, err, ErrUnderpriced)
		})
}

func test_NonceOutOfOrder(t *testing.T, tx types.TxData) {
	t.Run(fmt.Sprintf("NonceOutOfOrder_%v", txTypeName(tx)), func(t *testing.T) {
		// setup validation context
		opt := testTransactionsOption()
		signer := types.NewPragueSigner(big.NewInt(1))

		// set nonce lower than the current account nonce
		setNonce(t, tx, 1)
		setGasFeeCap(t, tx, opt.currentGasPrice)

		// sign txs with sender
		_, address, signedTx := testSignTx(t, tx)

		// setup low nonce
		testDb := newTestTxPoolStateDb()
		testDb.nonces[address] = 2
		opt.currentState = testDb

		// validate transaction
		err := validateTx(signedTx, signer, opt)
		require.ErrorIs(t, err, ErrNonceTooLow)
	})
}

func test_InsufficientFunds(t *testing.T, tx types.TxData) {
	t.Run(fmt.Sprintf("InsufficientFunds_%v", txTypeName(tx)), func(t *testing.T) {
		// setup validation context
		opt := testTransactionsOption()

		// setup transaction
		setGasFeeCap(t, tx, opt.currentGasPrice)
		setGas(t, tx, 1)

		// sign txs with sender
		signer, address, signedTx := testSignTx(t, tx)

		// setup low balance
		testDb := newTestTxPoolStateDb()
		testDb.balances[address] = uint256.NewInt(0)
		opt.currentState = testDb

		// validate transaction
		err := validateTx(signedTx, signer, opt)
		require.ErrorIs(t, err, ErrInsufficientFunds)
	})
}

func test_CannotAffordIntrinsicGas(t *testing.T, tx types.TxData) {
	t.Run(fmt.Sprintf("CannotAffordIntrinsicGas_%v", txTypeName(tx)), func(t *testing.T) {
		opt := testTransactionsOption()

		// setup tx to fail intrinsic gas calculation
		setGas(t, tx, 1)

		// --- needed for execution up to relevant check ---
		// set tx for execution
		setGasFeeCap(t, tx, opt.currentGasPrice)
		// sign txs with sender
		signer, address, signedTx := testSignTx(t, tx)
		// ---

		// setup enough balance
		testDb := newTestTxPoolStateDb()
		testDb.balances[address] = uint256.NewInt(math.MaxUint64)
		opt.currentState = testDb

		// validate transaction
		err := validateTx(signedTx, signer, opt)
		require.ErrorIs(t, err, ErrIntrinsicGas)
	})
}

func test_CannotAffordFloorDataGas(t *testing.T, tx types.TxData, data []byte) {
	t.Run(fmt.Sprintf("CannotAffordFloorDataGas_%v", txTypeName(tx)), func(t *testing.T) {
		opt := testTransactionsOption()
		opt.eip7623 = true

		// setup tx to fail intrinsic gas calculation
		setData(t, tx, data[:txSlotSize])
		floorDataGas, err := core.FloorDataGas(types.NewTx(tx).Data())
		require.NoError(t, err)
		setGas(t, tx, floorDataGas-1)
		opt.currentMaxGas = floorDataGas

		// --- needed for execution up to relevant check ---
		// set tx for execution
		setGasFeeCap(t, tx, opt.currentGasPrice)
		// sign txs with sender
		signer, address, signedTx := testSignTx(t, tx)
		// setup enough balance
		testDb := newTestTxPoolStateDb()
		testDb.balances[address] = uint256.NewInt(math.MaxUint64)
		opt.currentState = testDb
		// ---

		// validate transaction
		err = validateTx(signedTx, signer, opt)
		require.ErrorIs(t, err, ErrFloorDataGas)
	})
}

////////////////////////////////////////////////////////////////////////////////
// Helper functions for testing.

// allTxTypes returns a list of all transaction types for testing.
func allTxTypes() []types.TxData {
	return []types.TxData{
		makeLegacyTx(),
		makeAccessListTx(),
		makeDynamicFeeTx(),
		makeBlobTx(nil, nil),
		makeSetCodeTx([]types.SetCodeAuthorization{{}}),
	}
}

func testSignTx(t *testing.T, tx types.TxData) (types.Signer, common.Address, *types.Transaction) {
	key, err := crypto.GenerateKey()
	address := crypto.PubkeyToAddress(key.PublicKey)
	require.NoError(t, err)
	signer := types.NewPragueSigner(big.NewInt(1))
	signedTx, err := types.SignTx(types.NewTx(tx), signer, key)
	require.NoError(t, err)
	return signer, address, signedTx
}

// txTypeName returns the name of the transaction type for logging purposes.
func txTypeName(tx types.TxData) string {
	switch tx.(type) {
	case *types.LegacyTx:
		return "LegacyTx"
	case *types.AccessListTx:
		return "AccessListTx"
	case *types.DynamicFeeTx:
		return "DynamicFeeTx"
	case *types.BlobTx:
		return "BlobTx"
	case *types.SetCodeTx:
		return "SetCodeTx"
	default:
		panic("unexpected transaction type")
	}
}

func setNonce(tt *testing.T, tx types.TxData, nonce uint64) {
	setTxField(tt, tx, "Nonce", nonce, nonce, nonce, nonce, nonce)
}

// setGasFeeCap sets the gas fee cap for a transaction. For legacy and access list
// transactions, it sets the gas price.
func setGasTipCap(tt *testing.T, tx types.TxData, gasTipCap *big.Int) {
	u256, _ := uint256.FromBig(gasTipCap)
	switch tx := tx.(type) {
	case *types.LegacyTx:
		tx.GasPrice = gasTipCap
	case *types.AccessListTx:
		tx.GasPrice = gasTipCap
	case *types.DynamicFeeTx:
		tx.GasTipCap = gasTipCap
	case *types.BlobTx:
		tx.GasTipCap = u256
	case *types.SetCodeTx:
		tx.GasTipCap = u256
	default:
		tt.Fatalf("unexpected transaction type: %T", tx)
	}
}

// setGasFeeCap sets the gas fee cap for a transaction. For legacy and access list
// transactions, it sets the gas price.
func setGasFeeCap(tt *testing.T, tx types.TxData, gasFeeCap *big.Int) {
	u256, _ := uint256.FromBig(gasFeeCap)
	switch tx := tx.(type) {
	case *types.LegacyTx:
		tx.GasPrice = gasFeeCap
	case *types.AccessListTx:
		tx.GasPrice = gasFeeCap
	case *types.DynamicFeeTx:
		tx.GasFeeCap = gasFeeCap
	case *types.BlobTx:
		tx.GasFeeCap = u256
	case *types.SetCodeTx:
		tx.GasFeeCap = u256
	default:
		tt.Fatalf("unexpected transaction type: %T", tx)
	}
}

// setGas sets the gas limit for a transaction.
func setGas(tt *testing.T, tx types.TxData, gas uint64) {
	setTxField(tt, tx, "Gas", gas, gas, gas, gas, gas)
}

// Helper function to add oversized data to a transaction.
func setData(tt *testing.T, tx types.TxData, data []byte) {
	setTxField(tt, tx, "Data", data, data, data, data, data)
}

// Helper function to set the "To" field of a transaction to nil.
func setToNil(tt *testing.T, tx types.TxData) {
	switch tx := tx.(type) {
	case *types.LegacyTx:
		tx.To = nil
	case *types.AccessListTx:
		tx.To = nil
	case *types.DynamicFeeTx:
		tx.To = nil
	case *types.BlobTx:
		tx.To = common.Address{}
	case *types.SetCodeTx:
		tx.To = common.Address{}
	default:
		tt.Fatalf("unexpected transaction type: %T", tx)
	}
}

// Helper function to set the "Value" field of a transaction to a negative value.
// for blob and setCode transactions, it sets the value to zero since they use uint256.
func negativeValue(tt *testing.T, tx types.TxData) {
	setTxField(tt, tx, "Value", big.NewInt(-1), big.NewInt(-1), big.NewInt(-1),
		uint256.NewInt(0), uint256.NewInt(0))
}

// setTxField sets a field of a transaction to a specific value.
func setTxField(tt *testing.T, tx types.TxData, field string, value ...any) {
	switch tx := tx.(type) {
	case *types.LegacyTx:
		assignField(tt, tx, field, value[0])
	case *types.AccessListTx:
		assignField(tt, tx, field, value[1])
	case *types.DynamicFeeTx:
		assignField(tt, tx, field, value[2])
	case *types.BlobTx:
		if len(value) > 3 {
			assignField(tt, tx, field, value[3])
		}
	case *types.SetCodeTx:
		if len(value) > 4 {
			assignField(tt, tx, field, value[4])
		}
	default:
		tt.Fatalf("unexpected transaction type: %T", tx)
	}
}

// assignField assigns a value to a field of a transaction using reflection.
// It checks if the field is valid and can be set before assigning the value.
// If the field is not valid or cannot be set, it fails the test.
func assignField(tt *testing.T, tx any, field string, value any) {
	v := reflect.ValueOf(tx).Elem()
	f := v.FieldByName(field)
	if !f.IsValid() || !f.CanSet() {
		tt.Fatalf("invalid field %s for type %T", field, tx)
	}
	f.Set(reflect.ValueOf(value))
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

// legacyTx
func makeLegacyTx() types.TxData {
	return &types.LegacyTx{
		Nonce:    0,
		To:       &common.Address{},
		Value:    big.NewInt(0),
		Gas:      0,
		GasPrice: big.NewInt(0),
	}
}

// accessListTx
func makeAccessListTx() types.TxData {
	return &types.AccessListTx{
		ChainID:  big.NewInt(0),
		Nonce:    0,
		To:       &common.Address{},
		Value:    big.NewInt(0),
		Gas:      0,
		GasPrice: big.NewInt(0),
	}
}

// dynamicFeeTx
func makeDynamicFeeTx() types.TxData {
	return &types.DynamicFeeTx{
		ChainID:   big.NewInt(0),
		Nonce:     0,
		To:        &common.Address{},
		Value:     big.NewInt(0),
		Gas:       0,
		GasTipCap: big.NewInt(0),
		GasFeeCap: big.NewInt(0),
	}
}

// blobTx
func makeBlobTx(hashes []common.Hash, sidecar *types.BlobTxSidecar) types.TxData {
	return &types.BlobTx{
		ChainID:    uint256.NewInt(0),
		Nonce:      0,
		Value:      uint256.NewInt(0),
		Gas:        0,
		GasFeeCap:  uint256.NewInt(0),
		GasTipCap:  uint256.NewInt(0),
		BlobFeeCap: uint256.NewInt(0),
		BlobHashes: hashes,
		Sidecar:    sidecar,
	}
}

// setCodeTx
func makeSetCodeTx(authList []types.SetCodeAuthorization) types.TxData {
	return &types.SetCodeTx{
		ChainID:  uint256.NewInt(0),
		Nonce:    0,
		To:       common.Address{},
		AuthList: authList,
	}
}
