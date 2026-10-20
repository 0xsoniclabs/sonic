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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/format"
	"go/scanner"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
	"golang.org/x/tools/go/types/typeutil"
)

// symbol is a package-level go-ethereum declaration, a non-Go file of a
// go-ethereum package, or a module providing reachable code. Two symbols are
// equal if their text is equal; for declarations the text is the imports they
// refer to and the gofmt-ed source, without comments and without blank lines
// outside raw string literals.
type symbol struct {
	text string
	loc  string
}

func (s *symbol) hash() string {
	if s == nil {
		return "-"
	}
	h := sha256.Sum256([]byte(s.text))
	return hex.EncodeToString(h[:6])
}

type declNode struct {
	node  ast.Node
	info  *types.Info
	sonic bool
}

// side is the go-ethereum surface of one Sonic revision. consensus holds the
// used symbols that are also used on the consensus path, modules the source
// directory of every dependency module, and missing the parts of the consensus
// path that the revision lacks.
type side struct {
	symbols      map[string]*symbol
	modules      map[string]string
	used         map[string]bool
	direct       map[string]bool
	why          map[string][]string
	consensus    map[string]bool
	consensusWhy map[string][]string
	missing      []string
	funcs        int
}

// builder collects a side. Unlike the side, it refers to the loaded program.
// decls holds the package-level declarations to follow, of go-ethereum and of
// the other modules, files the file keys of each go-ethereum package, and
// initVars the variables used once a package is initialized: all of a package
// outside go-ethereum, and those of go-ethereum whose initializer calls a
// function.
type builder struct {
	*side
	decls    map[string][]declNode
	files    map[string][]string
	initVars map[string][]string
	origin   map[string]origin
	queue    []string
	cut      func(key string) bool
}

func (s *side) summary() string {
	var decls, modules int
	for k := range s.used {
		switch {
		case isModule(k):
			modules++
		case !strings.Contains(k, " func "):
			decls++
		}
	}
	return fmt.Sprintf("%d functions, %d other declarations and files, %d dependency modules; %d on the consensus path",
		s.funcs, decls, modules, len(s.consensus))
}

// analyze loads the main packages matching patterns in dir and returns the
// go-ethereum symbols they use, and those used on the consensus path. For every
// used symbol accepted by explain, the result also records why it is used.
func analyze(dir string, patterns []string, path consensusPath, explain func(key string) bool) (*side, error) {
	cfg := &packages.Config{
		Mode: packages.LoadAllSyntax | packages.NeedModule | packages.NeedEmbedFiles,
		Dir:  dir,
		// Each revision resolves its dependencies by its own go.mod, whatever
		// the workspace or flags of the caller.
		Env: append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=readonly"),
	}
	initial, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(initial) > 0 {
		return nil, fmt.Errorf("loading %v in %s failed", patterns, dir)
	}
	prog, ssaPkgs := ssautil.AllPackages(initial, ssa.InstantiateGenerics)
	prog.Build()

	var roots []*ssa.Function
	for _, p := range ssaPkgs {
		if p != nil && p.Pkg.Name() == "main" {
			roots = append(roots, p.Func("main"), p.Func("init"))
		}
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("no main package among %v", patterns)
	}

	b := &builder{
		side: &side{
			symbols:      map[string]*symbol{},
			modules:      map[string]string{},
			used:         map[string]bool{},
			direct:       map[string]bool{},
			why:          map[string][]string{},
			consensusWhy: map[string][]string{},
		},
		decls:    map[string][]declNode{},
		files:    map[string][]string{},
		initVars: map[string][]string{},
		origin:   map[string]origin{},
	}
	byTypes := map[*types.Package]*packages.Package{}
	var mainModule string
	packages.Visit(initial, nil, func(p *packages.Package) {
		byTypes[p.Types] = p
		switch {
		case isGeth(p.PkgPath):
			b.index(p)
		case p.Module != nil:
			b.indexOther(p)
		}
		switch m := p.Module; {
		case m != nil && m.Main:
			mainModule = m.Path
		case m != nil && m.Replace != nil:
			b.modules[m.Path] = m.Replace.Dir
		case m != nil:
			b.modules[m.Path] = m.Dir
		}
	})

	r := reachable(prog, roots, nil)
	b.use(r, byTypes)
	for k := range b.used {
		if strings.Contains(k, " func ") {
			b.funcs++
		}
		if explain != nil && explain(k) {
			b.why[k] = b.chain(k, r)
		}
	}

	consensusRoots, skips, missing := path.resolve(prog, mainModule, b.symbols)
	lim := &limits{seed: &r.runtime, mod: mainModule, modules: map[*types.Package]string{}, skip: map[*ssa.Function]bool{}, cut: path.cut}
	for t, p := range byTypes {
		if p.Module != nil {
			lim.modules[t] = p.Module.Path
		}
	}
	for _, f := range skips {
		lim.skip[f] = true
	}
	c := &builder{
		side:     &side{symbols: b.symbols, used: map[string]bool{}, direct: map[string]bool{}},
		decls:    b.decls,
		files:    b.files,
		initVars: b.initVars,
		origin:   map[string]origin{},
		cut:      path.cut,
	}
	rc := reachable(prog, consensusRoots, lim)
	c.use(rc, byTypes)
	b.consensus, b.missing = c.used, missing
	for k := range c.used {
		if explain != nil && explain(k) {
			b.consensusWhy[k] = c.chain(k, rc)
		}
	}
	return b.side, nil
}

