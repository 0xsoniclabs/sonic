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

package rpc_test

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The RPC module reaches the node only through rpc/core, and it can be
// deleted, or the legacy stack beside it, without touching the other. These
// tests hold the layout of its packages and the import rules that keep it so.

const repository = "github.com/0xsoniclabs/sonic/"

// namespaceImports are the module packages a namespace may import. A namespace
// never imports another; what two of them share moves to exec or format.
var namespaceImports = []string{"core", "dispatch", "chain", "exec", "format", "oracle"}

// layout lists, for each package of the module, the module packages its
// non-test files may import.
var layout = map[string][]string{
	"rpc/core":      {},
	"rpc/dispatch":  {},
	"rpc/chain":     {"core", "dispatch"},
	"rpc/exec":      {"chain", "dispatch", "core", "oracle"},
	"rpc/format":    {"core"},
	"rpc/oracle":    {"core"},
	"rpc/eth":       namespaceImports,
	"rpc/filters":   namespaceImports,
	"rpc/trace":     namespaceImports,
	"rpc/debug":     namespaceImports,
	"rpc/txpool":    namespaceImports,
	"rpc/net":       namespaceImports,
	"rpc/web3":      namespaceImports,
	"rpc/sonic":     namespaceImports,
	"rpc/dag":       namespaceImports,
	"rpc/abft":      namespaceImports,
	"rpc/testonly":  namespaceImports,
	"rpc/transport": {"dispatch"},
	"rpc/tests":     {"core", "dispatch", "chain", "exec", "oracle"},
	"rpc": {"core", "dispatch", "chain", "exec", "format", "oracle", "eth", "filters",
		"trace", "debug", "txpool", "net", "web3", "sonic", "dag", "abft", "testonly", "transport"},
}

// nodeLibraries are the node packages every package of the module may import:
// the data model the node hands out through rpc/core.
var nodeLibraries = []string{
	"evmcore",
	"opera",
	"inter",
	"inter/state",
	"inter/iblockproc",
	"utils",
}

// leafExceptions are node packages of consensus-critical definitions, which
// must not be copied, each allowed in one package of the module.
var leafExceptions = map[string]string{
	"gossip/blockproc/bundle":              "rpc/sonic",  // plan encoding, plan hash, envelope gas formula
	"gossip/blockproc/priorities/registry": "rpc/eth",    // system contract addresses of eth_config
	"gossip/blockproc/subsidies/registry":  "rpc/eth",    // system contract addresses of eth_config
	"gossip/blockproc/subsidies":           "rpc/format", // a sponsored transaction's zero effective gas price
	"gossip/gasprice/gaspricelimits":       "rpc/exec",   // the suggested price of a new transaction
	"gossip/evmstore":                      "rpc/tests",  // the Carmen state wrapper of test support
}

// gethKeepList is the part of go-ethereum the module uses: the data model,
// messages and the EVM, the tracers, the chain config and the logger. Its RPC
// server, node and event packages stay out.
var gethKeepList = []string{
	"common",
	"common/hexutil",
	"common/math",
	"core/types",
	"rlp",
	"crypto",
	"crypto/kzg4844",
	"core",
	"core/vm",
	"core/tracing",
	"eth/tracers",
	"eth/tracers/logger",
	"eth/tracers/native",
	"trie",
	"params",
	"accounts/abi",
	"log",
}

// gethOnlyIn restricts a kept go-ethereum package to some packages.
var gethOnlyIn = map[string][]string{
	"trie": {"rpc/eth"}, // hashes the transactions of a simulated block
}

// lachesisTypes are the lachesis-base packages of the index and hash types the
// node libraries hand out.
var lachesisTypes = []string{
	"github.com/Fantom-foundation/lachesis-base/hash",
	"github.com/Fantom-foundation/lachesis-base/inter/idx",
}

// legacyPackages are the legacy RPC stack. The module imports none of them and
// they import nothing of the module.
var legacyPackages = []string{
	"api",
	"api/ethapi",
	"api/sonicapi",
	"api/testapi",
	"api/rpctest",
	"gossip/filters",
	"gossip/gasprice",
	"txtrace",
}

// legacyGossipFiles are the files of the gossip package that belong to the
// legacy RPC stack.
var legacyGossipFiles = []string{
	"gossip/api.go",
	"gossip/ethapi_backend.go",
	"gossip/gpo_backend.go",
}

func TestLayout_EveryModulePackageHasARow(t *testing.T) {
	for _, pkg := range modulePackages(t) {
		_, found := layout[pkg]
		require.True(t, found, "package %s has no row in the layout", pkg)
	}
}

func TestLayout_ModulePackagesImportOnlyWhatTheirRowLists(t *testing.T) {
	for _, pkg := range modulePackages(t) {
		for _, imported := range nonTestImports(t, pkg) {
			module, isModule := strings.CutPrefix(imported, repository+"rpc/")
			if imported == repository+"rpc" {
				module, isModule = "", true
			}
			if isModule {
				require.Contains(t, layout[pkg], module, "%s imports %s, which its row does not list", pkg, imported)
				continue
			}
			require.NoError(t, checkExternalImport(pkg, imported))
		}
	}
}

func TestLayout_LegacyStackImportsNothingOfTheModule(t *testing.T) {
	for _, pkg := range legacyPackages {
		for _, imported := range allImports(t, pkg) {
			require.False(t, isModulePackage(imported), "legacy package %s imports %s", pkg, imported)
		}
	}
	for _, file := range legacyGossipFiles {
		for _, imported := range fileImports(t, file) {
			require.False(t, isModulePackage(imported), "legacy file %s imports %s", file, imported)
		}
	}
}

