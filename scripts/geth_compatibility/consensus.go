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

package main

import (
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/types/typeutil"
)

// consensusPath describes the code whose results all nodes have to agree on.
// Roots are functions of the main module, written like finding keys relative
// to it. The analysis does not enter the skipped functions of the main module,
// nor the cut go-ethereum symbols or packages: this code reaches them, but they
// cannot affect its results.
type consensusPath struct {
	roots []string
	skips []string
	cuts  []string
}

var sonicConsensus = consensusPath{
	roots: []string{
		"gossip func consensusCallbackBeginBlockFn", // forms and executes blocks
		"eventcheck func Checkers.Validate",         // decides which events are valid
		"gossip func Store.ApplyGenesis",            // writes the initial state
	},
	skips: []string{
		// Only imports genesis files without a Carmen (S5) section, which
		// predate Sonic's networks.
		"gossip/evmstore func Store.ImportLegacyEvmData",
	},
	cuts: []string{
		// Sonic always sets vm.Config.Interpreter (LFVM, SFVM from Brio) and
		// block processing sets no tracer, so go-ethereum's interpreter only
		// runs for tracing and eth_simulateV1.
		"core/vm func NewEvmInterpreter",
		"core/vm type evmInterpreter",
		"core/tracing",
		// Logging and metrics only observe the processing.
		"log",
		"metrics",
	},
}

func (c consensusPath) cut(key string) bool {
	return slices.ContainsFunc(c.cuts, func(p string) bool { return covers(p, key) })
}

func covers(p, key string) bool {
	return key == p || strings.HasPrefix(key, p+" ") || strings.HasPrefix(key, p+"/")
}

func (c consensusPath) resolve(prog *ssa.Program, mod string, symbols map[string]*symbol) (roots, skips []*ssa.Function, missing []string) {
	find := func(kind string, keys []string) []*ssa.Function {
		var funcs []*ssa.Function
		for _, key := range keys {
			if f := findFunc(prog, mod, key); f != nil {
				funcs = append(funcs, f)
			} else {
				missing = append(missing, kind+" "+key)
			}
		}
		return funcs
	}
	roots, skips = find("root", c.roots), find("skip", c.skips)
	found := map[string]bool{}
	for key := range symbols {
		for _, p := range c.cuts {
			if covers(p, key) {
				found[p] = true
			}
		}
	}
	for _, p := range c.cuts {
		if !found[p] {
			missing = append(missing, "cut "+p)
		}
	}
	return roots, skips, missing
}

// limits restricts a reachability analysis to a part of the program. The types
// of seed are runtime types from the start, but only for the interfaces of the
// main module mod and of go-ethereum, through which the program connects its
// parts, and for those of other modules if they are of the same module. Code
// calls the interfaces of the standard library, and unnamed ones, on all kinds
// of values, so only the types converted in reachable code implement them.
// Functions in skip, and go-ethereum functions and types whose key matches cut,
// are never reachable.
type limits struct {
	seed    *typeutil.Map
	mod     string
	modules map[*types.Package]string
	skip    map[*ssa.Function]bool
	cut     func(key string) bool
}

func findFunc(prog *ssa.Program, mod, key string) *ssa.Function {
	rel, name, ok := strings.Cut(key, " func ")
	if !ok {
		return nil
	}
	p := prog.ImportedPackage(mod + "/" + rel)
	if p == nil {
		return nil
	}
	recv, method, ok := strings.Cut(name, ".")
	if !ok {
		return p.Func(name)
	}
	obj, _ := p.Pkg.Scope().Lookup(recv).(*types.TypeName)
	if obj == nil {
		return nil
	}
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil
	}
	for m := range named.Methods() {
		if m.Name() == method {
			return prog.FuncValue(m)
		}
	}
	return nil
}

func fnKey(f *ssa.Function) (string, bool) {
	for f.Parent() != nil {
		f = f.Parent()
	}
	if o := f.Origin(); o != nil {
		f = o
	}
	obj, ok := f.Object().(*types.Func)
	if !ok {
		return "", false
	}
	return objKey(obj)
}

func typeKey(t types.Type) (string, bool) {
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	n, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return "", false
	}
	return objKey(n.Origin().Obj())
}

// admits reports whether the seed type t counts as an implementation of iface,
// given that it has its methods.
func (l *limits) admits(t, iface types.Type) bool {
	pkg := namedPkg(iface)
	if pkg == nil {
		return false
	}
	switch {
	case isStd(pkg.Path()):
		return false
	case isGeth(pkg.Path()) || l.modules[pkg] == l.mod:
		return true
	}
	return l.modules[namedPkg(t)] == l.modules[pkg]
}

func namedPkg(t types.Type) *types.Package {
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	if n, ok := types.Unalias(t).(*types.Named); ok {
		return n.Obj().Pkg()
	}
	return nil
}

func (r *reacher) indexSeed(seed *typeutil.Map) {
	seed.Iterate(func(t types.Type, _ any) {
		if types.IsInterface(t) || hasTypeParam(t) || r.cutType(t) {
			return
		}
		mset := r.prog.MethodSets.MethodSet(t)
		for i := range mset.Len() {
			id := mset.At(i).Obj().Id()
			r.seedByMethod[id] = append(r.seedByMethod[id], t)
		}
	})
}

func (r *reacher) cutFunc(f *ssa.Function) bool {
	if r.lim == nil {
		return false
	}
	if r.lim.skip[f] {
		return true
	}
	key, ok := fnKey(f)
	return ok && r.lim.cut(key)
}

func (r *reacher) cutType(t types.Type) bool {
	if r.lim == nil {
		return false
	}
	key, ok := typeKey(t)
	return ok && r.lim.cut(key)
}