// use marks the go-ethereum symbols, files and modules used by the functions of
// r, and the declarations these refer to. The syntax of a package initializer
// is the initializers of the package's variables.
func (b *builder) use(r *reach, byTypes map[*types.Package]*packages.Package) {
	type top struct {
		decl    *ast.FuncDecl
		pkg     *packages.Package
		reached *ssa.Function
	}
	var tops []top
	seen := map[*ssa.Function]bool{}
	for _, fn := range r.order {
		t := fn
		for t.Parent() != nil {
			t = t.Parent()
		}
		if o := t.Origin(); o != nil {
			t = o
		}
		if seen[t] || t.Pkg == nil {
			continue
		}
		seen[t] = true
		if t.Synthetic == "package initializer" {
			for _, k := range b.initVars[t.Pkg.Pkg.Path()] {
				b.mark(k, false, origin{fn: fn})
			}
			continue
		}
		decl, ok := t.Syntax().(*ast.FuncDecl)
		if !ok {
			continue
		}
		if p := byTypes[t.Pkg.Pkg]; p != nil && p.Module != nil {
			tops = append(tops, top{decl, p, fn})
		}
	}
	gethPkgs := map[*packages.Package]bool{}
	for _, t := range tops {
		switch {
		case isGeth(t.pkg.PkgPath):
			b.mark(funcKey(t.pkg.PkgPath, t.decl), false, origin{fn: t.reached})
			if !gethPkgs[t.pkg] {
				gethPkgs[t.pkg] = true
				for _, k := range b.files[t.pkg.PkgPath] {
					b.mark(k, false, origin{fn: t.reached})
				}
			}
		case !t.pkg.Module.Main:
			key := "module " + t.pkg.Module.Path
			b.symbols[key] = &symbol{text: moduleVersion(t.pkg.Module)}
			b.mark(key, false, origin{fn: t.reached})
		}
	}
	for _, t := range tops {
		b.walk(t.decl, t.pkg.TypesInfo, t.pkg.Module.Main, origin{fn: t.reached})
	}
	for len(b.queue) > 0 {
		k := b.queue[0]
		b.queue = b.queue[1:]
		for _, d := range b.decls[k] {
			b.walk(d.node, d.info, d.sonic, origin{key: k})
		}
	}
}

// origin records what made a symbol used: a reachable function, or the
// declaration of another used symbol.
type origin struct {
	fn  *ssa.Function
	key string
}

func (b *builder) chain(key string, r *reach) []string {
	var out []string
	seen := map[string]bool{}
	for !seen[key] {
		seen[key] = true
		o, ok := b.origin[key]
		if !ok {
			break
		}
		if o.key != "" {
			out = append(out, shortName(o.key))
			key = o.key
			continue
		}
		for f := o.fn; f != nil; f = r.via[f] {
			out = append(out, shortName(f.String())+r.explain(f))
		}
		break
	}
	return out
}

