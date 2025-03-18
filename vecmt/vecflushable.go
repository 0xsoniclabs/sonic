package vecmt

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/lachesis-base/kvdb"
)

var (
	errClosed         = errors.New("vecflushable - database closed")
	errNotImplemented = errors.New("vecflushable - not implemented")
)

// mapConst is an approximation of the number of extra bytes used by native go
// maps when adding an item to a map.
const mapConst = 100

func mapMemEst(keyS, valueS int) int {
	return mapConst + keyS + valueS
}

// VecFlushable is a fast, append only, Flushable intended for the vecengine.
// It does not implement all of the Flushable interface, just what is needed by
// the vecengine.
type VecFlushable struct {
	modified   map[string][]byte
	underlying backedMap
	memSize    int
}

func WrapByVecFlushable(parent kvdb.Store, sizeLimit int) *VecFlushable {
	if parent == nil {
		panic("nil parent")
	}
	return &VecFlushable{
		modified:   make(map[string][]byte),
		underlying: *newBackedMap(parent, sizeLimit, kvdb.IdealBatchSize),
	}
}

func (w *VecFlushable) clearModified() {
	w.modified = make(map[string][]byte)
	w.memSize = 0
}

func (w *VecFlushable) Has(key []byte) (bool, error) {
	if w.modified == nil {
		return false, errClosed
	}
	_, ok := w.modified[string(key)]
	if ok {
		return true, nil
	}
	return w.underlying.has(key)
}

func (w *VecFlushable) Get(key []byte) ([]byte, error) {
	if w.modified == nil {
		return nil, errClosed
	}
	if val, ok := w.modified[string(key)]; ok {
		return common.CopyBytes(val), nil
	}
	return w.underlying.get(key)
}

func (w *VecFlushable) Put(key []byte, value []byte) error {
	if value == nil || key == nil {
		return errors.New("vecflushable: key or value is nil")
	}
	w.modified[string(key)] = common.CopyBytes(value)
	w.memSize += mapMemEst(len(key), len(value))
	return nil
}

func (w *VecFlushable) NotFlushedPairs() int {
	return len(w.modified)
}

func (w *VecFlushable) NotFlushedSizeEst() int {
	return w.memSize
}

func (w *VecFlushable) Flush() error {
	if w.modified == nil {
		return errClosed
	}

	for key, val := range w.modified {
		w.underlying.add(key, val)
	}

	err := w.underlying.mayUnload()
	if err != nil {
		return err
	}

	w.clearModified()
	return nil
}

func (w *VecFlushable) DropNotFlushed() {
	w.clearModified()
}

func (w *VecFlushable) Close() error {
	if w.modified == nil {
		return errClosed
	}
	w.DropNotFlushed()
	w.modified = nil
	return w.underlying.close()
}

func (w *VecFlushable) Drop() {
	panic(errNotImplemented)
}

func (db *VecFlushable) AncientDatadir() (string, error) {
	panic(errNotImplemented)
}

/* Some methods are not implemented and panic when called */

func (w *VecFlushable) Delete(key []byte) error {
	panic(errNotImplemented)
}

func (w *VecFlushable) GetSnapshot() (kvdb.Snapshot, error) {
	panic(errNotImplemented)
}

func (w *VecFlushable) NewIterator(prefix []byte, start []byte) kvdb.Iterator {
	panic(errNotImplemented)
}

func (w *VecFlushable) Stat() (string, error) {
	panic(errNotImplemented)
}

func (w *VecFlushable) Compact(start []byte, limit []byte) error {
	panic(errNotImplemented)
}

func (w *VecFlushable) NewBatch() kvdb.Batch {
	panic(errNotImplemented)
}
