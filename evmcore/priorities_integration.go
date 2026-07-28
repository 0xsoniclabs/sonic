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
	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
)

// PrioritiesIntegrationImplementation uses the priority registry to determine
// if a transaction is prioritized.
type PrioritiesIntegrationImplementation struct {
	rules  opera.Rules
	chain  StateReader
	state  state.StateDB
	signer types.Signer
}

// newPrioritizedChecker creates a new checker instance. This instance is capable
// of querying the priority registry to determine if a transaction is
// prioritized. While the transaction-priorities feature is disabled no
// transaction is prioritized and the registry is not queried.
func newPrioritizedChecker(
	rules opera.Rules,
	chain StateReader,
	state state.StateDB,
	signer types.Signer,
) utils.TransactionCheckFunc {
	impl := &PrioritiesIntegrationImplementation{
		rules:  rules,
		chain:  chain,
		state:  state,
		signer: signer,
	}
	return impl.isPrioritized
}

func (p *PrioritiesIntegrationImplementation) isPrioritized(tx *types.Transaction) bool {
	currentBlock := p.chain.CurrentBlock()

	// Create a EVM processor instance to run the getPriority query.
	blockContext := NewEVMBlockContext(currentBlock.Header(), p.chain, nil)
	vmConfig := opera.GetVmConfig(p.rules)
	vm := vm.NewEVM(blockContext, p.state, p.chain.CurrentConfig(), vmConfig)

	// Query the priority registry contract to determine the transaction's priority.
	priority, err := priorities.GetPriority(p.rules.Upgrades, vm, p.signer, tx)
	if err != nil {
		log.Warn("Error checking if tx is prioritized", "tx", tx.Hash(), "err", err)
		return false
	}
	return priority.IsPrioritized()
}
