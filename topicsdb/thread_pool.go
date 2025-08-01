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
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/utils/dbutil/threads"
)

// withThreadPool wraps the index and limits its threads in use
type withThreadPool struct {
	*index
}

// FindInBlocks returns all log records of block range by pattern. 1st pattern element is an address.
func (tt *withThreadPool) FindInBlocks(ctx context.Context, from, to idx.Block, pattern [][]common.Hash) (logs []*types.Log, err error) {
	err = tt.ForEachInBlocks(
		ctx,
		from, to,
		pattern,
		func(l *types.Log) bool {
			logs = append(logs, l)
			return true
		})

	return
}

// ForEachInBlocks matches log records of block range by pattern. 1st pattern element is an address.
func (tt *withThreadPool) ForEachInBlocks(ctx context.Context, from, to idx.Block, pattern [][]common.Hash, onLog func(*types.Log) (gonext bool)) error {
	if 0 < to && to < from {
		return nil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	pattern, err := limitPattern(pattern)
	if err != nil {
		return err
	}

	onMatched := func(rec *logrec) (gonext bool, err error) {
		rec.fetch(tt.table.Logrec)
		if rec.err != nil {
			err = rec.err
			return
		}
		gonext = onLog(rec.result)
		return
	}

	splitby := 0
	parallels := 0
	for i := range pattern {
		parallels += len(pattern[i])
		if len(pattern[splitby]) < len(pattern[i]) {
			splitby = i
		}
	}
	rest := pattern[splitby]
	parallels -= len(rest)

	if parallels >= threads.GlobalPool.Cap() {
		return ErrTooBigTopics
	}

	for len(rest) > 0 {
		got, release := threads.GlobalPool.Lock(parallels + len(rest))
		if got <= parallels {
			release(got)
			select {
			case <-time.After(time.Millisecond):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		onDbIterator := func() {
			release(1)
		}

		pattern[splitby] = rest[:got-parallels]
		rest = rest[got-parallels:]
		err = tt.searchParallel(ctx, pattern, uint64(from), uint64(to), onMatched, onDbIterator)
		if err != nil {
			return err
		}
	}

	return nil
}
