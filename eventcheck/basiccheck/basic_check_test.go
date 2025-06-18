package basiccheck

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestChecker_checkTxs_AcceptsValidTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	event := inter.NewMockEventPayloadI(ctrl)

	valid := types.NewTx(&types.LegacyTx{To: &common.Address{}, Gas: 21000})
	require.NoError(t, validateTx(valid, 0))

	event.EXPECT().Transactions().Return(types.Transactions{valid}).AnyTimes()
	event.EXPECT().Payload().Return(&inter.Payload{}).AnyTimes()
	event.EXPECT().Version().Return(uint8(0))

	err := New().checkTxs(event)
	require.NoError(t, err)
}

func TestChecker_checkTxs_DetectsIssuesInTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	event := inter.NewMockEventPayloadI(ctrl)

	invalid := types.NewTx(&types.LegacyTx{
		Value: big.NewInt(-1),
	})

	event.EXPECT().Transactions().Return(types.Transactions{invalid}).AnyTimes()
	event.EXPECT().Payload().Return(&inter.Payload{}).AnyTimes()
	event.EXPECT().Version().Return(uint8(0))

	err := New().checkTxs(event)
	require.Error(t, err)
}

func TestChecker_checkTxs_UsesCorrectIntrinsicGasCalculation(t *testing.T) {
	ctrl := gomock.NewController(t)
	event := inter.NewMockEventPayloadI(ctrl)

	valid := types.NewTx(&types.SetCodeTx{
		To:       common.Address{},
		Gas:      21_000,
		AuthList: []types.SetCodeAuthorization{{}}})
	// in a real life scenario no SetCodeTx would be created with a version < 3
	require.NoError(t, validateTx(valid, 2))

	event.EXPECT().Transactions().Return(types.Transactions{valid}).AnyTimes()
	event.EXPECT().Payload().Return(&inter.Payload{}).AnyTimes()
	event.EXPECT().Version().Return(uint8(2))

	err := New().checkTxs(event)
	require.NoError(t, err)

	require.Error(t, validateTx(valid, 3))

	event.EXPECT().Transactions().Return(types.Transactions{valid}).AnyTimes()
	event.EXPECT().Payload().Return(&inter.Payload{}).AnyTimes()
	event.EXPECT().Version().Return(uint8(3))

	err = New().checkTxs(event)
	require.Error(t, err)
}

func TestChecker_checkTxs_RejectsLegacyCreateCallsBecauseOfInitCodeWordGas(t *testing.T) {
	tx := types.NewTx(&types.LegacyTx{
		To:   nil,    // This is a create transaction
		Gas:  54_072, // enough for a create cost, but not enough for the init code word gas
		Data: []byte("this is a string that is longer than 32 bytes, so it will cost more"),
	})
	require.NoError(t, validateTx(tx, 2))
	require.Error(t, validateTx(tx, 3))
}
