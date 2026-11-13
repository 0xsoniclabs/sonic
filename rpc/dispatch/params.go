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
	"fmt"
	"io"
	"reflect"
)

// paramsLayout is how the positional parameters of a method fill the fields
// of its parameter struct: the n-th parameter is the n-th field. A pointer
// field is optional and all optional fields trail, which is go-ethereum's
// rule for a pointer argument, so a missing or null optional parameter is a
// nil field.
type paramsLayout struct {
	// optional tells for each field whether its parameter may be missing.
	optional []bool
}

// newParamsLayout reads the layout of a parameter struct, and panics on a
// shape it cannot serve: a field that is not exported, or a required field
// after an optional one.
func newParamsLayout(paramsType reflect.Type) paramsLayout {
	if paramsType.Kind() != reflect.Struct {
		panic(fmt.Sprintf("parameters %v are not a struct", paramsType))
	}
	var layout paramsLayout
	for field := range paramsType.Fields() {
		if !field.IsExported() {
			panic(fmt.Sprintf("parameter %s of %v is not exported", field.Name, paramsType))
		}
		optional := field.Type.Kind() == reflect.Pointer
		if !optional && len(layout.optional) > 0 && layout.optional[len(layout.optional)-1] {
			panic(fmt.Sprintf("required parameter %s of %v follows an optional one", field.Name, paramsType))
		}
		layout.optional = append(layout.optional, optional)
	}
	return layout
}

// decode fills the fields of params, a pointer to the parameter struct, from
// the raw parameters of a request. Its errors are worded as go-ethereum's
// server words them; missing or null parameters are an empty list.
func (l paramsLayout) decode(raw json.RawMessage, params any) error {
	fields := reflect.ValueOf(params).Elem()
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	count := 0
	switch {
	case err == io.EOF || token == nil && err == nil:
	case err != nil:
		return err
	case token == json.Delim('['):
		if count, err = l.decodeArray(decoder, fields); err != nil {
			return err
		}
	default:
		return errors.New("non-array args")
	}
	for i := count; i < len(l.optional); i++ {
		if !l.optional[i] {
			return fmt.Errorf("missing value for required argument %d", i)
		}
	}
	return nil
}

// decodeArray decodes the elements of a parameter array into the fields, and
// answers how many there were.
func (l paramsLayout) decodeArray(decoder *json.Decoder, fields reflect.Value) (int, error) {
	count := 0
	for ; decoder.More(); count++ {
		if count >= len(l.optional) {
			return count, fmt.Errorf("too many arguments, want at most %d", len(l.optional))
		}
		if err := decoder.Decode(fields.Field(count).Addr().Interface()); err != nil {
			return count, fmt.Errorf("invalid argument %d: %v", count, err)
		}
	}
	_, err := decoder.Token()
	return count, err
}