func shortName(s string) string {
	s = strings.ReplaceAll(s, gethModule+"/", "")
	s = strings.ReplaceAll(s, gethModule, "go-ethereum")
	return strings.ReplaceAll(s, "github.com/0xsoniclabs/sonic/", "sonic/")
}

// reachable returns the functions reachable from roots. A function is reachable
// if a reachable function refers to it. A method is also reachable if reachable
// code calls it through an interface that a runtime type implements. Runtime
// types are the types converted to an interface in reachable code and all types
// reflection can derive from them. If a value is handed to a package that looks
// up methods by reflection, all exported methods of its type and of the types
// derived from it are reachable.
//
// If lim is not nil, it restricts the analysis to a part of the program.
func reachable(prog *ssa.Program, roots []*ssa.Function, lim *limits) *reach {
	r := &reacher{
		reach: &reach{
			funcs:   map[*ssa.Function]bool{},
			via:     map[*ssa.Function]*ssa.Function{},
			viaType: map[*ssa.Function]methodVia{},
		},
		prog:     prog,
		byKey:    map[string][]*calls{},
		byMethod: map[string][]types.Type{},
		sinks:    map[*types.Package]bool{},
		pending:  map[*types.Package][]handover{},
		lim:      lim,
	}
	hasher := typeutil.MakeHasher()
	r.runtime.SetHasher(hasher)
	r.reflected.SetHasher(hasher)
	r.calls.SetHasher(hasher)
	if lim != nil && lim.seed != nil {
		r.seedByMethod = map[string][]types.Type{}
		r.indexSeed(lim.seed)
	}
	for _, f := range roots {
		r.add(f)
	}
	var operands []*ssa.Value
	for i := 0; i < len(r.order); i++ {
		r.cur = r.order[i]
		for _, b := range r.cur.Blocks {
			for _, instr := range b.Instrs {
				switch instr := instr.(type) {
				case *ssa.MakeInterface:
					r.convert(instr)
				case ssa.CallInstruction:
					r.call(instr.Common())
				}
				operands = instr.Operands(operands[:0])
				for _, op := range operands {
					if fn, ok := (*op).(*ssa.Function); ok {
						r.add(fn)
					}
				}
			}
		}
	}
	return r.reach
}

// reach is the result of reachable. order lists the reachable functions in the
// order they were found, breadth first from the roots. via maps a function to
// the function that made it reachable, and viaType a method to the receiver
// type that made it reachable. runtime and reflected lead from a type back to
// the one it was derived from.
type reach struct {
	funcs     map[*ssa.Function]bool
	order     []*ssa.Function
	via       map[*ssa.Function]*ssa.Function
	viaType   map[*ssa.Function]methodVia
	runtime   typeutil.Map
	reflected typeutil.Map
}

// methodVia is the receiver type of a method reached through the interface
// iface, or by reflection if iface is nil.
type methodVia struct {
	recv  types.Type
	iface types.Type
}

// typeVia is the type a runtime type was derived from, nil for a type converted
// to an interface, and for a type reachable by reflection the package doing it.
type typeVia struct {
	from types.Type
	sink string
}

func (r *reach) explain(f *ssa.Function) string {
	v, ok := r.viaType[f]
	switch {
	case !ok:
		return ""
	case v.iface != nil:
		return fmt.Sprintf("  (called through %s on %s, converted to an interface)",
			shortName(v.iface.String()), strings.Join(typePath(&r.runtime, v.recv), " <- "))
	default:
		tv, _ := r.reflected.At(v.recv).(typeVia)
		return fmt.Sprintf("  (exported method of %s, handed to %s, which calls methods by reflection)",
			strings.Join(typePath(&r.reflected, v.recv), " <- "), shortName(tv.sink))
	}
}

func typePath(m *typeutil.Map, t types.Type) []string {
	var path []string
	for t != nil && len(path) < 50 {
		if _, ok := t.(*types.Tuple); !ok {
			path = append(path, shortName(t.String()))
		}
		v, _ := m.At(t).(typeVia)
		t = v.from
	}
	return path
}

