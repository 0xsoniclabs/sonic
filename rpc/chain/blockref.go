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

// BlockNumber is a block number or tag as a client sends it. The tags are
// negative values; inside the module a block number is a uint64, and a uint64
// cast to BlockNumber above math.MaxInt64 becomes a tag.
type BlockNumber int64

// The tags, with the values go-ethereum gives them.
const (
	EarliestBlockNumber  = BlockNumber(-5)
	SafeBlockNumber      = BlockNumber(-4)
	FinalizedBlockNumber = BlockNumber(-3)
	LatestBlockNumber    = BlockNumber(-2)
	PendingBlockNumber   = BlockNumber(-1)
)

// UnmarshalJSON accepts a tag or a hex quantity, as go-ethereum does.
func (bn *BlockNumber) UnmarshalJSON(data []byte) error {
	input := strings.TrimSpace(string(data))
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		input = input[1 : len(input)-1]
	}
	if tag, ok := parseTag(input); ok {
		*bn = tag
		return nil
	}
	n, err := hexutil.DecodeUint64(input)
	if err != nil {
		return err
	}
	if n > math.MaxInt64 {
		return errors.New("block number larger than int64")
	}
	*bn = BlockNumber(n)
	return nil
}

// MarshalText writes the tag or the hex quantity.
func (bn BlockNumber) MarshalText() ([]byte, error) {
	return []byte(bn.String()), nil
}

func (bn BlockNumber) String() string {
	switch bn {
	case EarliestBlockNumber:
		return "earliest"
	case LatestBlockNumber:
		return "latest"
	case PendingBlockNumber:
		return "pending"
	case FinalizedBlockNumber:
		return "finalized"
	case SafeBlockNumber:
		return "safe"
	}
	if bn < 0 {
		return fmt.Sprintf("<invalid %d>", int64(bn))
	}
	return hexutil.Uint64(bn).String()
}

func parseTag(s string) (BlockNumber, bool) {
	switch s {
	case "earliest":
		return EarliestBlockNumber, true
	case "latest":
		return LatestBlockNumber, true
	case "pending":
		return PendingBlockNumber, true
	case "finalized":
		return FinalizedBlockNumber, true
	case "safe":
		return SafeBlockNumber, true
	}
	return 0, false
}

// BlockRef selects a block by number, tag or hash, as a client sends it.
type BlockRef struct {
	Number           *BlockNumber `json:"blockNumber,omitempty"`
	Hash             *common.Hash `json:"blockHash,omitempty"`
	RequireCanonical bool         `json:"requireCanonical,omitempty"`
}

// NumberRef selects a block by number or tag.
func NumberRef(bn BlockNumber) BlockRef {
	return BlockRef{Number: &bn}
}

// HashRef selects a block by hash.
func HashRef(hash common.Hash) BlockRef {
	return BlockRef{Hash: &hash}
}

// LatestRef is the selector a method assumes when the client sends none.
func LatestRef() BlockRef {
	return NumberRef(LatestBlockNumber)
}

// OrLatest returns the selector, or latest if the client sent none.
func OrLatest(ref *BlockRef) BlockRef {
	if ref == nil {
		return LatestRef()
	}
	return *ref
}

// UnmarshalJSON accepts an object with blockNumber or blockHash, a tag, a
// hash or a hex quantity, as go-ethereum's BlockNumberOrHash does.
func (r *BlockRef) UnmarshalJSON(data []byte) error {
	type plain BlockRef
	var obj plain
	if err := json.Unmarshal(data, &obj); err == nil {
		if obj.Number != nil && obj.Hash != nil {
			return errors.New("cannot specify both BlockHash and BlockNumber, choose one or the other")
		}
		*r = BlockRef(obj)
		return nil
	}
	var input string
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}
	if tag, ok := parseTag(input); ok {
		*r = NumberRef(tag)
		return nil
	}
	if len(input) == 66 {
		var hash common.Hash
		if err := hash.UnmarshalText([]byte(input)); err != nil {
			return err
		}
		*r = HashRef(hash)
		return nil
	}
	n, err := hexutil.DecodeUint64(input)
	if err != nil {
		return err
	}
	if n > math.MaxInt64 {
		return errors.New("blocknumber too high")
	}
	*r = NumberRef(BlockNumber(n))
	return nil
}

func (r BlockRef) String() string {
	switch {
	case r.Number != nil:
		return r.Number.String()
	case r.Hash != nil:
		return r.Hash.String()
	}
	return "nil"
}

// IsPending reports whether the selector is the pending tag.
func (r BlockRef) IsPending() bool {
	return r.Number != nil && *r.Number == PendingBlockNumber
}
