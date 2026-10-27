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

// Package analysis finds the go-ethereum symbols that a program uses and
// compares them between two versions of the program.
package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/types"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

const gethModule = "github.com/ethereum/go-ethereum"

// Side is the go-ethereum surface of one version of the program. Symbols holds
// the text of every declaration, file and cgo preamble of go-ethereum and of the
// modules compared like it, and of every other dependency module, by key, Used
// the keys the program uses, and Consensus those it uses on the consensus path.
// Geth is the version of go-ethereum and Modules those of the modules compared
// like it, and Packages lists the packages of these modules that are loaded, as
// in keys. Upgrades lists the lowercase names of the program's upgrades, and
// Missing the parts of the config that the program lacks.
type Side struct {
	Geth      string
	Modules   []string
	Packages  map[string]bool
	Symbols   map[string]string
	Used      map[string]bool
	Consensus map[string]bool
	Upgrades  []string
	Missing   []string
}

// Analyze loads the main packages of c in dir and returns their side.
func Analyze(dir string, c Config) (*Side, error) {
	cfg := loadConfig(dir, packages.LoadAllSyntax)
	if c.Platform != "" {
		if err := checkPlatform(cfg, c.Platform); err != nil {
			return nil, err
		}
	}
	initial, err := packages.Load(cfg, c.Mains...)
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(initial) > 0 {
		return nil, fmt.Errorf("loading %v in %s failed", c.Mains, dir)
	}
	prog, ssaPkgs := ssautil.AllPackages(initial, ssa.InstantiateGenerics)
	prog.Build()
	var roots []*ssa.Function
	for _, p := range ssaPkgs {
		if p != nil && p.Pkg.Name() == "main" {
			roots = append(roots, p.Func("main"), p.Func("init"))
		}
	}

	ix := newIndex()
	side := &Side{Symbols: ix.symbols}
	lim := &limits{ix: ix, skip: map[*ssa.Function]bool{}, cuts: c.Cuts, codecs: c.Codecs}
	modules := map[string]string{}
	packages.Visit(initial, nil, func(p *packages.Package) {
		ix.pkgs[p.Types] = p
		switch m := p.Module; {
		case m == nil:
			return
		case isGeth(p.PkgPath):
			ix.addCompared(p)
			side.Geth = version(m)
		case slices.Contains(c.Modules, m.Path):
			ix.addCompared(p)
			modules[m.Path] = version(m)
		case m.Main:
			ix.addOther(p)
			lim.mod = m.Path
		default:
			ix.addOther(p)
			ix.symbols["module "+m.Path] = moduleText(m)
		}
		ix.modules[p.Types] = p.Module.Path
	})
	side.Modules = slices.Sorted(maps.Values(modules))
	side.Packages = map[string]bool{}
	for p := range ix.compared {
		side.Packages[pkgKey(p)] = true
	}
	consensusRoots, skips, upgrades, missing := c.resolve(prog, lim.mod, ix.symbols)
	side.Upgrades, side.Missing = upgrades, missing

	r := reach(prog, roots, &limits{ix: ix})
	side.Used = ix.use(r.order, nil, nil)

	for _, f := range skips {
		lim.skip[f] = true
	}
	lim.seed = &r.runtime
	cr := reach(prog, consensusRoots, lim)
	side.Consensus = ix.use(cr.order, c.Cuts, cr.vars)
	return side, nil
}

// Complete adds to side the declarations, files and cgo preambles of the
// compared packages that other uses but side has not loaded, by loading them in
// dir, the program of side. Then side lacks a symbol of other only if its
// package lacks it, or cannot be loaded in dir.
func Complete(dir string, side, other *Side) error {
	var paths []string
	for key := range other.Used {
		pkg, _, _ := strings.Cut(key, " ")
		if path := importPath(pkg); !isModule(key) && !side.Packages[pkg] && !slices.Contains(paths, path) {
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return nil
	}
	pkgs, err := packages.Load(loadConfig(dir, packages.LoadSyntax), paths...)
	if err != nil {
		return err
	}
	ix := newIndex()
	ix.symbols = side.Symbols
	for _, p := range pkgs {
		if len(p.Errors) == 0 && p.Module != nil {
			ix.addCompared(p)
			side.Packages[pkgKey(p.PkgPath)] = true
		}
	}
	return nil
}

func loadConfig(dir string, mode packages.LoadMode) *packages.Config {
	return &packages.Config{
		Mode: mode | packages.NeedModule | packages.NeedEmbedFiles,
		Dir:  dir,
		// Each version resolves its dependencies by its own go.mod, whatever
		// the workspace or flags of the caller.
		Env: append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=readonly"),
	}
}

// checkPlatform returns an error unless the toolchain of cfg builds for
// platform with cgo.
func checkPlatform(cfg *packages.Config, platform string) error {
	cmd := exec.Command("go", "env", "GOOS", "GOARCH", "CGO_ENABLED")
	cmd.Dir, cmd.Env = cfg.Dir, cfg.Env
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("go env: %w", err)
	}
	env := strings.Fields(string(out))
	if len(env) != 3 {
		return fmt.Errorf("go env printed %q", out)
	}
	if got := env[0] + "/" + env[1]; got != platform || env[2] != "1" {
		return fmt.Errorf("the toolchain builds for %s with CGO_ENABLED=%s, want %s with cgo", got, env[2], platform)
	}
	return nil
}

