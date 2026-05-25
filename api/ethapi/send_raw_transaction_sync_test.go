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

var (
	now   = time.Now
	since = time.Since
)

// newTestTx creates a signed EIP-155 transaction for use in tests.
// chainID defaults to 1 if nil.
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

// setupSendRawSyncAPI creates a mock backend and PublicTransactionPoolAPI for unit tests.
func setupSendRawSyncAPI(t *testing.T) (*MockBackend, *PublicTransactionPoolAPI) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockBackend := NewMockBackend(ctrl)
	mockBackend.EXPECT().ChainID().Return(big.NewInt(1)).AnyTimes()
	api := NewPublicTransactionPoolAPI(mockBackend, &AddrLocker{})
	return mockBackend, api
}

func TestSendRawTransactionSync_ReturnsReceipt(t *testing.T) {
	mockBackend, api := setupSendRawSyncAPI(t)

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

	mockBackend.EXPECT().GetPoolNonce(gomock.Any(), gomock.Any()).Return(uint64(0), nil)
	mockBackend.EXPECT().RPCTxFeeCap().Return(float64(0))
	mockBackend.EXPECT().UnprotectedAllowed().Return(false)
	mockBackend.EXPECT().SendTx(gomock.Any(), gomock.Any()).Return(nil)
	mockBackend.EXPECT().GetTransaction(gomock.Any(), txHash).Return(tx, uint64(1), uint64(0), nil)
	mockBackend.EXPECT().BlockByNumber(gomock.Any(), gomock.Any()).Return(block, nil)
	mockBackend.EXPECT().FetchReceiptsForBlock(block).Return(types.Receipts{receipt})
	mockBackend.EXPECT().ChainConfig(gomock.Any()).Return(&params.ChainConfig{ChainID: big.NewInt(1)}).AnyTimes()

	result, err := api.SendRawTransactionSync(context.Background(), encoded, nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, hexutil.Uint(types.ReceiptStatusSuccessful), result["status"])
}

func TestSendRawTransactionSync_NonceGap_ReturnsCode6(t *testing.T) {
	mockBackend, api := setupSendRawSyncAPI(t)

	// tx has nonce=5 but pool expects nonce=0 → gap
	_, encoded := newTestTx(t, 5, nil)

	mockBackend.EXPECT().GetPoolNonce(gomock.Any(), gomock.Any()).Return(uint64(0), nil)
	// SendTx must NOT be called on nonce gap

	result, err := api.SendRawTransactionSync(context.Background(), encoded, nil)

	require.Nil(t, result)
	require.Error(t, err)
	var syncErr *sendRawSyncError
	require.True(t, errors.As(err, &syncErr), "expected sendRawSyncError, got %T: %v", err, err)
	require.Equal(t, errCodeSendRawSyncNonceGap, syncErr.ErrorCode())
	require.Equal(t, hexutil.Uint64(0), syncErr.ErrorData())
}

func TestSendRawTransactionSync_Timeout_TxInPool_ReturnsCode5(t *testing.T) {
	mockBackend, api := setupSendRawSyncAPI(t)

	tx, encoded := newTestTx(t, 0, nil)
	txHash := tx.Hash()
	timeoutMs := hexutil.Uint64(10) // very short, 10ms

	mockBackend.EXPECT().GetPoolNonce(gomock.Any(), gomock.Any()).Return(uint64(0), nil)
	mockBackend.EXPECT().RPCTxFeeCap().Return(float64(0))
	mockBackend.EXPECT().UnprotectedAllowed().Return(false)
	mockBackend.EXPECT().SendTx(gomock.Any(), gomock.Any()).Return(nil)
	// GetTransaction always returns nil (not confirmed)
	mockBackend.EXPECT().GetTransaction(gomock.Any(), txHash).Return(nil, uint64(0), uint64(0), nil).AnyTimes()
	// tx still in pool after timeout
	mockBackend.EXPECT().GetPoolTransaction(txHash).Return(tx)

	result, err := api.SendRawTransactionSync(context.Background(), encoded, &timeoutMs)

	require.Nil(t, result)
	require.Error(t, err)
	var syncErr *sendRawSyncError
	require.True(t, errors.As(err, &syncErr))
	require.Equal(t, errCodeSendRawSyncQueued, syncErr.ErrorCode())
}

