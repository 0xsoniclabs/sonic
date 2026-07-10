// Copyright 2026 Sonic Operations Ltd
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

package ethapi

import (
	"context"
	"errors"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newTestTx creates a signed EIP-155 transaction for use in tests.
// chainID defaults to 1 if nil.
// it returns a transaction and its marshalling.
func newTestTx(t *testing.T, nonce uint64, chainID *big.Int) (*types.Transaction, hexutil.Bytes) {
	t.Helper()
	if chainID == nil {
		chainID = big.NewInt(1)
	}
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	to := crypto.PubkeyToAddress(key.PublicKey)
	tx, err := types.SignTx(
		types.NewTx(&types.LegacyTx{
			Nonce:    nonce,
			Gas:      21000,
			GasPrice: big.NewInt(1e9),
			To:       &to,
		}),
		types.NewEIP155Signer(chainID),
		key,
	)
	require.NoError(t, err)
	encoded, err := tx.MarshalBinary()
	require.NoError(t, err)
	return tx, hexutil.Bytes(encoded)
}

// setupSendRawSyncAPI creates a mock backend and PublicTransactionPoolAPI for
// unit tests, with the given default and maximum sync-wait timeouts.
func setupSendRawSyncAPI(t *testing.T, defaultTimeout, maxTimeout time.Duration) (*MockBackend, *PublicTransactionPoolAPI) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockBackend := NewMockBackend(ctrl)
	mockBackend.EXPECT().ChainID().Return(big.NewInt(1)).AnyTimes()
	mockBackend.EXPECT().RPCTxSyncDefaultTimeout().Return(defaultTimeout).AnyTimes()
	mockBackend.EXPECT().RPCTxSyncMaxTimeout().Return(maxTimeout).AnyTimes()
	api := NewPublicTransactionPoolAPI(mockBackend, &AddrLocker{})
	return mockBackend, api
}

// expectSuccessfulSubmission registers the backend expectations for a
// transaction that passes the nonce check and is accepted by the pool.
func expectSuccessfulSubmission(mockBackend *MockBackend) {
	mockBackend.EXPECT().GetPoolNonce(gomock.Any(), gomock.Any()).Return(uint64(0), nil)
	mockBackend.EXPECT().RPCTxFeeCap().Return(float64(0))
	mockBackend.EXPECT().UnprotectedAllowed().Return(false)
	mockBackend.EXPECT().SendTx(gomock.Any(), gomock.Any()).Return(nil)
}

func TestSendRawTransactionSync_ReturnsReceipt(t *testing.T) {
	mockBackend, api := setupSendRawSyncAPI(t, 1*time.Second, 2*time.Second)

	tx, encoded := newTestTx(t, 0, nil)
	txHash := tx.Hash()

	block := &evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{Number: big.NewInt(1)},
	}
	receipt := &types.Receipt{
		Status:  types.ReceiptStatusSuccessful,
		TxHash:  txHash,
		GasUsed: 21000,
	}

	expectSuccessfulSubmission(mockBackend)
	mockBackend.EXPECT().GetTransaction(gomock.Any(), txHash).Return(tx, uint64(1), uint64(0), nil)
	mockBackend.EXPECT().BlockByNumber(gomock.Any(), gomock.Any()).Return(block, nil)
	mockBackend.EXPECT().FetchReceiptsForBlock(block).Return(types.Receipts{receipt})
	mockBackend.EXPECT().ChainConfig(gomock.Any()).Return(&params.ChainConfig{ChainID: big.NewInt(1)}).AnyTimes()

	result, err := api.SendRawTransactionSync(context.Background(), encoded, nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, hexutil.Uint(receipt.Status), result["status"])
	require.Equal(t, receipt.TxHash, result["transactionHash"])
	require.Equal(t, hexutil.Uint64(receipt.GasUsed), result["gasUsed"])
}

