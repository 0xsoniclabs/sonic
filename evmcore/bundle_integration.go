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

package evmcore

import (
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/core/types"
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

	// TODO: check whether bundle is permanently blocked by trial-running it

	return true
}
