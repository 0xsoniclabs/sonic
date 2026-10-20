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
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const oldCore = `package core

import _ "embed"

//go:embed data.txt
var data string

type Config struct {
	A int
}

const (
	First = iota
	Limit
)

func Apply(c Config) int { return helper(c.A) }

func helper(x int) int { return x + Limit }

func Stable() int {
	return 1
}

func Unused() int { return 1 }

type Runner struct{}

func (Runner) Run() int { return 1 }

func (Runner) Idle() int { return 1 }

type Service struct{}

func (Service) Serve() int { return 1 }
`

const newCore = `package core

import _ "embed"

//go:embed data.txt
var data string

type Config struct {
	A int
	B int
}

const (
	First = iota
	Second
	Limit
)

func Apply(c Config) int { return helper(c.A) + added() }

func helper(x int) int { return x * Limit }

// Stable gained comments only.
func Stable() int {
	// Nothing to see here.
	return 1
}

func Unused() int { return 2 }

type Runner struct{}

func (Runner) Run() int { return 2 }

func (Runner) Idle() int { return 2 }

type Service struct{}

func (Service) Serve() int { return 2 }

func added() int { return 0 }
`

const rpc = `package rpc

import "reflect"

type API struct {
	Service any
}

func Register(api API) {
	v := reflect.ValueOf(api.Service)
	for i := range v.NumMethod() {
		v.Method(i).Call(nil)
	}
}
`

const app = `package main

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/rpc"
)

type runner interface{ Run() int }

func main() {
	var r runner = core.Runner{}
	println(core.Apply(core.Config{A: 1}), core.Stable(), r.Run())
	rpc.Register(rpc.API{Service: core.Service{}})
}
`

func appMod(geth string) string {
	return "module example.com/app\n\ngo 1.26\n\nrequire github.com/ethereum/go-ethereum v0.0.0\n\nreplace github.com/ethereum/go-ethereum => " + geth + "\n"
}

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAnalyze_ReportsExactlyTheReachableDifferences(t *testing.T) {
	dir := t.TempDir()
	gethMod := "module github.com/ethereum/go-ethereum\n\ngo 1.26\n"
	writeTree(t, dir, map[string]string{
		"geth-old/go.mod":        gethMod,
		"geth-old/core/core.go":  oldCore,
		"geth-old/core/data.txt": "old",
		"geth-old/rpc/rpc.go":    rpc,
		"geth-new/go.mod":        gethMod,
		"geth-new/core/core.go":  newCore,
		"geth-new/core/data.txt": "new",
		"geth-new/rpc/rpc.go":    rpc,
		"app-old/go.mod":         appMod("../geth-old"),
		"app-old/main.go":        app,
		"app-new/go.mod":         appMod("../geth-new"),
		"app-new/main.go":        app,
	})
	const runner, service = "core func Runner.Run", "core func Service.Serve"
	explain := func(key string) bool { return key == runner || key == service }
	base, err := analyze(filepath.Join(dir, "app-old"), []string{"."}, consensusPath{}, explain)
	if err != nil {
		t.Fatal(err)
	}
	head, err := analyze(filepath.Join(dir, "app-new"), []string{"."}, consensusPath{}, explain)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]string{}
	direct := map[string]bool{}
	for _, f := range compare(base, head) {
		got[f.key] = f.status
		direct[f.key] = f.direct
	}
	want := map[string]string{
		"core func Apply":       statusChanged,
		"core func helper":      statusChanged,
		"core func added":       statusAdded,
		runner:                  statusChanged,
		service:                 statusChanged,
		"core const Limit":      statusChanged,
		"core type Config":      statusChanged,
		"core file data.txt":    statusChanged,
		"core func Stable":      "",
		"core func Unused":      "",
		"core func Runner.Idle": "",
		"core const Second":     "",
	}
	for key, status := range want {
		if got[key] != status {
			t.Errorf("%s: got status %q, want %q", key, got[key], status)
		}
	}
	for key := range got {
		if _, ok := want[key]; !ok {
			t.Errorf("unexpected finding %s (%s)", key, got[key])
		}
	}
	if !direct["core func Apply"] || direct["core func helper"] {
		t.Errorf("Apply is called by the app and helper is not, got direct %v", direct)
	}

	chain := head.why[runner]
	if len(chain) < 2 || !strings.Contains(chain[0], "called through example.com/app.runner") || chain[len(chain)-1] != "example.com/app.main" {
		t.Errorf("want Runner.Run explained by the interface call in main, got %q", chain)
	}
	chain = head.why[service]
	if len(chain) < 2 || !strings.Contains(chain[0], "handed to rpc") || chain[len(chain)-1] != "example.com/app.main" {
		t.Errorf("want Service.Serve explained by the reflection in rpc, got %q", chain)
	}
}

