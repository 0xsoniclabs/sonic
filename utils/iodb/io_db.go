package iodb

import (
	"io"

	"github.com/0xsoniclabs/consensus/common/bigendian"
	"github.com/0xsoniclabs/consensus/kvdb"

	"github.com/0xsoniclabs/sonic/utils/ioread"
)

func NewIterator(reader io.Reader) kvdb.Iterator {
	return &Iterator{
		reader: reader,
	}
}

type Iterator struct {
	reader     io.Reader
	key, value []byte
	err        error
}

func (it *Iterator) Next() bool {
	if it.err != nil {
		return false
	}
	var lenB [4]byte
	it.err = ioread.ReadAll(it.reader, lenB[:])
	if it.err == io.EOF {
		it.err = nil
		return false
	}
	if it.err != nil {
		return false
	}

	lenKey := bigendian.BytesToUint32(lenB[:])
	key := make([]byte, lenKey)
	it.err = ioread.ReadAll(it.reader, key)
	if it.err != nil {
		return false
	}

	it.err = ioread.ReadAll(it.reader, lenB[:])
	if it.err != nil {
		return false
	}

	lenValue := bigendian.BytesToUint32(lenB[:])
	value := make([]byte, lenValue)
	it.err = ioread.ReadAll(it.reader, value)
	if it.err != nil {
		return false
	}

	it.key = key
	it.value = value
	return true
}

func (it *Iterator) Error() error {
	return it.err
}

func (it *Iterator) Key() []byte {
	return it.key
}

func (it *Iterator) Value() []byte {
	return it.value
}

func (it *Iterator) Release() {
	it.reader = nil
}