type reacher struct {
	*reach
	prog *ssa.Program
	cur  *ssa.Function

	calls    typeutil.Map
	byKey    map[string][]*calls
	byMethod map[string][]types.Type

	sinks   map[*types.Package]bool
	pending map[*types.Package][]handover

	lim          *limits
	seedByMethod map[string][]types.Type
}

// calls records the methods of an interface called in reachable code, and the
// runtime types implementing it.
type calls struct {
	iface   types.Type
	methods map[string]*types.Func
	impls   []types.Type
}

// handover is a type converted to an interface in fn and handed to a package.
type handover struct {
	t  types.Type
	fn *ssa.Function
}

func (r *reacher) add(f *ssa.Function) bool {
	if f == nil || r.funcs[f] || r.cutFunc(f) {
		return false
	}
	r.funcs[f] = true
	if r.cur != nil {
		r.via[f] = r.cur
	}
	r.order = append(r.order, f)
	return true
}

func (r *reacher) convert(mi *ssa.MakeInterface) {
	t := mi.X.Type()
	r.addType(t, nil, "")
	for _, p := range handedTo(mi) {
		if p != nil && p.Path() == "reflect" {
			p = pkgOf(r.cur)
		}
		switch {
		case p == nil:
		case r.sinks[p]:
			r.addType(t, nil, p.Path())
		default:
			r.pending[p] = append(r.pending[p], handover{t, r.cur})
		}
	}
}

func (r *reacher) call(c *ssa.CallCommon) {
	if c.IsInvoke() {
		r.addCall(c.Value.Type(), c.Method)
	}
	if p := pkgOf(r.cur); p != nil && p.Path() != "reflect" && r.cur.Syntax() != nil && looksUpMethod(c) {
		r.addSink(p)
	}
}

func (r *reacher) addSink(p *types.Package) {
	if r.sinks[p] {
		return
	}
	r.sinks[p] = true
	cur := r.cur
	for _, h := range r.pending[p] {
		r.cur = h.fn
		r.addType(h.t, nil, p.Path())
	}
	r.cur = cur
	delete(r.pending, p)
}

// addType records that values of type t, derived from the type from, exist at
// run time. Reflection can obtain the values of their exported fields and
// elements, so their types are runtime types too; it cannot call methods of
// values read from unexported fields. If sink is not empty, reflection in that
// package can also call the exported methods of these values and of their method
// results, and create arguments for them.
func (r *reacher) addType(t, from types.Type, sink string) {
	seen := &r.runtime
	if sink != "" {
		seen = &r.reflected
	}
	if t == nil || hasTypeParam(t) || seen.At(t) != nil || r.cutType(t) {
		return
	}
	seen.Set(t, typeVia{from, sink})
	mset := r.prog.MethodSets.MethodSet(t)
	if sink != "" {
		r.addType(t, from, "")
		for i := range mset.Len() {
			sel := mset.At(i)
			sig := sel.Type().(*types.Signature)
			if sig.TypeParams().Len() > 0 || !sel.Obj().Exported() {
				continue
			}
			if f := r.prog.MethodValue(sel); r.add(f) {
				r.viaType[f] = methodVia{recv: t}
			}
			r.addType(sig.Params(), t, "")
			r.addType(sig.Results(), t, sink)
		}
	} else if !types.IsInterface(t) {
		r.addImpl(t, mset)
	}
	switch u := t.(type) {
	case *types.Alias:
		r.addType(types.Unalias(u), t, sink)
	case *types.Pointer:
		r.addType(u.Elem(), t, sink)
	case *types.Slice:
		r.addType(u.Elem(), t, sink)
	case *types.Array:
		r.addType(u.Elem(), t, sink)
	case *types.Chan:
		r.addType(u.Elem(), t, sink)
	case *types.Map:
		r.addType(u.Key(), t, sink)
		r.addType(u.Elem(), t, sink)
	case *types.Signature:
		if sink != "" {
			r.addType(u.Params(), t, "")
			r.addType(u.Results(), t, sink)
		}
	case *types.Named:
		r.addType(types.NewPointer(u), t, sink)
		r.addType(u.Underlying(), t, sink)
	case *types.Struct:
		for i := range u.NumFields() {
			if f := u.Field(i); f.Exported() {
				r.addType(f.Type(), t, sink)
			}
		}
	case *types.Tuple:
		for i := range u.Len() {
			r.addType(u.At(i).Type(), t, sink)
		}
	}
}

