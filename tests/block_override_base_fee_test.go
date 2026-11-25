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

package tests

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// gasPriceAndBaseFeeCode returns the GASPRICE and BASEFEE values seen by the
// EVM as two 32-byte words.
var gasPriceAndBaseFeeCode = hexutil.Bytes{
	byte(vm.GASPRICE), byte(vm.PUSH1), 0x00, byte(vm.MSTORE),
	byte(vm.BASEFEE), byte(vm.PUSH1), 0x20, byte(vm.MSTORE),
	byte(vm.PUSH1), 0x40, byte(vm.PUSH1), 0x00, byte(vm.RETURN),
}

// feeObservation is what a call observed about its fees.
type feeObservation struct {
	GasPrice *big.Int
	BaseFee  *big.Int
}

// feeCall runs a call against the fee observer contract with the given block
// overrides and returns what the contract observed.
type feeCall func(
	t *testing.T,
	client *rpc.Client,
	args ethapi.TransactionArgs,
	block uint64,
	state ethapi.StateOverride,
	overrides *ethapi.BlockOverrides,
) (feeObservation, error)

// TestBlockOverride_BaseFeeDeterminesGasPrice checks that RPC methods taking
// block overrides price the call with the overridden base fee, so that the
// EVM never sees a GASPRICE below BASEFEE for a dynamic-fee call, and that
// the fee checks are made against the overridden base fee.
func TestBlockOverride_BaseFeeDeterminesGasPrice(t *testing.T) {
	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	t.Parallel()

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	header, err := client.HeaderByNumber(t.Context(), nil)
	require.NoError(t, err)
	blockNumber := header.Number.Uint64()
	chainBaseFee := header.BaseFee
	require.NotNil(t, chainBaseFee)
	require.Positive(t, chainBaseFee.Sign(), "the chain base fee must be positive")

	observer := common.Address{0xfe, 0xe0}
	funded := common.Address{0xfe, 0xe1}
	unfunded := common.Address{0xfe, 0xe2}
	balance := (*hexutil.U256)(new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(30)))
	state := ethapi.StateOverride{
		observer: {Code: &gasPriceAndBaseFeeCode},
		funded:   {Balance: &balance},
	}

	gas := hexutil.Uint64(100_000)
	big0 := big.NewInt(0)
	add := func(a *big.Int, b int64) *big.Int { return new(big.Int).Add(a, big.NewInt(b)) }
	mul := func(a *big.Int, b int64) *big.Int { return new(big.Int).Mul(a, big.NewInt(b)) }

	highBaseFee := mul(chainBaseFee, 10)
	lowBaseFee := big.NewInt(1)

	type fees struct {
		gasPrice, maxFee, maxTip *big.Int
	}
	tests := map[string]struct {
		baseFee *big.Int // nil means no base fee override
		from    common.Address
		fees    fees
		// wantGasPrice is checked if wantErr is empty.
		wantGasPrice *big.Int
		wantErr      string
		// wantEstimateErr replaces wantErr for eth_estimateGas, if set.
		wantEstimateErr string
	}{
		"no fees, no override": {
			fees:         fees{},
			wantGasPrice: big0,
		},
		"no fees, high base fee": {
			baseFee:      highBaseFee,
			wantGasPrice: big0,
		},
		"no fees, unfunded sender, high base fee": {
			baseFee:      highBaseFee,
			from:         unfunded,
			wantGasPrice: big0,
		},
		"dynamic fee, no override": {
			fees:         fees{maxFee: mul(chainBaseFee, 2), maxTip: big.NewInt(1)},
			wantGasPrice: add(chainBaseFee, 1),
		},
		"dynamic fee, base fee raised above chain base fee": {
			baseFee:      highBaseFee,
			fees:         fees{maxFee: mul(highBaseFee, 2), maxTip: big.NewInt(1)},
			wantGasPrice: add(highBaseFee, 1),
		},
		"dynamic fee, base fee lowered below chain base fee": {
			baseFee:      lowBaseFee,
			fees:         fees{maxFee: mul(chainBaseFee, 2), maxTip: big.NewInt(1)},
			wantGasPrice: add(lowBaseFee, 1),
		},
		"dynamic fee, base fee overridden to zero": {
			baseFee:      big0,
			fees:         fees{maxFee: mul(chainBaseFee, 2), maxTip: big.NewInt(7)},
			wantGasPrice: big.NewInt(7),
		},
		"dynamic fee, max fee caps the price": {
			baseFee:      highBaseFee,
			fees:         fees{maxFee: add(highBaseFee, 5), maxTip: big.NewInt(100)},
			wantGasPrice: add(highBaseFee, 5),
		},
		"dynamic fee, max fee equal to base fee": {
			baseFee:      highBaseFee,
			fees:         fees{maxFee: highBaseFee},
			wantGasPrice: highBaseFee,
		},
		"dynamic fee, only tip": {
			baseFee: highBaseFee,
			fees:    fees{maxTip: big.NewInt(1)},
			wantErr: "max priority fee per gas higher than max fee per gas",
		},
		"dynamic fee, max fee below overridden base fee": {
			baseFee: highBaseFee,
			fees:    fees{maxFee: add(highBaseFee, -1)},
			wantErr: "max fee per gas less than block base fee",
		},
		"dynamic fee, max fee above chain but below overridden base fee": {
			baseFee: highBaseFee,
			fees:    fees{maxFee: mul(chainBaseFee, 2)},
			wantErr: "max fee per gas less than block base fee",
		},
		"dynamic fee, max fee below chain but above overridden base fee": {
			baseFee:      lowBaseFee,
			fees:         fees{maxFee: add(chainBaseFee, -1), maxTip: add(chainBaseFee, -1)},
			wantGasPrice: add(chainBaseFee, -1),
		},
		"dynamic fee, tip above max fee": {
			baseFee: highBaseFee,
			fees:    fees{maxFee: highBaseFee, maxTip: add(highBaseFee, 1)},
			wantErr: "max priority fee per gas higher than max fee per gas",
		},
		"dynamic fee, unfunded sender": {
			baseFee: highBaseFee,
			from:    unfunded,
			fees:    fees{maxFee: highBaseFee},
			wantErr: "insufficient funds",
			// The estimation caps the gas by what the sender can afford.
			wantEstimateErr: "gas required exceeds allowance (0)",
		},
		"legacy, gas price above overridden base fee": {
			baseFee:      highBaseFee,
			fees:         fees{gasPrice: add(highBaseFee, 10)},
			wantGasPrice: add(highBaseFee, 10),
		},
		"legacy, gas price equal to overridden base fee": {
			baseFee:      highBaseFee,
			fees:         fees{gasPrice: highBaseFee},
			wantGasPrice: highBaseFee,
		},
		"legacy, gas price below overridden base fee": {
			baseFee: highBaseFee,
			fees:    fees{gasPrice: add(highBaseFee, -1)},
			wantErr: "max fee per gas less than block base fee",
		},
		"legacy, gas price below chain but above overridden base fee": {
			baseFee:      lowBaseFee,
			fees:         fees{gasPrice: add(chainBaseFee, -1)},
			wantGasPrice: add(chainBaseFee, -1),
		},
	}

	methods := map[string]feeCall{
		"eth_call":        feeObservingEthCall,
		"debug_traceCall": feeObservingDebugTraceCall,
		"eth_simulateV1":  feeObservingSimulateV1,
		"eth_estimateGas": feeObservingEstimateGas,
	}

	for methodName, method := range methods {
		t.Run(methodName, func(t *testing.T) {
			for name, test := range tests {
				t.Run(name, func(t *testing.T) {
					from := test.from
					if from == (common.Address{}) {
						from = funded
					}
					args := ethapi.TransactionArgs{
						From:                 &from,
						To:                   &observer,
						Gas:                  &gas,
						GasPrice:             toHexBig(test.fees.gasPrice),
						MaxFeePerGas:         toHexBig(test.fees.maxFee),
						MaxPriorityFeePerGas: toHexBig(test.fees.maxTip),
					}
					var overrides *ethapi.BlockOverrides
					wantBaseFee := chainBaseFee
					if test.baseFee != nil {
						overrides = &ethapi.BlockOverrides{BaseFee: toHexBig(test.baseFee)}
						wantBaseFee = test.baseFee
					}
					// A call that names no gas price runs with a base fee of
					// zero, as in go-ethereum. eth_simulateV1 keeps the base
					// fee of its simulated block.
					if test.wantGasPrice != nil && test.wantGasPrice.Sign() == 0 && methodName != "eth_simulateV1" {
						wantBaseFee = big0
					}

					wantErr := test.wantErr
					if methodName == "eth_estimateGas" && test.wantEstimateErr != "" {
						wantErr = test.wantEstimateErr
					}

					got, err := method(t, client.Client(), args, blockNumber, state, overrides)
					if wantErr != "" {
						require.ErrorContains(t, err, wantErr)
						return
					}
					require.NoError(t, err)
					if got.GasPrice == nil {
						return // the method does not report the observed fees
					}
					require.Equal(t, test.wantGasPrice.String(), got.GasPrice.String(), "GASPRICE")
					require.Equal(t, wantBaseFee.String(), got.BaseFee.String(), "BASEFEE")
				})
			}
		})
	}

	// trace_call takes block overrides in its config but does not apply them,
	// so it keeps pricing and running the call with the block's base fee.
	t.Run("trace_call ignores block overrides", func(t *testing.T) {
		from := funded
		args := ethapi.TransactionArgs{
			From:                 &from,
			To:                   &observer,
			Gas:                  &gas,
			MaxFeePerGas:         toHexBig(mul(chainBaseFee, 2)),
			MaxPriorityFeePerGas: toHexBig(big.NewInt(1)),
		}
		got, err := feeObservingTraceCall(t, client.Client(), args, blockNumber, state,
			&ethapi.BlockOverrides{BaseFee: toHexBig(highBaseFee)})
		require.NoError(t, err)
		require.Equal(t, add(chainBaseFee, 1).String(), got.GasPrice.String(), "GASPRICE")
		require.Equal(t, chainBaseFee.String(), got.BaseFee.String(), "BASEFEE")
	})
}

