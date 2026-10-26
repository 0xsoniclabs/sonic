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

package chain

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// BlockNumber is a block number as a client sends it: a number, or one of
// the tags, held as negative values. It is for client input only; inside the
// module a block number is a uint64, because a uint64 above MaxInt64 cast to
// a BlockNumber becomes a tag.
type BlockNumber int64

// The tags, with the values go-ethereum gives them.
const (
	EarliestBlockNumber  = BlockNumber(-5)
	SafeBlockNumber      = BlockNumber(-4)
	FinalizedBlockNumber = BlockNumber(-3)
	LatestBlockNumber    = BlockNumber(-2)
	PendingBlockNumber   = BlockNumber(-1)
)

var tags = map[string]BlockNumber{
	"earliest":  EarliestBlockNumber,
	"latest":    LatestBlockNumber,
	"pending":   PendingBlockNumber,
	"finalized": FinalizedBlockNumber,
	"safe":      SafeBlockNumber,
}

// UnmarshalJSON accepts a tag or a hex quantity, with the errors of
// go-ethereum's rpc.BlockNumber.
func (number *BlockNumber) UnmarshalJSON(data []byte) error {
	input := strings.TrimSpace(string(data))
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		input = input[1 : len(input)-1]
	}
	if tag, found := tags[input]; found {
		*number = tag
		return nil
	}
	value, err := hexutil.DecodeUint64(input)
	if err != nil {
		return err
	}
	if value > math.MaxInt64 {
		return errors.New("block number larger than int64")
	}
	*number = BlockNumber(value)
	return nil
}

// IsTag tells whether the number is one of the tags.
func (number BlockNumber) IsTag() bool {
	return number < 0
}

// MarshalText encodes a tag by its name and a number as a hex quantity.
func (number BlockNumber) MarshalText() ([]byte, error) {
	return []byte(number.String()), nil
}

func (number BlockNumber) String() string {
	for name, tag := range tags {
		if number == tag {
			return name
		}
	}
	if number < 0 {
		return fmt.Sprintf("<invalid %d>", number)
	}
	return hexutil.Uint64(number).String()
}

// BlockReference names a block by number, tag or hash, as a client sends it.
// It is go-ethereum's rpc.BlockNumberOrHash with the same JSON forms: a tag, a
// quantity, a 32-byte hash, or an object with blockNumber or blockHash.
type BlockReference struct {
	Number           *BlockNumber `json:"blockNumber,omitempty"`
	Hash             *common.Hash `json:"blockHash,omitempty"`
	RequireCanonical bool         `json:"requireCanonical,omitempty"`
}

// ReferenceToNumber returns a reference to a number or tag.
func ReferenceToNumber(number BlockNumber) BlockReference {
	return BlockReference{Number: &number}
}

// ReferenceToHash returns a reference to a hash.
func ReferenceToHash(hash common.Hash) BlockReference {
	return BlockReference{Hash: &hash}
}

// Latest is the reference a method uses when the client sends none.
func Latest() BlockReference {
	return ReferenceToNumber(LatestBlockNumber)
}

// UnmarshalJSON decodes the forms of go-ethereum's rpc.BlockNumberOrHash, with
// its errors.
func (reference *BlockReference) UnmarshalJSON(data []byte) error {
	type plain BlockReference
	var object plain
	if err := json.Unmarshal(data, &object); err == nil {
		if object.Number != nil && object.Hash != nil {
			return errors.New("cannot specify both BlockHash and BlockNumber, choose one or the other")
		}
		*reference = BlockReference(object)
		return nil
	}
	var input string
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}
	if tag, found := tags[input]; found {
		*reference = ReferenceToNumber(tag)
		return nil
	}
	if len(input) == 66 {
		var hash common.Hash
		if err := hash.UnmarshalText([]byte(input)); err != nil {
			return err
		}
		*reference = ReferenceToHash(hash)
		return nil
	}
	value, err := hexutil.DecodeUint64(input)
	if err != nil {
		return err
	}
	if value > math.MaxInt64 {
		return errors.New("blocknumber too high")
	}
	*reference = ReferenceToNumber(BlockNumber(value))
	return nil
}

// String describes the reference as go-ethereum does, for log lines.
func (reference BlockReference) String() string {
	switch {
	case reference.Number != nil:
		return reference.Number.String()
	case reference.Hash != nil:
		return reference.Hash.String()
	}
	return "nil"
}
