// Copyright 2014 The go-ethereum Authors
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

package gasprice

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// To ensure that the effective gas tip calculation is consistent across different versions
// of the codebase, we have this helper function that can be used in all places where the
// effective gas tip needs to be calculated.
// Based on geth 1.16.9 which is used in sonic 2.1.6
// https://github.com/0xsoniclabs/go-ethereum/blob/879ee1854e078ae6a6958f7cd40094377fd90b98/core/types/transaction.go#L358

// EffectiveGasTip returns the effective miner gasTipCap for the given base fee.
// Note: if the effective gasTipCap would be negative, this method
// returns ErrGasFeeCapTooLow, and value is undefined.
func EffectiveGasTip(tx *types.Transaction, baseFee *big.Int) (*big.Int, error) {
	dst := new(uint256.Int)
	base := new(uint256.Int)
	if baseFee != nil {
		if base.SetFromBig(baseFee) {
			return nil, types.ErrUint256Overflow
		}
	}
	err := calcEffectiveGasTip(tx, dst, base)
	return dst.ToBig(), err
}

// calcEffectiveGasTip calculates the effective gas tip of the transaction and
// saves the result to dst.
func calcEffectiveGasTip(tx *types.Transaction, dst *uint256.Int, baseFee *uint256.Int) error {
	if baseFee == nil {
		if dst.SetFromBig(tx.GasTipCap()) {
			return types.ErrUint256Overflow
		}
		return nil
	}

	var err error
	if dst.SetFromBig(tx.GasFeeCap()) {
		return types.ErrUint256Overflow
	}
	if dst.Cmp(baseFee) < 0 {
		err = types.ErrGasFeeCapTooLow
	}

	dst.Sub(dst, baseFee)
	gasTipCap := new(uint256.Int)
	if gasTipCap.SetFromBig(tx.GasTipCap()) {
		return types.ErrUint256Overflow
	}
	if gasTipCap.Cmp(dst) < 0 {
		dst.Set(gasTipCap)
	}
	return err
}
