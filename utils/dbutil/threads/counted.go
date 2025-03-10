package threads

import (
	"github.com/0xsoniclabs/kvdb"

	"github.com/0xsoniclabs/sonic/logger"
)

type (
	countedFullDbProducer struct {
		kvdb.FullDBProducer
	}

	countedStore struct {
		kvdb.Store
	}

	countedIterator struct {
		kvdb.Iterator
		release func(count int)
	}
)

// CountedFullDBProducer obtains one thread from the GlobalPool for each opened iterator.
func CountedFullDBProducer(dbs kvdb.FullDBProducer) kvdb.FullDBProducer {
	return &countedFullDbProducer{dbs}
}

func (p *countedFullDbProducer) OpenDB(name string) (kvdb.Store, error) {
	s, err := p.FullDBProducer.OpenDB(name)
	return &countedStore{s}, err
}

var notifier = logger.New("threads-pool")

func (s *countedStore) NewIterator(prefix []byte, start []byte) kvdb.Iterator {
	got, release := GlobalPool.Lock(1)
	if got < 1 {
		notifier.Log.Warn("Too many DB iterators")
	}

	return &countedIterator{
		Iterator: s.Store.NewIterator(prefix, start),
		release:  release,
	}
}

func (it *countedIterator) Release() {
	it.Iterator.Release()
	it.release(1)
}
