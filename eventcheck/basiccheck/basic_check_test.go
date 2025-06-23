package basiccheck

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
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

func TestChecker_IntrinsicGas_LegacyCalculationDoesNotAccountForInitDataOrAuthList(t *testing.T) {

	tests := map[string]*types.Transaction{
		"legacyTx": types.NewTx(&types.LegacyTx{
			To:  nil,
			Gas: 21_000,
			// some data that takes
			Data: []byte("this is a string that is longer than 32 bytes, so it will cost more"),
		}),
		"setCodeTx": types.NewTx(&types.SetCodeTx{
			To:       common.Address{},
			Gas:      21_000,
			AuthList: []types.SetCodeAuthorization{{}}}),
	}

	for name, tx := range tests {
		t.Run(name, func(t *testing.T) {
			costLegacy, err := intrinsicGasLegacy(tx.Data(), tx.AccessList(), tx.To() == nil)
			require.NoError(t, err)

			// in sonic, Homestead, Istanbul and Shanghai are always active
			costNew, err := core.IntrinsicGas(tx.Data(), tx.AccessList(),
				tx.SetCodeAuthorizations(), tx.To() == nil, true, true, true)
			require.NoError(t, err)
			require.Greater(t, costNew, costLegacy)
		})
	}
}
