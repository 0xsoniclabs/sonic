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

package ethapi

import (
	"encoding/json"
	"strconv"
	"testing"
)

const maxResultSize = 25 * 1024 * 1024

func TestNewJsonResultBuffer(t *testing.T) {
	b, err := NewJsonResultBuffer(maxResultSize)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if b != nil && b.Cap() != bufferStartSize {
		t.Errorf("expected buffer size to be %d, got %d", bufferStartSize, b.Cap())
	}
}

func TestAddOneObject(t *testing.T) {
	b, err := NewJsonResultBuffer(maxResultSize)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	obj := testStruct{Str: "test"}

	err = b.AddObject(obj)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	result, err := b.GetResult()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	var newObj []testStruct
	err = json.Unmarshal(result, &newObj)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if newObj[0].Str != "test" {
		t.Errorf("expected str to be test, got %v", newObj[0].Str)

	}
}

func TestAddMoreObjects(t *testing.T) {
	buffer, err := NewJsonResultBuffer(maxResultSize)
	if err != nil {
		t.Fatalf("failed to create JsonResultBuffer: %v", err)
	}

	for i := 0; i < 10; i++ {
		object := testStruct{Str: "test" + strconv.Itoa(i)}
		if err := buffer.AddObject(object); err != nil {
			t.Fatalf("failed to add object: %v", err)
		}
	}

	result, err := buffer.GetResult()
	if err != nil {
		t.Fatalf("failed to get result: %v", err)
	}

	var objects []testStruct
	if err := json.Unmarshal(result, &objects); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if len(objects) != 10 {
		t.Errorf("expected 10 objects, got %v", len(objects))
	}

	if objects[9].Str != "test9" {
		t.Errorf("expected last object to be 'test9', got %v", objects[9].Str)
	}
}

func TestAddObjectOverLimit(t *testing.T) {
	b, err := NewJsonResultBuffer(10)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	obj := testStruct{Str: "test string"}

	err = b.AddObject(obj)
	if err != ErrResponseTooLarge {
		t.Errorf("expected ErrResponseTooLarge, got %v", err)
	}
}

type testStruct struct {
	Str string
}
