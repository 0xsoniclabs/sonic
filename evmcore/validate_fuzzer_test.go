package evmcore

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// FuzzValidateTransaction fuzzes the validateTx function with randomly generated transactions.
func FuzzValidateTransaction(f *testing.F) {
	// Seed corpus with a few valid-looking values
	//    nonce: 0, gas: 21000, feeCap: 1_000_000_000, tip: 1_000_000_000, data: "hi"
	f.Add(uint64(0), uint64(21000), int64(1_000_000_000), int64(1_000_000_000), []byte("hi"))

	f.Fuzz(func(t *testing.T, nonce uint64, gas uint64, feeCap int64, tip int64, data []byte) {
		// Skip gas limits that are too low
		if gas < 21000 || gas > 30_000_000 {
			t.Skip()
		}
		// Disallow negative values that would crash NewTx
		if feeCap < 0 || tip < 0 {
			t.Skip()
		}

		// Avoid overflows on big.Int
		feeCapBig := big.NewInt(feeCap)
		tipBig := big.NewInt(tip)

		// Randomly choose a transaction type
		var tx *types.Transaction
		switch rand.Intn(5) {
		case 0: // Legacy Tx
			tx = types.NewTx(&types.LegacyTx{
				Nonce:    nonce,
				Gas:      gas,
				GasPrice: feeCapBig,
				To:       &common.Address{},
				Value:    big.NewInt(0),
				Data:     data,
			})
		case 1: // AccessList Tx
			tx = types.NewTx(&types.AccessListTx{
				ChainID:    big.NewInt(1),
				Nonce:      nonce,
				Gas:        gas,
				GasPrice:   feeCapBig,
				To:         &common.Address{},
				Value:      big.NewInt(0),
				Data:       data,
				AccessList: types.AccessList{},
			})
		case 2: // DynamicFee Tx
			tx = types.NewTx(&types.DynamicFeeTx{
				ChainID:    big.NewInt(1),
				Nonce:      nonce,
				Gas:        gas,
				GasFeeCap:  feeCapBig,
				GasTipCap:  tipBig,
				To:         &common.Address{},
				Value:      big.NewInt(0),
				Data:       data,
				AccessList: types.AccessList{},
			})
		case 3: // Blob Transaction
			tx = types.NewTx(&types.BlobTx{
				ChainID:    uint256.NewInt(1),
				Nonce:      nonce,
				Gas:        gas,
				GasFeeCap:  uint256.MustFromBig(feeCapBig),
				GasTipCap:  uint256.MustFromBig(tipBig),
				To:         common.Address{},
				Value:      uint256.NewInt(0),
				Data:       data,
				AccessList: types.AccessList{},
			})
		case 4: // SetCode Transaction
			tx = types.NewTx(&types.SetCodeTx{
				ChainID:    uint256.NewInt(1),
				Nonce:      nonce,
				Gas:        gas,
				GasFeeCap:  uint256.MustFromBig(feeCapBig),
				GasTipCap:  uint256.MustFromBig(tipBig),
				To:         common.Address{},
				Value:      uint256.NewInt(0),
				Data:       data,
				AccessList: types.AccessList{},
				AuthList:   []types.SetCodeAuthorization{{}},
			})
		}

		sender := types.NewPragueSigner(big.NewInt(1))
		opt := testTransactionsOption()

		// Validate the transaction
		_ = validateTx(tx, sender, opt)
	})
}
