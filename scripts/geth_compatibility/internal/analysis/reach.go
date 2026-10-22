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
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/types/typeutil"
	"golang.org/x/tools/refactor/satisfy"
)

// reacher finds the functions reachable from a set of roots. A function is
// reachable if a reachable function refers to it, directly or through the
// package variables it reads: their initializers and the init functions that
// write them, also through the maps, slices and pointers they hold, or in the
// functions these init functions call. A method is also reachable if reachable
// code calls it through an interface that a runtime type implements. Runtime
// types are the types converted to an interface in reachable code, including
// the initializers of these variables, and all types reflection can derive
// from them. If a value is handed to a package that looks up methods by
// reflection, all exported methods of its type and of the types derived from
// it are reachable.
//
// x/tools' rta package does the same without the reflection, but panics on
// generic methods such as math/rand/v2.Rand.N.
type reacher struct {
	prog    *ssa.Program
	lim     *limits
	cur     *ssa.Function
	funcs   map[*ssa.Function]bool
	order   []*ssa.Function
	vars    map[types.Object]bool
	writers map[types.Object][]*ssa.Function
	finder  satisfy.Finder

	runtime   typeutil.Map
	reflected typeutil.Map

	// calls maps an interface called in reachable code to its *calls, and
	// byKey indexes them by their first method. byMethod indexes the runtime
	// types by their methods, and seed those of lim.seed.
	calls    typeutil.Map
	byKey    map[string][]*calls
	byMethod map[string][]types.Type
	seed     map[string][]types.Type

	// sinks are the packages looking up methods by reflection, and pending
	// holds the types handed to the other packages until they turn out to be.
	sinks   map[*types.Package]bool
	pending map[*types.Package][]types.Type
}

type calls struct {
	iface   types.Type
	methods map[string]*types.Func
	impls   []types.Type
}

func reach(prog *ssa.Program, roots []*ssa.Function, lim *limits) *reacher {
	r := &reacher{
		prog:     prog,
		lim:      lim,
		funcs:    map[*ssa.Function]bool{},
		vars:     map[types.Object]bool{},
		writers:  initWriters(prog, lim.ix),
		byKey:    map[string][]*calls{},
		byMethod: map[string][]types.Type{},
		seed:     map[string][]types.Type{},
		sinks:    map[*types.Package]bool{},
		pending:  map[*types.Package][]types.Type{},
	}
	hasher := typeutil.MakeHasher()
	r.runtime.SetHasher(hasher)
	r.reflected.SetHasher(hasher)
	r.calls.SetHasher(hasher)
	if lim.seed != nil {
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
					switch v := (*op).(type) {
					case *ssa.Function:
						r.add(v)
					case *ssa.Global:
						// A store replaces the initial value.
						if st, ok := instr.(*ssa.Store); !ok || op != &st.Addr {
							r.addVar(v.Object())
						}
					}
				}
			}
		}
	}
	return r
}

func (r *reacher) add(f *ssa.Function) {
	if f != nil && !r.funcs[f] && !r.lim.cutFunc(f) {
		r.funcs[f] = true
		r.order = append(r.order, f)
	}
}

func (r *reacher) addVar(v types.Object) {
	if r.vars[v] {
		return
	}
	r.vars[v] = true
	key, ok := r.lim.ix.uncutKey(r.lim.cuts, v)
	if !ok {
		return
	}
	for _, f := range r.writers[v] {
		r.add(f)
	}
	for _, d := range r.lim.ix.decls[key] {
		r.addConversions(d)
		ast.Inspect(d.node, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncLit:
				for _, f := range r.prog.Package(v.Pkg()).Func("init").AnonFuncs {
					if f.Syntax() == n {
						r.add(f)
					}
				}
				return false
			case *ast.Ident:
				switch obj := d.info.Uses[n].(type) {
				case *types.Func:
					if recv := obj.Signature().Recv(); recv != nil && types.IsInterface(recv.Type()) {
						r.addCall(recv.Type(), obj)
					} else {
						r.add(r.prog.FuncValue(obj.Origin()))
					}
				case *types.Var:
					r.addVar(obj)
				}
			}
			return true
		})
	}
}

// addConversions adds the types that the declaration d of a package variable
// converts to interfaces.
func (r *reacher) addConversions(d decl) {
	gen, ok := d.node.(*ast.GenDecl)
	if !ok {
		gen = &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{d.node.(*ast.ValueSpec)}}
	}
	r.finder.Result = nil
	r.finder.Find(d.info, []*ast.File{{Decls: []ast.Decl{gen}}})
	for c := range r.finder.Result {
		if !types.IsInterface(c.RHS) {
			r.addType(c.RHS, false)
		}
	}
}

