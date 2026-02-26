package bundle_validate

import (
	"errors"
	"math/big"
	"testing"

	bp "github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestBundle_ValidateBundle_ReportsErrorWhen(t *testing.T) {

	tests := map[string]struct {
		tx  *types.Transaction
		err error
	}{
		"bundle transaction does not target bundle address": {
			tx:  types.NewTx(&types.AccessListTx{}),
			err: ErrMissingBundleAddress,
		},
		"bundle without execution plan": {
			tx: types.NewTx(&types.AccessListTx{
				To: &bp.BundleAddress,
			}),
			err: ErrEmptyExecutionPlan,
		},
		"bundle with invalid plan": {
			tx: types.NewTx(&types.AccessListTx{
				To:   &bp.BundleAddress,
				Data: []byte{0x01, 0x02, 0x03}, // invalid RLP encoding for the bundle
			}),
			err: ErrInvalidExecutionPlan,
		},
		"bundle with valid plan but invalid execution plan": {
			tx: types.NewTx(&types.AccessListTx{
				To:   &bp.BundleAddress,
				Data: makeBundleWithBlobTx(),
			}),
			err: ErrFailedToExtractExecutionPlan,
		},
		"bundle with valid plan but empty execution plan": {
			tx: types.NewTx(&types.AccessListTx{
				To:   &bp.BundleAddress,
				Data: makeBundleWithoutTxs(),
			}),
			err: ErrEmptyExecutionPlan,
		},
		"bundle with faulty transaction": {
			tx: types.NewTx(&types.AccessListTx{
				To:   &bp.BundleAddress,
				Data: makeBundleWithTxWithoutBundleOnlyMarker(),
			}),
			err: ErrFailedToValidateTransaction,
		},
		"bundle with invalid payment transaction": {
			tx: types.NewTx(&types.AccessListTx{
				To:   &bp.BundleAddress,
				Data: makeBundleWithInvalidPaymentTx(t),
			}),
			err: ErrInvalidPaymentTransaction,
		},
		"bundle with low gas limit": {
			tx: types.NewTx(&types.AccessListTx{
				To:  &bp.BundleAddress,
				Gas: 100, // gas limit too low
				Data: bp.Encode(bp.TransactionBundle{
					Version: 1,
					Bundle: types.Transactions{
						types.NewTx(&types.AccessListTx{
							Gas: 50, // gas of the transaction in the bundle
							AccessList: types.AccessList{
								{Address: bp.BundleOnly},
							},
						}),
					},
					Payment: types.NewTx(&types.AccessListTx{
						Gas: 51, //	 gas of the payment transaction
						To:  &bp.BundleAddress,
					}),
					Flags: 0,
				}),
			}),
			err: ErrBundleGasLimitTooLow,
		},
		"bundle overpriced": {
			tx: types.NewTx(&types.AccessListTx{
				To:       &bp.BundleAddress,
				GasPrice: big.NewInt(99), // price too high
				Data: bp.Encode(bp.TransactionBundle{
					Version: 1,
					Bundle: types.Transactions{
						types.NewTx(&types.AccessListTx{
							GasPrice: big.NewInt(98), // tx price lower than bundle price
							AccessList: types.AccessList{
								{Address: bp.BundleOnly},
							},
						}),
					},
					Payment: types.NewTx(&types.AccessListTx{
						GasPrice: big.NewInt(99), // payment price lower than bundle price
						To:       &bp.BundleAddress,
					}),
					Flags: 0,
				}),
			}),
			err: ErrBundleOverpriced,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			mockSigner := bp.NewMockSigner(ctrl)
			mockSigner.EXPECT().Sender(gomock.Any()).Return(common.Address{0x42}, nil).AnyTimes()
			mockSigner.EXPECT().Sender(gomock.Any()).Return(common.Address{0x43}, nil).AnyTimes()
			mockSigner.EXPECT().Hash(gomock.Any()).Return(common.Hash{0x01}).AnyTimes()
			mockSigner.EXPECT().Hash(gomock.Any()).Return(common.Hash{0x02}).AnyTimes()

			_, err := ValidateTransactionBundle(test.tx, mockSigner)
			require.ErrorIs(t, err, test.err)
		})
	}
}

func makeBundleWithBlobTx() []byte {
	bundle := bp.TransactionBundle{
		Version: 1,
		Bundle: types.Transactions{
			// blob transaction is not supported in the execution plan
			types.NewTx(&types.BlobTx{
				AccessList: types.AccessList{{
					Address:     bp.BundleOnly,
					StorageKeys: []common.Hash{common.Hash{0x01, 0x02, 0x03}}, // dummy hash
				}},
			}),
		},
		Payment: makeDefaultPaymentTx(),
		Flags:   0,
	}

	return bp.Encode(bundle)
}

func makeBundleWithoutTxs() []byte {
	bundle := bp.TransactionBundle{
		Version: 1,
		Bundle:  types.Transactions{},
		Payment: makeDefaultPaymentTx(),
		Flags:   0,
	}

	return bp.Encode(bundle)
}

