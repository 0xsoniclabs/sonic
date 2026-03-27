package ioread

import (
	"bytes"
	"io"
	"testing"
)

func TestReadAll_ExactSize(t *testing.T) {
	data := []byte("hello")
	reader := bytes.NewReader(data)
	buf := make([]byte, 5)
	err := ReadAll(reader, buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(buf, data) {
		t.Fatalf("expected %v, got %v", data, buf)
	}
}

func TestReadAll_EmptyBuffer(t *testing.T) {
	reader := bytes.NewReader([]byte("data"))
	buf := make([]byte, 0)
	err := ReadAll(reader, buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadAll_EOF(t *testing.T) {
	data := []byte("hi")
	reader := bytes.NewReader(data)
	buf := make([]byte, 10)
	err := ReadAll(reader, buf)
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

// slowReader delivers one byte at a time
type slowReader struct {
	data []byte
	pos  int
}

func (r *slowReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	p[0] = r.data[r.pos]
	r.pos++
	return 1, nil
}

func TestReadAll_SlowReader(t *testing.T) {
	data := []byte("abcdef")
	reader := &slowReader{data: data}
	buf := make([]byte, len(data))
	err := ReadAll(reader, buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(buf, data) {
		t.Fatalf("expected %v, got %v", data, buf)
	}
}

type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (int, error) {
	return 0, r.err
}

func TestReadAll_Error(t *testing.T) {
	expectedErr := io.ErrUnexpectedEOF
	reader := &errorReader{err: expectedErr}
	buf := make([]byte, 5)
	err := ReadAll(reader, buf)
	if err != expectedErr {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}
