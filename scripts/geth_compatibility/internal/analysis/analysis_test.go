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

// analyzeBoth analyzes the app against two versions of go-ethereum, made of
// the files of oldGeth and newGeth.
func analyzeBoth(t *testing.T, c Config, oldGeth, newGeth, app map[string]string) (*Side, *Side) {
	t.Helper()
	dir := t.TempDir()
	var sides []*Side
	for _, geth := range []map[string]string{oldGeth, newGeth} {
		c.Mains = []string{"."}
		side, err := Analyze(writeProgram(t, filepath.Join(dir, "v"+string(rune('0'+len(sides)))), geth, app), c)
		if err != nil {
			t.Fatal(err)
		}
		sides = append(sides, side)
	}
	return sides[0], sides[1]
}

// writeProgram writes the module of the app and the go-ethereum module made of
// the files of geth to root, and returns the directory of the app. The files
// of the app may replace its go.mod.
func writeProgram(t *testing.T, root string, geth, app map[string]string) string {
	t.Helper()
	files := map[string]string{
		"geth/go.mod": "module github.com/ethereum/go-ethereum\n\ngo 1.26\n",
		"app/go.mod":  "module example.com/app\n\ngo 1.26\n\nrequire github.com/ethereum/go-ethereum v0.0.0\n\nreplace github.com/ethereum/go-ethereum => ../geth\n",
	}
	for name, content := range geth {
		files["geth/"+name] = content
	}
	for name, content := range app {
		files["app/"+name] = content
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(root, "app")
}

func keysOf(fs []Finding) []string {
	var keys []string
	for _, f := range fs {
		keys = append(keys, f.Key)
	}
	return keys
}

func TestAnalyze_ReportsExactlyTheReachableDifferences(t *testing.T) {
	base, head := analyzeBoth(t, Config{},
		map[string]string{"core/core.go": oldCore, "core/data.txt": "old", "rpc/rpc.go": rpc},
		map[string]string{"core/core.go": newCore, "core/data.txt": "new", "rpc/rpc.go": rpc},
		map[string]string{"main.go": app})

	findings := Compare(base, head)
	want := []string{
		"core const Limit", "core file data.txt", "core func Apply", "core func Runner.Run",
		"core func Service.Serve", "core func added", "core func helper", "core type Config",
	}
	if got := keysOf(findings); !slices.Equal(got, want) {
		t.Errorf("got findings %v, want %v", got, want)
	}
	for _, f := range findings {
		if (f.Base == "-") != (f.Key == "core func added") || f.Head == "-" {
			t.Errorf("%s: got hashes %s and %s", f.Key, f.Base, f.Head)
		}
	}
	if base.Geth != "../geth" {
		t.Errorf("got go-ethereum %q, want the replacement", base.Geth)
	}
}

func TestAnalyze_ComparesTheNativeFilesTheBuildIncludes(t *testing.T) {
	const core = "package core\n\n/*\n#include \"./lib/src/sum.c\"\n#include \"ext.h\"\n*/\nimport \"C\"\n\nfunc Sum() int { return int(C.sum()) }\n"
	const sum = "#include \"value.h\"\n\nstatic int sum(void) { return VALUE; }\n"
	const appPkg = "package main\n\nimport \"github.com/ethereum/go-ethereum/core\"\n\nfunc main() { println(core.Sum()) }\n"
	base, head := analyzeBoth(t, Config{},
		map[string]string{"core/core.go": core, "core/ext.h": "", "core/lib/src/sum.c": sum, "core/lib/src/value.h": "#define VALUE 1\n", "core/lib/src/tests.c": "int old;\n"},
		map[string]string{"core/core.go": core, "core/ext.h": "", "core/lib/src/sum.c": sum, "core/lib/src/value.h": "#define VALUE 2\n", "core/lib/src/tests.c": "int new;\n"},
		map[string]string{"main.go": appPkg})

	want := []string{"core file lib/src/value.h"}
	if got := keysOf(Compare(base, head)); !slices.Equal(got, want) {
		t.Errorf("got findings %v, want %v", got, want)
	}
	if _, ok := head.Symbols["core file lib/src/sum.c"]; !ok {
		t.Error("want the included sum.c as a file of core")
	}
}

func TestAnalyze_ComparesCgoPreambles(t *testing.T) {
	core := func(v string) string {
		return "package core\n\n/*\n#cgo CFLAGS: -O2\nstatic int sum(void) { return " + v + "; }\n*/\nimport \"C\"\n\nfunc Sum() int { return int(C.sum()) }\n"
	}
	const appPkg = "package main\n\nimport \"github.com/ethereum/go-ethereum/core\"\n\nfunc main() { println(core.Sum()) }\n"
	base, head := analyzeBoth(t, Config{},
		map[string]string{"core/core.go": core("1")},
		map[string]string{"core/core.go": core("2")},
		map[string]string{"main.go": appPkg})

	want := []string{"core cgo core.go"}
	if got := keysOf(Compare(base, head)); !slices.Equal(got, want) {
		t.Errorf("got findings %v, want %v", got, want)
	}
}

func TestAnalyze_FollowsAliasesPromotedFieldsAndEmbeddedVariables(t *testing.T) {
	core := func(v string) string {
		return "package core\n\nimport \"github.com/ethereum/go-ethereum/data\"\n\n" +
			"type Inner struct{}\n\nfunc (Inner) Run() int { return " + v + " }\n\n" +
			"type Promoted struct{}\n\nfunc (Promoted) Run() int { return " + v + " }\n\ntype hidden struct{ P Promoted }\n\n" +
			"type Outer struct {\n\tIn Inner\n\thidden\n}\n\ntype Alias = Outer\n\n" +
			"func Text() string { return data.Text }\n"
	}
	const data = "package data\n\nimport _ \"embed\"\n\n//go:embed text.txt\nvar Text string\n"
	const appPkg = `package main

import "github.com/ethereum/go-ethereum/core"

type runner interface{ Run() int }

func use(v any) {
	if r, ok := v.(runner); ok {
		println(r.Run())
	}
}

func main() {
	var a core.Alias
	use(a)
	println(core.Text())
}
`
	base, head := analyzeBoth(t, Config{},
		map[string]string{"core/core.go": core("1"), "data/data.go": data, "data/text.txt": "old"},
		map[string]string{"core/core.go": core("2"), "data/data.go": data, "data/text.txt": "new"},
		map[string]string{"main.go": appPkg})

	want := []string{"core func Inner.Run", "core func Promoted.Run", "data file text.txt"}
	if got := keysOf(Compare(base, head)); !slices.Equal(got, want) {
		t.Errorf("got findings %v, want %v", got, want)
	}
}

func TestAnalyze_MarksTheConsensusPath(t *testing.T) {
	core := func(v, field string) string {
		return "package core\n\nimport \"fmt\"\n\ntype Config struct{ A int" + field + " }\n\n" +
			"func Apply(c Config) int { return c.A + " + v + " }\n\n" +
			"func Traced() int { return " + v + " }\n\n" +
			"func Served() int { return " + v + " }\n\n" +
			"func Legacy() int { return " + v + " }\n\n" +
			"type Name struct{}\n\nfunc (Name) String() string { return \"" + v + "\" }\n\n" +
			"type interpreter struct{}\n\nfunc (interpreter) run() int { return " + v + " }\n\nfunc Step() int { return interpreter{}.run() }\n\n" +
			"type Runner struct{}\n\nfunc (Runner) Run() int { return " + v + " }\n\n" +
			"var Gas = gas()\n\nfunc gas() int { return base() + step + Traced() }\n\nfunc base() int { return " + v + " }\n\nconst step = " + v + "\n\n" +
			"var Table = func() int { return rows() }()\n\nfunc rows() int { return width() }\n\nfunc width() int { return " + v + " }\n\n" +
			"var Startup = startup()\n\nfunc startup() int { return " + v + " }\n\n" +
			"type Jumps [1]func() int\n\ntype JumpsRef = *Jumps\n\nvar Ops JumpsRef = newOps(" + v + ")\n\nfunc newOps(int) JumpsRef { return &Jumps{op} }\n\nfunc op() int { return " + v + " }\n\n" +
			"var Written = written()\n\nfunc written() int { return deeper() + " + v + " }\n\nfunc deeper() int { return " + v + " }\n\n" +
			"type Doer interface{ Do() int }\n\ntype Mul struct{}\n\nfunc (Mul) Do() int { return " + v + " }\n\nvar doer Doer = Mul{}\n\nvar Done = doer.Do()\n\n" +
			"type Label struct{}\n\nfunc (Label) String() string { return \"" + v + "\" }\n\nvar label fmt.Stringer = Label{}\n\nfunc Labelled() string { return label.String() }\n\n" +
			"var Addrs []int\n\nfunc init() { Addrs = append(Addrs, fill()) }\n\nfunc fill() int { return " + v + " }\n\n" +
			"var noted int\n\nfunc init() { noted = note() }\n\nfunc note() int { return " + v + " }\n\n" +
			"var Seeded int\n\nfunc init() { seed() }\n\nfunc seed() { Seeded = seeded() }\n\nfunc seeded() int { return " + v + " }\n\n" +
			"var Codes = map[int]int{}\n\nfunc init() { Codes[0] = code() }\n\nfunc code() int { return " + v + " }\n\n" +
			"type Box struct{ X int }\n\nvar Boxed = &Box{}\n\nfunc init() { Boxed.X = box() }\n\nfunc box() int { return " + v + " }\n\n" +
			"var Slots = make([]int, 1)\n\nfunc init() { fillSlots(Slots) }\n\nfunc fillSlots(s []int) { s[0] = slot() }\n\nfunc slot() int { return " + v + " }\n\n" +
			"var Sizes = map[int]int{}\n\nvar tally int\n\nfunc init() { tally = Sizes[0] + sized() }\n\nfunc sized() int { return " + v + " }\n\n" +
			"const Size = " + v + "\n\nvar Cells [Size]int\n"
	}
	const chainPkg = `package chain

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core"
)

type Runner interface{ Run() int }

type Importer struct{}

type Upgrades struct{ Brio, Canto bool }

func Process(r Runner, s fmt.Stringer) int {
	println(s.String(), core.Ops, core.Labelled())
	core.Written = 3
	return core.Apply(core.Config{}) + r.Run() + core.Traced() + core.Step() + Importer{}.Legacy() + core.Gas + core.Table + legacy + core.Done + len(core.Addrs) +
		core.Seeded + core.Codes[0] + core.Boxed.X + core.Slots[0] + core.Sizes[0] + len(core.Cells)
}

var legacy = Importer{}.Legacy()

func (Importer) Legacy() int { return core.Legacy() }
`
	const appPkg = `package main

import (
	"example.com/app/chain"
	"github.com/ethereum/go-ethereum/core"
)

func main() {
	println(chain.Process(core.Runner{}, core.Name{}), core.Served(), core.Startup)
}
`
	c := Config{
		Roots:    []string{"chain func Process"},
		Skips:    []string{"chain func Importer.Legacy"},
		Cuts:     []string{"core func Traced", "core type interpreter", "core type Jumps"},
		Upgrades: "chain type Upgrades",
	}
	base, head := analyzeBoth(t, c,
		map[string]string{"core/core.go": core("1", "")},
		map[string]string{"core/core.go": core("2", "; B int")},
		map[string]string{"main.go": appPkg, "chain/c.go": chainPkg})

	got := map[string]bool{}
	for _, f := range Compare(base, head) {
		got[f.Key] = f.Consensus
	}
	want := map[string]bool{
		"core func Apply":           true,
		"core type Config":          true,
		"core func Runner.Run":      true,
		"core func Traced":          false,
		"core func interpreter.run": false,
		"core func Served":          false,
		"core func Legacy":          false,
		"core func Name.String":     false,
		"core func base":            true,
		"core const step":           true,
		"core func width":           true,
		"core func startup":         false,
		"core var Ops":              false,
		"core func op":              false,
		"core func written":         false,
		"core func deeper":          false,
		"core func Mul.Do":          true,
		"core func Label.String":    true,
		"core func fill":            true,
		"core func note":            false,
		"core func seeded":          true,
		"core func code":            true,
		"core func box":             true,
		"core func slot":            true,
		"core func sized":           false,
		"core const Size":           true,
	}
	if !maps.Equal(got, want) {
		t.Errorf("got consensus path %v, want %v", got, want)
	}
	if want := []string{"brio", "canto"}; !slices.Equal(head.Upgrades, want) {
		t.Errorf("got upgrades %v, want %v", head.Upgrades, want)
	}
}

func TestAnalyze_ListsThePartsOfTheConfigTheProgramLacks(t *testing.T) {
	const chainPkg = "package chain\n\nimport \"github.com/ethereum/go-ethereum/core\"\n\nfunc Process() int { return core.Run() }\n"
	const appPkg = "package main\n\nimport \"example.com/app/chain\"\n\nfunc main() { println(chain.Process()) }\n"
	dir := writeProgram(t, t.TempDir(),
		map[string]string{"core/core.go": "package core\n\nfunc Run() int { return 1 }\n"},
		map[string]string{"main.go": appPkg, "chain/chain.go": chainPkg})
	c := Config{
		Mains:    []string{"."},
		Roots:    []string{"chain func Process", "chain func Gone"},
		Skips:    []string{"chain func Gone"},
		Cuts:     []string{"core func Run", "core func Gone"},
		Upgrades: "chain type Gone",
	}
	side, err := Analyze(dir, c)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"root chain func Gone", "skip chain func Gone", "cut core func Gone", "upgrades chain type Gone"}
	if !slices.Equal(side.Missing, want) {
		t.Errorf("got missing %v, want %v", side.Missing, want)
	}
}

