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
	"runtime/debug"

	"golang.org/x/sync/errgroup"
)

// Group runs the parallel work of one call. A child that panics becomes an
// error that cancels its siblings; the call that waits for the group then
// answers as if it had panicked itself.
type Group struct {
	group *errgroup.Group
}

// NewGroup creates a group whose children share a context cancelled by the
// first failure.
func NewGroup(ctx context.Context) (*Group, context.Context) {
	group, ctx := errgroup.WithContext(ctx)
	return &Group{group: group}, ctx
}

// Go runs a child.
func (g *Group) Go(child func() error) {
	g.group.Go(func() (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				err = &childCrash{recovered: recovered, stack: debug.Stack()}
			}
		}()
		return child()
	})
}

// Wait waits for all children and answers the first failure.
func (g *Group) Wait() error {
	return g.group.Wait()
}

// childCrash is the panic of a child, carried to the call as an error.
type childCrash struct {
	recovered any
	stack     []byte
}

func (c *childCrash) Error() string {
	return fmt.Sprintf("panic: %v", c.recovered)
}