// initWriters maps the package variables to the init functions of prog that
// write them, themselves or in the functions they call outside the standard
// library, function literals included.
func initWriters(prog *ssa.Program, ix *index) map[types.Object][]*ssa.Function {
	writers := map[types.Object][]*ssa.Function{}
	for _, p := range prog.AllPackages() {
		if ix.modules[p.Pkg] == "" {
			continue
		}
		for name, m := range p.Members {
			init, ok := m.(*ssa.Function)
			if !ok || !strings.HasPrefix(name, "init#") {
				continue
			}
			vars := map[types.Object]bool{}
			seen := map[*ssa.Function]bool{init: true}
			for queue := []*ssa.Function{init}; len(queue) > 0; queue = queue[1:] {
				written, callees := ix.written(queue[0])
				for _, v := range written {
					vars[v] = true
				}
				for _, f := range callees {
					if !seen[f] && !ix.isStd(f) {
						seen[f] = true
						queue = append(queue, f)
					}
				}
			}
			for v := range vars {
				writers[v] = append(writers[v], init)
			}
		}
	}
	return writers
}

// written returns the package variables that f and its function literals
// write, by referring to them other than by loading their values or by writing
// through the references they load from them, and the functions f calls.
func (ix *index) written(f *ssa.Function) (vars []types.Object, callees []*ssa.Function) {
	for _, b := range f.Blocks {
		for _, instr := range b.Instrs {
			if c, ok := instr.(ssa.CallInstruction); ok {
				if g := c.Common().StaticCallee(); g != nil {
					callees = append(callees, g)
				}
			}
			for _, op := range instr.Operands(nil) {
				if g, ok := (*op).(*ssa.Global); ok {
					if u, ok := instr.(*ssa.UnOp); !ok || u.Op != token.MUL || ix.writesThrough(u, map[ssa.Value]bool{}) {
						vars = append(vars, g.Object())
					}
				}
			}
		}
	}
	for _, a := range f.AnonFuncs {
		w, _ := ix.written(a)
		vars = append(vars, w...)
	}
	return vars, callees
}

// writesThrough reports whether the map, slice or pointer v is written
// through: stored to, updated, cleared or copied into, also by way of the
// values derived from it, or by a function outside the standard library that
// it is handed to.
func (ix *index) writesThrough(v ssa.Value, seen map[ssa.Value]bool) bool {
	if seen[v] || v.Referrers() == nil {
		return false
	}
	seen[v] = true
	for _, ref := range *v.Referrers() {
		switch ref := ref.(type) {
		case *ssa.Store:
			if ref.Addr == v {
				return true
			}
		case *ssa.MapUpdate:
			if ref.Map == v {
				return true
			}
		case ssa.CallInstruction:
			c := ref.Common()
			if b, ok := c.Value.(*ssa.Builtin); ok {
				if slices.Contains([]string{"clear", "copy", "delete"}, b.Name()) && c.Args[0] == v {
					return true
				}
			} else if f := c.StaticCallee(); f != nil && f.Blocks != nil && !ix.isStd(f) {
				for i, a := range c.Args {
					if a == v && ix.writesThrough(f.Params[i], seen) {
						return true
					}
				}
			}
		case *ssa.UnOp:
			if ref.Op == token.MUL && ix.writesThrough(ref, seen) {
				return true
			}
		case *ssa.FieldAddr, *ssa.IndexAddr, *ssa.Field, *ssa.Index, *ssa.Lookup, *ssa.Slice, *ssa.ChangeType, *ssa.Phi:
			if ix.writesThrough(ref.(ssa.Value), seen) {
				return true
			}
		}
	}
	return false
}

func (ix *index) isStd(f *ssa.Function) bool {
	p := pkgOf(f)
	return p != nil && ix.modules[p] == ""
}

func (r *reacher) convert(mi *ssa.MakeInterface) {
	t := mi.X.Type()
	r.addType(t, false)
	for _, p := range handedTo(mi) {
		if p != nil && p.Path() == "reflect" {
			p = pkgOf(r.cur)
		}
		switch {
		case p == nil:
		case r.sinks[p]:
			r.addType(t, true)
		default:
			r.pending[p] = append(r.pending[p], t)
		}
	}
}

func (r *reacher) call(c *ssa.CallCommon) {
	if c.IsInvoke() {
		r.addCall(c.Value.Type(), c.Method)
	}
	if p := pkgOf(r.cur); p != nil && p.Path() != "reflect" && r.cur.Syntax() != nil && looksUpMethod(c) && !r.sinks[p] {
		r.sinks[p] = true
		for _, t := range r.pending[p] {
			r.addType(t, true)
		}
		delete(r.pending, p)
	}
}

