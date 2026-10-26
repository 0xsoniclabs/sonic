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

// Package transport serves the dispatcher over HTTP, WebSocket and IPC with
// go-ethereum's JSON-RPC envelope.
package transport

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// Kind is a transport.
type Kind string

// The transports.
const (
	HTTP Kind = "http"
	WS   Kind = "ws"
	IPC  Kind = "ipc"
)

// Router serves the calls of an endpoint.
type Router interface {
	Serve(ctx context.Context, kind Kind, call dispatch.Call) (json.RawMessage, error)
}

// Forwarder answers a call with the node's own handler.
type Forwarder func(ctx context.Context, call dispatch.Call) (json.RawMessage, error)

// FrontConfig says what each transport serves.
type FrontConfig struct {
	// Modules are the namespaces enabled per transport (--http.api,
	// --ws.api); an empty list enables every namespace, and IPC enables all.
	Modules map[Kind][]string
	// Forward answers, on IPC, the calls the module does not serve: the
	// namespaces that stay with the node.
	Forward Forwarder
	// ForwardEverywhere are the methods forwarded on every transport that
	// enables their namespace: the p2p facts.
	ForwardEverywhere []string
}

// Front routes the calls of every transport: to the dispatcher for the
// methods the module serves, and to the node's handler for the p2p facts and,
// on IPC only, for the namespaces that stay with the node.
type Front struct {
	d          *dispatch.Dispatcher
	cfg        FrontConfig
	enabled    map[Kind]map[string]bool
	everywhere map[string]bool
}

// NewFront creates the router and registers rpc_modules.
func NewFront(d *dispatch.Dispatcher, cfg FrontConfig) *Front {
	f := &Front{d: d, cfg: cfg, enabled: make(map[Kind]map[string]bool), everywhere: make(map[string]bool)}
	for kind, modules := range cfg.Modules {
		if kind == IPC || len(modules) == 0 {
			continue
		}
		set := make(map[string]bool)
		for _, m := range modules {
			set[m] = true
		}
		f.enabled[kind] = set
	}
	for _, m := range cfg.ForwardEverywhere {
		f.everywhere[m] = true
	}
	d.Add("rpc_modules", dispatch.Func(func(ctx context.Context, _ dispatch.None) (any, error) {
		kind, _ := ctx.Value(kindKey{}).(Kind)
		return f.modules(ctx, kind)
	}))
	return f
}

type kindKey struct{}

// isEnabled reports whether a transport serves a namespace; the rpc namespace
// is served everywhere, as by go-ethereum's server.
func (f *Front) isEnabled(kind Kind, namespace string) bool {
	set, restricted := f.enabled[kind]
	return !restricted || set[namespace] || namespace == "rpc"
}

// Serve routes a call of a transport.
func (f *Front) Serve(ctx context.Context, kind Kind, call dispatch.Call) (json.RawMessage, error) {
	ctx = context.WithValue(ctx, kindKey{}, kind)
	name := call.Method
	namespace := dispatch.NamespaceOf(name)
	switch {
	case strings.HasSuffix(name, "_unsubscribe"):
		// Every namespace unsubscribes, as in go-ethereum.
		return f.d.Serve(ctx, call)
	case strings.HasSuffix(name, "_subscribe"):
		if _, stream := dispatch.NotifierFrom(ctx); stream && !f.isEnabled(kind, namespace) {
			subKind, err := dispatch.SubscriptionKind(call.Params)
			if err != nil {
				return nil, &dispatch.Error{Code: dispatch.CodeInvalidParams, Message: err.Error()}
			}
			return nil, dispatch.SubscriptionNotFound(namespace, subKind)
		}
		return f.d.Serve(ctx, call)
	}
	if !f.isEnabled(kind, namespace) {
		return nil, dispatch.MethodNotFound(name)
	}
	if !f.d.Has(name) && f.cfg.Forward != nil && (kind == IPC || f.everywhere[name]) {
		return f.d.Forward(ctx, call, f.cfg.Forward)
	}
	return f.d.Serve(ctx, call)
}

// modules answers rpc_modules: the namespaces a transport serves, those of the
// node's handler included on IPC.
func (f *Front) modules(ctx context.Context, kind Kind) (map[string]string, error) {
	served := make(map[string]bool)
	for _, ns := range f.d.Namespaces() {
		served[ns] = true
	}
	for _, m := range f.cfg.ForwardEverywhere {
		served[dispatch.NamespaceOf(m)] = true
	}
	if kind == IPC && f.cfg.Forward != nil {
		raw, err := f.cfg.Forward(ctx, dispatch.Call{Method: "rpc_modules"})
		if err != nil {
			return nil, err
		}
		var node map[string]string
		if err := json.Unmarshal(raw, &node); err != nil {
			return nil, err
		}
		for ns := range node {
			served[ns] = true
		}
	}
	names := make([]string, 0, len(served))
	for ns := range served {
		names = append(names, ns)
	}
	sort.Strings(names)
	modules := make(map[string]string, len(names))
	for _, ns := range names {
		if f.isEnabled(kind, ns) {
			modules[ns] = "1.0"
		}
	}
	return modules, nil
}
