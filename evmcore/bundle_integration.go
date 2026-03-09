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
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	params "github.com/ethereum/go-ethereum/params"
)

//go:generate mockgen -source=bundle_integration.go -destination=bundle_integration_mock.go -package=evmcore

// bundleChecker is an interface for checking if a bundle transaction is pending
// for execution. A bundle is pending if it has not yet been processed, its
// block range is not yet exceeded, and it is not permanently blocked due to an
// on-chain state mutation (e.g. a mandatory transaction in the bundle using a
// nonce that has already been used).
//
// This interface facilitates testing and decouples the bundle integration
// logic from the transaction pool.
type bundleChecker interface {
	isPending(tx *types.Transaction) bool
}

// BundleIntegrationImplementation uses the chain and state to determine if a
// bundle transaction is still pending for execution or obsolete.
type BundleIntegrationImplementation struct {
	rules  opera.Rules
	chain  StateReader
	state  state.StateDB
	signer types.Signer
}

// newBundleChecker creates a new BundleChecker instance.
func newBundleChecker(
	rules opera.Rules,
	chain StateReader,
	state state.StateDB,
	signer types.Signer,
) bundleChecker {
	return &BundleIntegrationImplementation{
		rules:  rules,
		chain:  chain,
		state:  state,
		signer: signer,
	}
}

func (s *BundleIntegrationImplementation) isPending(tx *types.Transaction) bool {
	// If transaction bundling is disabled, all bundles should be dropped.
	if !s.chain.CurrentRules().Upgrades.TransactionBundles {
		return false
	}

	// Invalid bundles should be dropped.
	_, plan, err := bundle.ValidateTransactionBundle(tx, s.signer)
	if err != nil {
		return false
	}

	// Drop the bundle if it is obsolete.
	currentBlock := s.chain.CurrentBlock().Number.Uint64()
	if plan.Latest < currentBlock {
		return false
	}

	// Drop the bundle if it has been processed.
	if s.chain.HasBundleBeenProcessed(plan.Hash()) {
		return false
	}

	// Remove permanently blocked bundles.
	chain := preCheckChainAdapter{
		chainState: s.chain,
		stateDB:    s.state,
	}
	bundleState := GetBundleState(&chain, tx)
	return bundleState != BundleStatePermanentlyBlocked
}

type preCheckChainAdapter struct {
	chainState StateReader
	stateDB    state.StateDB
}

func (a *preCheckChainAdapter) GetCurrentNetworkRules() opera.Rules {
	return a.chainState.CurrentRules()
}

func (a *preCheckChainAdapter) StateDB() state.StateDB {
	return a.stateDB
}

func (a *preCheckChainAdapter) GetLatestHeader() *EvmHeader {
	return &a.chainState.CurrentBlock().EvmHeader
}

func (a *preCheckChainAdapter) Header(hash common.Hash, number uint64) *EvmHeader {
	return a.chainState.Header(hash, number)
}

func (a *preCheckChainAdapter) GetEvmChainConfig(blockHeight idx.Block) *params.ChainConfig {
	return a.chainState.CurrentConfig()
}
