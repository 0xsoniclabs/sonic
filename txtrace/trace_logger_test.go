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

package txtrace

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
)

func TestToParityError_ConvertsGethErrors(t *testing.T) {
	tests := map[string]struct {
		err  error
		want string
	}{
		"out of gas": {
			err:  vm.ErrOutOfGas,
			want: "Out of gas",
		},
		"code store out of gas": {
			err:  vm.ErrCodeStoreOutOfGas,
			want: "Out of gas",
		},
		"gas uint64 overflow": {
			err:  vm.ErrGasUintOverflow,
			want: "Out of gas",
		},
		"max code size exceeded": {
			err:  vm.ErrMaxCodeSizeExceeded,
			want: "Out of gas",
		},
		"invalid jump": {
			err:  vm.ErrInvalidJump,
			want: "Bad jump destination",
		},
		"reverted": {
			err:  vm.ErrExecutionReverted,
			want: "Reverted",
		},
		"return data out of bounds": {
			err:  vm.ErrReturnDataOutOfBounds,
			want: "Out of bounds",
		},
		"precompiled failed": {
			err:  errors.New("precompiled failed"),
			want: "Built-in failed",
		},
		"invalid input length": {
			err:  errors.New("invalid input length"),
			want: "Built-in failed",
		},
		"wrapped out of gas, write protection": {
			err:  fmt.Errorf("%w: %v", vm.ErrOutOfGas, vm.ErrWriteProtection),
			want: "Out of gas",
		},
		"wrapped out of gas, gas uint64 overflow": {
			err:  fmt.Errorf("%w: %v", vm.ErrOutOfGas, vm.ErrGasUintOverflow),
			want: "Out of gas",
		},
		"wrapped out of gas, max initcode size exceeded": {
			err: fmt.Errorf("%w: %v", vm.ErrOutOfGas,
				fmt.Errorf("%w: code size %v limit %v", vm.ErrMaxInitCodeSizeExceeded, 98305, 49152)),
			want: "Out of gas",
		},
		"stack underflow": {
			err:  errors.New("stack underflow (0 <=> 2)"),
			want: "Stack underflow",
		},
		"stack underflow with other operands": {
			err:  errors.New("stack underflow (5 <=> 7)"),
			want: "Stack underflow",
		},
		"invalid opcode INVALID": {
			err:  errors.New("invalid opcode: INVALID"),
			want: "Bad instruction",
		},
		"invalid opcode not defined": {
			err:  errors.New("invalid opcode: opcode 0xef not defined"),
			want: "Bad instruction",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := toParityError(test.err.Error())
			require.True(t, ok)
			require.Equal(t, test.want, got)
		})
	}
}

func TestToParityError_KeepsUnknownErrors(t *testing.T) {
	for _, gethError := range []string{
		"",
		"some unknown error",
		"write protection",
		"Out of gas",
		"not out of gas: write protection",
	} {
		t.Run(gethError, func(t *testing.T) {
			_, ok := toParityError(gethError)
			require.False(t, ok)
		})
	}
}

func TestActionTrace_ProcessOutput_ReportsParityErrorNames(t *testing.T) {
	tests := map[string]struct {
		err  error
		want string
	}{
		"wrapped out of gas": {
			err:  fmt.Errorf("%w: %v", vm.ErrOutOfGas, vm.ErrWriteProtection),
			want: "Out of gas",
		},
		"stack underflow": {
			err:  errors.New("stack underflow (0 <=> 2)"),
			want: "Stack underflow",
		},
		"invalid opcode": {
			err:  errors.New("invalid opcode: INVALID"),
			want: "Bad instruction",
		},
		"unknown error": {
			err:  errors.New("some unknown error"),
			want: "some unknown error",
		},
	}
	for name, test := range tests {
		for _, traceType := range []string{CALL, CREATE} {
			t.Run(name+"/"+traceType, func(t *testing.T) {
				to := common.Address{1}
				trace := CreateActionTrace(common.Hash{}, *big.NewInt(1), common.Hash{}, 0, traceType)
				trace.Action = NewAddressAction(common.Address{2}, 100, nil, &to, hexutil.Big{}, nil)
				trace.Result = &TraceActionResult{}

				trace.processOutput(nil, test.err, false)

				require.Equal(t, test.want, trace.Error)
				require.Nil(t, trace.Result)
			})
		}
	}
}