func toHexBig(v *big.Int) *hexutil.Big {
	if v == nil {
		return nil
	}
	return (*hexutil.Big)(new(big.Int).Set(v))
}

func decodeFeeObservation(data []byte) (feeObservation, error) {
	if len(data) != 64 {
		return feeObservation{}, fmt.Errorf("unexpected result length %d, want 64", len(data))
	}
	return feeObservation{
		GasPrice: new(big.Int).SetBytes(data[:32]),
		BaseFee:  new(big.Int).SetBytes(data[32:]),
	}, nil
}

func feeObservingEthCall(
	t *testing.T, client *rpc.Client, args ethapi.TransactionArgs, block uint64,
	state ethapi.StateOverride, overrides *ethapi.BlockOverrides,
) (feeObservation, error) {
	var res hexutil.Bytes
	err := client.CallContext(t.Context(), &res, "eth_call",
		args, hexutil.EncodeUint64(block), state, overrides)
	if err != nil {
		return feeObservation{}, err
	}
	return decodeFeeObservation(res)
}

func feeObservingDebugTraceCall(
	t *testing.T, client *rpc.Client, args ethapi.TransactionArgs, block uint64,
	state ethapi.StateOverride, overrides *ethapi.BlockOverrides,
) (feeObservation, error) {
	config := ethapi.TraceCallConfig{StateOverrides: &state, BlockOverrides: overrides}
	var res struct {
		Failed      bool   `json:"failed"`
		ReturnValue string `json:"returnValue"`
	}
	err := client.CallContext(t.Context(), &res, "debug_traceCall",
		args, hexutil.EncodeUint64(block), config)
	if err != nil {
		return feeObservation{}, err
	}
	if res.Failed {
		return feeObservation{}, fmt.Errorf("traced call failed")
	}
	data, err := hexutil.Decode("0x" + strings.TrimPrefix(res.ReturnValue, "0x"))
	if err != nil {
		return feeObservation{}, err
	}
	return decodeFeeObservation(data)
}

