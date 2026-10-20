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

package evmstore

import (
	"errors"
	"fmt"

	cc "github.com/0xsoniclabs/carmen/go/common"
	"github.com/0xsoniclabs/carmen/go/common/amount"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
)

// burnAddress is the conventional burn address 0x000000000000000000000000000000000000dEaD.
var burnAddress = cc.Address(common.HexToAddress("0x000000000000000000000000000000000000dEaD"))

// TotalSupplySnapshot captures the total supply and the balances of the
// addresses considered as burned, all taken from the state of a single block.
type TotalSupplySnapshot struct {
	Block uint64
	// Total is the sum of the balances of all accounts.
	Total amount.Amount
	// ZeroBalance is the balance of the zero address.
	ZeroBalance amount.Amount
	// DeadBalance is the balance of the 0x...dEaD address.
	DeadBalance amount.Amount
}

// ErrTotalSupplyUnavailable is returned if no total supply snapshot is available.
var ErrTotalSupplyUnavailable = errors.New("total supply is not available")

// UpdateTotalSupply records the total supply snapshot of the current state and
// attributes it to the given block. It must be called right after the block was
// committed to the state and before the next block is processed, such that the
// state is guaranteed to be the one of the given block. On failure the stored
// snapshot is cleared, so no value of an older block is reported.
func (s *Store) UpdateTotalSupply(block idx.Block) error {
	snapshot, err := s.readTotalSupply(uint64(block))
	if err != nil {
		s.totalSupply.Store(nil)
		return err
	}
	s.totalSupply.Store(snapshot)
	return nil
}

// GetTotalSupply returns the snapshot of the last processed block.
func (s *Store) GetTotalSupply() (*TotalSupplySnapshot, error) {
	snapshot := s.totalSupply.Load()
	if snapshot == nil {
		return nil, ErrTotalSupplyUnavailable
	}
	return snapshot, nil
}

func (s *Store) readTotalSupply(block uint64) (*TotalSupplySnapshot, error) {
	state := s.carmenState
	if state == nil || s.liveStateDb == nil {
		return nil, fmt.Errorf("unable to read total supply - EvmStore is not open")
	}
	// If the live StateDB has collected an error, EndBlock skips applying the
	// block's changes, so the state may still be the one of a previous block.
	if err := s.liveStateDb.Check(); err != nil {
		return nil, fmt.Errorf("live state is not consistent with block %d: %w", block, err)
	}
	total, err := state.TotalSupply()
	if err != nil {
		return nil, fmt.Errorf("failed to get total supply: %w", err)
	}
	zero, err := state.GetBalance(cc.Address{})
	if err != nil {
		return nil, fmt.Errorf("failed to get balance of the zero address: %w", err)
	}
	dead, err := state.GetBalance(burnAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance of the dead address: %w", err)
	}
	return &TotalSupplySnapshot{
		Block:       block,
		Total:       total,
		ZeroBalance: zero,
		DeadBalance: dead,
	}, nil
}