// Missing returns the parts of the config that neither side has, so that one
// renamed between them can be listed under both names, and the upgrades type
// if the head lacks it.
func Missing(base, head *Side) []string {
	return slices.DeleteFunc(slices.Clone(head.Missing), func(m string) bool {
		return !strings.HasPrefix(m, "upgrades ") && !slices.Contains(base.Missing, m)
	})
}

// Finding is a used symbol whose text differs between two sides. Base and
// Head are hashes of its texts, "-" for a side that lacks it.
type Finding struct {
	Key       string
	Base      string
	Head      string
	Consensus bool
}

// Compare returns the findings of the symbols used by either side, ordered by
// key: those of go-ethereum first, the dependency modules last.
func Compare(base, head *Side) []Finding {
	keys := maps.Clone(base.Used)
	maps.Copy(keys, head.Used)
	var out []Finding
	for k := range keys {
		b, inBase := base.Symbols[k]
		h, inHead := head.Symbols[k]
		if inBase && inHead && b == h {
			continue
		}
		out = append(out, Finding{k, hash(b, inBase), hash(h, inHead), base.Consensus[k] || head.Consensus[k]})
	}
	slices.SortFunc(out, func(a, b Finding) int {
		if c := rank(a.Key) - rank(b.Key); c != 0 {
			return c
		}
		return strings.Compare(a.Key, b.Key)
	})
	return out
}

// rank orders the keys of go-ethereum before those of other modules, whose
// package path starts with an element containing a dot, and these before the
// dependency modules.
func rank(key string) int {
	pkg, _, _ := strings.Cut(key, " ")
	first, _, _ := strings.Cut(pkg, "/")
	switch {
	case isModule(key):
		return 2
	case strings.Contains(first, "."):
		return 1
	}
	return 0
}

func hash(text string, ok bool) string {
	if !ok {
		return "-"
	}
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:6])
}

func version(m *packages.Module) string {
	if r := m.Replace; r != nil {
		return strings.TrimSpace(r.Path + " " + r.Version)
	}
	return m.Path + " " + m.Version
}

func moduleText(m *packages.Module) string {
	if m.Replace != nil {
		return m.Path + " => " + version(m)
	}
	return version(m)
}

// objKey is the key of a package-level declaration outside the standard
// library. Outside go-ethereum it starts with the full package path, whose
// first element has a dot, unlike the relative paths of go-ethereum packages.
func (ix *index) objKey(obj types.Object) (string, bool) {
	if obj == nil || ix.modules[obj.Pkg()] == "" {
		return "", false
	}
	rel := pkgKey(obj.Pkg().Path())
	scope := obj.Pkg().Scope()
	switch obj := obj.(type) {
	case *types.Func:
		obj = obj.Origin()
		recv := obj.Signature().Recv()
		if recv == nil {
			return rel + " func " + obj.Name(), true
		}
		named := namedOf(recv.Type())
		if named == nil {
			return "", false
		}
		if types.IsInterface(named) {
			return rel + " type " + named.Origin().Obj().Name(), true
		}
		return rel + " func " + named.Origin().Obj().Name() + "." + obj.Name(), true
	case *types.TypeName:
		if obj.Parent() == scope {
			return rel + " type " + obj.Name(), true
		}
	case *types.Var:
		if !obj.IsField() && obj.Parent() == scope {
			return rel + " var " + obj.Name(), true
		}
	case *types.Const:
		if obj.Parent() == scope {
			return rel + " const " + obj.Name(), true
		}
	}
	return "", false
}

func funcKey(pkgPath string, d *ast.FuncDecl) string {
	name := d.Name.Name
	if d.Recv != nil && len(d.Recv.List) > 0 {
		name = recvName(d.Recv.List[0].Type) + "." + name
	}
	return pkgKey(pkgPath) + " func " + name
}

func recvName(e ast.Expr) string {
	for {
		switch x := e.(type) {
		case *ast.StarExpr:
			e = x.X
		case *ast.IndexExpr:
			e = x.X
		case *ast.IndexListExpr:
			e = x.X
		case *ast.ParenExpr:
			e = x.X
		case *ast.Ident:
			return x.Name
		default:
			return "?"
		}
	}
}

func isModule(key string) bool {
	return strings.HasPrefix(key, "module ")
}

func isGeth(pkgPath string) bool {
	return pkgPath == gethModule || strings.HasPrefix(pkgPath, gethModule+"/")
}

// importPath is the import path of pkg, the package of a key.
func importPath(pkg string) string {
	switch first, _, _ := strings.Cut(pkg, "/"); {
	case pkg == "go-ethereum":
		return gethModule
	case strings.Contains(first, "."):
		return pkg
	}
	return gethModule + "/" + pkg
}

// pkgKey is the package of a key: relative to go-ethereum for its packages,
// "go-ethereum" for its root, and the full path for other packages.
func pkgKey(pkgPath string) string {
	if !isGeth(pkgPath) {
		return pkgPath
	}
	if rel, ok := strings.CutPrefix(pkgPath, gethModule+"/"); ok {
		return rel
	}
	return "go-ethereum"
}
