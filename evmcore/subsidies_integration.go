// Copyright 2025 Sonic Operations Ltd
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
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
)

//go:generate mockgen -source=subsidies_integration.go -destination=subsidies_integration_mock.go -package=evmcore

// SubsidiesChecker is an interface for checking if a transaction is sponsored
// by the subsidies contract.
// it does not include [subsidies.IsCovered] directly to avoid creating dependencies
// on state for an operation which is pure.
//
// This interface facilitates testing and decouples the subsidies integration
// logic from the transaction pool.
type SubsidiesChecker interface {
	IsSponsored(tx *types.Transaction) bool
}

// SubsidiesIntegrationImplementation uses the subsidies contract to determine
// if a transaction is sponsored.
type SubsidiesIntegrationImplementation struct {
	rules  opera.Rules
	chain  StateReader
	state  state.StateDB
	signer types.Signer
}

// NewSubsidiesChecker creates a new SubsidiesChecker instance.
// This instance is capable of executing the subsidies contract to determine
// if a transaction is sponsored.
func NewSubsidiesChecker(
	rules opera.Rules,
	chain StateReader,
	state state.StateDB,
	signer types.Signer,
) SubsidiesChecker {
	return &SubsidiesIntegrationImplementation{
		rules:  rules,
		chain:  chain,
		state:  state,
		signer: signer,
	}
}

func (s *SubsidiesIntegrationImplementation) IsSponsored(tx *types.Transaction) bool {
	currentBlock := s.chain.CurrentBlock()
	baseFee := s.chain.GetCurrentBaseFee()

	// Create a EVM processor instance to run the IsCovered query.
	blockContext := NewEVMBlockContext(currentBlock.Header(), s.chain, nil)
	vmConfig := opera.GetVmConfig(s.rules)
	vm := vm.NewEVM(blockContext, s.state, s.chain.Config(), vmConfig)

	// Query the subsidies registry contract to determine if the transaction is sponsored.
	isSponsored, _, err := subsidies.IsCovered(s.rules.Upgrades, vm, s.signer, tx, baseFee)
	if err != nil {
		log.Warn("Error checking if tx is sponsored", "tx", tx.Hash(), "err", err)
		return false
	}
	return isSponsored
}
