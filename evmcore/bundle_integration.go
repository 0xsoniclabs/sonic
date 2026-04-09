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
	"github.com/ethereum/go-ethereum/core/types"
)

// newBundlesChecker constructs a checker with the available state to determine
// if a bundle transaction is pending.
func newBundlesChecker(
	// TODO: add required pool facilities to serve data to the bundle checker
	_ opera.Rules,
	_ StateReader,
	_ state.StateDB,
	_ types.Signer,
) utils.TransactionCheckFunc {
	// TODO: implement a real checker that can determine if a bundle transaction is pending
	return func(*types.Transaction) bool {
		return false
	}
}
