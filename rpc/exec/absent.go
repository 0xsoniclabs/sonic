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

package exec

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
)

// Absent tells whether an error says that the item does not exist: a block,
// transaction or receipt Core does not have, or a hash of no served block.
func Absent(err error) bool {
	return errors.Is(err, core.ErrNotFound) || errors.Is(err, chain.ErrUnknownHash)
}

// NullIfAbsent answers an absent item as no value and no error, the null
// result of the methods that answer null for what does not exist.
func NullIfAbsent[Value any](value Value, err error) (Value, error) {
	if Absent(err) {
		var zero Value
		return zero, nil
	}
	return value, err
}

var (
	errHeaderNotFound  = errors.New("header not found")
	errUnknownSelector = errors.New("unknown header selector")
)

// StateLookupError words a failure to find the block of a state as the
// reference's state lookup worded it: "header not found", and "unknown header
// selector" for a reference with neither number nor hash (F3).
func StateLookupError(err error) error {
	switch {
	case Absent(err):
		return errHeaderNotFound
	case errors.Is(err, chain.ErrNoSelector):
		return errUnknownSelector
	}
	return err
}

// UnknownHashError is the wording of a block hash of no served block, shared
// by several methods.
func UnknownHashError(hash common.Hash) error {
	return fmt.Errorf("block with hash %s not found", hash.String())
}
