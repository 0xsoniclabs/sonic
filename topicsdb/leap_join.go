package topicsdb

import (
	"context"

	"github.com/0xsoniclabs/sonic/utils/leap"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type withLeapJoin struct {
	*index
}

// NewWithLeapJoin creates an Index instance using the leap join algorithm for
// log filtering.
func NewWithLeapJoin(db kvdb.Store) Index {
	return &withLeapJoin{newIndex(db)}
}

// FindInBlocks returns all log records of block range by pattern.
// 1st pattern element is an address. Results are listed in the order of the
// log topic index, which is BlockNumber > TxHash > LogIndex.
func (i *withLeapJoin) FindInBlocks(
	ctx context.Context,
	from, to idx.Block,
	pattern [][]common.Hash,
) ([]*types.Log, error) {
	// Validate range
	if 0 < to && to < from {
		return nil, nil
	}

	// Check the context, stop if already cancelled
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Clean up the pattern. This restricts the pattern to cover at most 5 topics,
	// and removes duplicates within each topic set. If in the end the pattern is
	// empty, an error is returned.
	pattern, err := limitPattern(pattern)
	if err != nil {
		return nil, err
	}
	//fmt.Printf("\nFinding logs in blocks %d..%d by pattern %v\n", from, to, pattern)

	// Build the leap join plan.
	iterators := make([]leap.Iterator[logrec], 0, len(pattern))
	for position, topics := range pattern {
		//fmt.Printf("Building iterator for position %d topics %v\n", position, topics)
		it := newTopicIterator(topics, position, from, to, i.table.Topic)
		if it != nil {
			iterators = append(iterators, it)
			defer it.Release()
		} else {
			//fmt.Printf("No iterator for position %d topics %v\n", position, topics)
		}
	}

	// Execute the leap join.
	var logs []*types.Log
	for logrec := range leap.JoinFunc(logRecLess, iterators...) {
		// Stop if the context is cancelled
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		//fmt.Printf("Resolving log record %v\n", logrec.ID)
		// Resolve the log record.
		logrec.fetch(i.table.Logrec)
		if logrec.err != nil {
			return nil, logrec.err
		}
		logs = append(logs, logrec.result)
	}

	//fmt.Printf("Found total %d logs\n", len(logs))

	return logs, nil
}

func newTopicIterator(
	topics []common.Hash,
	position int,
	from, to idx.Block,
	table kvdb.Store,
) leap.Iterator[logrec] {
	if len(topics) == 0 {
		return nil
	}

	iters := make([]leap.Iterator[logrec], 0, len(topics))
	for _, topic := range topics {
		iters = append(iters, newIndexIterator(topic, position, from, to, table))
	}
	if len(iters) == 1 {
		return iters[0]
	}
	return leap.UnionFunc(logRecLess, iters...)
}

var _ leap.Iterator[logrec] = (*topicIndexIterator)(nil)

type topicIndexIterator struct {
	prefix   []byte
	table    kvdb.Store
	from, to idx.Block
	iter     kvdb.Iterator
}

func newIndexIterator(
	topic common.Hash,
	position int,
	from, to idx.Block,
	table kvdb.Store,
) leap.Iterator[logrec] {
	prefix := append(topic.Bytes(), posToBytes(uint8(position))...)
	/*
		return leap.Prefetch(&topicIndexIterator{
			prefix: prefix,
			table:  table,
			from:   from,
			to:     to,
		})
	*/
	return &topicIndexIterator{
		prefix: prefix,
		table:  table,
		from:   from,
		to:     to,
	}
}

func (it *topicIndexIterator) Next() bool {
	// The actual DB iterator is lazy-initialized to perform this only when
	// needed and in potentially in parallel to other iterators.
	if it.iter == nil {
		it.iter = it.table.NewIterator(it.prefix, uintToBytes(uint64(it.from)))
	}
	res := it.next()
	//fmt.Printf("  Called Next() on iterator for position %d - result: %t\n", bytesToPos(it.prefix[len(it.prefix)-1:]), res)
	return res
}

func (it *topicIndexIterator) next() bool {
	if it.iter == nil {
		return false
	}
	for it.iter.Next() {
		// Skip invalid keys.
		if len(it.iter.Key()) == topicKeySize {
			// Stop if we are past the 'to' block.
			id := extractLogrecID(it.iter.Key())
			return id.BlockNumber() <= uint64(it.to)
		}
		//fmt.Printf("    Skipping entry due to invalid key size: %d\n", len(it.iter.Key()))
	}
	//fmt.Printf("  Iterator exhausted for position %d\n", bytesToPos(it.prefix[len(it.prefix)-1:]))
	return false
}

func (it *topicIndexIterator) Cur() logrec {
	if it.iter == nil {
		return logrec{}
	}
	id := extractLogrecID(it.iter.Key())
	topicCount := bytesToPos(it.iter.Value())
	return *newLogrec(id, topicCount)
}

func (it *topicIndexIterator) Seek(target logrec) bool {
	if it.iter != nil {
		it.iter.Release()
	}
	it.iter = it.table.NewIterator(it.prefix, target.ID.Bytes())
	return it.Next()
}

func (it *topicIndexIterator) Release() {
	if it.iter != nil {
		it.iter.Release()
		it.iter = nil
	}
}