func TestAnalyze_MarksTheConsensusPath(t *testing.T) {
	dir := t.TempDir()
	gethMod := "module github.com/ethereum/go-ethereum\n\ngo 1.26\n"
	core := func(v, field string) string {
		return "package core\n\ntype Config struct{ A int" + field + " }\n\n" +
			"func Apply(c Config) int { return c.A + " + v + " }\n\n" +
			"func Traced() int { return " + v + " }\n\n" +
			"func Served() int { return " + v + " }\n\n" +
			"func Legacy() int { return " + v + " }\n\n" +
			"type Name struct{}\n\nfunc (Name) String() string { return \"" + v + "\" }\n\n" +
			"type Runner struct{}\n\nfunc (Runner) Run() int { return " + v + " }\n"
	}
	const chainPkg = `package chain

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core"
)

type Runner interface{ Run() int }

type Importer struct{}

func Process(r Runner, s fmt.Stringer) int {
	println(s.String())
	return core.Apply(core.Config{}) + r.Run() + core.Traced() + Importer{}.Legacy()
}

func (Importer) Legacy() int { return core.Legacy() }
`
	const appPkg = `package main

import (
	"example.com/app/chain"
	"github.com/ethereum/go-ethereum/core"
)

func main() {
	println(chain.Process(core.Runner{}, core.Name{}), core.Served())
}
`
	writeTree(t, dir, map[string]string{
		"geth-old/go.mod":       gethMod,
		"geth-old/core/core.go": core("1", ""),
		"geth-new/go.mod":       gethMod,
		"geth-new/core/core.go": core("2", "; B int"),
		"app-old/go.mod":        appMod("../geth-old"),
		"app-old/main.go":       appPkg,
		"app-old/chain/c.go":    chainPkg,
		"app-new/go.mod":        appMod("../geth-new"),
		"app-new/main.go":       appPkg,
		"app-new/chain/c.go":    chainPkg,
	})
	path := consensusPath{
		roots: []string{"chain func Process", "chain func Gone"},
		skips: []string{"chain func Importer.Legacy"},
		cuts:  []string{"core func Traced", "core func Gone"},
	}
	const runner = "core func Runner.Run"
	explain := func(key string) bool { return key == runner }
	base, err := analyze(filepath.Join(dir, "app-old"), []string{"."}, path, explain)
	if err != nil {
		t.Fatal(err)
	}
	head, err := analyze(filepath.Join(dir, "app-new"), []string{"."}, path, explain)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]bool{}
	for _, f := range compare(base, head) {
		got[f.key] = f.consensus
	}
	want := map[string]bool{
		"core func Apply":       true,
		"core type Config":      true,
		runner:                  true,
		"core func Traced":      false,
		"core func Served":      false,
		"core func Legacy":      false,
		"core func Name.String": false,
	}
	if !maps.Equal(got, want) {
		t.Errorf("got consensus path %v, want %v", got, want)
	}
	if want := []string{"root chain func Gone", "cut core func Gone"}; !slices.Equal(head.missing, want) {
		t.Errorf("got missing %v, want %v", head.missing, want)
	}
	chain := head.consensusWhy[runner]
	if len(chain) < 2 || !strings.Contains(chain[0], "called through example.com/app/chain.Runner") || chain[len(chain)-1] != "example.com/app/chain.Process" {
		t.Errorf("want Runner.Run on the consensus path through the interface call in Process, got %q", chain)
	}
}

