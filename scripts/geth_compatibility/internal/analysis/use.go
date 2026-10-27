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
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

// use returns the symbols used by the functions in funcs: their declarations,
// the declarations these refer to, the files of the compared packages
// declaring any of them, and the modules providing the functions, except
// package initializers without variables to initialize, or the values of the
// variables and constants these declarations read. The syntax of a
// package initializer is the initializers of the package's variables.
// Symbols covered by cuts, and variables of a cut type, are not used. If vars
// is not nil, of the package variables outside vars, whose values funcs do not
// read, only the declared types are followed, such as an array length that
// len folds to a constant.
func (ix *index) use(funcs []*ssa.Function, cuts []string, vars map[types.Object]bool) map[string]bool {
	u := &user{index: ix, cuts: cuts, vars: vars, seen: map[string]bool{}}
	tops := map[*ssa.Function]bool{}
	for _, fn := range funcs {
		t := topLevel(fn)
		if t.Synthetic == "package initializer" {
			// fn is the initializer or a literal in the initializer of a
			// variable, whose declaration covers the literal's syntax.
			t = fn
		}
		if tops[t] || t.Pkg == nil {
			continue
		}
		tops[t] = true
		p := ix.pkgs[t.Pkg.Pkg]
		if p == nil || p.Module == nil {
			continue
		}
		initializer := t.Synthetic == "package initializer"
		if !initializer || len(p.TypesInfo.InitOrder) > 0 {
			u.markModule(p)
		}
		if initializer {
			for _, k := range ix.initVars[p.PkgPath] {
				u.mark(k, true)
			}
			continue
		}
		d, ok := t.Syntax().(*ast.FuncDecl)
		if !ok {
			continue
		}
		if ix.compared[p.PkgPath] {
			u.mark(funcKey(p.PkgPath, d), true)
		}
		u.walk(d, p.TypesInfo)
	}
	for len(u.queue) > 0 {
		q := u.queue[0]
		u.queue = u.queue[1:]
		for _, d := range ix.decls[q.key] {
			s, isValue := d.spec.(*ast.ValueSpec)
			switch {
			case !isValue:
				u.walk(d.spec, d.info)
			case q.follow:
				u.markModule(ix.pkgs[d.info.Defs[s.Names[0]].Pkg()])
				u.walk(s, d.info)
			case s.Type != nil:
				u.walk(s.Type, d.info)
			}
		}
	}

	used := map[string]bool{}
	for k := range u.seen {
		if _, ok := ix.symbols[k]; ok {
			used[k] = true
		}
	}
	for k := range used {
		pkg, _, _ := strings.Cut(k, " ")
		for _, f := range ix.files[pkg] {
			if !covered(cuts, f) {
				used[f] = true
			}
		}
	}
	return used
}

type user struct {
	*index
	cuts  []string
	vars  map[types.Object]bool
	seen  map[string]bool
	queue []queued
}

type queued struct {
	key    string
	follow bool
}

// mark uses the symbol key and the declarations it refers to, of a variable
// only those of its declared type unless follow.
func (u *user) mark(key string, follow bool) {
	_, isSymbol := u.symbols[key]
	if u.seen[key] || !isSymbol && u.decls[key] == nil || covered(u.cuts, key) {
		return
	}
	u.seen[key] = true
	u.queue = append(u.queue, queued{key, follow})
}

// markModule uses the module providing p, unless its declarations are compared,
// as those of go-ethereum, or it is the main module.
func (u *user) markModule(p *packages.Package) {
	if !u.compared[p.PkgPath] && !p.Module.Main {
		u.mark("module "+p.Module.Path, true)
	}
}

func (u *user) walk(node ast.Node, info *types.Info) {
	ast.Inspect(node, func(n ast.Node) bool {
		if e, ok := n.(ast.Expr); ok {
			u.markType(info.TypeOf(e))
		}
		switch n := n.(type) {
		case *ast.Ident:
			obj := info.Uses[n]
			if key, ok := u.uncutKey(u.cuts, obj); ok {
				_, isVar := obj.(*types.Var)
				u.mark(key, !isVar || u.vars == nil || u.vars[obj])
			}
		case *ast.SelectorExpr:
			if sel := info.Selections[n]; sel != nil {
				u.markType(sel.Recv())
			}
		}
		return true
	})
}

func (u *user) markType(t types.Type) {
	switch t := t.(type) {
	case *types.Alias:
		u.markType(types.Unalias(t))
	case *types.Named:
		if key, ok := u.objKey(t.Origin().Obj()); ok {
			u.mark(key, true)
		}
		for a := range t.TypeArgs().Types() {
			u.markType(a)
		}
	default:
		for _, c := range components(t) {
			u.markType(c)
		}
	}
}

// topLevel is the package-level function that declares f, the origin of an
// instantiated generic function.
func topLevel(f *ssa.Function) *ssa.Function {
	for f.Parent() != nil {
		f = f.Parent()
	}
	if o := f.Origin(); o != nil {
		f = o
	}
	return f
}