func TestSendRawTransactionSync_Timeout_TxNotInPool_ReturnsCode4(t *testing.T) {
	mockBackend, api := setupSendRawSyncAPI(t)

	tx, encoded := newTestTx(t, 0, nil)
	txHash := tx.Hash()
	timeoutMs := hexutil.Uint64(10) // very short, 10ms

	mockBackend.EXPECT().GetPoolNonce(gomock.Any(), gomock.Any()).Return(uint64(0), nil)
	mockBackend.EXPECT().RPCTxFeeCap().Return(float64(0))
	mockBackend.EXPECT().UnprotectedAllowed().Return(false)
	mockBackend.EXPECT().SendTx(gomock.Any(), gomock.Any()).Return(nil)
	// GetTransaction always returns nil
	mockBackend.EXPECT().GetTransaction(gomock.Any(), txHash).Return(nil, uint64(0), uint64(0), nil).AnyTimes()
	// tx not in pool after timeout
	mockBackend.EXPECT().GetPoolTransaction(txHash).Return(nil)

	result, err := api.SendRawTransactionSync(context.Background(), encoded, &timeoutMs)

	require.Nil(t, result)
	require.Error(t, err)
	var syncErr *sendRawSyncError
	require.True(t, errors.As(err, &syncErr))
	require.Equal(t, errCodeSendRawSyncTimeout, syncErr.ErrorCode())
}

func TestSendRawTransactionSync_SendTxError_PropagatesError(t *testing.T) {
	mockBackend, api := setupSendRawSyncAPI(t)

	_, encoded := newTestTx(t, 0, nil)
	poolErr := errors.New("insufficient funds")

	mockBackend.EXPECT().GetPoolNonce(gomock.Any(), gomock.Any()).Return(uint64(0), nil)
	mockBackend.EXPECT().RPCTxFeeCap().Return(float64(0))
	mockBackend.EXPECT().UnprotectedAllowed().Return(false)
	mockBackend.EXPECT().SendTx(gomock.Any(), gomock.Any()).Return(poolErr)

	result, err := api.SendRawTransactionSync(context.Background(), encoded, nil)

	require.Nil(t, result)
	require.ErrorIs(t, err, poolErr)
}

func TestSendRawTransactionSync_InvalidRLP_ReturnsDecodeError(t *testing.T) {
	_, api := setupSendRawSyncAPI(t)

	invalidEncoded := hexutil.Bytes([]byte{0xde, 0xad, 0xbe, 0xef})

	result, err := api.SendRawTransactionSync(context.Background(), invalidEncoded, nil)

	require.Nil(t, result)
	require.Error(t, err)
	// No backend calls expected — gomock will fail the test if any are made
}

func TestSendRawTransactionSync_CustomTimeout_IsHonored(t *testing.T) {
	mockBackend, api := setupSendRawSyncAPI(t)

	tx, encoded := newTestTx(t, 0, nil)
	txHash := tx.Hash()
	timeoutMs := hexutil.Uint64(50) // 50ms, well under the 2s default

	mockBackend.EXPECT().GetPoolNonce(gomock.Any(), gomock.Any()).Return(uint64(0), nil)
	mockBackend.EXPECT().RPCTxFeeCap().Return(float64(0))
	mockBackend.EXPECT().UnprotectedAllowed().Return(false)
	mockBackend.EXPECT().SendTx(gomock.Any(), gomock.Any()).Return(nil)
	mockBackend.EXPECT().GetTransaction(gomock.Any(), txHash).Return(nil, uint64(0), uint64(0), nil).AnyTimes()
	mockBackend.EXPECT().GetPoolTransaction(txHash).Return(nil)

	start := now()
	_, err := api.SendRawTransactionSync(context.Background(), encoded, &timeoutMs)
	elapsed := since(start)

	require.Error(t, err)
	// Should return well before the 2s default timeout
	require.Less(t, elapsed.Milliseconds(), int64(1500), "should have returned before default 2s timeout")
}

func TestSendRawTransactionSync_ContextCanceled_ReturnsError(t *testing.T) {
	mockBackend, api := setupSendRawSyncAPI(t)

	tx, encoded := newTestTx(t, 0, nil)
	txHash := tx.Hash()

	ctx, cancel := context.WithCancel(context.Background())

	mockBackend.EXPECT().GetPoolNonce(gomock.Any(), gomock.Any()).Return(uint64(0), nil)
	mockBackend.EXPECT().RPCTxFeeCap().Return(float64(0))
	mockBackend.EXPECT().UnprotectedAllowed().Return(false)
	mockBackend.EXPECT().SendTx(gomock.Any(), gomock.Any()).Return(nil)
	mockBackend.EXPECT().GetTransaction(gomock.Any(), txHash).DoAndReturn(
		func(ctx context.Context, _ common.Hash) (*types.Transaction, uint64, uint64, error) {
			cancel() // cancel the parent context on first poll
			return nil, 0, 0, nil
		},
	).AnyTimes()
	mockBackend.EXPECT().GetPoolTransaction(txHash).Return(nil).AnyTimes()

	_, err := api.SendRawTransactionSync(ctx, encoded, nil)

	require.Error(t, err)
}
