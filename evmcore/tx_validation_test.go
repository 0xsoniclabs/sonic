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

func TestValidateTx_BeforeEip2718_RejectsNonLegacyTransactions(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		if _, ok := tx.(*types.LegacyTx); ok {
			continue // Skip legacy transactions
		}
		t.Run(name, func(t *testing.T) {
			err := validateTx(types.NewTx(tx), validationOptions{eip2718: false})
			require.ErrorIs(t, ErrTxTypeNotSupported, err)
		})
	}
}

// getTestTransactionsOption returns a set of options to adjust the validation of transactions
// so that it would accept all types of transactions, considering them as local transactions
// with a min tip of 1, current base fee of 1, and a current max gas of 100_000.
func getTestTransactionsOption() validationOptions {
	return validationOptions{
		eip2718:        true,
		eip1559:        true,
		shanghai:       true,
		eip4844:        true,
		eip7623:        true,
		eip7702:        true,
		currentMaxGas:  100_000,
		currentBaseFee: big.NewInt(1),
		minTip:         big.NewInt(1),
		isLocal:        true,
		signer:         types.NewPragueSigner(big.NewInt(1)),
	}
}

func TestValidateTx_RejectsTxBasedOnTypeAndActiveRevision(t *testing.T) {
	tests := map[string]struct {
		tx        *types.Transaction
		configure func(validationOptions) validationOptions
	}{
		"accessList tx before eip2718": {
			tx: types.NewTx(&types.AccessListTx{}),
			configure: func(opts validationOptions) validationOptions {
				opts.eip2718 = false
				return opts
			},
		},
		"dynamic fee tx before eip1559": {
			tx: types.NewTx(&types.DynamicFeeTx{}),
			configure: func(opts validationOptions) validationOptions {
				opts.eip2718 = true
				opts.eip1559 = false
				return opts
			},
		},
		"blob tx before eip4844": {
			tx: types.NewTx(makeBlobTx(nil, nil)),
			configure: func(opts validationOptions) validationOptions {
				opts.eip2718 = true
				opts.eip4844 = false
				return opts
			},
		},
		"setCode tx before eip7702": {
			tx: types.NewTx(&types.SetCodeTx{}),
			configure: func(opts validationOptions) validationOptions {
				opts.eip2718 = true
				opts.eip7702 = false
				return opts
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateTx(test.tx, test.configure(getTestTransactionsOption()))
			require.Equal(t, ErrTxTypeNotSupported, err)
		})
	}
}

func TestValidateTx_Nonce_RejectsTxWith(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("older nonce/%v", name), func(t *testing.T) {
			// setup validation context
			opt := getTestTransactionsOption()

			// set up to reach nonce check
			setGasPriceOrFeeCap(t, tx, opt.minTip)

			// set nonce lower than the current account nonce
			currentNonce := uint64(2)
			setNonce(t, tx, currentNonce-1)

			// sign txs with sender and set current balance for account
			address, signedTx := signTxForTest(t, tx)
			testDb := newTestTxPoolStateDb()
			testDb.nonces[address] = currentNonce
			opt.currentState = testDb

			// validate transaction
			err := validateTx(signedTx, opt)
			require.ErrorIs(t, err, ErrNonceTooLow)
		})
	}
}

func TestValidateTx_Value_RejectsTxWith(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("negative value/%v", name), func(t *testing.T) {
			if isBlobOrSetCode(tx) {
				t.Skip("blob and setCode transactions cannot have negative value because they use uint256 Value")
			}
			setValueToNegative(t, tx)
			err := validateTx(types.NewTx(tx), getTestTransactionsOption())
			require.ErrorIs(t, err, ErrNegativeValue)
		})
	}
}