// addImpl matches the new concrete runtime type t against the interfaces called
// so far. Interfaces are indexed by their first method, so that only the types
// having it are checked.
func (r *reacher) addImpl(t types.Type, mset *types.MethodSet) {
	for i := range mset.Len() {
		id := mset.At(i).Obj().Id()
		r.byMethod[id] = append(r.byMethod[id], t)
		for _, c := range r.byKey[id] {
			if types.Implements(t, c.iface.Underlying().(*types.Interface)) {
				c.impls = append(c.impls, t)
				for _, m := range c.methods {
					r.addMethod(t, m, c.iface)
				}
			}
		}
	}
}

func (r *reacher) addCall(t types.Type, m *types.Func) {
	if hasTypeParam(t) {
		return
	}
	c, _ := r.calls.At(t).(*calls)
	if c == nil {
		iface := t.Underlying().(*types.Interface)
		c = &calls{iface: t, methods: map[string]*types.Func{}}
		r.calls.Set(t, c)
		key := iface.Method(0).Id()
		r.byKey[key] = append(r.byKey[key], c)
		for _, impl := range r.byMethod[key] {
			if types.Implements(impl, iface) {
				c.impls = append(c.impls, impl)
			}
		}
		for _, impl := range r.seedByMethod[key] {
			if r.lim.admits(impl, t) && types.Implements(impl, iface) {
				c.impls = append(c.impls, impl)
			}
		}
	}
	if c.methods[m.Id()] != nil {
		return
	}
	c.methods[m.Id()] = m
	for _, impl := range c.impls {
		r.addMethod(impl, m, t)
	}
}

func (r *reacher) addMethod(recv types.Type, m *types.Func, iface types.Type) {
	sel := r.prog.MethodSets.MethodSet(recv).Lookup(m.Pkg(), m.Name())
	if sel == nil {
		return
	}
	if f := r.prog.MethodValue(sel); r.add(f) {
		r.viaType[f] = methodVia{recv, iface}
	}
}

// handedTo returns the packages v is handed to: those of the functions it is
// passed to, also as a variadic argument, and those declaring the structs it is
// stored in.
func handedTo(v ssa.Value) []*types.Package {
	var out []*types.Package
	for _, ref := range *v.Referrers() {
		switch ref := ref.(type) {
		case ssa.CallInstruction:
			if f := ref.Common().StaticCallee(); f != nil {
				out = append(out, pkgOf(f))
			}
		case *ssa.Store:
			switch addr := ref.Addr.(type) {
			case *ssa.FieldAddr:
				if p, ok := addr.X.Type().Underlying().(*types.Pointer); ok {
					if n, ok := types.Unalias(p.Elem()).(*types.Named); ok {
						out = append(out, n.Obj().Pkg())
					}
				}
			case *ssa.IndexAddr:
				if alloc, ok := addr.X.(*ssa.Alloc); ok {
					for _, ref := range *alloc.Referrers() {
						if s, ok := ref.(*ssa.Slice); ok {
							out = append(out, handedTo(s)...)
						}
					}
				}
			}
		}
	}
	return out
}

func looksUpMethod(c *ssa.CallCommon) bool {
	var f *types.Func
	if c.IsInvoke() {
		f = c.Method
	} else if callee := c.StaticCallee(); callee != nil {
		f, _ = callee.Object().(*types.Func)
	}
	return f != nil && f.Pkg() != nil && f.Pkg().Path() == "reflect" && (f.Name() == "Method" || f.Name() == "MethodByName")
}

func pkgOf(f *ssa.Function) *types.Package {
	if o := f.Origin(); o != nil {
		f = o
	}
	if f.Pkg == nil {
		return nil
	}
	return f.Pkg.Pkg
}