// The dispatcher starts every goroutine of the module, so that shutdown can
// wait for all of them and a panic in one is recovered.
func TestLayout_OnlyTheDispatcherStartsGoroutines(t *testing.T) {
	for _, pkg := range modulePackages(t) {
		if pkg == "rpc/dispatch" {
			continue
		}
		for _, file := range importPackage(t, pkg).GoFiles {
			path := filepath.Join(repositoryPath(pkg), file)
			parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			require.NoError(t, err)
			ast.Inspect(parsed, func(node ast.Node) bool {
				_, isGo := node.(*ast.GoStmt)
				require.False(t, isGo, "%s starts a goroutine; use the dispatcher's Go or Loop", path)
				return true
			})
		}
	}
}

func TestLayout_CheckExternalImport_RefusesWhatTheRulesExclude(t *testing.T) {
	tests := map[string]struct {
		pkg, imported string
		allowed       bool
	}{
		"standard library":         {"rpc/eth", "context", true},
		"third party":              {"rpc/transport", "github.com/gorilla/websocket", true},
		"node library":             {"rpc/eth", repository + "evmcore", true},
		"other node package":       {"rpc/eth", repository + "gossip", false},
		"legacy package":           {"rpc/eth", repository + "api/ethapi", false},
		"leaf exception":           {"rpc/eth", repository + "gossip/blockproc/subsidies/registry", true},
		"leaf exception elsewhere": {"rpc/format", repository + "gossip/blockproc/subsidies/registry", false},
		"kept go-ethereum":         {"rpc/eth", "github.com/ethereum/go-ethereum/core/types", true},
		"go-ethereum rpc":          {"rpc/eth", "github.com/ethereum/go-ethereum/rpc", false},
		"restricted go-ethereum":   {"rpc/format", "github.com/ethereum/go-ethereum/trie", false},
		"lachesis types":           {"rpc/eth", "github.com/Fantom-foundation/lachesis-base/inter/idx", true},
		"other lachesis":           {"rpc/eth", "github.com/Fantom-foundation/lachesis-base/abft", false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := checkExternalImport(test.pkg, test.imported)
			require.Equal(t, test.allowed, err == nil, "%v", err)
		})
	}
}

// checkExternalImport tells whether a package of the module may import a
// package from outside the module.
func checkExternalImport(pkg, imported string) error {
	if gethPackage, isGeth := strings.CutPrefix(imported, "github.com/ethereum/go-ethereum/"); isGeth {
		if !slices.Contains(gethKeepList, gethPackage) {
			return importError(pkg, imported, "go-ethereum package outside the keep list")
		}
		if only, restricted := gethOnlyIn[gethPackage]; restricted && !slices.Contains(only, pkg) {
			return importError(pkg, imported, "go-ethereum package restricted to "+strings.Join(only, ", "))
		}
		return nil
	}
	if strings.HasPrefix(imported, "github.com/Fantom-foundation/lachesis-base/") && !slices.Contains(lachesisTypes, imported) {
		return importError(pkg, imported, "lachesis-base package beyond its index and hash types")
	}
	nodePackage, isNode := strings.CutPrefix(imported, repository)
	if !isNode || slices.Contains(nodeLibraries, nodePackage) || leafExceptions[nodePackage] == pkg {
		return nil
	}
	return importError(pkg, imported, "node package outside the node libraries and the leaf exceptions")
}

type layoutError struct{ message string }

func (e layoutError) Error() string { return e.message }

func importError(pkg, imported, reason string) error {
	return layoutError{pkg + " imports " + imported + ": " + reason}
}

func isModulePackage(path string) bool {
	return path == repository+"rpc" || strings.HasPrefix(path, repository+"rpc/")
}

// modulePackages lists the packages under rpc/ with non-test files, relative
// to the repository.
func modulePackages(t *testing.T) []string {
	t.Helper()
	var packages []string
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return err
		}
		if entry.Name() == "testdata" {
			return filepath.SkipDir
		}
		pkg, err := build.ImportDir(path, 0)
		if err != nil {
			if _, noGo := err.(*build.NoGoError); noGo {
				return nil
			}
			return err
		}
		if len(pkg.GoFiles) > 0 {
			packages = append(packages, filepath.ToSlash(filepath.Join("rpc", path)))
		}
		return nil
	})
	require.NoError(t, err)
	return packages
}

// nonTestImports lists the imports of a package's non-test files. Test files
// may import more: go-ethereum's RPC server is the oracle of the envelope
// tests.
func nonTestImports(t *testing.T, pkg string) []string {
	t.Helper()
	return importPackage(t, pkg).Imports
}

func allImports(t *testing.T, pkg string) []string {
	t.Helper()
	imported := importPackage(t, pkg)
	return slices.Concat(imported.Imports, imported.TestImports, imported.XTestImports)
}

func importPackage(t *testing.T, pkg string) *build.Package {
	t.Helper()
	imported, err := build.ImportDir(repositoryPath(pkg), 0)
	require.NoError(t, err)
	return imported
}

func fileImports(t *testing.T, file string) []string {
	t.Helper()
	dir, name := filepath.Split(repositoryPath(file))
	context := build.Default
	context.ReadDir = func(string) ([]fs.FileInfo, error) {
		info, err := os.Stat(filepath.Join(dir, name))
		return []fs.FileInfo{info}, err
	}
	imported, err := context.ImportDir(dir, 0)
	require.NoError(t, err)
	return imported.Imports
}

// repositoryPath turns a path relative to the repository into one relative to
// this package's directory.
func repositoryPath(path string) string {
	return filepath.Join("..", filepath.FromSlash(path))
}
