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
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/0xsoniclabs/sonic/"

// layout is the Layout table of the plan as data: the module packages each
// package of the module may import. A change to the table changes this map in
// the same commit.
var layout = map[string][]string{
	"rpc/core":      {},
	"rpc/dispatch":  {},
	"rpc/chain":     {"rpc/core", "rpc/dispatch"},
	"rpc/exec":      {"rpc/chain", "rpc/dispatch", "rpc/core", "rpc/oracle"},
	"rpc/format":    {"rpc/core"},
	"rpc/oracle":    {"rpc/core"},
	"rpc/transport": {"rpc/dispatch"},
	"rpc/tests":     {"rpc/core", "rpc/dispatch", "rpc/chain", "rpc/exec", "rpc/oracle"},
	"rpc": {"rpc/core", "rpc/dispatch", "rpc/chain", "rpc/exec", "rpc/format", "rpc/oracle",
		"rpc/eth", "rpc/filters", "rpc/trace", "rpc/debug", "rpc/txpool", "rpc/net", "rpc/web3", "rpc/sonic",
		"rpc/transport", "rpc/tests"},
}

// namespaces import the shared packages, never another namespace.
var namespaces = []string{"rpc/eth", "rpc/filters", "rpc/trace", "rpc/debug", "rpc/txpool", "rpc/net", "rpc/web3", "rpc/sonic"}

func init() {
	for _, ns := range namespaces {
		layout[ns] = []string{"rpc/dispatch", "rpc/chain", "rpc/exec", "rpc/format", "rpc/oracle", "rpc/core"}
	}
}

// nodeLibraries are the node packages the module uses as libraries.
var nodeLibraries = []string{"evmcore", "opera", "inter", "inter/state", "txtrace", "utils"}

// leafExceptions are the consensus-critical definitions a package may import
// from gossip, each in the one package that needs it.
var leafExceptions = map[string][]string{
	"rpc/sonic":  {"gossip/blockproc/bundle"},
	"rpc/eth":    {"gossip/blockproc/priorities/registry", "gossip/blockproc/subsidies/registry"},
	"rpc/tests":  {"gossip/evmstore"},
	"rpc/format": {"gossip/blockproc/subsidies"},
	"rpc/exec":   {"gossip/gasprice/gaspricelimits"},
}

// gethKeep is the module's import budget of go-ethereum.
var gethKeep = []string{
	"common", "common/hexutil", "common/math", "core/types", "rlp", "crypto", "crypto/kzg4844",
	"core", "core/vm", "core/tracing", "eth/tracers", "eth/tracers/logger", "eth/tracers/native",
	"params", "accounts/abi", "log",
}

// thirdParty are the other modules the module uses.
var thirdParty = []string{
	"github.com/0xsoniclabs/carmen/go/common",
	"github.com/0xsoniclabs/carmen/go/common/immutable",
	"github.com/0xsoniclabs/carmen/go/state",         // rpc/tests: the Carmen state of test support
	"github.com/0xsoniclabs/carmen/go/state/gostate", // rpc/tests
	"github.com/Fantom-foundation/lachesis-base/inter/idx",
	"github.com/Fantom-foundation/lachesis-base/utils/piecefunc",
	"github.com/davecgh/go-spew/spew",
	"github.com/gorilla/websocket",
	"github.com/holiman/uint256",
	"github.com/prometheus/client_golang/prometheus",
}

// testOnly may be imported by test files: the assertion libraries, and
// go-ethereum's rpc package as the oracle of envelope compatibility.
var testOnly = []string{
	"github.com/stretchr/testify",
	"go.uber.org/mock",
	"github.com/ethereum/go-ethereum/rpc",
	"github.com/ethereum/go-ethereum/ethclient",
	"pgregory.net/rapid",
}

type goFile struct {
	pkg  string // module-relative directory
	path string
	test bool
	ast  *ast.File
}

func moduleFiles(t *testing.T) []goFile {
	t.Helper()
	var files []goFile
	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		full, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		dir := filepath.ToSlash(filepath.Join("rpc", filepath.Dir(path)))
		files = append(files, goFile{pkg: strings.TrimSuffix(dir, "/."), path: path, test: strings.HasSuffix(path, "_test.go"), ast: full})
		return nil
	})
	if err != nil {
		t.Fatalf("walking the module: %v", err)
	}
	return files
}

func imports(f goFile) []string {
	var out []string
	for _, imp := range f.ast.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		out = append(out, p)
	}
	return out
}

func isStdlib(p string) bool {
	first, _, _ := strings.Cut(p, "/")
	return !strings.Contains(first, ".")
}

func hasPrefixIn(p string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if p == prefix || strings.HasPrefix(p, prefix+"/") {
			return true
		}
	}
	return false
}

