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

package dispatch

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// BlockNumber is a block number or one of the tags. Its JSON form is that of
// go-ethereum's rpc.BlockNumber.
type BlockNumber int64

const (
	EarliestBlockNumber  = BlockNumber(-5)
	SafeBlockNumber      = BlockNumber(-4)
	FinalizedBlockNumber = BlockNumber(-3)
	LatestBlockNumber    = BlockNumber(-2)
	PendingBlockNumber   = BlockNumber(-1)
)

// UnmarshalJSON parses a block number or one of the tags "safe", "finalized",
// "latest", "earliest" and "pending".
func (bn *BlockNumber) UnmarshalJSON(data []byte) error {
	input := strings.TrimSpace(string(data))
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		input = input[1 : len(input)-1]
	}
	switch input {
	case "earliest":
		*bn = EarliestBlockNumber
		return nil
	case "latest":
		*bn = LatestBlockNumber
		return nil
	case "pending":
		*bn = PendingBlockNumber
		return nil
	case "finalized":
		*bn = FinalizedBlockNumber
		return nil
	case "safe":
		*bn = SafeBlockNumber
		return nil
	}
	number, err := hexutil.DecodeUint64(input)
	if err != nil {
		return err
	}
	if number > math.MaxInt64 {
		return errors.New("block number larger than int64")
	}
	*bn = BlockNumber(number)
	return nil
}

// MarshalText encodes the tags as strings and other numbers as hex.
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
	default:
		if bn < 0 {
			return fmt.Sprintf("<invalid %d>", bn)
		}
		return hexutil.Uint64(bn).String()
	}
}

// Int64 returns the block number as int64.
func (bn BlockNumber) Int64() int64 { return int64(bn) }

// IsTag reports whether the number is one of the tags.
func (bn BlockNumber) IsTag() bool { return bn < 0 }

// IsLatest reports whether the number selects the head: latest, pending,
// safe or finalized.
func (bn BlockNumber) IsLatest() bool {
	return bn == LatestBlockNumber || bn == PendingBlockNumber ||
		bn == FinalizedBlockNumber || bn == SafeBlockNumber
}

// BlockRef names a block by number, tag or hash. Its JSON form is that of
// go-ethereum's rpc.BlockNumberOrHash.
type BlockRef struct {
	Number           *BlockNumber `json:"blockNumber,omitempty"`
	Hash             *common.Hash `json:"blockHash,omitempty"`
	RequireCanonical bool         `json:"requireCanonical,omitempty"`
}

// RefNumber returns a reference to a block number.
func RefNumber(number BlockNumber) BlockRef { return BlockRef{Number: &number} }

// RefHash returns a reference to a block hash.
func RefHash(hash common.Hash) BlockRef { return BlockRef{Hash: &hash} }

// RefLatest is the reference to the head.
func RefLatest() BlockRef { return RefNumber(LatestBlockNumber) }

func (r *BlockRef) UnmarshalJSON(data []byte) error {
	type erased BlockRef
	e := erased{}
	err := json.Unmarshal(data, &e)
	if err == nil {
		if e.Number != nil && e.Hash != nil {
			return errors.New("cannot specify both BlockHash and BlockNumber, choose one or the other")
		}
		r.Number = e.Number
		r.Hash = e.Hash
		r.RequireCanonical = e.RequireCanonical
		return nil
	}
	var input string
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}
	tag := func(bn BlockNumber) error {
		r.Number = &bn
		return nil
	}
	switch input {
	case "earliest":
		return tag(EarliestBlockNumber)
	case "latest":
		return tag(LatestBlockNumber)
	case "pending":
		return tag(PendingBlockNumber)
	case "finalized":
		return tag(FinalizedBlockNumber)
	case "safe":
		return tag(SafeBlockNumber)
	}
	if len(input) == 66 {
		hash := common.Hash{}
		if err := hash.UnmarshalText([]byte(input)); err != nil {
			return err
		}
		r.Hash = &hash
		return nil
	}
	number, err := hexutil.DecodeUint64(input)
	if err != nil {
		return err
	}
	if number > math.MaxInt64 {
		return errors.New("blocknumber too high")
	}
	return tag(BlockNumber(number))
}

func (r *BlockRef) String() string {
	if r.Number != nil {
		return r.Number.String()
	}
	if r.Hash != nil {
		return r.Hash.String()
	}
	return "nil"
}

// OrLatest returns the reference, or the head if it is absent.
func OrLatest(ref *BlockRef) BlockRef {
	if ref == nil {
		return RefLatest()
	}
	return *ref
}
