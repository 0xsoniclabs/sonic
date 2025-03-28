package fileshash

import (
	"io"

	"github.com/0xsoniclabs/consensus/consensus"
)

func Wrap(backend func(string) (io.Reader, error), maxMemoryUsage uint64, roots map[string]consensus.Hash) func(string) (io.Reader, error) {
	return func(name string) (io.Reader, error) {
		root, ok := roots[name]
		if !ok {
			return nil, ErrRootNotFound
		}
		f, err := backend(name)
		if err != nil {
			return nil, err
		}
		return WrapReader(f, maxMemoryUsage, root), nil
	}
}