// addType records that values of type t exist at run time. Reflection can
// obtain the values of their exported fields and elements, so their types are
// runtime types too, and of the exported fields promoted from embedded fields,
// so these count as runtime types; it cannot call methods of values read from
// other unexported fields. If reflected, reflection can also call the exported
// methods of these values and of their method results, and create arguments
// for them.
func (r *reacher) addType(t types.Type, reflected bool) {
	t = types.Unalias(t)
	seen := &r.runtime
	if reflected {
		seen = &r.reflected
	}
	if t == nil || hasTypeParam(t) || seen.At(t) != nil || r.lim.cutType(t) {
		return
	}
	seen.Set(t, true)
	mset := r.prog.MethodSets.MethodSet(t)
	if reflected {
		r.addType(t, false)
		for i := range mset.Len() {
			sel := mset.At(i)
			sig := sel.Type().(*types.Signature)
			if sig.TypeParams().Len() == 0 && sel.Obj().Exported() {
				r.add(r.prog.MethodValue(sel))
				r.addType(sig.Params(), false)
				r.addType(sig.Results(), true)
			}
		}
	} else if !types.IsInterface(t) {
		r.addImpl(t, mset)
	}
	switch u := t.(type) {
	case *types.Signature:
		if reflected {
			r.addType(u.Params(), false)
			r.addType(u.Results(), true)
		}
	case *types.Named:
		r.addType(types.NewPointer(u), reflected)
		r.addType(u.Underlying(), reflected)
	case *types.Struct:
		for f := range u.Fields() {
			if f.Exported() || f.Embedded() {
				r.addType(f.Type(), reflected)
			}
		}
	case *types.Interface:
	default:
		for _, c := range components(t) {
			r.addType(c, reflected)
		}
	}
}

// addImpl matches the new concrete runtime type t against the interfaces called
// so far.
func (r *reacher) addImpl(t types.Type, mset *types.MethodSet) {
	for i := range mset.Len() {
		id := mset.At(i).Obj().Id()
		r.byMethod[id] = append(r.byMethod[id], t)
		for _, c := range r.byKey[id] {
			if types.Implements(t, c.iface.Underlying().(*types.Interface)) {
				c.impls = append(c.impls, t)
				for _, m := range c.methods {
					r.addMethod(t, m)
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
		for _, impl := range r.seed[key] {
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
		r.addMethod(impl, m)
	}
}

func (r *reacher) addMethod(recv types.Type, m *types.Func) {
	if sel := r.prog.MethodSets.MethodSet(recv).Lookup(m.Pkg(), m.Name()); sel != nil {
		r.add(r.prog.MethodValue(sel))
	}
}

func (r *reacher) indexSeed(seed *typeutil.Map) {
	seed.Iterate(func(t types.Type, _ any) {
		if types.IsInterface(t) || hasTypeParam(t) || r.lim.cutType(t) {
			return
		}
		mset := r.prog.MethodSets.MethodSet(t)
		for i := range mset.Len() {
			id := mset.At(i).Obj().Id()
			r.seed[id] = append(r.seed[id], t)
		}
	})
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
		for a := range t.TypeArgs().Types() {
			if hasTypeParam(a) {
				return true
			}
		}
		return false
	}
	return slices.ContainsFunc(components(t), hasTypeParam)
}

// components returns the types that the unnamed type t is composed of: the
// elements of pointers, slices, arrays, channels and maps, the keys of maps,
// the members of tuples, the parameters and results of signatures, the fields
// of structs, and the methods and embedded types of interfaces.
func components(t types.Type) []types.Type {
	var out []types.Type
	switch t := t.(type) {
	case *types.Pointer:
		out = append(out, t.Elem())
	case *types.Slice:
		out = append(out, t.Elem())
	case *types.Array:
		out = append(out, t.Elem())
	case *types.Chan:
		out = append(out, t.Elem())
	case *types.Map:
		out = append(out, t.Key(), t.Elem())
	case *types.Signature:
		out = append(out, t.Params(), t.Results())
	case *types.Tuple:
		for v := range t.Variables() {
			out = append(out, v.Type())
		}
	case *types.Struct:
		for f := range t.Fields() {
			out = append(out, f.Type())
		}
	case *types.Interface:
		for m := range t.ExplicitMethods() {
			out = append(out, m.Type())
		}
		out = append(out, slices.Collect(t.EmbeddedTypes())...)
	}
	return out
}
