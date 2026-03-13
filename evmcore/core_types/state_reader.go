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

package coretypes

import (
	"math/big"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
)

//go:generate mockgen -source=state_reader.go -destination=state_reader_mock.go -package=coretypes

// StateReader provides the state of blockchain and current gas limit to do
// some pre checks in tx pool and event subscribers.
type StateReader interface {
	CurrentBlock() *EvmBlock
	Block(hash common.Hash, number uint64) *EvmBlock
	CurrentStateDB() (state.StateDB, error)
	CurrentBaseFee() *big.Int
	CurrentMaxGasLimit() uint64
	SubscribeNewBlock(ch chan<- ChainHeadNotify) event.Subscription
	CurrentConfig() *params.ChainConfig
	CurrentRules() opera.Rules
	Header(hash common.Hash, number uint64) *EvmHeader
	HasBundleBeenProcessed(execPlanHash common.Hash) bool
}
