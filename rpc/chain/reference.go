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
	"math"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// BlockNumber is a block number as a client gives it: a number, or a tag
// held as a negative value. The tags have go-ethereum's values, which the
// node's log query code reads, so a number above math.MaxInt64 cast to a
// BlockNumber becomes a tag; inside the module block numbers are uint64.
type BlockNumber int64

// The block tags.
const (
	EarliestBlockNumber  = BlockNumber(-5)
	SafeBlockNumber      = BlockNumber(-4)
	FinalizedBlockNumber = BlockNumber(-3)
	LatestBlockNumber    = BlockNumber(-2)
	PendingBlockNumber   = BlockNumber(-1)
)

// tags are the block tags by their names.
var tags = map[string]BlockNumber{
	"earliest":  EarliestBlockNumber,
	"latest":    LatestBlockNumber,
	"pending":   PendingBlockNumber,
	"finalized": FinalizedBlockNumber,
	"safe":      SafeBlockNumber,
}

// UnmarshalJSON decodes a tag name or a hex number, quoted or not, with the
// errors of go-ethereum's decoding.
func (n *BlockNumber) UnmarshalJSON(data []byte) error {
	input := strings.TrimSpace(string(data))
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		input = input[1 : len(input)-1]
	}
	if tag, isTag := tags[input]; isTag {
		*n = tag
		return nil
	}
	number, err := hexutil.DecodeUint64(input)
	if err != nil {
		return err
	}
	if number > math.MaxInt64 {
		return errors.New("block number larger than int64")
	}
	*n = BlockNumber(number)
	return nil
}

// BlockReference names a block by number, by tag or by hash, as a client
// gives it. Neither set is a missing selector.
type BlockReference struct {
	Number *BlockNumber
	Hash   *common.Hash
}

// ReferenceTo names a block by number or tag.
func ReferenceTo(number BlockNumber) BlockReference {
	return BlockReference{Number: &number}
}

// UnmarshalJSON decodes an object with a number or a hash, a tag name, a hex
// number or a hash, with the errors of go-ethereum's decoding. The object's
// requireCanonical is decoded for its errors only: every block is canonical.
func (r *BlockReference) UnmarshalJSON(data []byte) error {
	var object struct {
		BlockNumber      *BlockNumber `json:"blockNumber,omitempty"`
		BlockHash        *common.Hash `json:"blockHash,omitempty"`
		RequireCanonical bool         `json:"requireCanonical,omitempty"`
	}
	if err := json.Unmarshal(data, &object); err == nil {
		if object.BlockNumber != nil && object.BlockHash != nil {
			return errors.New("cannot specify both BlockHash and BlockNumber, choose one or the other")
		}
		*r = BlockReference{Number: object.BlockNumber, Hash: object.BlockHash}
		return nil
	}
	var input string
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}
	if tag, isTag := tags[input]; isTag {
		*r = ReferenceTo(tag)
		return nil
	}
	if len(input) == 66 {
		var hash common.Hash
		if err := hash.UnmarshalText([]byte(input)); err != nil {
			return err
		}
		*r = BlockReference{Hash: &hash}
		return nil
	}
	number, err := hexutil.DecodeUint64(input)
	if err != nil {
		return err
	}
	if number > math.MaxInt64 {
		return errors.New("blocknumber too high")
	}
	*r = ReferenceTo(BlockNumber(number))
	return nil
}