// TestSendRawTransactionSync_ErrorCases covers all pre-confirmation failure
// paths: request validation, nonce checks, and pool submission errors.
func TestSendRawTransactionSync_ErrorCases(t *testing.T) {
	poolErr := errors.New("insufficient funds")
	nonceErr := errors.New("pool nonce lookup failed")
	zero := hexutil.Uint64(0)

	tests := map[string]struct {
		txNonce    uint64
		invalidRLP bool
		timeoutMs  *hexutil.Uint64
		setupMocks func(*MockBackend)
		// wantCode/wantData checked only when wantCode != 0; wantData
		// receives the submitted tx to derive expected error data.
		wantCode    int
		wantData    func(tx *types.Transaction) interface{}
		wantErrText string
	}{
		"invalid RLP returns decode error before any backend call": {
			invalidRLP:  true,
			wantErrText: "rlp: value size exceeds available input length",
		},
		"zero timeout is rejected before submission": {
			timeoutMs:   &zero,
			wantErrText: "timeout must be greater than zero",
		},
		"pool nonce lookup error is propagated": {
			setupMocks: func(mockBackend *MockBackend) {
				mockBackend.EXPECT().GetPoolNonce(gomock.Any(), gomock.Any()).Return(uint64(0), nonceErr)
			},
			wantErrText: nonceErr.Error(),
		},
		"nonce gap returns code 6 without pool submission": {
			// tx has nonce=5 but pool expects nonce=0 → gap;
			// SendTx must NOT be called.
			txNonce: 5,
			setupMocks: func(mockBackend *MockBackend) {
				mockBackend.EXPECT().GetPoolNonce(gomock.Any(), gomock.Any()).Return(uint64(0), nil)
			},
			wantCode: errCodeSendRawSyncNonceGap,
			wantData: func(*types.Transaction) interface{} {
				return hexutil.Uint64(0)
			},
			wantErrText: "nonce gap",
		},
		"pool rejection returns code 5": {
			setupMocks: func(mockBackend *MockBackend) {
				mockBackend.EXPECT().GetPoolNonce(gomock.Any(), gomock.Any()).Return(uint64(0), nil)
				mockBackend.EXPECT().RPCTxFeeCap().Return(float64(0))
				mockBackend.EXPECT().UnprotectedAllowed().Return(false)
				mockBackend.EXPECT().SendTx(gomock.Any(), gomock.Any()).Return(poolErr)
			},
			wantCode: errCodeSendRawSyncRejected,
			wantData: func(tx *types.Transaction) interface{} {
				return tx.Hash()
			},
			wantErrText: poolErr.Error(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mockBackend, api := setupSendRawSyncAPI(t, 1*time.Second, 2*time.Second)
			if test.setupMocks != nil {
				test.setupMocks(mockBackend)
			}

			tx, encoded := newTestTx(t, test.txNonce, nil)
			if test.invalidRLP {
				encoded = hexutil.Bytes([]byte{0xde, 0xad, 0xbe, 0xef})
			}

			result, err := api.SendRawTransactionSync(context.Background(), encoded, test.timeoutMs)

			require.Nil(t, result)
			require.ErrorContains(t, err, test.wantErrText)
			if test.wantCode != 0 {
				var syncErr *sendRawSyncError
				require.True(t, errors.As(err, &syncErr), "expected sendRawSyncError, got %T: %v", err, err)
				require.Equal(t, test.wantCode, syncErr.ErrorCode())
				require.Equal(t, test.wantData(tx), syncErr.ErrorData())
			}
		})
	}
}

