package evmcore

import (
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

// TestValidateTx_RejectsWhen tests various scenarios where
// the validateTx function should reject a transaction because of the different
// eip flags.
func TestValidateTx_RejectsWhen(t *testing.T) {
	tests := map[string]struct {
		tx   *types.Transaction // Transaction data to validate.
		opts validationOptions  // Validation options (e.g., EIP flags).
		err  error              // Expected error.
	}{
		"non legacy tx submitted before eip2718": {
			tx: types.NewTx(&types.DynamicFeeTx{}),
			opts: validationOptions{
				eip2718: false,
			},
			err: ErrTxTypeNotSupported,
		},
		"dynamic fee tx submitted before eip1559": {
			tx: types.NewTx(&types.DynamicFeeTx{}),
			opts: validationOptions{
				eip2718: true,
				eip1559: false,
			},
			err: ErrTxTypeNotSupported,
		},
		"blob tx submitted before eip4844": {
			tx: types.NewTx(makeBlobTx(nil, nil)),
			opts: validationOptions{
				eip2718: true,
				eip4844: false,
			},
			err: ErrTxTypeNotSupported,
		},
		"blob tx with hashes": {
			tx: types.NewTx(makeBlobTx([]common.Hash{{0x01}}, nil)),
			opts: validationOptions{
				eip2718: true,
				eip4844: true,
			},
			err: ErrTxTypeNotSupported,
		},
		"blob tx with sidecar": {
			tx: types.NewTx(makeBlobTx(nil,
				&types.BlobTxSidecar{Commitments: []kzg4844.Commitment{{0x01}}})),
			opts: validationOptions{
				eip2718: true,
				eip4844: true,
			},
			err: ErrTxTypeNotSupported,
		},
		"setCode tx submitted before 7702": {
			tx: types.NewTx(&types.SetCodeTx{}),
			opts: validationOptions{
				eip2718: true,
				eip7702: false,
			},
			err: ErrTxTypeNotSupported,
		},
		"setCode tx submitted with an empty auth list": {
			tx: types.NewTx(&types.SetCodeTx{}),
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
		eip1559:        true,
		eip2718:        true,
		eip4844:        true,
		eip7623:        true,
		eip7702:        true,
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
			require.ErrorContains(t, err, ErrMaxInitCodeSizeExceeded.Error())
		})
	}
}

