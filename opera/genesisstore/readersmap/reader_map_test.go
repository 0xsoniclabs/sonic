package readersmap

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestWrap_Success(t *testing.T) {
	units := []Unit{
		{Name: "a", ReaderProvider: func() (io.Reader, error) { return bytes.NewReader(nil), nil }},
		{Name: "b", ReaderProvider: func() (io.Reader, error) { return bytes.NewReader(nil), nil }},
	}
	m, err := Wrap(units)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m))
	}
}

func TestWrap_Empty(t *testing.T) {
	m, err := Wrap(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(m))
	}
}

func TestWrap_DuplicateName(t *testing.T) {
	units := []Unit{
		{Name: "a", ReaderProvider: func() (io.Reader, error) { return nil, nil }},
		{Name: "a", ReaderProvider: func() (io.Reader, error) { return nil, nil }},
	}
	_, err := Wrap(units)
	if !errors.Is(err, ErrDupFile) {
		t.Fatalf("expected ErrDupFile, got %v", err)
	}
}

func TestMap_Open_Found(t *testing.T) {
	data := []byte("hello")
	units := []Unit{
		{Name: "test", ReaderProvider: func() (io.Reader, error) { return bytes.NewReader(data), nil }},
	}
	m, _ := Wrap(units)

	r, err := m.Open("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	buf, _ := io.ReadAll(r)
	if !bytes.Equal(buf, data) {
		t.Fatalf("expected %q, got %q", data, buf)
	}
}

func TestMap_Open_NotFound(t *testing.T) {
	m, _ := Wrap(nil)
	_, err := m.Open("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMap_Open_ProviderError(t *testing.T) {
	provErr := errors.New("provider failed")
	units := []Unit{
		{Name: "bad", ReaderProvider: func() (io.Reader, error) { return nil, provErr }},
	}
	m, _ := Wrap(units)

	_, err := m.Open("bad")
	if !errors.Is(err, provErr) {
		t.Fatalf("expected provider error, got %v", err)
	}
}