func hasTypeParam(t types.Type) bool {
	switch t := t.(type) {
	case *types.TypeParam, *types.Union:
		return true
	case *types.Alias:
		return hasTypeParam(types.Unalias(t))
	case *types.Named:
		if t.TypeParams().Len() > 0 && t.TypeArgs().Len() == 0 {
			return true
		}
		for i := range t.TypeArgs().Len() {
			if hasTypeParam(t.TypeArgs().At(i)) {
				return true
			}
		}
	case *types.Pointer:
		return hasTypeParam(t.Elem())
	case *types.Slice:
		return hasTypeParam(t.Elem())
	case *types.Array:
		return hasTypeParam(t.Elem())
	case *types.Chan:
		return hasTypeParam(t.Elem())
	case *types.Map:
		return hasTypeParam(t.Key()) || hasTypeParam(t.Elem())
	case *types.Signature:
		return hasTypeParam(t.Params()) || hasTypeParam(t.Results())
	case *types.Tuple:
		for i := range t.Len() {
			if hasTypeParam(t.At(i).Type()) {
				return true
			}
		}
	case *types.Struct:
		for i := range t.NumFields() {
			if hasTypeParam(t.Field(i).Type()) {
				return true
			}
		}
	case *types.Interface:
		for i := range t.NumMethods() {
			if hasTypeParam(t.Method(i).Type()) {
				return true
			}
		}
	}
	return false
}

func (b *builder) index(p *packages.Package) {
	rel := relPkg(p.PkgPath)
	goFiles := map[string]bool{}
	for _, f := range p.GoFiles {
		goFiles[f] = true
	}
	for _, f := range p.Syntax {
		for _, d := range f.Decls {
			pos := p.Fset.Position(d.Pos())
			if !goFiles[pos.Filename] {
				continue // cgo-generated
			}
			loc := fmt.Sprintf("%s:%d", filepath.Join(pkgDir(p.PkgPath), filepath.Base(pos.Filename)), pos.Line)
			switch d := d.(type) {
			case *ast.FuncDecl:
				b.add(funcKey(p.PkgPath, d), d, loc, p.Fset, p.TypesInfo)
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						node := &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{spec}}
						b.add(rel+" type "+spec.Name.Name, node, loc, p.Fset, p.TypesInfo)
					case *ast.ValueSpec:
						named := slices.ContainsFunc(spec.Names, func(n *ast.Ident) bool { return n.Name != "_" })
						for _, n := range spec.Names {
							if n.Name == "_" && (d.Tok != token.VAR || named) {
								continue
							}
							if c, ok := p.TypesInfo.Defs[n].(*types.Const); ok {
								b.addText(rel+" const "+n.Name, fmt.Sprintf("const %s %s = %s", n.Name, c.Type(), c.Val().ExactString()), loc)
							} else if d.Tok == token.VAR {
								key := rel + " var " + n.Name
								node := &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{spec}}
								b.add(key, node, loc, p.Fset, p.TypesInfo)
								if callsFunction(spec.Values, p.TypesInfo) {
									b.initVars[p.PkgPath] = append(b.initVars[p.PkgPath], key)
								}
							}
						}
					}
				}
			}
		}
	}

	if len(p.GoFiles) == 0 {
		return
	}
	root := filepath.Dir(p.GoFiles[0])
	files := slices.Clone(p.EmbedFiles)
	if slices.ContainsFunc(p.OtherFiles, isNativeSource) {
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && isNativeSource(path) {
				files = append(files, path)
			}
			return nil
		})
	}
	for _, path := range files {
		text, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		name, _ := filepath.Rel(root, path)
		key := rel + " file " + filepath.ToSlash(name)
		b.addText(key, string(text), filepath.Join(pkgDir(p.PkgPath), name))
		b.files[p.PkgPath] = append(b.files[p.PkgPath], key)
	}
}

// indexOther records the package-level types, variables and constants of p,
// a package outside go-ethereum, so that the go-ethereum declarations they
// refer to are used if they are.
func (b *builder) indexOther(p *packages.Package) {
	if isStd(p.PkgPath) {
		return
	}
	for _, f := range p.Syntax {
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range g.Specs {
				var names []*ast.Ident
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					names = []*ast.Ident{spec.Name}
				case *ast.ValueSpec:
					names = spec.Names
				}
				for _, n := range names {
					key := fmt.Sprintf("%s %s %s", p.PkgPath, g.Tok, n.Name)
					b.decls[key] = append(b.decls[key], declNode{spec, p.TypesInfo, p.Module.Main})
					if g.Tok == token.VAR {
						b.initVars[p.PkgPath] = append(b.initVars[p.PkgPath], key)
					}
				}
			}
		}
	}
}

