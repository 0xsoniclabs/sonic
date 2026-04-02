package bundle

import "io"

//go:generate mockgen -source=io_test.go -destination=io_test_mock.go -package=bundle

type Reader interface {
	io.Reader
}

type Writer interface {
	io.Writer
}
