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
	require.NoError(t, validateTx(valid))

	event.EXPECT().Transactions().Return(types.Transactions{valid}).AnyTimes()
	event.EXPECT().Payload().Return(&inter.Payload{}).AnyTimes()

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

	err := New().checkTxs(event)
	require.Error(t, err)
}
