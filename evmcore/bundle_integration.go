// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package evmcore

import (
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type bundlePoolStatus int

const (
	bundlePending bundlePoolStatus = iota
	bundleQueued
	bundleRejected
)

// newBundlesChecker constructs a checker with the available state to determine
// if a bundle transaction is pending.
func newBundlesChecker(
	rules opera.Rules,
	chain StateReader,
	state state.StateDB,
) utils.TransactionCheckFunc {

	adapter := chainAddapter{
		rules:   rules,
		chain:   chain,
		stateDb: state,
	}
	return func(tx *types.Transaction) bool {
		return trialRunBundle(tx, adapter, state)
	}
}

type chainAddapter struct {
	rules   opera.Rules
	chain   StateReader
	stateDb state.StateDB
}

// GetCurrentNetworkRules implements [ChainState].
func (c chainAddapter) GetCurrentNetworkRules() opera.Rules {
	return c.rules
}

func (c chainAddapter) GetEvmChainConfig(idx.Block) *params.ChainConfig {
	return c.chain.CurrentConfig()
}

func (c chainAddapter) GetLatestHeader() *EvmHeader {
	return &c.chain.CurrentBlock().EvmHeader
}

func (c chainAddapter) Header(hash common.Hash, number uint64) *EvmHeader {
	return c.chain.Header(hash, number)
}

func (c chainAddapter) StateDB() state.StateDB {
	return c.stateDb
}
