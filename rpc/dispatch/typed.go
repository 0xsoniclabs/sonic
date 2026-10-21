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
	"context"
	"fmt"
)

// The Add helpers register a handler with typed parameters. The dispatcher
// decodes the parameters in order, as go-ethereum does, and counts them against
// the names given: a name ending in "?" is optional, an absent or null optional
// parameter arrives as the zero value of its type (nil for a pointer).

func checkArity(name string, arity int, params []string) {
	if len(params) != arity {
		panic(fmt.Sprintf("rpc: method %s: %d parameter names for %d parameters", name, len(params), arity))
	}
}

// Add0 registers a method without parameters.
func Add0[R any](d *Dispatcher, name string, h func(ctx context.Context) (R, error)) {
	d.Add(name, func(ctx context.Context, args Args) (any, error) {
		if err := args.Check(); err != nil {
			return nil, err
		}
		return h(ctx)
	})
}

// Add1 registers a method with 1 parameter.
func Add1[A, R any](d *Dispatcher, name string, h func(ctx context.Context, a A) (R, error), params ...string) {
	checkArity(name, 1, params)
	d.Add(name, func(ctx context.Context, args Args) (any, error) {
		a, err := decode[A](args, 0)
		if err != nil {
			return nil, err
		}
		if err := args.Check(); err != nil {
			return nil, err
		}
		return h(ctx, a)
	}, params...)
}

// Add2 registers a method with 2 parameters.
func Add2[A, B, R any](d *Dispatcher, name string, h func(ctx context.Context, a A, b B) (R, error), params ...string) {
	checkArity(name, 2, params)
	d.Add(name, func(ctx context.Context, args Args) (any, error) {
		a, err := decode[A](args, 0)
		if err != nil {
			return nil, err
		}
		b, err := decode[B](args, 1)
		if err != nil {
			return nil, err
		}
		if err := args.Check(); err != nil {
			return nil, err
		}
		return h(ctx, a, b)
	}, params...)
}

// Add3 registers a method with 3 parameters.
func Add3[A, B, C, R any](d *Dispatcher, name string, h func(ctx context.Context, a A, b B, c C) (R, error), params ...string) {
	checkArity(name, 3, params)
	d.Add(name, func(ctx context.Context, args Args) (any, error) {
		a, err := decode[A](args, 0)
		if err != nil {
			return nil, err
		}
		b, err := decode[B](args, 1)
		if err != nil {
			return nil, err
		}
		c, err := decode[C](args, 2)
		if err != nil {
			return nil, err
		}
		if err := args.Check(); err != nil {
			return nil, err
		}
		return h(ctx, a, b, c)
	}, params...)
}

// Add4 registers a method with 4 parameters.
func Add4[A, B, C, D, R any](d *Dispatcher, name string, h func(ctx context.Context, a A, b B, c C, d D) (R, error), params ...string) {
	checkArity(name, 4, params)
	d.Add(name, func(ctx context.Context, args Args) (any, error) {
		a, err := decode[A](args, 0)
		if err != nil {
			return nil, err
		}
		b, err := decode[B](args, 1)
		if err != nil {
			return nil, err
		}
		c, err := decode[C](args, 2)
		if err != nil {
			return nil, err
		}
		d, err := decode[D](args, 3)
		if err != nil {
			return nil, err
		}
		if err := args.Check(); err != nil {
			return nil, err
		}
		return h(ctx, a, b, c, d)
	}, params...)
}
