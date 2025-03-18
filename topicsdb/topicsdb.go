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
