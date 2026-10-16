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
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/core"
)

const modulePath = "github.com/0xsoniclabs/sonic"

// moduleFiles parses the imports and statements of the module's Go files
// (without tests, unless asked for).
func moduleFiles(t *testing.T, withTests bool) map[string]*ast.File {
	t.Helper()
	files := map[string]*ast.File{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if !withTests && strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		require.NoError(t, err, path)
		files[filepath.ToSlash(path)] = f
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, files)
	return files
}

func imports(f *ast.File) []string {
	var res []string
	for _, imp := range f.Imports {
		res = append(res, strings.Trim(imp.Path.Value, `"`))
	}
	return res
}

// I1: nothing under rpc/ imports gossip, api/* or go-ethereum/rpc. The
// exceptions are leaf packages of consensus-critical definitions that must not
// be copied, each tied to the one package that needs it.
var importExceptions = map[string][]string{
	// plan encoding, plan hash and envelope gas formula
	modulePath + "/gossip/blockproc/bundle": {"sonic/"},
	// addresses of system contracts, reported by eth_config
	modulePath + "/gossip/blockproc/priorities/registry": {"eth/"},
	modulePath + "/gossip/blockproc/subsidies/registry":  {"eth/"},
	// the constructor of the Carmen state wrapper; test support only
	modulePath + "/gossip/evmstore": {"tests/"},
}

func TestImports_NothingImportsTheNode(t *testing.T) {
	forbidden := []string{
		modulePath + "/gossip",
		modulePath + "/api",
		"github.com/ethereum/go-ethereum/rpc",
		"github.com/ethereum/go-ethereum/node",
		"github.com/ethereum/go-ethereum/metrics", // MON1: Prometheus client_golang only
		"github.com/ethereum/go-ethereum/event",
		"github.com/ethereum/go-ethereum/accounts/keystore",
		"github.com/ethereum/go-ethereum/accounts/scwallet",
		"github.com/ethereum/go-ethereum/accounts/external",
	}
	for path, f := range moduleFiles(t, false) {
		for _, imp := range imports(f) {
			for _, bad := range forbidden {
				if imp != bad && !strings.HasPrefix(imp, bad+"/") {
					continue
				}
				allowedIn, ok := importExceptions[imp]
				allowed := ok && slices.ContainsFunc(allowedIn, func(p string) bool { return strings.HasPrefix(path, p) })
				require.True(t, allowed, "%s imports %s", path, imp)
			}
		}
	}
}

// go-ethereum stays for the data model, the EVM and the tracers (section 3).
var gethKeepList = []string{
	"common", "common/hexutil", "common/math", "core/types", "rlp", "crypto", "crypto/kzg4844",
	"core", "core/vm", "core/tracing", "eth/tracers", "eth/tracers/logger", "eth/tracers/native", "eth/tracers/js",
	"params", "accounts/abi", "log", "common/lru",
}

func TestImports_GethOnlyForWhatItIsKeptFor(t *testing.T) {
	const prefix = "github.com/ethereum/go-ethereum/"
	for path, f := range moduleFiles(t, false) {
		for _, imp := range imports(f) {
			if !strings.HasPrefix(imp, prefix) {
				continue
			}
			require.Contains(t, gethKeepList, strings.TrimPrefix(imp, prefix), "%s imports %s", path, imp)
		}
	}
}

// D1: request goroutines are created by the dispatcher or its Group only.
func TestNoBareGoStatements_OutsideTheDispatcher(t *testing.T) {
	for path, f := range moduleFiles(t, false) {
		if strings.HasPrefix(path, "dispatch/") {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if g, ok := n.(*ast.GoStmt); ok {
				t.Errorf("%s: bare go statement at %v; use Dispatcher.Go, Loop or Group", path, g.Pos())
			}
			return true
		})
	}
}

// O2: the fork's SetExecutionTimeLimit is not used by the module.
func TestNoExecutionTimeLimit(t *testing.T) {
	for path, f := range moduleFiles(t, true) {
		if strings.HasSuffix(path, "lint_test.go") {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && id.Name == "SetExecutionTimeLimit" {
				t.Errorf("%s uses SetExecutionTimeLimit", path)
			}
			return true
		})
	}
}

// The interface to the node is narrow: it does not grow during the port, and
// has no setter but SendTx (section 2).
func TestCore_SurfaceIsTheSpecifiedOne(t *testing.T) {
	typ := reflect.TypeOf((*core.Core)(nil)).Elem()
	var names []string
	for i := range typ.NumMethod() {
		names = append(names, typ.Method(i).Name)
	}
	sort.Strings(names)
	want := []string{
		"BundleInfo", "Block", "BlockNumberByHash", "ChainInfo", "FindLogs", "Head", "Header",
		"OracleInputs", "PendingHeads", "PoolContent", "PoolContentFrom", "PoolNonce", "PoolStats",
		"PoolTransaction", "PoolTransactions", "Progress", "Receipts", "Rules", "SendTx", "State",
		"SubscribeHead", "SubscribeTxs", "TxByHash",
	}
	sort.Strings(want)
	require.Equal(t, want, names)

	for _, name := range names {
		for _, prefix := range []string{"Set", "Put", "Write", "Delete", "Remove", "Update", "Commit"} {
			require.False(t, strings.HasPrefix(name, prefix), "Core has a mutating method %s", name)
		}
	}
}
