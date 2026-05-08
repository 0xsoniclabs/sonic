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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// bundlePoolStatus represents the status of a bundle in the transaction pool,
// which can be pending, queued, or rejected. This type is used to
// manage transitions from queued to pending, pending to queued, queued to
// rejected, and pending to rejected, based on the evaluation of the bundle's
// executability.
type bundlePoolStatus int

const (
	// bundlePending, the bundle is executable with the current state
	bundlePending bundlePoolStatus = iota
	// bundleQueued, the bundle is temporarily non-executable
	bundleQueued
	// bundleRejected, the bundle is permanently non-executable
	bundleRejected
)

// newBundlesChecker creates a function that checks the status of a bundle in
// the transaction pool. This function translates BundleState into bundlePoolStatus,
// for use in promotion/demotion/eviction of bundles in the transaction pool.
func newBundlesChecker(
	cache BundleEvaluator,
	chain StateReader,
	stateDB state.StateDB,
) func(*types.Transaction) bundlePoolStatus {

	adapter := chainAdapter{
		chain:   chain,
		stateDb: stateDB,
	}
	return func(tx *types.Transaction) bundlePoolStatus {
		bundleState := cache.GetBundleState(adapter, stateDB, tx)
		if bundleState.Executable {
			return bundlePending
		} else if bundleState.TemporarilyBlocked {
			return bundleQueued
		} else {
			return bundleRejected
		}
	}
}

type chainAdapter struct {
	chain   StateReader
	stateDb state.StateDB
}

// GetCurrentNetworkRules implements [ChainState].
func (c chainAdapter) GetCurrentNetworkRules() opera.Rules {
	return c.chain.CurrentRules()
}

func (c chainAdapter) GetCurrentChainConfig() *params.ChainConfig {
	return c.chain.CurrentConfig()
}

func (c chainAdapter) GetLatestHeader() *EvmHeader {
	return &c.chain.CurrentBlock().EvmHeader
}

func (c chainAdapter) Header(hash common.Hash, number uint64) *EvmHeader {
	return c.chain.Header(hash, number)
}

func (c chainAdapter) StateDB() state.StateDB {
	return c.stateDb
}
