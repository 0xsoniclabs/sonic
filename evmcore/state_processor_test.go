package evmcore

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"go.uber.org/mock/gomock"
)

func TestApplyTransaction_InternalTransactionsSkipBaseFeeCharges(t *testing.T) {
	for _, internal := range []bool{true, false} {
		t.Run("internal="+fmt.Sprint(internal), func(t *testing.T) {
			ctxt := gomock.NewController(t)
			state := state.NewMockStateDB(ctxt)

			any := gomock.Any()
			state.EXPECT().GetBalance(any).Return(uint256.NewInt(0))
			state.EXPECT().SubBalance(any, any, any)
			if !internal {
				state.EXPECT().GetNonce(any)
				state.EXPECT().GetCode(any)
			}

			evm := vm.NewEVM(vm.BlockContext{}, state, &params.ChainConfig{}, vm.Config{})
			gp := new(core.GasPool).AddGas(1000000)

			// The transaction will fail for various reasons, but for this test
			// this is not relevant. We just want to check if the base fee
			// configuration flag is updated to match the SkipAccountChecks flag.
			_, _, _, err := applyTransaction(&core.Message{
				SkipNonceChecks:  internal,
				SkipFromEOACheck: internal,
				GasPrice:         big.NewInt(0),
				Value:            big.NewInt(0),
			}, gp, state, nil, nil, nil, evm, nil)
			if err == nil {
				t.Errorf("expected transaction to fail")
			}

			if want, got := internal, evm.Config.NoBaseFee; want != got {
				t.Fatalf("want %v, got %v", want, got)
			}
		})
	}
}
