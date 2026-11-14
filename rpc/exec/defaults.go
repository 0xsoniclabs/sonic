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

package exec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// SetDefaults fills what the arguments of a new transaction leave out: the
// fees from the oracle's suggestion and the head, the value, the pool's
// nonce of the sender, the gas by an estimation at the head, and the chain
// id. It checks what the transaction cannot hold, in the legacy order.
func (e *Environment) SetDefaults(ctx context.Context, head *evmcore.EvmBlock, args *CallArguments) error {
	if err := e.setFeeDefaults(head, args); err != nil {
		return err
	}
	if args.Value == nil {
		args.Value = new(hexutil.Big)
	}
	if args.Nonce == nil {
		nonce := e.reader.PoolNonce(args.Sender())
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if err := args.checkCallData(); err != nil {
		return err
	}
	if args.Gas == nil {
		// The estimation takes the data, not the input, as the legacy
		// defaulting does; clients see the same estimate.
		estimated, err := e.EstimateGas(ctx, head, Estimation{
			Arguments: CallArguments{
				From:                 args.From,
				To:                   args.To,
				GasPrice:             args.GasPrice,
				MaxFeePerGas:         args.MaxFeePerGas,
				MaxPriorityFeePerGas: args.MaxPriorityFeePerGas,
				Value:                args.Value,
				Data:                 args.Data,
				AccessList:           args.AccessList,
				BlobFeeCap:           args.BlobFeeCap,
				BlobHashes:           args.BlobHashes,
				AuthorizationList:    args.AuthorizationList,
			},
			Block: chain.ReferenceTo(chain.PendingBlockNumber),
		})
		if err != nil {
			return err
		}
		args.Gas = &estimated
	}
	if args.ChainID == nil {
		args.ChainID = (*hexutil.Big)(e.chainID)
		return nil
	}
	if have := args.ChainID.ToInt(); have.Cmp(e.chainID) != 0 {
		return fmt.Errorf("chainId does not match node's (have=%v, want=%v)", have, e.chainID)
	}
	return nil
}

// setFeeDefaults prices a transaction after London by its caps, defaulting
// the tip to the oracle's suggestion and the fee cap to the tip over the
// suggested price, and before London by the gas price.
func (e *Environment) setFeeDefaults(head *evmcore.EvmBlock, args *CallArguments) error {
	if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return errBothPricings
	}
	if args.GasPrice != nil && args.AuthorizationList != nil {
		return errors.New("both gasPrice and authorizationList specified")
	}
	if args.MaxPriorityFeePerGas != nil && args.MaxFeePerGas != nil {
		return checkFeeCaps(args)
	}
	london := e.ChainConfig(head.NumberU64()).IsLondon(head.Number)
	if !london || args.GasPrice != nil {
		if args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil {
			return errors.New("maxFeePerGas or maxPriorityFeePerGas specified but london is not active yet")
		}
		if args.GasPrice == nil {
			price, err := e.oracle.SuggestTip(oracle.DefaultCertainty)
			if err != nil {
				return err
			}
			args.GasPrice = (*hexutil.Big)(price.Add(price, e.SuggestedGasPrice(head)))
		}
		return nil
	}
	if args.MaxPriorityFeePerGas == nil {
		tip, err := e.oracle.SuggestTip(oracle.DefaultCertainty)
		if err != nil {
			return err
		}
		args.MaxPriorityFeePerGas = (*hexutil.Big)(tip)
	}
	if args.MaxFeePerGas == nil {
		feeCap := new(big.Int).Add(args.MaxPriorityFeePerGas.ToInt(), e.SuggestedGasPrice(head))
		args.MaxFeePerGas = (*hexutil.Big)(feeCap)
	}
	return checkFeeCaps(args)
}

func checkFeeCaps(args *CallArguments) error {
	if args.MaxFeePerGas.ToInt().Cmp(args.MaxPriorityFeePerGas.ToInt()) < 0 {
		return fmt.Errorf("maxFeePerGas (%v) < maxPriorityFeePerGas (%v)", args.MaxFeePerGas, args.MaxPriorityFeePerGas)
	}
	return nil
}

// checkCallData refuses data and input that differ, and a creation without
// code or with what only a call can carry.
func (a *CallArguments) checkCallData() error {
	if a.Data != nil && a.Input != nil && !bytes.Equal(*a.Data, *a.Input) {
		return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	}
	if a.To != nil {
		return nil
	}
	switch {
	case a.BlobHashes != nil || a.BlobFeeCap != nil:
		return errors.New(`missing "to" in blob transaction`)
	case len(a.CallData()) == 0:
		return errors.New(`contract creation without any data provided`)
	case a.AuthorizationList != nil:
		return errors.New(`authorizationList provided for contract creation, but "to" field is missing`)
	}
	return nil
}
