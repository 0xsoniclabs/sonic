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
	"log/slog"
	"math"
	"math/big"
	"time"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
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
	isSponsored(tx *types.Transaction) bool
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

func (s *SubsidiesIntegrationImplementation) isSponsored(tx *types.Transaction) bool {
	currentBlock := s.chain.CurrentBlock()
	baseFee := s.chain.GetCurrentBaseFee()
	blockContext := vm.BlockContext{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash: func(number uint64) common.Hash {
			block := s.chain.GetBlock(common.Hash{}, number)
			if block != nil {
				return block.Hash
			}
			return common.Hash{}
		},
		BlockNumber: new(big.Int).Add(currentBlock.Number, common.Big1),
		Time:        uint64(time.Now().Unix()),
		Difficulty:  big.NewInt(0),
		BaseFee:     baseFee,
		GasLimit:    math.MaxInt64,
		Random:      &common.Hash{}, // < signals Revision >= Merge
		BlobBaseFee: big.NewInt(1),  // TODO issue #147
	}

	// Create a EVM processor instance to run the IsCovered query.
	vmConfig := opera.GetVmConfig(s.rules)
	vm := vm.NewEVM(blockContext, s.state, s.chain.Config(), vmConfig)
	isSponsored, _, err := subsidies.IsCovered(s.rules.Upgrades, vm, s.signer, tx, baseFee)
	if err != nil {
		slog.Warn("Error checking if tx is sponsored", "tx", tx.Hash(), "err", err)
		return false
	}
	return isSponsored
}

// static assert interface implementation
var _ SubsidiesChecker = &SubsidiesIntegrationImplementation{}