func TestValidateTx_GasPriceAndTip_RejectsTxWith(t *testing.T) {
	extremelyLargeN := new(big.Int).Lsh(big.NewInt(1), 256)

	// GasPrice/GasFeeCap tests
	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("gas fee longer than 256 bits/%s", name), func(t *testing.T) {
			if isBlobOrSetCode(tx) {
				t.Skip("blob and setCode transactions cannot have gas price larger than uint256")
			}
			setGasPriceOrFeeCap(t, tx, extremelyLargeN)
			err := validateTx(types.NewTx(tx), getTestTransactionsOption())
			require.ErrorIs(t, err, ErrFeeCapVeryHigh)
		})
	}

	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("gas price lower than base fee/%v", name), func(t *testing.T) {
			// setup validation context
			opt := getTestTransactionsOption()
			opt.currentBaseFee = big.NewInt(2)

			// gas fee cap should be higher than current gas price
			setGasPriceOrFeeCap(t, tx, big.NewInt(1))

			// sign txs with sender
			_, signedTx := signTxForTest(t, tx)

			// validate transaction
			err := validateTx(signedTx, opt)
			require.ErrorIs(t, err, ErrUnderpriced)
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

			err := validateTx(types.NewTx(tx), getTestTransactionsOption())

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

	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("gas tip lower than pool min tip/%v", name), func(t *testing.T) {

			// setup validation context
			opt := getTestTransactionsOption()
			opt.isLocal = false
			opt.minTip = big.NewInt(2)

			// setup low tip cap
			lowTipCap := new(big.Int).Sub(opt.minTip, big.NewInt(1))
			// fee cap needs to be greater than or equal to tip cap
			setEffectiveTip(t, tx, lowTipCap)

			// --- needed for execution up to relevant check ---
			setGasPriceOrFeeCap(t, tx, lowTipCap)
			// sign txs with sender
			_, signedTx := signTxForTest(t, tx)
			opt.locals = newAccountSet(opt.signer)
			// --- needed for execution up to relevant check ---

			// validate transaction
			err := validateTx(signedTx, opt)
			require.ErrorIs(t, err, ErrUnderpriced)
		})
	}

	// GasFeeCap and GasTipCap test
	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("gas fee lower than gas tip/%v", name), func(t *testing.T) {

			setGasPriceOrFeeCap(t, tx, big.NewInt(1))
			setEffectiveTip(t, tx, big.NewInt(2))

			setGas(t, tx, 53000)
			address, signedTx := signTxForTest(t, tx)
			opt := getTestTransactionsOption()
			opt.locals = newAccountSet(opt.signer)

			testDb := newTestTxPoolStateDb()
			testDb.balances[address] = uint256.NewInt(math.MaxUint64)
			opt.currentState = testDb

			err := validateTx(signedTx, opt)
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

func TestValidateTx_Gas_RejectsTxWith(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("current max gas lower than tx gas/%v", name), func(t *testing.T) {
			opt := getTestTransactionsOption()
			opt.currentMaxGas = 1

			setGas(t, tx, 2)

			err := validateTx(types.NewTx(tx), opt)
			require.ErrorIs(t, err, ErrGasLimit)
		})
	}

	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("tx gas lower than intrinsic gas/%v", name), func(t *testing.T) {
			opt := getTestTransactionsOption()

			// setup tx to fail intrinsic gas calculation
			setGas(t, tx, getIntrinsicGasForTest(t, tx, &opt)-1)

			// --- needed for execution up to relevant check ---
			// set tx for execution
			setGasPriceOrFeeCap(t, tx, opt.minTip)
			// sign txs with sender
			address, signedTx := signTxForTest(t, tx)
			// setup enough balance
			testDb := newTestTxPoolStateDb()
			testDb.balances[address] = uint256.NewInt(math.MaxUint64)
			opt.currentState = testDb
			// ---

			// validate transaction
			err := validateTx(signedTx, opt)
			require.ErrorIs(t, err, ErrIntrinsicGas)
		})
	}

	// EIP-7623
	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("gas lower than floor data gas/%v", name), func(t *testing.T) {
			opt := getTestTransactionsOption()

			// setup tx to fail intrinsic gas calculation
			someData := make([]byte, txSlotSize)
			setData(t, tx, someData)
			floorDataGas, err := core.FloorDataGas(someData)
			require.NoError(t, err)
			setGas(t, tx, floorDataGas-1)
			opt.currentMaxGas = floorDataGas

			// --- needed for execution up to relevant check ---
			// set tx for execution
			setGasPriceOrFeeCap(t, tx, opt.minTip)
			// sign txs with sender
			address, signedTx := signTxForTest(t, tx)
			// setup enough balance
			testDb := newTestTxPoolStateDb()
			testDb.balances[address] = uint256.NewInt(math.MaxUint64)
			opt.currentState = testDb
			// ---

			// validate transaction
			err = validateTx(signedTx, opt)
			require.ErrorIs(t, err, ErrFloorDataGas)
		})
	}

	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("floor data gas not checked before eip7623/%v", name), func(t *testing.T) {
			opt := getTestTransactionsOption()
			opt.eip7623 = false

			// setup tx to fail intrinsic gas calculation
			someData := make([]byte, txSlotSize)
			setData(t, tx, someData)
			floorDataGas, err := core.FloorDataGas(someData)
			require.NoError(t, err)
			setGas(t, tx, floorDataGas-1)
			opt.currentMaxGas = floorDataGas

			// --- needed for execution up to relevant check ---
			// set tx for execution
			setGasPriceOrFeeCap(t, tx, opt.minTip)
			// sign txs with sender
			address, signedTx := signTxForTest(t, tx)
			// setup enough balance
			testDb := newTestTxPoolStateDb()
			testDb.balances[address] = uint256.NewInt(math.MaxUint64)
			opt.currentState = testDb
			// ---

			// validate transaction
			err = validateTx(signedTx, opt)
			require.NoError(t, err)

		})
	}
}

