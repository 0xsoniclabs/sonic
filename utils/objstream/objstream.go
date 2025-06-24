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

// This package provides reader and writer utilities for encoding sequences of
// serializable objects into a single byte stream. The format is simply
//
//	(<size-of-object><serialized-object>)*
//
// where
//
//	<size-of-object>    .. is a variable length encoded size of hte object data
//	<serialized-object> .. is a variable-sized encoded object
//
// An example use case for this is the encoding of a sequence of proto buffer
// objects. The encoding of proto buffer do not include any special termination
// characters. Thus, when encoding a sequence of proto buffer objects, the
// receiver needs to know when one object ends and the next one starts. This
// package provides a simple way to encode and decode such sequences.
//
// Example:
//
//		  type Obj struct { ... }
//
//		  func (o Obj) Serialize() ([]byte, error) {
//	       // do the encoding
//		  }
//
//		  func ( *Obj) Deserialize(data []byte) error {
//	       // do the decoding
//		  }
//
// you can use the `Reader` and `Writer` to encode and decode sequences of `Obj`
// objects.
package objstream

//go:generate mockgen -source=objstream.go -destination=objstream_mock.go -package=objstream

import (
	"bufio"
	"encoding/binary"
	"io"
)

// Reader reads objects from an input stream.
type Reader[T Deserializer] struct {
	in *bufio.Reader
}

// NewReader creates a new reader that reads objects from the given input stream.
func NewReader[T Deserializer](in io.Reader) Reader[T] {
	return Reader[T]{in: bufio.NewReader(in)}
}

// Read reads the next object from the input stream. It returns an error if
// there is an issue in the underlying stream or if the object could not be
// decoded.
func (r *Reader[T]) Read(target T) error {
	// read length
	length, err := binary.ReadUvarint(r.in)
	if err != nil {
		return err
	}

	// read encoded object
	encodedObject := make([]byte, length)
	if _, err := io.ReadFull(r.in, encodedObject); err != nil {
		return err
	}

	// decode
	return target.Deserialize(encodedObject)
}

// Writer writes objects to an output stream.
type Writer[T Serializer] struct {
	out io.Writer
}

// NewWriter creates a new writer that writes objects to the given output stream.
func NewWriter[T Serializer](out io.Writer) Writer[T] {
	return Writer[T]{out: out}
}

// Write writes the given object to the output stream. It returns an error if
// there is an issue in the underlying stream or if the object could not be
// encoded.
func (w *Writer[T]) Write(source T) error {
	encodedObject, err := source.Serialize()
	if err != nil {
		return err
	}

	// write length
	encodedLength := binary.AppendUvarint(nil, uint64(len(encodedObject)))
	if _, err := w.out.Write(encodedLength); err != nil {
		return err
	}

	// write encoded object
	_, err = w.out.Write(encodedObject)
	return err
}

// Serializer is an interface that objects need to implement to be serialized.
type Serializer interface {
	Serialize() ([]byte, error)
}

// Deserializer is an interface that objects need to implement to be deserialized.
type Deserializer interface {
	Deserialize([]byte) error
}