func TestMissing_ListsWhatNeitherSideHasAndTheUpgradesTheHeadLacks(t *testing.T) {
	base := &Side{Missing: []string{"cut core func New", "cut core func Gone"}}
	head := &Side{Missing: []string{"cut core func Old", "cut core func Gone", "upgrades chain type Upgrades"}}
	want := []string{"cut core func Gone", "upgrades chain type Upgrades"}
	if got := Missing(base, head); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAnalyze_FollowsDeclarationsOutsideGoEthereum(t *testing.T) {
	core := func(v, field string) string {
		return "package core\n\n" +
			"type Embedded struct{ A int" + field + " }\n\n" +
			"type Field struct{ A int" + field + " }\n\n" +
			"var Default = " + v + "\n\n" +
			"var InitOnly = " + v + "\n\n" +
			"var Unused = " + v + "\n\n" +
			"const Limit = " + v + "\n\n" +
			"const Step = " + v + "\n"
	}
	const flagsPkg = `package flags

import "github.com/ethereum/go-ethereum/core"

var Max = core.Default

type Holder struct{ F core.Field }
`
	const appMod = `module example.com/app

go 1.26

require (
	flags v0.0.0
	github.com/ethereum/go-ethereum v0.0.0
	shapes v0.0.0
)

replace (
	flags => ../flags
	github.com/ethereum/go-ethereum => ../geth
	shapes => ../shapes
)
`
	const appPkg = `package main

import (
	"flags"
	"shapes"

	"github.com/ethereum/go-ethereum/core"
)

type wrapper struct{ core.Embedded }

const maxSize = core.Limit * 2

const (
	first = core.Step + iota
	second
)

var _ = register(core.InitOnly)

func register(int) int { return 0 }

func use(any) {}

func main() {
	var w wrapper
	w.A = 1
	use(flags.Holder{})
	use(shapes.Point{})
	println(w.A, maxSize, second, flags.Max)
}
`
	base, head := analyzeBoth(t, Config{},
		map[string]string{"core/core.go": core("1", "")},
		map[string]string{"core/core.go": core("2", "; B int")},
		map[string]string{
			"go.mod":              appMod,
			"main.go":             appPkg,
			"../flags/go.mod":     "module flags\n\ngo 1.26\n\nrequire github.com/ethereum/go-ethereum v0.0.0\n",
			"../flags/flags.go":   flagsPkg,
			"../shapes/go.mod":    "module shapes\n\ngo 1.26\n",
			"../shapes/shapes.go": "package shapes\n\ntype Point struct{ X int }\n",
		})

	want := []string{"core const Limit", "core const Step", "core type Embedded", "core type Field", "core var Default", "core var InitOnly"}
	if got := keysOf(Compare(base, head)); !slices.Equal(got, want) {
		t.Errorf("got findings %v, want %v", got, want)
	}
	if !head.Used["module flags"] {
		t.Error("want module flags used, whose initializer sets the variable main reads")
	}
	if head.Used["module shapes"] {
		t.Error("want module shapes unused, as it has no code to run")
	}
}

func TestAnalyze_ComparesImportsInitializersAndRawStrings(t *testing.T) {
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
	base, head := analyzeBoth(t, Config{},
		map[string]string{"core/core.go": core("crypto/sha256", "1", "first\n\nlast")},
		map[string]string{"core/core.go": core("crypto/sha512", "2", "first\nlast")},
		map[string]string{"main.go": appPkg})

	want := []string{"core const Weight", "core func Banner", "core func Hash", "core var _", "core var computed", "core var pair"}
	if got := keysOf(Compare(base, head)); !slices.Equal(got, want) {
		t.Errorf("got findings %v, want %v", got, want)
	}
	if text := head.Symbols["core var _"]; strings.Contains(text, "pair") {
		t.Errorf("want the blank name next to pair left to core var pair, got core var _ %q", text)
	}
}

func TestAnalyze_ComparesEmbedDirectives(t *testing.T) {
	geth := func(first, second string) map[string]string {
		core := "package core\n\nimport _ \"embed\"\n\n" +
			"//go:embed " + first + "\nvar A string\n\n" +
			"var (\n\t//go:embed " + second + "\n\tB string\n\n\t// C is next to " + second + ".\n\t//go:embed c.txt\n\tC string\n)\n\n" +
			"func Read() string { return A + B + C }\n"
		return map[string]string{"core/core.go": core, "core/a.txt": "a", "core/b.txt": "b", "core/c.txt": "c"}
	}
	const appPkg = "package main\n\nimport \"github.com/ethereum/go-ethereum/core\"\n\nfunc main() { println(core.Read()) }\n"
	base, head := analyzeBoth(t, Config{}, geth("a.txt", "b.txt"), geth("b.txt", "a.txt"), map[string]string{"main.go": appPkg})

	want := []string{"core var A", "core var B"}
	if got := keysOf(Compare(base, head)); !slices.Equal(got, want) {
		t.Errorf("got findings %v, want %v", got, want)
	}
}

func TestAnalyze_MarksTheModulesProvidingReadValues(t *testing.T) {
	const chainPkg = "package chain\n\nimport (\n\t\"limits\"\n\t\"shapes\"\n\t\"slots\"\n\t\"steps\"\n)\n\n" +
		"func Process() int {\n\tvar p shapes.Point\n\tn := limits.Limit + steps.Step + p.X\n\tfor _, s := range slots.Slots {\n\t\tn += s\n\t}\n\treturn n\n}\n"
	const appPkg = "package main\n\nimport \"example.com/app/chain\"\n\nfunc main() { println(chain.Process()) }\n"
	dir := t.TempDir()
	var sides []*Side
	for _, v := range []string{"1", "2"} {
		app := writeProgram(t, filepath.Join(dir, "v"+v), nil, map[string]string{
			"go.mod": "module example.com/app\n\ngo 1.26\n\nrequire (\n\tgithub.com/ethereum/go-ethereum v0.0.0\n\tlimits v0.0.0\n\tshapes v0.0.0\n\tslots v0.0.0\n\tsteps v0.0.0\n)\n\n" +
				"replace (\n\tgithub.com/ethereum/go-ethereum => ../geth\n\tlimits => ../limits" + v + "\n\tshapes => ../shapes" + v + "\n\tslots => ../slots" + v + "\n\tsteps => ../steps" + v + "\n)\n",
			"main.go":                    appPkg,
			"chain/chain.go":             chainPkg,
			"../limits" + v + "/go.mod":  "module limits\n\ngo 1.26\n",
			"../limits" + v + "/l.go":    "package limits\n\nvar Limit = 1" + v + "\n",
			"../shapes" + v + "/go.mod":  "module shapes\n\ngo 1.26\n",
			"../shapes" + v + "/s.go":    "package shapes\n\ntype Point struct{ X int }\n",
			"../slots" + v + "/go.mod":   "module slots\n\ngo 1.26\n",
			"../slots" + v + "/slots.go": "package slots\n\nvar Slots [" + v + "]int\n",
			"../steps" + v + "/go.mod":   "module steps\n\ngo 1.26\n",
			"../steps" + v + "/steps.go": "package steps\n\nconst Step = 1" + v + "\n",
		})
		side, err := Analyze(app, Config{Mains: []string{"."}, Roots: []string{"chain func Process"}})
		if err != nil {
			t.Fatal(err)
		}
		sides = append(sides, side)
	}

	got := map[string]bool{}
	for _, f := range Compare(sides[0], sides[1]) {
		got[f.Key] = f.Consensus
	}
	if want := map[string]bool{"module limits": true, "module slots": true, "module steps": true}; !maps.Equal(got, want) {
		t.Errorf("got findings %v, want %v", got, want)
	}
}

func TestCompare_OrdersByKeyWithModulesLast(t *testing.T) {
	keys := []string{"rpc func B", "module example.com/a", "core func A", "module example.com/b", "p2p func C"}
	base := &Side{Symbols: map[string]string{}, Used: map[string]bool{}}
	head := &Side{Symbols: map[string]string{}, Used: map[string]bool{}}
	for _, k := range keys {
		base.Symbols[k], head.Symbols[k] = "old", "new"
		base.Used[k], head.Used[k] = true, true
	}
	want := []string{"core func A", "p2p func C", "rpc func B", "module example.com/a", "module example.com/b"}
	if got := keysOf(Compare(base, head)); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
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
	l := &limits{ix: newIndex(), mod: "example.com/app"}
	for path, mod := range map[string]string{
		"example.com/app/chain": "example.com/app",
		"example.com/app/node":  "example.com/app",
		"example.com/db":        "example.com/db",
		"example.com/db/table":  "example.com/db",
		"example.com/fmt":       "example.com/fmt",
		gethModule + "/core":    gethModule,
		gethModule + "/core/vm": gethModule,
	} {
		l.ix.modules[pkg(path)] = mod
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