func TestSendRawTransactionSync_TimeoutHandling(t *testing.T) {
	uint64Ptr := func(v uint64) *hexutil.Uint64 {
		u := hexutil.Uint64(v)
		return &u
	}

	tests := map[string]struct {
		defaultTimeout time.Duration
		maxTimeout     time.Duration
		timeoutMs      *hexutil.Uint64
		// effective timeout the implementation is expected to apply
		wantTimeout time.Duration
	}{
		"nil timeout uses default": {
			defaultTimeout: 200 * time.Millisecond,
			maxTimeout:     10 * time.Second,
			timeoutMs:      nil,
			wantTimeout:    200 * time.Millisecond,
		},
		"custom timeout below max is honored": {
			defaultTimeout: 10 * time.Second,
			maxTimeout:     10 * time.Second,
			timeoutMs:      uint64Ptr(100),
			wantTimeout:    100 * time.Millisecond,
		},
		"custom timeout equal to max is honored": {
			defaultTimeout: 10 * time.Second,
			maxTimeout:     300 * time.Millisecond,
			timeoutMs:      uint64Ptr(300),
			wantTimeout:    300 * time.Millisecond,
		},
		"custom timeout above max is clamped to max": {
			defaultTimeout: 10 * time.Second,
			maxTimeout:     200 * time.Millisecond,
			timeoutMs:      uint64Ptr(60_000),
			wantTimeout:    200 * time.Millisecond,
		},
		"huge timeout does not overflow and is clamped to max": {
			defaultTimeout: 10 * time.Second,
			maxTimeout:     200 * time.Millisecond,
			timeoutMs:      uint64Ptr(math.MaxUint64),
			wantTimeout:    200 * time.Millisecond,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mockBackend, api := setupSendRawSyncAPI(t, test.defaultTimeout, test.maxTimeout)

			tx, encoded := newTestTx(t, 0, nil)
			txHash := tx.Hash()

			expectSuccessfulSubmission(mockBackend)
			// GetTransaction always reports not-yet-confirmed, so the call
			// can only end by hitting the effective timeout.
			mockBackend.EXPECT().GetTransaction(gomock.Any(), txHash).Return(nil, uint64(0), uint64(0), nil).AnyTimes()

			start := time.Now()
			result, err := api.SendRawTransactionSync(context.Background(), encoded, test.timeoutMs)
			elapsed := time.Since(start)

			require.Nil(t, result)
			require.Error(t, err)
			var syncErr *sendRawSyncError
			require.True(t, errors.As(err, &syncErr), "expected sendRawSyncError, got %T: %v", err, err)
			require.Equal(t, errCodeSendRawSyncTimeout, syncErr.ErrorCode())
			require.Equal(t, txHash, syncErr.ErrorData())

			require.GreaterOrEqual(t, elapsed, test.wantTimeout,
				"returned before the effective timeout elapsed")
			// Generous upper bound: well below the not-chosen timeouts
			// (default/max of 10s) while tolerating scheduling jitter.
			require.Less(t, elapsed, test.wantTimeout+2*time.Second,
				"did not return close to the effective timeout")
		})
	}
}

func TestSendRawTransactionSync_BlockUnavailable_ReturnsError(t *testing.T) {
	blockErr := errors.New("block lookup failed")

	tests := map[string]struct {
		block   *evmcore.EvmBlock
		err     error
		wantErr string
	}{
		"lookup error is propagated": {
			block:   nil,
			err:     blockErr,
			wantErr: blockErr.Error(),
		},
		"missing block yields explicit error": {
			block:   nil,
			err:     nil,
			wantErr: "block is unavailable",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mockBackend, api := setupSendRawSyncAPI(t, 1*time.Second, 2*time.Second)

			tx, encoded := newTestTx(t, 0, nil)
			txHash := tx.Hash()

			expectSuccessfulSubmission(mockBackend)
			mockBackend.EXPECT().GetTransaction(gomock.Any(), txHash).Return(tx, uint64(1), uint64(0), nil)
			mockBackend.EXPECT().BlockByNumber(gomock.Any(), gomock.Any()).Return(test.block, test.err)

			result, err := api.SendRawTransactionSync(context.Background(), encoded, nil)

			require.Nil(t, result)
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestSendRawTransactionSync_ContextCanceled_ReturnsTimeoutError(t *testing.T) {
	mockBackend, api := setupSendRawSyncAPI(t, 10*time.Second, 10*time.Second)

	tx, encoded := newTestTx(t, 0, nil)
	txHash := tx.Hash()

	ctx, cancel := context.WithCancel(context.Background())

	expectSuccessfulSubmission(mockBackend)
	mockBackend.EXPECT().GetTransaction(gomock.Any(), txHash).DoAndReturn(
		func(ctx context.Context, _ common.Hash) (*types.Transaction, uint64, uint64, error) {
			cancel() // cancel the parent context on first poll
			return nil, 0, 0, nil
		},
	).AnyTimes()

	start := time.Now()
	result, err := api.SendRawTransactionSync(ctx, encoded, nil)
	elapsed := time.Since(start)

	require.Nil(t, result)
	require.Error(t, err)
	var syncErr *sendRawSyncError
	require.True(t, errors.As(err, &syncErr), "expected sendRawSyncError, got %T: %v", err, err)
	require.Equal(t, errCodeSendRawSyncTimeout, syncErr.ErrorCode())
	// Cancellation must interrupt the wait long before the 10s timeout.
	require.Less(t, elapsed, 5*time.Second)
}
