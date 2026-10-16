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

	"golang.org/x/sync/errgroup"
)

// Group runs the child goroutines of one request. A child that panics becomes
// an error of the group and cancels its siblings; Wait returns only after all
// children returned. A Group is the only way a handler may start a goroutine.
type Group struct {
	d  *Dispatcher
	eg *errgroup.Group
	//lint:ignore containedctx the group's context is the context of its children
	ctx context.Context
}

// Group creates a group whose children run under a context derived from ctx
// that ends when a child fails or Wait returns.
func (d *Dispatcher) Group(ctx context.Context) (*Group, context.Context) {
	eg, gctx := errgroup.WithContext(ctx)
	return &Group{d: d, eg: eg, ctx: gctx}, gctx
}

// Go starts a child. The child's panic is recovered, logged once with its
// stack, counted under the method of the request and returned by Wait.
func (g *Group) Go(fn func(ctx context.Context) error) {
	g.eg.Go(func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				label := MethodOf(g.ctx)
				err = g.d.recovered(label, "child of "+label, r)
			}
		}()
		return fn(g.ctx)
	})
}

// SetLimit bounds the number of children running at once.
func (g *Group) SetLimit(n int) { g.eg.SetLimit(n) }

// Wait waits for all children and returns the first error.
func (g *Group) Wait() error { return g.eg.Wait() }
