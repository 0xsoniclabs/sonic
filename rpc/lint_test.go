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
	"unicode"
)

const (
	modulePath = "github.com/0xsoniclabs/sonic"
	rpcPath    = modulePath + "/rpc"
)

// layout is the Layout table of the plan: for each package of the module, the
// packages of the module it may import. A namespace imports no other
// namespace, and dispatch imports no package of the node.
var layout = map[string][]string{
	"rpc/core":      {},
	"rpc/dispatch":  {},
	"rpc/chain":     {"rpc/core", "rpc/dispatch"},
	"rpc/exec":      {"rpc/chain", "rpc/dispatch", "rpc/core", "rpc/oracle"},
	"rpc/format":    {"rpc/core"},
	"rpc/oracle":    {"rpc/core"},
	"rpc/transport": {"rpc/dispatch"},
	"rpc/tests":     {"rpc/core", "rpc/dispatch", "rpc/chain", "rpc/exec", "rpc/oracle"},
	"rpc":           nil, // the module wires everything
}

// namespaces are the packages that serve one namespace each.
var namespaces = []string{"rpc/eth", "rpc/filters", "rpc/trace", "rpc/debug", "rpc/txpool", "rpc/net", "rpc/web3", "rpc/sonic"}

var namespaceImports = []string{"rpc/dispatch", "rpc/chain", "rpc/exec", "rpc/format", "rpc/oracle", "rpc/core"}

// nodeLibraries are the packages of the node any package of the module but
// dispatch may import.
var nodeLibraries = []string{"evmcore", "opera", "inter", "inter/state", "txtrace", "utils"}

// leafExceptions are the packages of consensus-critical definitions that must
// not be copied, each allowed in the one package that needs it.
var leafExceptions = map[string]string{
	"gossip/blockproc/bundle":              "rpc/sonic",
	"gossip/blockproc/priorities/registry": "rpc/eth",
	"gossip/blockproc/subsidies/registry":  "rpc/eth",
	"gossip/evmstore":                      "rpc/tests",
	"gossip/blockproc/subsidies":           "rpc/format",
	"gossip/gasprice/gaspricelimits":       "rpc/exec",
}

// gethKeepList is the part of go-ethereum the module keeps: the data model,
// the EVM and its tracers, the chain configuration and the logger. core is
// kept for the execution of messages (Message, ApplyMessage, GasPool).
var gethKeepList = []string{
	"common", "common/hexutil", "common/math", "core/types", "rlp", "crypto", "crypto/kzg4844",
	"core", "core/vm", "core/tracing", "eth/tracers", "eth/tracers/logger", "eth/tracers/native",
	"params", "accounts/abi", "log",
}

// thirdParty are the other dependencies of the module, each with its reason.
var thirdParty = []string{
	"github.com/prometheus/client_golang/prometheus", // metrics
	"github.com/rs/cors",                             // the CORS checks of go-ethereum's HTTP stack
	"github.com/gorilla/websocket",                   // the WebSocket transport
	"github.com/holiman/uint256",                     // balances and values
	"github.com/0xsoniclabs/carmen/go/common",        // witness proofs
	"github.com/Fantom-foundation/lachesis-base/inter/idx",
	"github.com/Fantom-foundation/lachesis-base/utils/piecefunc", // the curves of the gas price oracle
	"github.com/davecgh/go-spew/spew",                            // debug_printBlock
}

// testSupportImports are what rpc/tests may import beyond the module's own
// dependencies: the real Carmen state behind its core.
var testSupportImports = []string{
	"github.com/0xsoniclabs/carmen/go/state",
}

type sourceFile struct {
	path    string
	pkg     string // relative to the module, rpc/...
	test    bool
	syntax  *ast.File
	fileSet *token.FileSet
}

func parseModule(t *testing.T) []sourceFile {
	t.Helper()
	var files []sourceFile
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		fileSet := token.NewFileSet()
		syntax, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		directory := filepath.ToSlash(filepath.Dir(path))
		pkg := "rpc"
		if directory != "." {
			pkg = "rpc/" + directory
		}
		files = append(files, sourceFile{
			path:    path,
			pkg:     pkg,
			test:    strings.HasSuffix(path, "_test.go"),
			syntax:  syntax,
			fileSet: fileSet,
		})
		return nil
	})
	if err != nil {
		t.Fatalf("failed to parse the module: %v", err)
	}
	return files
}

func importsOf(file sourceFile) []string {
	var paths []string
	for _, spec := range file.syntax.Imports {
		path, _ := strconv.Unquote(spec.Path.Value)
		paths = append(paths, path)
	}
	return paths
}

func isStandardLibrary(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return !strings.Contains(first, ".")
}

