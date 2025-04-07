package evmcore

import (
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

// TestValidateTx_RejectsWhen tests various scenarios where
// the validateTx function should reject a transaction because of the different
// eip flags.
func TestValidateTx_RejectsBasedOnTxTypeAndActiveEip(t *testing.T) {
	tests := map[string]struct {
		tx   *types.Transaction // Transaction data to validate.
		opts validationOptions  // Validation options (e.g., EIP flags).
	}{
		"non legacy tx submitted before eip2718": {
			tx: types.NewTx(&types.DynamicFeeTx{}),
			opts: validationOptions{
				berlin: false,
			},
		},
		"accessList tx submitted before eip2718": {
			tx: types.NewTx(&types.AccessListTx{}),
			opts: validationOptions{
				// since in berlin
				berlin: false,
			},
		},
		"dynamic fee tx submitted before eip1559": {
			tx: types.NewTx(&types.DynamicFeeTx{}),
			opts: validationOptions{
				berlin: true,
				london: false,
			},
		},
		"blob tx submitted before eip4844": {
			tx: types.NewTx(makeBlobTx(nil, nil)),
			opts: validationOptions{
				berlin: true,
				cancun: false,
			},
		},
		"setCode tx submitted before 7702": {
			tx: types.NewTx(&types.SetCodeTx{}),
			opts: validationOptions{
				berlin: true,
				prague: false,
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateTx(test.tx, types.NewPragueSigner(big.NewInt(1)), test.opts)
			require.Equal(t, ErrTxTypeNotSupported, err)
		})
	}
}

// testTransactionsOption is a set of options to adjust the validation of transactions
func testTransactionsOption() validationOptions {
	return validationOptions{
		london:         true,
		berlin:         true,
		cancun:         true,
		prague:         true,
		shanghai:       true,
		currentMaxGas:  100_000,
		currentBaseFee: big.NewInt(1),
		minTip:         big.NewInt(1),
		isLocal:        true,
	}
}

// TestValidation_validateTx_RejectsOversizedData tests that validateTx
// rejects transactions with oversized data fields.
func TestValidateTx_RejectsOversizedData(t *testing.T) {
	oversizedData := make([]byte, txMaxSize+1) // Create oversized data.
	for name, tx := range getTxsFromAllTypes() {
		t.Run(name, func(t *testing.T) {
			setData(t, types.TxData(tx), oversizedData)
			err := validateTx(types.NewTx(tx), types.NewPragueSigner(big.NewInt(1)),
				testTransactionsOption())
			require.Equal(t, ErrOversizedData, err)
		})
	}
}

func TestValidation_validateTx_RejectsMaxInitCodeSize(t *testing.T) {
	maxInitCode := make([]byte, params.MaxInitCodeSize+1)
	for name, tx := range getTxsFromAllTypes() {
		t.Run(name, func(t *testing.T) {
			if isBlobOrSetCode(tx) {
				t.Skip("blob and setCode transactions cannot be used as create")
			}
			setData(t, types.TxData(tx), maxInitCode)
			setToNil(t, tx)
			err := validateTx(types.NewTx(tx), types.NewPragueSigner(big.NewInt(1)),
				testTransactionsOption())
			require.ErrorIs(t, err, ErrMaxInitCodeSizeExceeded)
		})
	}
}

func TestValidation_validateTx_RejectsNegativeValue(t *testing.T) {
	for name, tx := range getTxsFromAllTypes() {
		t.Run(name, func(t *testing.T) {
			if isBlobOrSetCode(tx) {
				t.Skip("blob and setCode transactions cannot have negative value because they use uint256 Value")
			}
			setValueToNegative(t, tx)
			err := validateTx(types.NewTx(tx), types.NewPragueSigner(big.NewInt(1)),
				testTransactionsOption())
			require.ErrorIs(t, err, ErrNegativeValue)
		})
	}
}

func TestValidation_validateTx_RejectsMaxGas(t *testing.T) {
	for name, tx := range getTxsFromAllTypes() {
		t.Run(name, func(t *testing.T) {
			opt := testTransactionsOption()
			opt.currentMaxGas = 1
			setGas(t, tx, 2)
			err := validateTx(types.NewTx(tx), types.NewPragueSigner(big.NewInt(1)), opt)
			require.ErrorIs(t, err, ErrGasLimit)
		})
	}
}

func TestValidation_validateTx_RejectsTooLargeGas(t *testing.T) {
	extremelyLargeN := new(big.Int).Lsh(big.NewInt(1), 256)
	for name, tx := range getTxsFromAllTypes() {
		t.Run(name, func(t *testing.T) {
			if isBlobOrSetCode(tx) {
				t.Skip("blob and setCode transactions cannot have gas larger than u256")
			}
			setGasFeeCap(t, tx, extremelyLargeN)
			err := validateTx(types.NewTx(tx), types.NewPragueSigner(big.NewInt(1)),
				testTransactionsOption())
			require.ErrorIs(t, err, ErrFeeCapVeryHigh)
		})
	}
}

func TestValidation_validateTx_RejectsTooLargeGasTip(t *testing.T) {
	extremelyLargeN := new(big.Int).Lsh(big.NewInt(1), 256)
	for name, tx := range getTxsFromAllTypes() {
		t.Run(name, func(t *testing.T) {
			// For legacy and access list transactions, the gas price is used as the gas tip cap,
			// it would be rejected as well in the gas fee cap check.
			// For blob and setCode transactions, the gas tip cap is of type uint256
			// so it can never have a bit length larger than 256.
			if isBlobOrSetCode(tx) || isLegacyOrAccessList(tx) {
				t.Skip("legacy, access list, blob and setCode transactions cannot have gas tip larger than u256")
			}
			// set gas fee cap too large
			setGasTipCap(t, tx, extremelyLargeN)

			err := validateTx(types.NewTx(tx), types.NewPragueSigner(big.NewInt(1)),
				testTransactionsOption())

			if _, ok := tx.(*types.DynamicFeeTx); ok {
				require.ErrorIs(t, err, ErrTipVeryHigh)
			} else {
				t.Fatal("unknown transaction type")
			}
		})
	}
}

func TestValidation_validateTx_RejectsGasFeeLessThanTip(t *testing.T) {
	for name, tx := range getTxsFromAllTypes() {
		t.Run(name, func(t *testing.T) {
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
}

func TestValidation_validateTx_RejectsTxWithInvalidSender(t *testing.T) {
	for name, tx := range getTxsFromAllTypes() {
		t.Run(name, func(t *testing.T) {
			signer := types.HomesteadSigner{}
			err := validateTx(types.NewTx(tx), signer, testTransactionsOption())
			require.ErrorIs(t, err, ErrInvalidSender)
		})
	}
}

func TestValidation_validateTx_RejectsUnderpricedLocal(t *testing.T) {
	for name, tx := range getTxsFromAllTypes() {
		t.Run(name, func(t *testing.T) {
			// setup validation context
			opt := testTransactionsOption()
			opt.isLocal = false

			// setup low tip cap
			lowTipCap := new(big.Int).Sub(opt.minTip, big.NewInt(1))
			// legacy and access list transactions do not have tip cap
			if !isLegacyOrAccessList(tx) {
				setGasTipCap(t, tx, lowTipCap)
			}
			// fee cap needs to be greater than or equal to tip cap
			setGasFeeCap(t, tx, lowTipCap)

			// sign txs with sender
			signer, _, signedTx := signTxForTest(t, tx)
			opt.locals = newAccountSet(signer)
			opt.minTip = big.NewInt(2)

			// validate transaction
			err := validateTx(signedTx, signer, opt)
			require.ErrorIs(t, err, ErrUnderpriced)
		})
	}
}

func TestValidation_validateTx_RejectsBaseFeeLowerThanChainLimit(t *testing.T) {
	for name, tx := range getTxsFromAllTypes() {
		t.Run(name, func(t *testing.T) {
			// setup validation context
			opt := testTransactionsOption()
			opt.currentBaseFee = big.NewInt(2)

			// gas fee cap should be higher than current gas price
			setGasFeeCap(t, tx, big.NewInt(1))

			// sign txs with sender
			signer, _, signedTx := signTxForTest(t, tx)

			// validate transaction
			err := validateTx(signedTx, signer, opt)
			require.ErrorIs(t, err, ErrUnderpriced)
		})

	}
}

func TestValidation_validateTx_RejectsNonceOutOfOrder(t *testing.T) {
	for name, tx := range getTxsFromAllTypes() {
		t.Run(name, func(t *testing.T) {
			// setup validation context
			opt := testTransactionsOption()
			signer := types.NewPragueSigner(big.NewInt(1))

			// set nonce lower than the current account nonce
			setNonce(t, tx, 1)
			setGasFeeCap(t, tx, opt.minTip)

			// sign txs with sender
			_, address, signedTx := signTxForTest(t, tx)

			// setup low nonce
			testDb := newTestTxPoolStateDb()
			testDb.nonces[address] = 2
			opt.currentState = testDb

			// validate transaction
			err := validateTx(signedTx, signer, opt)
			require.ErrorIs(t, err, ErrNonceTooLow)
		})
	}
}

func TestValidation_validateTx_RejectsInsufficientFunds(t *testing.T) {
	for name, tx := range getTxsFromAllTypes() {
		t.Run(name, func(t *testing.T) {
			// setup validation context
			opt := testTransactionsOption()

			// setup transaction
			setGasFeeCap(t, tx, opt.minTip)
			setGas(t, tx, 1)

			// sign txs with sender
			signer, address, signedTx := signTxForTest(t, tx)

			// setup low balance
			testDb := newTestTxPoolStateDb()
			testDb.balances[address] = uint256.NewInt(0)
			opt.currentState = testDb

			// validate transaction
			err := validateTx(signedTx, signer, opt)
			require.ErrorIs(t, err, ErrInsufficientFunds)
		})
	}
}

func TestValidation_validateTx_RejectsTransactionWithGasLowerThanIntrinsicGasCost(t *testing.T) {
	for name, tx := range getTxsFromAllTypes() {
		t.Run(name, func(t *testing.T) {
			opt := testTransactionsOption()

			// setup tx to fail intrinsic gas calculation
			setGas(t, tx, 1)

			// --- needed for execution up to relevant check ---
			// set tx for execution
			setGasFeeCap(t, tx, opt.minTip)
			// sign txs with sender
			signer, address, signedTx := signTxForTest(t, tx)

			// setup enough balance
			testDb := newTestTxPoolStateDb()
			testDb.balances[address] = uint256.NewInt(math.MaxUint64)
			opt.currentState = testDb
			// ---

			// validate transaction
			err := validateTx(signedTx, signer, opt)
			require.ErrorIs(t, err, ErrIntrinsicGas)
		})
	}
}

func TestValidation_validateTx_GasPriceIsLowerThanFloorDataGas(t *testing.T) {
	oversizedData := make([]byte, txMaxSize+1) // Create oversized data.
	for name, tx := range getTxsFromAllTypes() {
		t.Run(name, func(t *testing.T) {
			opt := testTransactionsOption()

			// setup tx to fail intrinsic gas calculation
			setData(t, tx, oversizedData[:txSlotSize])
			floorDataGas, err := core.FloorDataGas(types.NewTx(tx).Data())
			require.NoError(t, err)
			setGas(t, tx, floorDataGas-1)
			opt.currentMaxGas = floorDataGas

			// --- needed for execution up to relevant check ---
			// set tx for execution
			setGasFeeCap(t, tx, opt.minTip)
			// sign txs with sender
			signer, address, signedTx := signTxForTest(t, tx)
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
}

func TestValidation_validateTx_Success(t *testing.T) {
	tests := map[string]types.TxData{
		"Legacy": &types.LegacyTx{
			Nonce:    0,
			GasPrice: big.NewInt(1),
			Gas:      21000,
			To:       &common.Address{},
			Value:    big.NewInt(1),
		},
		"AccessList": &types.AccessListTx{
			Nonce:      0,
			GasPrice:   big.NewInt(1),
			Gas:        21000,
			To:         &common.Address{},
			Value:      big.NewInt(1),
			AccessList: types.AccessList{},
		},
		"DynamicFee": &types.DynamicFeeTx{
			Nonce:     0,
			GasTipCap: big.NewInt(1),
			GasFeeCap: big.NewInt(2),
			Gas:       21000,
			To:        &common.Address{},
			Value:     big.NewInt(1),
		},
		"Blob": &types.BlobTx{
			Nonce:     0,
			GasTipCap: uint256.NewInt(1),
			GasFeeCap: uint256.NewInt(2),
			Gas:       21000,
		},
		"SetCode": &types.SetCodeTx{
			Nonce:     0,
			GasTipCap: uint256.NewInt(1),
			GasFeeCap: uint256.NewInt(2),
			Gas:       46000, // needs more gas than other tx types because of the auth list
			AuthList:  []types.SetCodeAuthorization{{}},
		},
	}

	for name, tx := range tests {
		t.Run(name, func(t *testing.T) {
			// Sign the transaction
			signer, address, signedTx := signTxForTest(t, tx)

			// Set up sufficient balance and nonce
			testDb := newTestTxPoolStateDb()
			testDb.balances[address] = uint256.NewInt(math.MaxUint64)
			testDb.nonces[address] = 0

			opts := testTransactionsOption()
			opts.currentState = testDb

			// Validate the transaction
			err := validateTx(signedTx, signer, opts)
			require.NoError(t, err)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////
// Helper functions for testing.

// getTxsFromAllTypes returns a list of all transaction types for testing.
func getTxsFromAllTypes() map[string]types.TxData {
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
func signTxForTest(t *testing.T, tx types.TxData) (types.Signer, common.Address, *types.Transaction) {
	key, err := crypto.GenerateKey()
	address := crypto.PubkeyToAddress(key.PublicKey)
	require.NoError(t, err)
	signer := types.NewPragueSigner(big.NewInt(1))
	signedTx, err := types.SignTx(types.NewTx(tx), signer, key)
	require.NoError(t, err)
	return signer, address, signedTx
}

// setNonce sets the nonce for a transaction.
func setNonce(t *testing.T, tx types.TxData, nonce uint64) {
	// setTxField(t, tx, "Nonce", nonce, nonce, nonce, nonce, nonce)
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

// setGasFeeCap sets the gas fee cap for a transaction. For legacy and access list
// transactions, it sets the gas price.
func setGasTipCap(t *testing.T, tx types.TxData, gasTipCap *big.Int) {
	switch tx := tx.(type) {
	case *types.LegacyTx:
		t.Fatal("legacy transactions cannot have gas tip cap")
	case *types.AccessListTx:
		t.Fatal("access list transactions cannot have gas tip cap")
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
func setGasFeeCap(t *testing.T, tx types.TxData, gasFeeCap *big.Int) {
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

// setToNil is a helper function to set the "To" field of a transaction to nil.
func setToNil(t *testing.T, tx types.TxData) {
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