func makeBundleWithTxWithoutBundleOnlyMarker() []byte {
	// create a bundle with an execution plan that contains a transaction without the bundle-only marker
	bundle := bp.TransactionBundle{
		Version: 1,
		Bundle: types.Transactions{
			types.NewTx(&types.AccessListTx{}), // missing bundle-only marker in access list
		},
		Payment: makeDefaultPaymentTx(),
		Flags:   0,
	}

	return bp.Encode(bundle)
}

func makeBundleWithInvalidPaymentTx(t *testing.T) []byte {
	bundle := bp.TransactionBundle{
		Version: 1,
		Bundle: types.Transactions{
			types.NewTx(&types.AccessListTx{
				AccessList: types.AccessList{{
					Address:     bp.BundleOnly,
					StorageKeys: []common.Hash{common.Hash{0x01, 0x02, 0x03}}, // dummy hash
				}},
			}),
		},
		Payment: types.NewTx(&types.AccessListTx{To: nil}),
		Flags:   0,
	}

	return bp.Encode(bundle)
}

func makeDefaultPaymentTx() *types.Transaction {
	return types.NewTx(&types.AccessListTx{
		To: &bp.BundleAddress,
	})
}

func TestBundle_ValidateBundle_FailsToGetSenderOfBundleTx(t *testing.T) {
	tx := types.NewTx(&types.AccessListTx{
		To:       &bp.BundleAddress,
		GasPrice: big.NewInt(100), // price too high
		Data: bp.Encode(bp.TransactionBundle{
			Version: 1,
			Bundle: types.Transactions{
				types.NewTx(&types.AccessListTx{
					AccessList: types.AccessList{
						{Address: bp.BundleOnly},
					},
				}),
			},
			Payment: types.NewTx(&types.AccessListTx{
				To: &bp.BundleAddress,
			}),
			Flags: 0,
		}),
	})

	ctrl := gomock.NewController(t)
	mockSigner := bp.NewMockSigner(ctrl)
	mockSigner.EXPECT().Sender(gomock.Any()).Return(common.Address{}, errors.New("sender error"))

	_, err := ValidateTransactionBundle(tx, mockSigner)
	require.ErrorContains(t, err, "failed to derive sender of the bundle transaction")
}

func TestBundle_ValidateBundle_Succeeds(t *testing.T) {
	tx := types.NewTx(&types.AccessListTx{
		To:       &bp.BundleAddress,
		GasPrice: big.NewInt(100),
		Gas:      200, // sufficient gas limit
		Data: bp.Encode(bp.TransactionBundle{
			Version: 1,
			Bundle: types.Transactions{
				types.NewTx(&types.AccessListTx{
					GasPrice: big.NewInt(100),
					Gas:      50,
					AccessList: types.AccessList{
						{Address: bp.BundleOnly},
					},
				}),
			},
			Payment: types.NewTx(&types.AccessListTx{
				GasPrice: big.NewInt(100),
				Gas:      50,
				To:       &bp.BundleAddress,
			}),
			Flags: 0,
		}),
	})

	ctrl := gomock.NewController(t)
	mockSigner := bp.NewMockSigner(ctrl)
	mockSigner.EXPECT().Sender(gomock.Any()).Return(common.Address{0x42}, nil).AnyTimes()
	mockSigner.EXPECT().Sender(gomock.Any()).Return(common.Address{0x43}, nil).AnyTimes()
	mockSigner.EXPECT().Hash(gomock.Any()).Return(common.Hash{0x01}).AnyTimes()
	mockSigner.EXPECT().Hash(gomock.Any()).Return(common.Hash{0x02}).AnyTimes()

	_, err := ValidateTransactionBundle(tx, mockSigner)
	require.NoError(t, err)
}

func TestBundle_ValidatePaymentTx_ReportsErrorWhen(t *testing.T) {

	tests := map[string]struct {
		bundle bp.TransactionBundle
		err    error
	}{
		"payment transaction does not target bundle address": {
			bundle: bp.TransactionBundle{
				Version: 1,
				Bundle: types.Transactions{
					types.NewTx(&types.AccessListTx{
						AccessList: types.AccessList{
							{Address: bp.BundleOnly},
						},
					}),
				},
				Payment: types.NewTx(&types.AccessListTx{
					To: nil, // payment transaction with no recipient
				}),
				Flags: 0,
			},
			err: ErrPaymentDoesNotTargetBundleAddress,
		},
		"payment transaction with insufficient gas": {
			bundle: bp.TransactionBundle{
				Version: 1,
				Bundle: types.Transactions{
					types.NewTx(&types.AccessListTx{
						Gas: 50, // gas of the transaction in the bundle
						AccessList: types.AccessList{
							{Address: bp.BundleOnly},
						},
					}),
				},
				Payment: types.NewTx(&types.AccessListTx{
					Gas: 49, // gas of the payment transaction is less than the gas of the transaction in the bundle
					To:  &bp.BundleAddress,
				}),
				Flags: 0,
			},
			err: ErrPaymentGasTooLow,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockSigner := bp.NewMockSigner(ctrl)
			mockSigner.EXPECT().Sender(gomock.Any()).Return(common.Address{0x42}, nil)

			err := validatePaymentTx(
				test.bundle.Payment,
				test.bundle.Bundle,
				mockSigner,
				common.Address{0x42},
			)
			require.ErrorIs(t, err, test.err)
		})
	}
}

// "payment transaction sender does not match bundle sender": {