func underAny(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

// allowedModuleImports returns the packages of the module a package may
// import, nil for any.
func allowedModuleImports(pkg string) ([]string, bool) {
	if slices.Contains(namespaces, pkg) {
		return namespaceImports, true
	}
	allowed, found := layout[pkg]
	return allowed, found
}

func TestLayout_EveryPackageIsInTheTable(t *testing.T) {
	for _, file := range parseModule(t) {
		if file.test {
			continue
		}
		if _, found := allowedModuleImports(file.pkg); !found {
			t.Errorf("%s: package %s is not in the Layout table", file.path, file.pkg)
		}
	}
}

func TestLayout_ImportsFollowTheTable(t *testing.T) {
	for _, file := range parseModule(t) {
		if file.test {
			continue
		}
		allowed, _ := allowedModuleImports(file.pkg)
		for _, path := range importsOf(file) {
			if err := checkImport(file.pkg, path, allowed); err != "" {
				t.Errorf("%s: %s", file.path, err)
			}
		}
	}
}

func checkImport(pkg, path string, allowed []string) string {
	switch {
	case isStandardLibrary(path):
		return ""
	case strings.HasPrefix(path, rpcPath+"/") || path == rpcPath:
		target := strings.TrimPrefix(path, modulePath+"/")
		if allowed != nil && !slices.Contains(allowed, target) {
			return pkg + " imports " + target + ", which the Layout table does not list"
		}
		return ""
	case strings.HasPrefix(path, modulePath+"/"):
		return checkNodeImport(pkg, strings.TrimPrefix(path, modulePath+"/"))
	case strings.HasPrefix(path, "github.com/ethereum/go-ethereum"):
		library := strings.TrimPrefix(strings.TrimPrefix(path, "github.com/ethereum/go-ethereum"), "/")
		if !slices.Contains(gethKeepList, library) {
			return pkg + " imports go-ethereum's " + path + ", which is not on the keep list"
		}
		return ""
	case underAny(path, thirdParty):
		return ""
	case pkg == "rpc/tests" && underAny(path, testSupportImports):
		return ""
	}
	return pkg + " imports " + path + ", a dependency the module does not list"
}

func checkNodeImport(pkg, target string) string {
	if owner, found := leafExceptions[target]; found {
		if owner != pkg {
			return pkg + " imports " + target + ", allowed in " + owner + " only"
		}
		return ""
	}
	if pkg == "rpc/dispatch" || pkg == "rpc/transport" {
		return pkg + " imports the node package " + target
	}
	if underAny(target, nodeLibraries) {
		return ""
	}
	return pkg + " imports the node package " + target + ", which is no node library"
}

func TestLayout_TestsImportNoForbiddenPackage(t *testing.T) {
	for _, file := range parseModule(t) {
		if !file.test {
			continue
		}
		for _, path := range importsOf(file) {
			target := strings.TrimPrefix(path, modulePath+"/")
			if strings.HasPrefix(target, "api/") {
				t.Errorf("%s imports %s", file.path, path)
			}
			if strings.HasPrefix(target, "gossip") {
				if owner, found := leafExceptions[target]; !found || owner != file.pkg {
					t.Errorf("%s imports %s", file.path, path)
				}
			}
		}
	}
}

// TestLint_GoroutinesStartInTheDispatcher holds D1: every goroutine of the
// module starts in the dispatcher.
func TestLint_GoroutinesStartInTheDispatcher(t *testing.T) {
	for _, file := range parseModule(t) {
		if file.test || file.pkg == "rpc/dispatch" {
			continue
		}
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			if statement, ok := node.(*ast.GoStmt); ok {
				t.Errorf("%s: a go statement outside the dispatcher", file.fileSet.Position(statement.Pos()))
			}
			return true
		})
	}
}

// TestLint_NoExecutionTimeLimit holds O2: the request deadline is the
// dispatcher's, not the fork's global limit.
func TestLint_NoExecutionTimeLimit(t *testing.T) {
	for _, file := range parseModule(t) {
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			if selector, ok := node.(*ast.SelectorExpr); ok && selector.Sel.Name == "SetExecutionTimeLimit" {
				t.Errorf("%s: SetExecutionTimeLimit", file.fileSet.Position(selector.Pos()))
			}
			return true
		})
	}
}

// TestLint_NamespacesReachNoCore holds H8: a namespace receives the role
// interfaces it uses, never core.Core, and the environment hands out no Core.
func TestLint_NamespacesReachNoCore(t *testing.T) {
	for _, file := range parseModule(t) {
		if file.test {
			continue
		}
		if slices.Contains(namespaces, file.pkg) {
			ast.Inspect(file.syntax, func(node ast.Node) bool {
				if selector, ok := node.(*ast.SelectorExpr); ok && selector.Sel.Name == "Core" {
					if identifier, ok := selector.X.(*ast.Ident); ok && identifier.Name == "core" {
						t.Errorf("%s: a namespace names core.Core", file.fileSet.Position(selector.Pos()))
					}
				}
				return true
			})
		}
		if file.pkg == "rpc/exec" {
			checkEnvironmentFields(t, file)
		}
	}
}

