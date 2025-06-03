package kvdb2ethdb

import (
	"bytes"

	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

type Adapter struct {
	kvdb.Store
}

var _ ethdb.KeyValueStore = (*Adapter)(nil)

func Wrap(v kvdb.Store) *Adapter {
	return &Adapter{v}
}

// batch is a write-only memory batch that commits changes to its host
// database when Write is called. A batch cannot be used concurrently.
type batch struct {
	kvdb.Batch
}

// Replay replays the batch contents.
func (b *batch) Replay(w ethdb.KeyValueWriter) error {
	return b.Batch.Replay(w)
}

// NewBatch creates a write-only key-value store that buffers changes to its host
// database until a final write is called.
func (db *Adapter) NewBatch() ethdb.Batch {
	return &batch{db.Store.NewBatch()}
}

func (db *Adapter) NewBatchWithSize(int) ethdb.Batch {
	return db.NewBatch()
}

// NewIterator creates a binary-alphabetical iterator over a subset
// of database content with a particular key prefix, starting at a particular
// initial key (or after, if it does not exist).
func (db *Adapter) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return db.Store.NewIterator(prefix, start)
}

// DeleteRange deletes all of the keys (and values) in the range [start,end).
func (db *Adapter) DeleteRange(start, end []byte) error {
	iter := db.Store.NewIterator(nil, start)
	defer iter.Release()
	for iter.Next() {
		key := iter.Key()
		if bytes.Compare(key, end) >= 0 {
			break
		}
		if err := db.Delete(key); err != nil {
			return err
		}
	}
	return nil
}