func TestAnalyze_FollowsDeclarationsOutsideGoEthereum(t *testing.T) {
	dir := t.TempDir()
	gethMod := "module github.com/ethereum/go-ethereum\n\ngo 1.26\n"
	core := func(v, field string) string {
		return "package core\n\n" +
			"type Embedded struct{ A int" + field + " }\n\n" +
			"type Field struct{ A int" + field + " }\n\n" +
			"var Default = " + v + "\n\n" +
			"var InitOnly = " + v + "\n\n" +
			"var Unused = " + v + "\n\n" +
			"const Limit = " + v + "\n"
	}
	const flagsPkg = `package flags

import "github.com/ethereum/go-ethereum/core"

var Max = core.Default
`
	const appPkg = `package main

import (
	"example.com/app/flags"
	"github.com/ethereum/go-ethereum/core"
)

type wrapper struct{ core.Embedded }

type holder struct{ F core.Field }

const maxSize = core.Limit * 2

var _ = register(core.InitOnly)

func register(int) int { return 0 }

func use(any) {}

func main() {
	var w wrapper
	w.A = 1
	use(holder{})
	println(w.A, maxSize, flags.Max)
}
`
	writeTree(t, dir, map[string]string{
		"geth-old/go.mod":        gethMod,
		"geth-old/core/core.go":  core("1", ""),
		"geth-new/go.mod":        gethMod,
		"geth-new/core/core.go":  core("2", "; B int"),
		"app-old/go.mod":         appMod("../geth-old"),
		"app-old/main.go":        appPkg,
		"app-old/flags/flags.go": flagsPkg,
		"app-new/go.mod":         appMod("../geth-new"),
		"app-new/main.go":        appPkg,
		"app-new/flags/flags.go": flagsPkg,
	})
	const def = "core var Default"
	explain := func(key string) bool { return key == def }
	base, err := analyze(filepath.Join(dir, "app-old"), []string{"."}, consensusPath{}, explain)
	if err != nil {
		t.Fatal(err)
	}
	head, err := analyze(filepath.Join(dir, "app-new"), []string{"."}, consensusPath{}, explain)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]bool{}
	for _, f := range compare(base, head) {
		got[f.key] = f.direct
	}
	want := map[string]bool{
		"core type Embedded": true,
		"core type Field":    true,
		def:                  true,
		"core var InitOnly":  true,
		"core const Limit":   true,
	}
	if !maps.Equal(got, want) {
		t.Errorf("got findings (direct) %v, want %v", got, want)
	}
	if chain := head.why[def]; len(chain) == 0 || chain[0] != "example.com/app/flags var Max" {
		t.Errorf("want Default explained by the variable of flags, got %q", chain)
	}
}

