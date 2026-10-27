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
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/chain"
)

// DefaultTraceTimeout bounds a traced run that sets no timeout of its own,
// as in the reference.
const DefaultTraceTimeout = 5 * time.Second

// TraceCallConfig is the configuration of debug_traceCall, trace_call and
// trace_callMany: a tracer's, the overrides of the state and the block, and
// the position in the block the call runs at.
type TraceCallConfig struct {
	tracers.TraceConfig
	StateOverrides   *StateOverride
	BlockOverrides   *BlockOverrides
	TransactionIndex *hexutil.Uint `json:"txIndex"`
}

// Position is the position the call runs at, zero when none is given.
func (config *TraceCallConfig) Position() int {
	if config == nil || config.TransactionIndex == nil {
		return 0
	}
	return int(*config.TransactionIndex)
}

// CallBlock finds the block a traced call runs on, worded as the
// reference's lookup worded a block it did not find.
func (environment *Environment) CallBlock(head Head, reference chain.BlockReference) (*evmcore.EvmBlock, error) {
	if reference.Number == nil && reference.Hash == nil {
		return nil, errors.New("invalid arguments; neither block number nor hash specified")
	}
	block, err := NullIfAbsent(environment.FindBlock(head, reference))
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, errors.New("header not found")
	}
	return block, nil
}

// ToTracedCall is the message of a traced call and the legacy transaction
// that stands for it, as the reference built them.
func (arguments *CallArguments) ToTracedCall(gasCap uint64, baseFee *big.Int) (*types.Transaction, *core.Message, error) {
	message, err := arguments.ToMessage(gasCap, baseFee)
	if err != nil {
		return nil, nil, err
	}
	transaction := types.NewTx(&types.LegacyTx{
		To:       message.To,
		Nonce:    message.Nonce,
		Gas:      message.GasLimit,
		GasPrice: message.GasPrice,
		Value:    message.Value,
		Data:     message.Data,
	})
	return transaction, message, nil
}
