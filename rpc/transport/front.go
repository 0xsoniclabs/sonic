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

package transport

import (
	"context"
	"encoding/json"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// Forwarder hands a call to the node's legacy handler. It exists while methods
// are being ported; the composition root provides it.
type Forwarder interface {
	Forward(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, *dispatch.Error)
}

// Front routes a call to the dispatcher, or to the node's legacy handler.
//
// A method the dispatcher has is served by it. Anything else is unknown, with
// three exceptions the legacy handler keeps serving until it is removed: the
// namespaces and methods of the consensus internals and the node's wallets,
// which are for the IPC transport only, and the p2p facts, which every
// transport serves.
type Front struct {
	d      *dispatch.Dispatcher
	legacy Forwarder
	denied map[string]bool

	ipcNamespaces map[string]bool // namespaces served by the legacy handler on IPC
	ipcMethods    map[string]bool // methods served by the legacy handler on IPC
	allMethods    map[string]bool // methods served by the legacy handler on every transport
	extra         []string        // namespaces the legacy handler serves
}

// FrontOption configures a Front.
type FrontOption func(*Front)

// WithLegacy sets the handler that serves what the module does not.
func WithLegacy(f Forwarder) FrontOption { return func(fr *Front) { fr.legacy = f } }

// WithLegacyNamespaces lists the namespaces of the legacy handler, for the
// answer of rpc_modules.
func WithLegacyNamespaces(namespaces ...string) FrontOption {
	return func(fr *Front) { fr.extra = append(fr.extra, namespaces...) }
}

// WithLegacyOnIPC makes the IPC transport forward the namespaces and the
// methods to the legacy handler.
func WithLegacyOnIPC(namespaces []string, methods []string) FrontOption {
	return func(fr *Front) {
		for _, ns := range namespaces {
			fr.ipcNamespaces[ns] = true
		}
		for _, m := range methods {
			fr.ipcMethods[m] = true
		}
	}
}

// WithLegacyEverywhere makes every transport forward the methods to the legacy
// handler.
func WithLegacyEverywhere(methods ...string) FrontOption {
	return func(fr *Front) {
		for _, m := range methods {
			fr.allMethods[m] = true
		}
	}
}

// Namespaces returns the namespaces served on a transport that enables the
// given ones (all if none): those of the dispatcher and the legacy handler,
// and rpc.
func (f *Front) Namespaces(enabled func(string) bool) map[string]string {
	res := map[string]string{"rpc": "1.0"}
	for _, ns := range append(f.d.Namespaces(), f.extra...) {
		if enabled(ns) {
			res[ns] = "1.0"
		}
	}
	return res
}

// NewFront creates a front for the dispatcher. The methods eth_stop and
// ftm_stop are denied: the legacy filter API exports its Stop method, which
// any client could call to stop the filter event loop.
func NewFront(d *dispatch.Dispatcher, opts ...FrontOption) *Front {
	f := &Front{
		d:             d,
		denied:        map[string]bool{"eth_stop": true, "ftm_stop": true},
		ipcNamespaces: map[string]bool{},
		ipcMethods:    map[string]bool{},
		allMethods:    map[string]bool{},
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// Dispatcher returns the dispatcher of the front.
func (f *Front) Dispatcher() *dispatch.Dispatcher { return f.d }

type ipcKey struct{}

// withIPC marks the context of a request that came over IPC.
func withIPC(ctx context.Context) context.Context { return context.WithValue(ctx, ipcKey{}, true) }

func isIPC(ctx context.Context) bool {
	v, _ := ctx.Value(ipcKey{}).(bool)
	return v
}

// Call serves one method call.
func (f *Front) Call(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, *dispatch.Error) {
	if f.d.Has(method) {
		return f.d.Serve(ctx, method, params)
	}
	ns, _, _ := cutNamespace(method)
	forward := !f.denied[method] && f.legacy != nil &&
		(f.allMethods[method] || isIPC(ctx) && (f.ipcNamespaces[ns] || f.ipcMethods[method]))
	if !forward {
		return f.d.Serve(ctx, method, params) // unknown: -32601, counted as other
	}
	return f.legacy.Forward(ctx, method, params)
}

func cutNamespace(method string) (ns, rest string, found bool) {
	for i := 0; i < len(method); i++ {
		if method[i] == '_' {
			return method[:i], method[i+1:], true
		}
	}
	return method, "", false
}

// Subscribe serves an eth_subscribe request of a namespace.
func (f *Front) Subscribe(ctx context.Context, ns string, allowed bool, params json.RawMessage) (json.RawMessage, *dispatch.Error) {
	return f.d.Subscribe(ctx, ns, allowed, params)
}