func isNativeSource(path string) bool {
	switch filepath.Ext(path) {
	case ".c", ".h", ".cc", ".cpp", ".hpp", ".s", ".S":
		return true
	}
	return false
}

// add records the text of a declaration. Functions are followed through the
// program, the other declarations through the references of their syntax.
func (b *builder) add(key string, node ast.Node, loc string, fset *token.FileSet, info *types.Info) {
	stripComments(node)
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		buf.Reset()
		fmt.Fprintf(&buf, "unprintable: %v", err)
	}
	b.addText(key, importLines(node, info)+dropBlankLines(buf.Bytes()), loc)
	if _, isFunc := node.(*ast.FuncDecl); !isFunc {
		b.decls[key] = append(b.decls[key], declNode{node, info, false})
	}
}

// importLines lists the imports that node refers to, so that its text changes
// with the package an import name stands for.
func importLines(node ast.Node, info *types.Info) string {
	var lines []string
	ast.Inspect(node, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			if pn, ok := info.Uses[id].(*types.PkgName); ok {
				lines = append(lines, fmt.Sprintf("import %s %q\n", pn.Name(), pn.Imported().Path()))
			}
		}
		return true
	})
	slices.Sort(lines)
	return strings.Join(slices.Compact(lines), "")
}

// dropBlankLines removes the blank lines of src, except those inside raw
// string literals, which are part of their value.
func dropBlankLines(src []byte) string {
	file := token.NewFileSet().AddFile("", -1, len(src))
	var s scanner.Scanner
	s.Init(file, src, nil, 0)
	inLiteral := map[int]bool{}
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.STRING && lit[0] == '`' {
			first := file.Line(pos)
			for i := range strings.Count(lit, "\n") {
				inLiteral[first+1+i] = true
			}
		}
	}
	var text strings.Builder
	for i, l := range strings.Split(string(src), "\n") {
		if inLiteral[i+1] || strings.TrimSpace(l) != "" {
			text.WriteString(l)
			text.WriteByte('\n')
		}
	}
	return text.String()
}

// callsFunction reports whether evaluating exprs calls a function, other than
// a builtin or a function literal that is not called.
func callsFunction(exprs []ast.Expr, info *types.Info) bool {
	found := false
	for _, e := range exprs {
		ast.Inspect(e, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncLit:
				return false
			case *ast.CallExpr:
				if tv := info.Types[n.Fun]; !tv.IsType() && !tv.IsBuiltin() {
					found = true
				}
			}
			return !found
		})
	}
	return found
}

func (b *builder) addText(key, text, loc string) {
	if sym, ok := b.symbols[key]; ok {
		sym.text += text
		return
	}
	b.symbols[key] = &symbol{text: text, loc: loc}
}

func stripComments(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncDecl:
			n.Doc = nil
		case *ast.GenDecl:
			n.Doc = nil
		case *ast.Field:
			n.Doc, n.Comment = nil, nil
		case *ast.TypeSpec:
			n.Doc, n.Comment = nil, nil
		case *ast.ValueSpec:
			n.Doc, n.Comment = nil, nil
		}
		return true
	})
}

func (b *builder) mark(key string, fromSonic bool, from origin) {
	_, isSymbol := b.symbols[key]
	if !isSymbol && b.decls[key] == nil || b.cut != nil && b.cut(key) {
		return
	}
	if isSymbol && fromSonic {
		b.direct[key] = true
	}
	if _, seen := b.origin[key]; seen {
		return
	}
	b.origin[key] = from
	if isSymbol {
		b.used[key] = true
	}
	b.queue = append(b.queue, key)
}

func (b *builder) walk(node ast.Node, info *types.Info, fromSonic bool, from origin) {
	ast.Inspect(node, func(n ast.Node) bool {
		if e, ok := n.(ast.Expr); ok {
			b.markType(info.TypeOf(e), fromSonic, from)
		}
		switch n := n.(type) {
		case *ast.Ident:
			if key, ok := objKey(info.Uses[n]); ok {
				b.mark(key, fromSonic, from)
			}
		case *ast.SelectorExpr:
			if sel := info.Selections[n]; sel != nil {
				b.markType(sel.Recv(), fromSonic, from)
			}
		}
		return true
	})
}