// TestImports_FollowTheLayoutAndTheBudget holds the import rules of the
// module: between its packages only what the Layout table lists, beyond it the
// go-ethereum keep list, the node libraries and the leaf exceptions, and never
// gossip, api or go-ethereum's rpc package.
func TestImports_FollowTheLayoutAndTheBudget(t *testing.T) {
	for _, f := range moduleFiles(t) {
		allowed, known := layout[f.pkg]
		if !known {
			t.Errorf("%s: package %s is not in the Layout table", f.path, f.pkg)
			continue
		}
		for _, p := range imports(f) {
			switch {
			case isStdlib(p):
			case strings.HasPrefix(p, modulePath+"rpc"):
				target := strings.TrimPrefix(p, modulePath)
				if target == f.pkg || slices.Contains(allowed, target) {
					continue
				}
				// A test may use the support of tests and what it wires.
				if f.test && (target == "rpc/tests" || slices.Contains(layout["rpc/tests"], target)) {
					continue
				}
				t.Errorf("%s: %s may not import %s", f.path, f.pkg, target)
			case strings.HasPrefix(p, modulePath):
				target := strings.TrimPrefix(p, modulePath)
				if hasPrefixIn(target, nodeLibraries) {
					continue
				}
				if slices.Contains(leafExceptions[f.pkg], target) {
					continue
				}
				t.Errorf("%s: %s may not import the node package %s", f.path, f.pkg, target)
			case strings.HasPrefix(p, "github.com/ethereum/go-ethereum/"):
				sub := strings.TrimPrefix(p, "github.com/ethereum/go-ethereum/")
				if slices.Contains(gethKeep, sub) {
					continue
				}
				if f.test && hasPrefixIn(p, testOnly) {
					continue
				}
				t.Errorf("%s: go-ethereum package %s is outside the keep list", f.path, sub)
			default:
				if hasPrefixIn(p, thirdParty) {
					continue
				}
				if f.test && hasPrefixIn(p, testOnly) {
					continue
				}
				t.Errorf("%s: third-party package %s is not on the module's list", f.path, p)
			}
		}
		if f.pkg == "rpc/dispatch" {
			for _, p := range imports(f) {
				if strings.HasPrefix(p, modulePath) && !f.test {
					t.Errorf("%s: the dispatcher imports the node package %s", f.path, p)
				}
			}
		}
	}
}

// TestLint_NoBareGoStatementOutsideTheDispatcher holds D1: every goroutine of
// the module is started by the dispatcher.
func TestLint_NoBareGoStatementOutsideTheDispatcher(t *testing.T) {
	for _, f := range moduleFiles(t) {
		if f.test || f.pkg == "rpc/dispatch" {
			continue
		}
		ast.Inspect(f.ast, func(n ast.Node) bool {
			if _, ok := n.(*ast.GoStmt); ok {
				t.Errorf("%s: bare go statement; start goroutines through the dispatcher", f.path)
			}
			return true
		})
	}
}

// TestLint_NoExecutionTimeLimit holds O2: the dispatcher applies the request
// deadline, not the fork's SetExecutionTimeLimit.
func TestLint_NoExecutionTimeLimit(t *testing.T) {
	for _, f := range moduleFiles(t) {
		ast.Inspect(f.ast, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && id.Name == "SetExecutionTimeLimit" {
				t.Errorf("%s: SetExecutionTimeLimit is not used by the module", f.path)
			}
			return true
		})
	}
}

// TestLint_NamespacesReachCoreThroughRoles holds H8: no namespace names
// core.Core, and Env exports no field of Core's type.
func TestLint_NamespacesReachCoreThroughRoles(t *testing.T) {
	for _, f := range moduleFiles(t) {
		if f.test {
			continue
		}
		if slices.Contains(namespaces, f.pkg) {
			ast.Inspect(f.ast, func(n ast.Node) bool {
				if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "Core" {
					if id, ok := sel.X.(*ast.Ident); ok && id.Name == "core" {
						t.Errorf("%s: a namespace takes a role of Core, not core.Core", f.path)
					}
				}
				return true
			})
		}
		if f.pkg == "rpc/exec" {
			ast.Inspect(f.ast, func(n ast.Node) bool {
				spec, ok := n.(*ast.TypeSpec)
				if !ok || spec.Name.Name != "Env" {
					return true
				}
				for _, field := range spec.Type.(*ast.StructType).Fields.List {
					for _, name := range field.Names {
						if name.IsExported() {
							t.Errorf("%s: Env exports the field %s; it offers lookups instead", f.path, name.Name)
						}
					}
				}
				return false
			})
		}
	}
}