func TestValidation_validateTx_RejectsNegativeValue(t *testing.T) {
	for name, tx := range getTxsFromAllTypes() {
		t.Run(name, func(t *testing.T) {
			if isBlobOrSetCode(tx) {
				t.Skip("blob and setCode transactions cannot have negative value because they use uint256 Value")
			}
			negativeValue(t, tx)
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
			if isBlobOrSetCode(tx) {
				t.Skip("blob and setCode transactions cannot have gas larger than u256")
			}
			// set gas fee cap too large
			setGasTipCap(t, tx, extremelyLargeN)

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
			setGasTipCap(t, tx, lowTipCap)
			// fee cap needs to be greater than or equal to tip cap
			setGasFeeCap(t, tx, lowTipCap)

			// sign txs with sender
			signer, _, signedTx := testSignTx(t, tx)
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
			signer, _, signedTx := testSignTx(t, tx)

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
}

func TestValidation_validateTx_RejectsCannotAffordIntrinsicGas(t *testing.T) {
	for name, tx := range getTxsFromAllTypes() {
		t.Run(name, func(t *testing.T) {
			opt := testTransactionsOption()

			// setup tx to fail intrinsic gas calculation
			setGas(t, tx, 1)

			// --- needed for execution up to relevant check ---
			// set tx for execution
			setGasFeeCap(t, tx, opt.minTip)
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
}

func TestValidation_validateTx_RejectsCannotAffordFloorDataGas(t *testing.T) {
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
}

func TestValidation_validateTx_Success(t *testing.T) {
	tests := []types.TxData{
		&types.LegacyTx{
			Nonce:    0,
			GasPrice: big.NewInt(1),
			Gas:      21000,
			To:       &common.Address{},
			Value:    big.NewInt(1),
		},
		&types.AccessListTx{
			Nonce:      0,
			GasPrice:   big.NewInt(1),
			Gas:        21000,
			To:         &common.Address{},
			Value:      big.NewInt(1),
			AccessList: types.AccessList{},
		},
		&types.DynamicFeeTx{
			Nonce:     0,
			GasTipCap: big.NewInt(1),
			GasFeeCap: big.NewInt(2),
			Gas:       21000,
			To:        &common.Address{},
			Value:     big.NewInt(1),
		},
		&types.BlobTx{
			Nonce:     0,
			GasTipCap: uint256.NewInt(1),
			GasFeeCap: uint256.NewInt(2),
			Gas:       21000,
		},
		&types.SetCodeTx{
			Nonce:     0,
			GasTipCap: uint256.NewInt(1),
			GasFeeCap: uint256.NewInt(2),
			Gas:       46000, // needs more gas than other tx types because of the auth list
			AuthList:  []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range tests {
		t.Run(getTxTypeName(tx), func(t *testing.T) {
			// Sign the transaction
			signer, address, signedTx := testSignTx(t, tx)

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

// testSignTx generates a new key, signs the transaction with it, and returns
// the signer, address, and signed transaction.
func testSignTx(t *testing.T, tx types.TxData) (types.Signer, common.Address, *types.Transaction) {
	key, err := crypto.GenerateKey()
	address := crypto.PubkeyToAddress(key.PublicKey)
	require.NoError(t, err)
	signer := types.NewPragueSigner(big.NewInt(1))
	signedTx, err := types.SignTx(types.NewTx(tx), signer, key)
	require.NoError(t, err)
	return signer, address, signedTx
}

// getTxTypeName returns the name of the transaction type for logging purposes.
func getTxTypeName(tx types.TxData) string {
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

// setNonce sets the nonce for a transaction.
func setNonce(t *testing.T, tx types.TxData, nonce uint64) {
	setTxField(t, tx, "Nonce", nonce, nonce, nonce, nonce, nonce)
}

// setGasFeeCap sets the gas fee cap for a transaction. For legacy and access list
// transactions, it sets the gas price.
func setGasTipCap(t *testing.T, tx types.TxData, gasTipCap *big.Int) {
	bigIntToU256 := func(bigInt *big.Int) *uint256.Int {
		u256, overflow := uint256.FromBig(gasTipCap)
		if overflow {
			t.Fatalf("overflowed converting gasFeeCap to uint256")
		}
		return u256
	}
	switch tx := tx.(type) {
	case *types.LegacyTx:
		tx.GasPrice = gasTipCap
	case *types.AccessListTx:
		tx.GasPrice = gasTipCap
	case *types.DynamicFeeTx:
		tx.GasTipCap = gasTipCap
	case *types.BlobTx:
		tx.GasTipCap = bigIntToU256(gasTipCap)
	case *types.SetCodeTx:
		tx.GasTipCap = bigIntToU256(gasTipCap)
	default:
		t.Fatalf("unexpected transaction type: %T", tx)
	}
}

// setGasFeeCap sets the gas fee cap for a transaction. For legacy and access list
// transactions, it sets the gas price.
func setGasFeeCap(t *testing.T, tx types.TxData, gasFeeCap *big.Int) {
	bigIntToU256 := func(bigInt *big.Int) *uint256.Int {
		u256, overflow := uint256.FromBig(gasFeeCap)
		if overflow {
			t.Fatalf("overflowed converting gasFeeCap to uint256")
		}
		return u256
	}
	switch tx := tx.(type) {
	case *types.LegacyTx:
		tx.GasPrice = gasFeeCap
	case *types.AccessListTx:
		tx.GasPrice = gasFeeCap
	case *types.DynamicFeeTx:
		tx.GasFeeCap = gasFeeCap
	case *types.BlobTx:
		tx.GasFeeCap = bigIntToU256(gasFeeCap)
	case *types.SetCodeTx:
		tx.GasFeeCap = bigIntToU256(gasFeeCap)
	default:
		t.Fatalf("unexpected transaction type: %T", tx)
	}
}

// setGas sets the gas limit for a transaction.
func setGas(t *testing.T, tx types.TxData, gas uint64) {
	setTxField(t, tx, "Gas", gas, gas, gas, gas, gas)
}

// Helper function to add oversized data to a transaction.
func setData(t *testing.T, tx types.TxData, data []byte) {
	setTxField(t, tx, "Data", data, data, data, data, data)
}

// Helper function to set the "To" field of a transaction to nil.
func setToNil(t *testing.T, tx types.TxData) {
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
		t.Fatalf("unexpected transaction type: %T", tx)
	}
}

// Helper function to set the "Value" field of a transaction to a negative value.
// for blob and setCode transactions, it sets the value to zero since they use uint256.
func negativeValue(t *testing.T, tx types.TxData) {
	setTxField(t, tx, "Value", big.NewInt(-1), big.NewInt(-1), big.NewInt(-1),
		uint256.NewInt(0), uint256.NewInt(0))
}

// setTxField sets a field of a transaction to a specific value.
func setTxField(t *testing.T, tx types.TxData, field string, value ...any) {
	switch tx := tx.(type) {
	case *types.LegacyTx:
		assignField(t, tx, field, value[0])
	case *types.AccessListTx:
		assignField(t, tx, field, value[1])
	case *types.DynamicFeeTx:
		assignField(t, tx, field, value[2])
	case *types.BlobTx:
		if len(value) > 3 {
			assignField(t, tx, field, value[3])
		}
	case *types.SetCodeTx:
		if len(value) > 4 {
			assignField(t, tx, field, value[4])
		}
	default:
		t.Fatalf("unexpected transaction type: %T", tx)
	}
}

// assignField assigns a value to a field of a transaction using reflection.
// It checks if the field is valid and can be set before assigning the value.
// If the field is not valid or cannot be set, it fails the test.
func assignField(t *testing.T, tx any, field string, value any) {
	v := reflect.ValueOf(tx).Elem()
	f := v.FieldByName(field)
	if !f.IsValid() || !f.CanSet() {
		t.Fatalf("invalid field %s for type %T", field, tx)
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

// blobTx
func makeBlobTx(hashes []common.Hash, sidecar *types.BlobTxSidecar) types.TxData {
	return &types.BlobTx{
		BlobHashes: hashes,
		Sidecar:    sidecar,
	}
}