func (b *builder) markType(t types.Type, fromSonic bool, from origin) {
	switch t := t.(type) {
	case *types.Alias:
		b.markType(types.Unalias(t), fromSonic, from)
	case *types.Named:
		if key, ok := objKey(t.Origin().Obj()); ok {
			b.mark(key, fromSonic, from)
		}
		for i := range t.TypeArgs().Len() {
			b.markType(t.TypeArgs().At(i), fromSonic, from)
		}
	case *types.Pointer:
		b.markType(t.Elem(), fromSonic, from)
	case *types.Slice:
		b.markType(t.Elem(), fromSonic, from)
	case *types.Array:
		b.markType(t.Elem(), fromSonic, from)
	case *types.Chan:
		b.markType(t.Elem(), fromSonic, from)
	case *types.Map:
		b.markType(t.Key(), fromSonic, from)
		b.markType(t.Elem(), fromSonic, from)
	case *types.Signature:
		b.markType(t.Params(), fromSonic, from)
		b.markType(t.Results(), fromSonic, from)
	case *types.Tuple:
		for i := range t.Len() {
			b.markType(t.At(i).Type(), fromSonic, from)
		}
	case *types.Struct:
		for i := range t.NumFields() {
			b.markType(t.Field(i).Type(), fromSonic, from)
		}
	case *types.Interface:
		for i := range t.NumExplicitMethods() {
			b.markType(t.ExplicitMethod(i).Type(), fromSonic, from)
		}
		for i := range t.NumEmbeddeds() {
			b.markType(t.EmbeddedType(i), fromSonic, from)
		}
	}
}

// objKey is the key of a package-level declaration. Outside go-ethereum it
// starts with the full package path, whose first element has a dot, unlike the
// relative paths of go-ethereum packages.
func objKey(obj types.Object) (string, bool) {
	if obj == nil || obj.Pkg() == nil || isStd(obj.Pkg().Path()) {
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
		t := recv.Type()
		if p, ok := t.(*types.Pointer); ok {
			t = p.Elem()
		}
		named, ok := types.Unalias(t).(*types.Named)
		if !ok {
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

func isStd(pkgPath string) bool {
	first, _, _ := strings.Cut(pkgPath, "/")
	return !strings.Contains(first, ".")
}

func pkgDir(pkgPath string) string {
	return strings.TrimPrefix(strings.TrimPrefix(pkgPath, gethModule), "/")
}

func relPkg(pkgPath string) string {
	if d := pkgDir(pkgPath); d != "" {
		return d
	}
	return "go-ethereum"
}

func moduleVersion(m *packages.Module) string {
	if r := m.Replace; r != nil {
		return strings.TrimSpace(m.Path + " => " + r.Path + " " + r.Version)
	}
	return m.Path + " " + m.Version
}

const (
	statusChanged = "changed"
	statusAdded   = "added"
	statusRemoved = "removed"
)

type finding struct {
	key       string
	status    string
	base      *symbol
	head      *symbol
	direct    bool
	consensus bool
}

func (f finding) location() string {
	if f.head != nil {
		return f.head.loc
	}
	return f.base.loc
}

func (f finding) diff() textDiff {
	return diffSymbols(f.base, f.head)
}

func compare(base, head *side) []finding {
	keys := map[string]bool{}
	for k := range base.used {
		keys[k] = true
	}
	for k := range head.used {
		keys[k] = true
	}
	var out []finding
	for k := range keys {
		b, h := base.symbols[k], head.symbols[k]
		f := finding{key: k, base: b, head: h, direct: base.direct[k] || head.direct[k], consensus: base.consensus[k] || head.consensus[k]}
		switch {
		case b != nil && h != nil && b.text == h.text:
			continue
		case b != nil && h != nil:
			f.status = statusChanged
		case b != nil:
			f.status = statusRemoved
		case h != nil:
			f.status = statusAdded
		default:
			continue
		}
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool {
		if mi, mj := isModule(out[i].key), isModule(out[j].key); mi != mj {
			return mj
		}
		return out[i].key < out[j].key
	})
	return out
}
