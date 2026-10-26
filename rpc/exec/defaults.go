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
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
)

// SetDefaults fills in the fields a client left out of a transaction, as the
// reference's setDefaults did: the fees of a dynamic-fee transaction from the
// oracle's tip and the suggested gas price, a zero value, the pool's nonce,
// the estimated gas and the chain id. London is active from genesis on Sonic,
// so the reference's defaulting before London is not ported.
func (environment *Environment) SetDefaults(head Head, arguments *CallArguments, poolNonce func(common.Address) uint64) error {
	if err := environment.setFees(head, arguments); err != nil {
		return err
	}
	if arguments.Value == nil {
		arguments.Value = new(hexutil.Big)
	}
	if arguments.Nonce == nil {
		nonce := poolNonce(arguments.Sender())
		arguments.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if err := arguments.checkData(); err != nil {
		return err
	}
	if arguments.Gas == nil {
		estimated, err := environment.EstimateGas(head, Estimation{Arguments: arguments.estimationArguments(), Block: chain.Latest()})
		if err != nil {
			return err
		}
		arguments.Gas = &estimated
		log.Trace("Estimate gas usage automatically", "gas", arguments.Gas)
	}
	return arguments.setChainID(environment.ChainID())
}

func (environment *Environment) setFees(head Head, arguments *CallArguments) error {
	if arguments.GasPrice != nil && (arguments.MaxFeePerGas != nil || arguments.MaxPriorityFeePerGas != nil) {
		return errBothPriceStyles
	}
	// A set-code transaction cannot be a legacy transaction.
	if arguments.GasPrice != nil && arguments.AuthorizationList != nil {
		return errors.New("both gasPrice and authorizationList specified")
	}
	if arguments.GasPrice != nil {
		return nil
	}
	if arguments.MaxPriorityFeePerGas == nil {
		tip, err := environment.oracle.SuggestTip(head.Context(), oracle.DefaultCertainty)
		if err != nil {
			return err
		}
		arguments.MaxPriorityFeePerGas = (*hexutil.Big)(tip)
	}
	if arguments.MaxFeePerGas == nil {
		feeCap := new(big.Int).Add(arguments.MaxPriorityFeePerGas.ToInt(), SuggestedGasPrice(head))
		arguments.MaxFeePerGas = (*hexutil.Big)(feeCap)
	}
	if arguments.MaxFeePerGas.ToInt().Cmp(arguments.MaxPriorityFeePerGas.ToInt()) < 0 {
		return fmt.Errorf("maxFeePerGas (%v) < maxPriorityFeePerGas (%v)", arguments.MaxFeePerGas, arguments.MaxPriorityFeePerGas)
	}
	return nil
}

// checkData checks that data and input agree and that a contract creation
// has code and is no blob or set-code transaction (V5).
func (arguments *CallArguments) checkData() error {
	if arguments.Data != nil && arguments.Input != nil && !bytes.Equal(*arguments.Data, *arguments.Input) {
		return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	}
	if arguments.Recipient != nil {
		return nil
	}
	if arguments.BlobHashes != nil || arguments.BlobFeeCap != nil {
		return errors.New(`missing "to" in blob transaction`)
	}
	if len(arguments.CallData()) == 0 {
		return errors.New(`contract creation without any data provided`)
	}
	if arguments.AuthorizationList != nil {
		return errors.New(`authorizationList provided for contract creation, but "to" field is missing`)
	}
	return nil
}

// estimationArguments are the fields the gas of a transaction is estimated
// with. As in the reference, they take Data and not Input, so a transaction
// that sent only input is estimated without call data (section 6, F8).
func (arguments *CallArguments) estimationArguments() CallArguments {
	return CallArguments{
		From:                 arguments.From,
		Recipient:            arguments.Recipient,
		GasPrice:             arguments.GasPrice,
		MaxFeePerGas:         arguments.MaxFeePerGas,
		MaxPriorityFeePerGas: arguments.MaxPriorityFeePerGas,
		Value:                arguments.Value,
		Data:                 arguments.Data,
		AccessList:           arguments.AccessList,
		BlobFeeCap:           arguments.BlobFeeCap,
		BlobHashes:           arguments.BlobHashes,
		AuthorizationList:    arguments.AuthorizationList,
	}
}

func (arguments *CallArguments) setChainID(chainID *big.Int) error {
	if arguments.ChainID == nil {
		arguments.ChainID = (*hexutil.Big)(chainID)
		return nil
	}
	if have := (*big.Int)(arguments.ChainID); have.Cmp(chainID) != 0 {
		return fmt.Errorf("chainId does not match node's (have=%v, want=%v)", have, chainID)
	}
	return nil
}
