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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode"
)

// Handler serves one method. Its arguments arrive decoded and counted: a
// request whose parameters do not fit the method's Args never reaches it.
type Handler func(ctx context.Context, args Args) (any, error)

// Args are the decoded positional parameters of one request, in the order the
// method declares them.
type Args []any

// None declares a method without parameters.
type None struct{}

// Method is a handler with the positional parameters it takes.
type Method struct {
	params  []param
	handler Handler
}

// param is one positional parameter. A pointer parameter is optional, as in
// go-ethereum: it may be absent or null.
type param struct {
	name     string
	typ      reflect.Type
	optional bool
}

// Func makes a Method of a function that takes its positional parameters as
// the exported fields of the struct P, in field order. A field's parameter
// name is its `rpc` tag or its name with a lower-case first letter; a pointer
// field is optional.
func Func[P any](h func(ctx context.Context, p P) (any, error)) Method {
	typ := reflect.TypeFor[P]()
	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("dispatch: parameters of a method must be a struct, got %v", typ))
	}
	params := make([]param, typ.NumField())
	for i := range params {
		field := typ.Field(i)
		if !field.IsExported() {
			panic(fmt.Sprintf("dispatch: parameter field %s of %v is not exported", field.Name, typ))
		}
		name := field.Tag.Get("rpc")
		if name == "" {
			name = lowerFirst(field.Name)
		}
		params[i] = param{name: name, typ: field.Type, optional: field.Type.Kind() == reflect.Pointer}
	}
	return Method{
		params: params,
		handler: func(ctx context.Context, args Args) (any, error) {
			var p P
			v := reflect.ValueOf(&p).Elem()
			for i, arg := range args {
				v.Field(i).Set(reflect.ValueOf(arg))
			}
			return h(ctx, p)
		},
	}
}

func lowerFirst(s string) string {
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// decode reads the positional parameters of a request, after the first skip
// ones (the kind of a subscription). Absent and null parameters are allowed
// for optional ones only; the errors name the parameter they are about, by
// its position in the request.
func (m *Method) decode(raw json.RawMessage, skip int) (Args, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	args := make(Args, 0, len(m.params))
	tok, err := dec.Token()
	switch {
	case err == io.EOF || tok == nil && err == nil:
		// no parameters
	case err != nil:
		return nil, err
	case tok == json.Delim('['):
		for i := 0; dec.More(); i++ {
			if i < skip {
				var ignored json.RawMessage
				if err := dec.Decode(&ignored); err != nil {
					return nil, err
				}
				continue
			}
			if i-skip >= len(m.params) {
				return nil, fmt.Errorf("too many arguments, want at most %d", len(m.params)+skip)
			}
			p := m.params[i-skip]
			value := reflect.New(p.typ)
			if err := dec.Decode(value.Interface()); err != nil {
				return nil, fmt.Errorf("parameter %d (%s): %v", i, p.name, err)
			}
			args = append(args, value.Elem().Interface())
		}
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("non-array args")
	}
	for i := len(args); i < len(m.params); i++ {
		p := m.params[i]
		if !p.optional {
			return nil, fmt.Errorf("parameter %d (%s): missing value", i+skip, p.name)
		}
		args = append(args, reflect.Zero(p.typ).Interface())
	}
	return args, nil
}

// SubscriptionKind reads the first parameter of a subscription request, the
// kind of the subscription.
func SubscriptionKind(raw json.RawMessage) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if tok, _ := dec.Token(); tok != json.Delim('[') {
		return "", errors.New("non-array args")
	}
	v, _ := dec.Token()
	kind, ok := v.(string)
	if !ok {
		return "", errors.New("expected subscription name as first argument")
	}
	return kind, nil
}

// NamespaceOf returns the part of a method name before the first '_'.
func NamespaceOf(method string) string {
	ns, _, _ := strings.Cut(method, "_")
	return ns
}
