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

package rpc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// interfaceMethods are methods that implement an interface of the standard
// library or of go-ethereum; they are called through it.
var interfaceMethods = []string{
	"Error", "ErrorCode", "ErrorData", "MarshalJSON", "UnmarshalJSON", "MarshalText", "UnmarshalText", "String",
	"ServeHTTP", "Unwrap",
}

// testOnlyExports are the exported symbols only tests use, with the reason.
var testOnlyExports = map[string]string{}

// TestH4_ExportedSymbolsHaveCallers is the exported half of H4; staticcheck's
// U1000 checks the unexported half. An exported symbol of the module is named
// outside its declaration by code that is not a test.
func TestH4_ExportedSymbolsHaveCallers(t *testing.T) {
	type decl struct {
		pkg, name string
		pos       token.Pos
		method    bool
	}
	fset := token.NewFileSet()
	var decls []decl
	for _, f := range moduleFiles(t) {
		if f.test || f.pkg == "rpc/tests" {
			continue
		}
		file, err := parser.ParseFile(fset, f.path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range file.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Name.IsExported() {
					decls = append(decls, decl{pkg: f.pkg, name: d.Name.Name, pos: d.Name.Pos(), method: d.Recv != nil})
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							decls = append(decls, decl{pkg: f.pkg, name: s.Name.Name, pos: s.Name.Pos()})
						}
					case *ast.ValueSpec:
						for _, n := range s.Names {
							if n.IsExported() {
								decls = append(decls, decl{pkg: f.pkg, name: n.Name, pos: n.Pos()})
							}
						}
					}
				}
			}
		}
	}

	// The names used by code that is not a test, anywhere in the repository.
	used := map[string]bool{}
	root := ".."
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); path != root && (strings.HasPrefix(name, ".") || name == "build" || name == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.Contains(path, "/rpc/tests/") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}
		// The names a declaration introduces are not uses of them.
		declared := map[*ast.Ident]bool{}
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncDecl:
				declared[n.Name] = true
			case *ast.TypeSpec:
				declared[n.Name] = true
			case *ast.ValueSpec:
				for _, name := range n.Names {
					declared[name] = true
				}
			}
			return true
		})
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.SelectorExpr:
				used[n.Sel.Name] = true
			case *ast.Ident:
				if !declared[n] {
					used[n.Name] = true
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range decls {
		if used[d.name] || (d.method && slices.Contains(interfaceMethods, d.name)) {
			continue
		}
		if _, ok := testOnlyExports[d.pkg+"."+d.name]; ok {
			continue
		}
		t.Errorf("%s.%s has no caller outside tests", d.pkg, d.name)
	}
}