func TestAnalyze_ComparesImportsInitializersAndRawStrings(t *testing.T) {
	dir := t.TempDir()
	gethMod := "module github.com/ethereum/go-ethereum\n\ngo 1.26\n"
	core := func(hasher, v, banner string) string {
		return "package core\n\nimport hasher \"" + hasher + "\"\n\n" +
			"func Hash(b []byte) []byte { h := hasher.New(); h.Write(b); return h.Sum(nil) }\n\n" +
			"func Banner() string { return `" + banner + "` }\n\n" +
			"func register(v int) int { return v }\n\n" +
			"var _ = register(" + v + ")\n\n" +
			"const Weight = " + v + "\n\n" +
			"var registered = register(Weight)\n\n" +
			"var pair, _ = register(" + v + "), 0\n\n" +
			"var computed = func() int { return " + v + " }()\n\n" +
			"var unread = " + v + "\n"
	}
	const appPkg = `package main

import "github.com/ethereum/go-ethereum/core"

func main() {
	println(len(core.Hash(nil)), core.Banner())
}
`
	writeTree(t, dir, map[string]string{
		"geth-old/go.mod":       gethMod,
		"geth-old/core/core.go": core("crypto/sha256", "1", "first\n\nlast"),
		"geth-new/go.mod":       gethMod,
		"geth-new/core/core.go": core("crypto/sha512", "2", "first\nlast"),
		"app-old/go.mod":        appMod("../geth-old"),
		"app-old/main.go":       appPkg,
		"app-new/go.mod":        appMod("../geth-new"),
		"app-new/main.go":       appPkg,
	})
	base, err := analyze(filepath.Join(dir, "app-old"), []string{"."}, consensusPath{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	head, err := analyze(filepath.Join(dir, "app-new"), []string{"."}, consensusPath{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	got := keysOf(compare(base, head))
	want := []string{"core const Weight", "core func Banner", "core func Hash", "core var _", "core var computed", "core var pair"}
	if !slices.Equal(got, want) {
		t.Errorf("got findings %v, want %v", got, want)
	}
	if text := head.symbols["core var _"].text; strings.Contains(text, "pair") {
		t.Errorf("want the blank name next to pair left to core var pair, got core var _ %q", text)
	}
}

func TestLimits_Admits(t *testing.T) {
	pkgs := map[string]*types.Package{}
	pkg := func(path string) *types.Package {
		if pkgs[path] == nil {
			pkgs[path] = types.NewPackage(path, "p")
		}
		return pkgs[path]
	}
	named := func(path, name string) types.Type {
		return types.NewNamed(types.NewTypeName(0, pkg(path), name, nil), types.NewStruct(nil, nil), nil)
	}
	l := &limits{mod: "example.com/app", modules: map[*types.Package]string{}}
	for path, mod := range map[string]string{
		"example.com/app/chain": "example.com/app",
		"example.com/app/node":  "example.com/app",
		"example.com/db":        "example.com/db",
		"example.com/db/table":  "example.com/db",
		"example.com/fmt":       "example.com/fmt",
		gethModule + "/core":    gethModule,
		gethModule + "/core/vm": gethModule,
	} {
		l.modules[pkg(path)] = mod
	}
	tests := map[string]struct {
		impl, iface types.Type
		want        bool
	}{
		"main module interface":        {named("example.com/db/table", "T"), named("example.com/app/chain", "I"), true},
		"go-ethereum interface":        {types.NewPointer(named("example.com/app/node", "T")), named(gethModule+"/core/vm", "I"), true},
		"interface of the same module": {named("example.com/db/table", "T"), named("example.com/db", "I"), true},
		"interface of another module":  {named("example.com/db/table", "T"), named("example.com/fmt", "I"), false},
		"standard library interface":   {named("example.com/app/chain", "T"), named("io", "I"), false},
		"unnamed interface":            {named("example.com/app/chain", "T"), types.NewInterfaceType(nil, nil), false},
	}
	for name, test := range tests {
		if got := l.admits(test.impl, test.iface); got != test.want {
			t.Errorf("%s: got %v, want %v", name, got, test.want)
		}
	}
}

func TestCompare_OrdersByKeyWithModulesLast(t *testing.T) {
	keys := []string{"rpc func B", "module example.com/a", "core func A", "module example.com/b", "p2p func C"}
	base := &side{symbols: map[string]*symbol{}, used: map[string]bool{}}
	head := &side{symbols: map[string]*symbol{}, used: map[string]bool{}}
	for _, k := range keys {
		base.symbols[k], head.symbols[k] = &symbol{text: "old"}, &symbol{text: "new"}
		base.used[k], head.used[k] = true, true
	}
	var got []string
	for _, f := range compare(base, head) {
		got = append(got, f.key)
	}
	want := []string{"core func A", "p2p func C", "rpc func B", "module example.com/a", "module example.com/b"}
	if strings.Join(got, ", ") != strings.Join(want, ", ") {
		t.Errorf("got %v, want %v", got, want)
	}
}
