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
	"errors"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/common"
)

// newIndexIterator creates a new index iterator for the given topic, topic
// position and block range. The resulting iterator returns log records with the
// given topic at the given position in the log topics, and with a block number
// in the range [from, to], ordered by the log record key (which is ordered by
// block number, then tx hash, then log index).
func newIndexIterator(
	table kvdb.Iteratee,
	topic common.Hash,
	position int,
	from, to idx.Block,
) *indexIterator {
	prefix := append(topic.Bytes(), posToBytes(uint8(position))...)
	return &indexIterator{
		table:  table,
		prefix: prefix,
		from:   from,
		to:     to,
	}
}

// indexIterator is an adapter that implements the leap.Iterator[logrec] interface
// by wrapping a kvdb.Iterator over the index entries for a specific topic and
// topic position. It also tracks errors from the underlying iterator and
// provides them through the Error() method. This method should be called after
// the iteration is done to check if any error occurred during iteration.
type indexIterator struct {
	// table is the index table to iterate over. Iterators are created lazily
	// from this table when Next or Seek is called.
	table kvdb.Iteratee

	// prefix is the key prefix for the index entries to iterate over, which is
	// derived from the topic and topic position.
	prefix []byte

	// from and to define the block range to iterate over. The iterator returns
	// log records with block numbers in the range [from, to].
	from, to idx.Block

	// iter is the current underlying kvdb.Iterator for the index entries. It is
	// created lazily on the first Next or Seek call, and may be replaced on
	// Seek calls.
	iter kvdb.Iterator

	// current is the current element of the iteration, which is updated on each
	// Next or Seek calls. It is only valid if Next or Seek returned true, and
	// should not be accessed after Next or Seek returned false. It is returned
	// by the Current() method.
	current logrec

	// err tracks errors encountered during iteration.
	err error
}

func (it *indexIterator) Next() bool {
	// If there is an iteration error, stop the iteration.
	if it.err != nil {
		return false
	}

	// The underlying DB iterator is lazy-initialized to perform this only when
	// needed and potentially in parallel to other iterators.
	if it.iter == nil {
		if !it.initIter(uintToBytes(uint64(it.from))) {
			return false
		}
	}
	for it.iter.Next() {
		// Skip invalid keys. This is not an error, as in the key-value store
		// there may be other entries with the same prefix that are not valid
		// index entries, and we just skip those.
		key := it.iter.Key()
		if len(key) != topicKeySize {
			continue
		}

		// Abort with an error if the value field has an invalid length.
		value := it.iter.Value()
		if len(value) != 1 {
			it.err = errors.New("corrupted index entry: invalid value length")
			return false
		}

		// Make sure no error occurred during retrieving the key and value.
		if err := it.iter.Error(); err != nil {
			it.err = err
			return false
		}

		// Stop if we are past the 'to' block.
		id := extractLogrecID(key)
		if id.BlockNumber() > uint64(it.to) {
			break
		}

		// Update the current log record and return it.
		topicCount := bytesToPos(value)
		it.current = *newLogrec(id, topicCount)
		return true
	}

	it.Release()
	return false
}

func (it *indexIterator) Current() logrec {
	return it.current
}

func (it *indexIterator) Seek(target logrec) bool {
	// No seek if there is no underlying table.
	if it.table == nil {
		return false
	}

	// Skip seek if there is an iteration error, as the iterator may be in an
	// invalid state.
	if it.err != nil {
		return false
	}

	// Release the old iterator, if any.
	it.releaseIterator()

	// Re-initialize the iterator with the target log record ID as the new
	// starting point.
	it.initIter(target.ID.Bytes())
	return it.Next()
}

func (it *indexIterator) Error() error {
	return it.err
}

func (it *indexIterator) Release() {
	it.table = nil
	it.releaseIterator()
}

func (it *indexIterator) initIter(start []byte) bool {
	if it.table == nil {
		return false
	}
	it.iter = it.table.NewIterator(it.prefix, start)
	if it.iter == nil {
		it.err = errors.New("failed to create iterator")
		return false
	}
	return true
}

func (it *indexIterator) releaseIterator() {
	if it.iter != nil {
		// collect potential final errors before releasing the iterator
		it.err = errors.Join(it.err, it.iter.Error())
		it.iter.Release()
		it.iter = nil
	}
}