func TestValidateTx_Data_RejectsTxWith(t *testing.T) {
	oversizedData := make([]byte, txMaxSize+1)
	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("oversized data/%v", name), func(t *testing.T) {

			setData(t, types.TxData(tx), oversizedData)

			err := validateTx(types.NewTx(tx), getTestTransactionsOption())
			require.Equal(t, ErrOversizedData, err)
		})
	}

	// EIP-3860
	maxInitCode := make([]byte, params.MaxInitCodeSize+1)
	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("init code too large/%v", name), func(t *testing.T) {
			if isBlobOrSetCode(tx) {
				t.Skip("blob and setCode transactions cannot be used as create")
			}

			setData(t, types.TxData(tx), maxInitCode)
			setReceiverToNil(t, tx)

			err := validateTx(types.NewTx(tx), getTestTransactionsOption())
			require.ErrorIs(t, err, ErrMaxInitCodeSizeExceeded)
		})
	}

	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("init code size not checked before shanghai/%v", name), func(t *testing.T) {
			if isBlobOrSetCode(tx) {
				t.Skip("blob and setCode transactions cannot be used to initialize a contract")
			}
			opt := getTestTransactionsOption()
			opt.shanghai = false
			opt.eip4844 = false
			opt.eip7623 = false
			// needs extra gas to allow big data to be afforded.
			opt.currentMaxGas = 249_612

			setData(t, types.TxData(tx), maxInitCode)
			setReceiverToNil(t, tx)

			setGasPriceOrFeeCap(t, tx, opt.currentBaseFee)
			setGas(t, tx, opt.currentMaxGas) // enough gas
			address, signedTx := signTxForTest(t, tx)
			testDb := newTestTxPoolStateDb()
			testDb.balances[address] = uint256.NewInt(opt.currentMaxGas*opt.currentBaseFee.Uint64() + 1)
			opt.currentState = testDb

			err := validateTx(signedTx, opt)
			require.NoError(t, err)
		})
	}

}

func TestValidateTx_Signer_RejectsTxWith(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("invalid signature/%v", name), func(t *testing.T) {
			opt := getTestTransactionsOption()
			opt.signer = types.HomesteadSigner{}
			setSignatureValues(t, tx, big.NewInt(1), big.NewInt(2), big.NewInt(3))
			err := validateTx(types.NewTx(tx), getTestTransactionsOption())
			require.ErrorIs(t, err, ErrInvalidSender)
		})
	}

	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("invalid signer/%v", name), func(t *testing.T) {
			// sign txs with sender
			key, err := crypto.GenerateKey()
			require.NoError(t, err)
			signedTx, err := types.SignTx(types.NewTx(tx),
				types.NewPragueSigner(big.NewInt(2)), key)
			require.NoError(t, err)

			// validate transaction
			err = validateTx(signedTx, getTestTransactionsOption())
			require.ErrorIs(t, err, ErrInvalidSender)
		})
	}
}

