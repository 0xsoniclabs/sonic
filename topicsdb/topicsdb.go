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

package topicsdb

import (
	"context"
	"fmt"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const maxTopicsCount = 5 // count is limited hard to 5 by EVM (see LOG0...LOG4 ops)

var (
	ErrEmptyTopics     = fmt.Errorf("empty topics")
	ErrTooBigTopics    = fmt.Errorf("too many topics")
	ErrLogsNotRecorded = fmt.Errorf("logs are not being recorded")
)

//go:generate mockgen -source=topicsdb.go -package=topicsdb -destination=topicsdb_mock.go

type Index interface {
	FindInBlocks(ctx context.Context, from, to idx.Block, pattern [][]common.Hash) (logs []*types.Log, err error)
	Push(recs ...*types.Log) error
	Close()

	WrapTablesAsBatched() (unwrap func())
}

// NewWithThreadPool creates an Index instance consuming a limited number of threads.
func NewWithThreadPool(db kvdb.Store) Index {
	tt := newIndex(db)
	return &withThreadPool{tt}
}

func NewDummy() Index {
	return &dummyIndex{}
}

func limitPattern(pattern [][]common.Hash) (limited [][]common.Hash, err error) {
	if len(pattern) > (maxTopicsCount + 1) {
		limited = make([][]common.Hash, (maxTopicsCount + 1))
	} else {
		limited = make([][]common.Hash, len(pattern))
	}
	copy(limited, pattern)

	ok := false
	for i, variants := range limited {
		ok = ok || len(variants) > 0
		if len(variants) > 1 {
			limited[i] = uniqOnly(variants)
		}
	}
	if !ok {
		err = ErrEmptyTopics
		return
	}

	return
}

func uniqOnly(hh []common.Hash) []common.Hash {
	index := make(map[common.Hash]struct{}, len(hh))
	for _, h := range hh {
		index[h] = struct{}{}
	}

	uniq := make([]common.Hash, 0, len(index))
	for h := range index {
		uniq = append(uniq, h)
	}
	return uniq
}