func feeObservingSimulateV1(
	t *testing.T, client *rpc.Client, args ethapi.TransactionArgs, block uint64,
	state ethapi.StateOverride, overrides *ethapi.BlockOverrides,
) (feeObservation, error) {
	blockOverrides := map[string]any{}
	if overrides != nil && overrides.BaseFee != nil {
		blockOverrides["baseFeePerGas"] = overrides.BaseFee
	}
	opts := map[string]any{
		"blockStateCalls": []any{
			map[string]any{
				"blockOverrides": blockOverrides,
				"stateOverrides": state,
				"calls":          []any{args},
			},
		},
	}
	var res []struct {
		Calls []struct {
			ReturnData hexutil.Bytes `json:"returnData"`
			Status     string        `json:"status"`
			Error      *struct {
				Message string `json:"message"`
			} `json:"error"`
		} `json:"calls"`
	}
	err := client.CallContext(t.Context(), &res, "eth_simulateV1", opts, hexutil.EncodeUint64(block))
	if err != nil {
		return feeObservation{}, err
	}
	if len(res) != 1 || len(res[0].Calls) != 1 {
		return feeObservation{}, fmt.Errorf("unexpected simulation result shape: %+v", res)
	}
	call := res[0].Calls[0]
	if call.Error != nil {
		return feeObservation{}, fmt.Errorf("simulated call failed: %s", call.Error.Message)
	}
	return decodeFeeObservation(call.ReturnData)
}

// feeObservingEstimateGas only checks whether the call can be estimated, as
// eth_estimateGas does not return the output of the call.
func feeObservingEstimateGas(
	t *testing.T, client *rpc.Client, args ethapi.TransactionArgs, block uint64,
	state ethapi.StateOverride, overrides *ethapi.BlockOverrides,
) (feeObservation, error) {
	args.Gas = nil // let the estimation search for the gas limit
	var res hexutil.Uint64
	err := client.CallContext(t.Context(), &res, "eth_estimateGas",
		args, hexutil.EncodeUint64(block), state, overrides)
	if err != nil {
		return feeObservation{}, err
	}
	if res == 0 {
		return feeObservation{}, fmt.Errorf("estimated gas is zero")
	}
	return feeObservation{}, nil
}

func feeObservingTraceCall(
	t *testing.T, client *rpc.Client, args ethapi.TransactionArgs, block uint64,
	state ethapi.StateOverride, overrides *ethapi.BlockOverrides,
) (feeObservation, error) {
	config := ethapi.TraceCallConfig{StateOverrides: &state, BlockOverrides: overrides}
	var res struct {
		Output hexutil.Bytes `json:"output"`
	}
	err := client.CallContext(t.Context(), &res, "trace_call",
		args, []string{"trace"}, hexutil.EncodeUint64(block), config)
	if err != nil {
		return feeObservation{}, err
	}
	return decodeFeeObservation(res.Output)
}
