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

package analysis

import (
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/types/typeutil"
)

// Config names the main packages of a program and its consensus path, the code
// whose results all nodes have to agree on. Roots are functions of the main
// module, written like keys relative to it. The consensus path does not enter
// the skipped functions of the main module, nor the cut go-ethereum symbols or
// packages, nor variables of a cut type: this code reaches them, but they
// cannot affect its results. Upgrades is the struct type of the main module
// whose fields name the upgrades that gate changes of the consensus path.
// Modules lists the dependency modules whose declarations, files and cgo
// preambles are compared like those of go-ethereum, instead of by version.
type Config struct {
	Mains    []string
	Modules  []string
	Roots    []string
	Skips    []string
	Cuts     []string
	Upgrades string
}

var Sonic = Config{
	Mains: []string{"./cmd/sonicd", "./cmd/sonictool"},
	Modules: []string{
		"github.com/0xsoniclabs/carmen/go", // the state database
		"github.com/0xsoniclabs/tosca",     // the EVM implementations
	},
	Roots: []string{
		"gossip func consensusCallbackBeginBlockFn", // forms and executes blocks
		"eventcheck func Checkers.Validate",         // decides which events are valid
		"gossip func Store.ApplyGenesis",            // writes the initial state
	},
	Skips: []string{
		// Only imports genesis files without a Carmen (S5) section, which
		// predate Sonic's networks.
		"gossip/evmstore func Store.ImportLegacyEvmData",
	},
	Cuts: []string{
		// Sonic always sets vm.Config.Interpreter (SFVM) and block processing
		// sets no tracer, so go-ethereum's interpreter only runs for tracing
		// and eth_simulateV1.
		"core/vm func NewEvmInterpreter",
		"core/vm type evmInterpreter",
		// Only that interpreter reads the jump table, and EnableEIP only edits
		// it for vm.Config.ExtraEips, which Sonic does not set.
		"core/vm type JumpTable",
		"core/vm func EnableEIP",
		// Tracing hooks, logging and metrics only observe the processing.
		"core/tracing",
		"log",
		"metrics",
	},
	Upgrades: "opera type Upgrades",
}

func covered(cuts []string, key string) bool {
	return slices.ContainsFunc(cuts, func(p string) bool { return covers(p, key) })
}

func (ix *index) cutType(cuts []string, t types.Type) bool {
	n := namedOf(t)
	if n == nil {
		return false
	}
	key, ok := ix.objKey(n.Origin().Obj())
	return ok && covered(cuts, key)
}

func (ix *index) uncutKey(cuts []string, obj types.Object) (string, bool) {
	key, ok := ix.objKey(obj)
	if !ok || covered(cuts, key) {
		return "", false
	}
	if v, isVar := obj.(*types.Var); isVar && ix.cutType(cuts, v.Type()) {
		return "", false
	}
	return key, true
}

func covers(p, key string) bool {
	return key == p || strings.HasPrefix(key, p+" ") || strings.HasPrefix(key, p+"/") ||
		strings.HasPrefix(key, strings.Replace(p, " type ", " func ", 1)+".")
}

func (c Config) resolve(prog *ssa.Program, mod string, symbols map[string]string) (roots, skips []*ssa.Function, upgrades, missing []string) {
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
	roots, skips = find("root", c.Roots), find("skip", c.Skips)
	found := map[string]bool{}
	for key := range symbols {
		for _, p := range c.Cuts {
			found[p] = found[p] || covers(p, key)
		}
	}
	for _, p := range c.Cuts {
		if !found[p] {
			missing = append(missing, "cut "+p)
		}
	}
	if c.Upgrades != "" {
		if upgrades = findFields(prog, mod, c.Upgrades); upgrades == nil {
			missing = append(missing, "upgrades "+c.Upgrades)
		}
	}
	return roots, skips, upgrades, missing
}

// findFields returns the lowercase names of the fields of the struct type that
// key names in the main module mod.
func findFields(prog *ssa.Program, mod, key string) []string {
	rel, name, ok := strings.Cut(key, " type ")
	if !ok {
		return nil
	}
	p := prog.ImportedPackage(mod + "/" + rel)
	if p == nil {
		return nil
	}
	obj, _ := p.Pkg.Scope().Lookup(name).(*types.TypeName)
	if obj == nil {
		return nil
	}
	s, _ := obj.Type().Underlying().(*types.Struct)
	if s == nil {
		return nil
	}
	var names []string
	for f := range s.Fields() {
		names = append(names, strings.ToLower(f.Name()))
	}
	return names
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

// limits restricts a reachability analysis to a part of the program. The types
// of seed are runtime types from the start, but only for the interfaces of the
// main module mod and of go-ethereum, through which the program connects its
// parts, and for those of other modules if they are of the same module. Code
// calls the interfaces of the standard library, and unnamed ones, on all kinds
// of values, so only the types converted in reachable code implement them.
// Functions in skip, and go-ethereum functions and types covered by cuts, with
// the methods of those types, are never reachable, and neither the initializers
// of variables covered by cuts or of a cut type nor the init functions writing
// them are followed.
type limits struct {
	ix   *index
	seed *typeutil.Map
	mod  string
	skip map[*ssa.Function]bool
	cuts []string
}

func (l *limits) cutFunc(f *ssa.Function) bool {
	if l.skip[f] {
		return true
	}
	obj, ok := topLevel(f).Object().(*types.Func)
	if !ok {
		return false
	}
	key, ok := l.ix.objKey(obj)
	return ok && covered(l.cuts, key)
}

func (l *limits) cutType(t types.Type) bool {
	return l.ix.cutType(l.cuts, t)
}

// admits reports whether the seed type t counts as an implementation of iface,
// given that it has its methods.
func (l *limits) admits(t, iface types.Type) bool {
	pkg := namedPkg(iface)
	if pkg == nil {
		return false
	}
	mod := l.ix.modules[pkg]
	switch {
	case mod == "":
		return false
	case isGeth(pkg.Path()) || mod == l.mod:
		return true
	}
	return l.ix.modules[namedPkg(t)] == mod
}

func namedPkg(t types.Type) *types.Package {
	if n := namedOf(t); n != nil {
		return n.Obj().Pkg()
	}
	return nil
}

func namedOf(t types.Type) *types.Named {
	t = types.Unalias(t)
	if p, ok := t.(*types.Pointer); ok {
		t = types.Unalias(p.Elem())
	}
	n, _ := t.(*types.Named)
	return n
}
