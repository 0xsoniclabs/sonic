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
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

const gethModule = "github.com/ethereum/go-ethereum"

// Side is the go-ethereum surface of one version of the program. Symbols holds
// the text of every go-ethereum declaration, file and cgo preamble and of every
// dependency module by key, Used the keys the program uses, and Consensus those
// it uses on the consensus path. Upgrades lists the lowercase names of the
// program's upgrades, and Missing the parts of the config that the program
// lacks.
type Side struct {
	Geth      string
	Symbols   map[string]string
	Used      map[string]bool
	Consensus map[string]bool
	Upgrades  []string
	Missing   []string
}

// Analyze loads the main packages of c in dir and returns their side.
func Analyze(dir string, c Config) (*Side, error) {
	cfg := &packages.Config{
		Mode: packages.LoadAllSyntax | packages.NeedModule | packages.NeedEmbedFiles,
		Dir:  dir,
		// Each version resolves its dependencies by its own go.mod, whatever
		// the workspace or flags of the caller.
		Env: append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=readonly"),
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
	lim := &limits{ix: ix, skip: map[*ssa.Function]bool{}, cuts: c.Cuts}
	packages.Visit(initial, nil, func(p *packages.Package) {
		ix.pkgs[p.Types] = p
		switch m := p.Module; {
		case m == nil:
			return
		case isGeth(p.PkgPath):
			ix.addGeth(p)
			side.Geth = version(m)
		case m.Main:
			ix.addOther(p)
			lim.mod = m.Path
		default:
			ix.addOther(p)
			ix.symbols["module "+m.Path] = moduleText(m)
		}
		ix.modules[p.Types] = p.Module.Path
	})
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
// key with the dependency modules last.
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
		if ma, mb := isModule(a.Key), isModule(b.Key); ma != mb {
			if ma {
				return 1
			}
			return -1
		}
		return strings.Compare(a.Key, b.Key)
	})
	return out
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
	rel := obj.Pkg().Path()
	if isGeth(rel) {
		rel = relPkg(rel)
	}
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
	return relPkg(pkgPath) + " func " + name
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

func relPkg(pkgPath string) string {
	if rel, ok := strings.CutPrefix(pkgPath, gethModule+"/"); ok {
		return rel
	}
	return "go-ethereum"
}
