// Copyright 2026 Sonic Operations Ltd
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

package dispatch

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// Parameter is one positional parameter of a method: its name, whether a
// client must send it, and how it is decoded.
type Parameter struct {
	name     string
	required bool
	decode   func(raw json.RawMessage) (any, error)
	absent   func() any

	// undecoded marks the one parameter of a method that takes the whole
	// array undecoded.
	undecoded bool
}

// Required is a parameter a client must send. A JSON null decodes as the type
// decodes it, as go-ethereum's server does for a non-pointer parameter.
func Required[Value any](name string) Parameter {
	return Parameter{
		name:     name,
		required: true,
		decode: func(raw json.RawMessage) (any, error) {
			var value Value
			err := json.Unmarshal(raw, &value)
			return value, err
		},
	}
}

// Optional is a parameter a client may leave out or send as null; its
// argument is then a nil *Value.
func Optional[Value any](name string) Parameter {
	return Parameter{
		name: name,
		decode: func(raw json.RawMessage) (any, error) {
			var value *Value
			err := json.Unmarshal(raw, &value)
			return value, err
		},
		absent: func() any { return (*Value)(nil) },
	}
}

// wholeArray makes a method take its parameter array as it came, as the
// single argument of type json.RawMessage, for the calls the module hands on
// to the node.
func wholeArray() Parameter {
	return Parameter{name: "parameters", undecoded: true}
}

// Arguments are the decoded parameters of a call, in the order of the
// method's parameters. An absent optional parameter is a nil pointer.
type Arguments []any

// Argument returns the argument at a position as the type its parameter
// decodes to: Value for a Required[Value], *Value for an Optional[Value].
func Argument[Value any](arguments Arguments, position int) Value {
	return arguments[position].(Value)
}

// DecodeParameters decodes the JSON array of a call against the parameters of
// a method. Like go-ethereum's server it accepts an absent or null array and
// fills absent optional parameters.
func DecodeParameters(raw json.RawMessage, parameters []Parameter) (Arguments, error) {
	if len(parameters) == 1 && parameters[0].undecoded {
		return Arguments{raw}, nil
	}
	elements, err := splitArray(raw, len(parameters))
	if err != nil {
		return nil, err
	}
	arguments := make(Arguments, 0, len(parameters))
	for position, element := range elements {
		parameter := parameters[position]
		value, err := parameter.decode(element)
		if err != nil {
			return nil, InvalidParameters("parameter %d (%s): %v", position, parameter.name, err)
		}
		arguments = append(arguments, value)
	}
	for position := len(arguments); position < len(parameters); position++ {
		parameter := parameters[position]
		if parameter.required {
			return nil, InvalidParameters("missing value for required argument %d", position)
		}
		arguments = append(arguments, parameter.absent())
	}
	return arguments, nil
}

// splitArray returns the elements of a JSON array, refusing more than limit
// before it decodes one beyond it.
func splitArray(raw json.RawMessage, limit int) ([]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	switch {
	case errors.Is(err, io.EOF) || token == nil && err == nil:
		return nil, nil
	case err != nil:
		return nil, InvalidParameters("%v", err)
	case token != json.Delim('['):
		return nil, InvalidParameters("non-array args")
	}
	var elements []json.RawMessage
	for decoder.More() {
		if len(elements) >= limit {
			return nil, InvalidParameters("too many arguments, want at most %d", limit)
		}
		var element json.RawMessage
		if err := decoder.Decode(&element); err != nil {
			return nil, InvalidParameters("parameter %d: %v", len(elements), err)
		}
		elements = append(elements, element)
	}
	if _, err := decoder.Token(); err != nil {
		return nil, InvalidParameters("%v", err)
	}
	return elements, nil
}