func checkEnvironmentFields(t *testing.T, file sourceFile) {
	ast.Inspect(file.syntax, func(node ast.Node) bool {
		spec, ok := node.(*ast.TypeSpec)
		if !ok || spec.Name.Name != "Environment" {
			return true
		}
		for _, field := range spec.Type.(*ast.StructType).Fields.List {
			for _, name := range field.Names {
				if name.IsExported() {
					t.Errorf("%s: Environment exports the field %s", file.fileSet.Position(name.Pos()), name.Name)
				}
			}
		}
		return false
	})
}

// allowedShortNames are the names shorter than three letters the module may
// declare.
var allowedShortNames = []string{"_", "ok", "t", "b", "ID", "Go"}

// clippedWords are the clipped words a name may not contain; the full word is
// written instead.
var clippedWords = []string{"cfg", "req", "msg", "addr", "tx", "txs", "ref", "args", "ns", "idx", "num", "buf", "conn", "fn", "opts", "tmp", "params"}

// ownedNames are names that keep the spelling of their owner: the knobs of
// the node configuration and its TOML keys.
var ownedNames = []string{"RPCTxFeeCap", "AllowUnprotectedTxs", "TxSyncDefaultTimeout", "TxSyncMaxTimeout"}

// TestLint_NamesAreWholeWords holds H9 for the names the module declares.
func TestLint_NamesAreWholeWords(t *testing.T) {
	for _, file := range parseModule(t) {
		for _, name := range declaredNames(file.syntax) {
			if problem := checkName(name.Name); problem != "" {
				t.Errorf("%s: %s %s", file.fileSet.Position(name.Pos()), name.Name, problem)
			}
		}
	}
}

func checkName(name string) string {
	if name == "ctx" || name == "err" || slices.Contains(allowedShortNames, name) || slices.Contains(ownedNames, name) {
		return ""
	}
	if len(name) < 3 {
		return "is shorter than three letters"
	}
	for _, word := range splitWords(name) {
		if slices.Contains(clippedWords, strings.ToLower(word)) {
			return "contains the clipped word " + word
		}
	}
	return ""
}

// splitWords splits a camel-case name into its words.
func splitWords(name string) []string {
	var words []string
	runes := []rune(name)
	start := 0
	for index := 1; index < len(runes); index++ {
		lowerToUpper := unicode.IsLower(runes[index-1]) && unicode.IsUpper(runes[index])
		acronymEnd := unicode.IsUpper(runes[index-1]) && unicode.IsUpper(runes[index]) && index+1 < len(runes) && unicode.IsLower(runes[index+1])
		if lowerToUpper || acronymEnd || runes[index] == '_' {
			words = append(words, string(runes[start:index]))
			start = index
		}
	}
	words = append(words, string(runes[start:]))
	for index, word := range words {
		words[index] = strings.Trim(word, "_")
	}
	return words
}

// declaredNames returns the identifiers a file declares: functions and
// methods, receivers, parameters and results, types, fields and interface
// methods, variables and constants, and the variables of := and range.
// Methods that implement an interface of the standard library keep its names.
func declaredNames(file *ast.File) []*ast.Ident {
	var names []*ast.Ident
	addFields := func(fields *ast.FieldList) {
		if fields == nil {
			return
		}
		for _, field := range fields.List {
			names = append(names, field.Names...)
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch declaration := node.(type) {
		case *ast.FuncDecl:
			names = append(names, declaration.Name)
			addFields(declaration.Recv)
		case *ast.FuncType:
			addFields(declaration.Params)
			addFields(declaration.Results)
		case *ast.TypeSpec:
			names = append(names, declaration.Name)
		case *ast.StructType:
			addFields(declaration.Fields)
		case *ast.InterfaceType:
			addFields(declaration.Methods)
		case *ast.ValueSpec:
			names = append(names, declaration.Names...)
		case *ast.AssignStmt:
			if declaration.Tok == token.DEFINE {
				for _, expression := range declaration.Lhs {
					if identifier, ok := expression.(*ast.Ident); ok {
						names = append(names, identifier)
					}
				}
			}
		case *ast.RangeStmt:
			if declaration.Tok == token.DEFINE {
				for _, expression := range []ast.Expr{declaration.Key, declaration.Value} {
					if identifier, ok := expression.(*ast.Ident); ok {
						names = append(names, identifier)
					}
				}
			}
		}
		return true
	})
	return names
}