func TestValidateTx_Balance_RejectsTxWhen(t *testing.T) {
	for name, tx := range getTxsOfAllTypes() {
		t.Run(fmt.Sprintf("insufficient balance/%v", name), func(t *testing.T) {
			// setup validation context
			opt := getTestTransactionsOption()
			setValue(t, tx, big.NewInt(42))

			// --- needed for execution up to relevant check ---
			// setup transaction enough gas and fee cap to reach balance check
			setGasPriceOrFeeCap(t, tx, opt.currentBaseFee)
			setGas(t, tx, opt.currentMaxGas)

			// sign txs with sender
			address, signedTx := signTxForTest(t, tx)
			// ---

			// setup low balance
			testDb := newTestTxPoolStateDb()
			// balance = gas * fee cap + value
			zero := uint256.NewInt(0)
			txCost := zero.Mul(
				uint256.NewInt(opt.currentMaxGas),
				uint256.MustFromBig(opt.currentBaseFee),
			)
			txCost = zero.Add(txCost, uint256.MustFromBig(signedTx.Value()))
			testDb.balances[address] = zero.Sub(txCost, uint256.NewInt(1))
			opt.currentState = testDb

			// validate transaction
			err := validateTx(signedTx, opt)
			require.ErrorIs(t, err, ErrInsufficientFunds)
		})
	}
}

func TestValidateTx_Blobs_RejectsTxWith(t *testing.T) {
	// blob txs are not supported in sonic, so they must have empty hash list and sidecar

	t.Run("blob tx with non-empty blob hashes", func(t *testing.T) {
		tx := types.NewTx(makeBlobTx([]common.Hash{{0x01}}, nil))
		err := validateTx(tx, getTestTransactionsOption())
		require.ErrorIs(t, err, ErrTxTypeNotSupported)
	})

	t.Run("blob tx with non-empty sidecar", func(t *testing.T) {
		tx := types.NewTx(makeBlobTx(nil,
			&types.BlobTxSidecar{Commitments: []kzg4844.Commitment{{0x01}}}))
		err := validateTx(tx, getTestTransactionsOption())
		require.ErrorIs(t, err, ErrTxTypeNotSupported)
	})
}

func TestValidateTx_AuthorizationList_RejectsTxWith(t *testing.T) {
	t.Run("setCode tx with empty authorization list", func(t *testing.T) {
		tx := types.NewTx(&types.SetCodeTx{})
		err := validateTx(tx, getTestTransactionsOption())
		require.ErrorIs(t, err, ErrEmptyAuthorizations)
	})
}

func TestValidateTx_Success(t *testing.T) {
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
			address, signedTx := signTxForTest(t, tx)

			// Set up sufficient balance and nonce
			testDb := newTestTxPoolStateDb()
			testDb.balances[address] = uint256.NewInt(math.MaxUint64)
			testDb.nonces[address] = 0

			opts := getTestTransactionsOption()
			opts.currentState = testDb

			// Validate the transaction
			err := validateTx(signedTx, opts)
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
func signTxForTest(t *testing.T, tx types.TxData) (common.Address, *types.Transaction) {
	key, err := crypto.GenerateKey()
	address := crypto.PubkeyToAddress(key.PublicKey)
	require.NoError(t, err)
	signer := types.NewPragueSigner(big.NewInt(1))
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

func setSignatureValues(t *testing.T, tx types.TxData, v, r, s *big.Int) {
	switch tx := tx.(type) {
	case *types.LegacyTx:
		tx.V = v
		tx.R = r
		tx.S = s
	case *types.AccessListTx:
		tx.V = v
		tx.R = r
		tx.S = s
	case *types.DynamicFeeTx:
		tx.V = v
		tx.R = r
		tx.S = s
	case *types.BlobTx:
		tx.V = uint256.MustFromBig(v)
		tx.R = uint256.MustFromBig(r)
		tx.S = uint256.MustFromBig(s)
	case *types.SetCodeTx:
		tx.V = uint256.MustFromBig(v)
		tx.R = uint256.MustFromBig(r)
		tx.S = uint256.MustFromBig(s)
	default:
		t.Fatalf("unexpected transaction type: %T", tx)
	}
}

func getIntrinsicGasForTest(t *testing.T, tx types.TxData, opt *validationOptions) uint64 {
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
