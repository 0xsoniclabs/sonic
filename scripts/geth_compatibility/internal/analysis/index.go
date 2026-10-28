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
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

// index holds the declarations of the loaded program. symbols maps a key to
// its text: for a declaration the imports it refers to, the //go:embed
// directives of a variable, and its gofmt-ed source, without other comments
// and without blank lines outside raw string literals. decls holds the
// package-level declarations other than functions, of go-ethereum and of the
// other modules, to follow their references.
// initVars lists per package the variables used once it is initialized: all of
// a package outside go-ethereum, and those of go-ethereum whose initializer
// calls a function. files lists the keys of the files and cgo preambles of
// each go-ethereum package.
// modules maps the packages outside the standard library to their module.
type index struct {
	pkgs     map[*types.Package]*packages.Package
	modules  map[*types.Package]string
	symbols  map[string]string
	decls    map[string][]decl
	initVars map[string][]string
	files    map[string][]string
}

type decl struct {
	spec ast.Spec
	info *types.Info
}

func newIndex() *index {
	return &index{
		pkgs:     map[*types.Package]*packages.Package{},
		modules:  map[*types.Package]string{},
		symbols:  map[string]string{},
		decls:    map[string][]decl{},
		initVars: map[string][]string{},
		files:    map[string][]string{},
	}
}

func (ix *index) addGeth(p *packages.Package) {
	rel := relPkg(p.PkgPath)
	goFiles := map[string]bool{}
	for _, f := range p.GoFiles {
		goFiles[f] = true
	}
	for _, f := range p.Syntax {
		for _, d := range f.Decls {
			if !goFiles[p.Fset.Position(d.Pos()).Filename] {
				continue // cgo-generated
			}
			switch d := d.(type) {
			case *ast.FuncDecl:
				ix.add(funcKey(p.PkgPath, d), d, p)
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						ix.add(rel+" type "+spec.Name.Name, &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{spec}}, p)
					case *ast.ValueSpec:
						named := slices.ContainsFunc(spec.Names, func(n *ast.Ident) bool { return n.Name != "_" })
						embeds := embedDirectives(d.Doc, spec.Doc)
						for _, n := range spec.Names {
							if n.Name == "_" && (d.Tok != token.VAR || named) {
								continue
							}
							if c, ok := p.TypesInfo.Defs[n].(*types.Const); ok {
								ix.symbols[rel+" const "+n.Name] += fmt.Sprintf("const %s %s = %s", n.Name, c.Type(), c.Val().ExactString())
							} else if d.Tok == token.VAR {
								key := rel + " var " + n.Name
								ix.symbols[key] += embeds
								ix.add(key, &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{spec}}, p)
								if callsFunction(spec.Values, p.TypesInfo) {
									ix.addInitVar(p.PkgPath, key)
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
	var included []string
	for _, path := range p.GoFiles {
		if text := preamble(path); text != "" {
			key := rel + " cgo " + filepath.Base(path)
			ix.symbols[key] += text
			ix.files[rel] = append(ix.files[rel], key)
			included = append(included, includes(path, text)...)
		}
	}
	root := filepath.Dir(p.GoFiles[0])
	for _, path := range slices.Concat(p.EmbedFiles, nativeFiles(p, included)) {
		text, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		name, _ := filepath.Rel(root, path)
		key := rel + " file " + filepath.ToSlash(name)
		ix.symbols[key] += string(text)
		ix.files[rel] = append(ix.files[rel], key)
	}
}

// preamble returns the cgo preamble of the Go file at path, the comment above
// its import of "C" that holds C code and #cgo directives.
func preamble(path string) string {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return ""
	}
	var text strings.Builder
	for _, d := range f.Decls {
		g := d.(*ast.GenDecl)
		for _, s := range g.Specs {
			s := s.(*ast.ImportSpec)
			doc := s.Doc
			if doc == nil && len(g.Specs) == 1 {
				doc = g.Doc
			}
			if s.Path.Value != `"C"` || doc == nil {
				continue
			}
			for _, c := range doc.List {
				text.WriteString(c.Text)
				text.WriteByte('\n')
			}
		}
	}
	return text.String()
}

func (ix *index) addInitVar(pkg, key string) {
	if !slices.Contains(ix.initVars[pkg], key) {
		ix.initVars[pkg] = append(ix.initVars[pkg], key)
	}
}

// addOther records the package-level types, variables and constants of p, a
// package outside go-ethereum, so that the go-ethereum declarations they refer
// to are used if they are.
func (ix *index) addOther(p *packages.Package) {
	for _, f := range p.Syntax {
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			var last *ast.ValueSpec
			for _, spec := range g.Specs {
				var names []*ast.Ident
				switch s := spec.(type) {
				case *ast.TypeSpec:
					names = []*ast.Ident{s.Name}
				case *ast.ValueSpec:
					names = s.Names
					if g.Tok == token.CONST && len(s.Values) == 0 {
						spec = &ast.ValueSpec{Names: s.Names, Type: last.Type, Values: last.Values}
					} else {
						last = s
					}
				}
				for _, n := range names {
					key := fmt.Sprintf("%s %s %s", p.PkgPath, g.Tok, n.Name)
					ix.decls[key] = append(ix.decls[key], decl{spec, p.TypesInfo})
					if g.Tok == token.VAR {
						ix.addInitVar(p.PkgPath, key)
					}
				}
			}
		}
	}
}

var includeLine = regexp.MustCompile(`(?m)^\s*(?://\s*)?#\s*include\s+"([^"]+)"`)

// nativeFiles returns the C and assembly files of p, the files in included,
// and the files that any of them include with quotes, directly or indirectly.
func nativeFiles(p *packages.Package, included []string) []string {
	var files []string
	queue := slices.Concat(included, slices.DeleteFunc(slices.Clone(p.OtherFiles), func(path string) bool {
		return !isNativeSource(path)
	}))
	seen := map[string]bool{}
	for i := 0; i < len(queue); i++ {
		path := queue[i]
		if seen[path] {
			continue
		}
		seen[path] = true
		text, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		files = append(files, path)
		queue = append(queue, includes(path, string(text))...)
	}
	return files
}

func includes(path, text string) []string {
	var paths []string
	for _, m := range includeLine.FindAllStringSubmatch(text, -1) {
		paths = append(paths, filepath.Join(filepath.Dir(path), m[1]))
	}
	return paths
}

func isNativeSource(path string) bool {
	switch filepath.Ext(path) {
	case ".c", ".h", ".cc", ".cpp", ".hpp", ".s", ".S":
		return true
	}
	return false
}

func (ix *index) add(key string, node ast.Node, p *packages.Package) {
	stripComments(node)
	var buf bytes.Buffer
	if err := format.Node(&buf, p.Fset, node); err != nil {
		buf.Reset()
		fmt.Fprintf(&buf, "unprintable: %v", err)
	}
	ix.symbols[key] += importLines(node, p.TypesInfo) + dropBlankLines(buf.Bytes())
	if g, ok := node.(*ast.GenDecl); ok {
		ix.decls[key] = append(ix.decls[key], decl{g.Specs[0], p.TypesInfo})
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

// embedDirectives returns the //go:embed lines of docs, which choose the files
// a variable holds.
func embedDirectives(docs ...*ast.CommentGroup) string {
	var text strings.Builder
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		for _, c := range doc.List {
			if strings.HasPrefix(c.Text, "//go:embed") {
				text.WriteString(c.Text)
				text.WriteByte('\n')
			}
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
