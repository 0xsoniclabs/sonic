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

package sonic

// The test support the reference's bundle tests used, on the module's test
// core: a chain whose state holds the accounts, and the handlers over it.

import (
	"context"
	"encoding/json"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func toHexUint64(value uint64) *hexutil.Uint64 {
	converted := hexutil.Uint64(value)
	return &converted
}

func toHexBigInt(value *big.Int) *hexutil.Big {
	if value == nil {
		return nil
	}
	converted := hexutil.Big(*value)
	return &converted
}

// expectJsonEqual checks that a value encodes to the JSON expected, compared
// as values of its type.
func expectJsonEqual[Value any](t testing.TB, expected string, value Value) {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err, "failed to marshal %T to JSON", value)
	var actual, wanted Value
	require.NoError(t, json.Unmarshal(encoded, &actual), "failed to unmarshal JSON back to %T", value)
	require.NoError(t, json.Unmarshal([]byte(expected), &wanted), "failed to unmarshal the expected JSON to %T", value)
	if !reflect.DeepEqual(actual, wanted) {
		compact := strings.NewReplacer(" ", "", "\n", "", "\t", "").Replace(expected)
		t.Logf("Expected JSON: %s", compact)
		t.Logf("Actual JSON:   %s", string(encoded))
		t.FailNow()
	}
}

// backendBuilder collects the accounts of a test chain.
type backendBuilder struct {
	t        *testing.T
	accounts map[common.Address]tests.Account
}

func newBackendBuilder(t *testing.T) *backendBuilder {
	return &backendBuilder{t: t, accounts: map[common.Address]tests.Account{}}
}

func (builder *backendBuilder) WithAccount(address common.Address, account tests.Account) *backendBuilder {
	builder.accounts[address] = account
	return builder
}

// testBackend is a chain of one block over the state of the accounts. Its
// head has no base fee to speak of, as the reference's test chain had none,
// so that any explicit price is enough.
type testBackend struct {
	harness *tests.Harness
	head    *evmcore.EvmBlock
}

func (builder *backendBuilder) Build() *testBackend {
	carmenCore := tests.NewCarmenCore(builder.t, builder.accounts)
	head := carmenCore.AppendAndPublish(tests.BlockSpec{BaseFee: new(big.Int)})
	harness := tests.NewHarness(builder.t, carmenCore.FakeCore, func(dispatcher *dispatch.Dispatcher, environment *exec.Environment) {
		Register(dispatcher, environment)
	})
	harness.WaitForHead(head.NumberU64())
	return &testBackend{harness: harness, head: head}
}

func (backend *testBackend) CurrentBlock() *evmcore.EvmBlock {
	return backend.head
}

func (backend *testBackend) ChainID() *big.Int {
	return backend.harness.Environment.ChainID()
}

// bundleAPI calls the handlers of the namespace with Go values.
type bundleAPI struct {
	handler *handlers
}

func newBundleAPI(backend *testBackend) *bundleAPI {
	return &bundleAPI{handler: &handlers{environment: backend.harness.Environment}}
}

func (api *bundleAPI) PrepareBundle(ctx context.Context, proposal RPCExecutionProposal) (*RPCPreparedBundle, error) {
	result, err := api.handler.prepareBundle(ctx, dispatch.Arguments{proposal})
	if err != nil {
		return nil, err
	}
	return result.(*RPCPreparedBundle), nil
}
