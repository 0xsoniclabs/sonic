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

package api

import (
	"math/big"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/api/legacy"
	"github.com/0xsoniclabs/sonic/api/testapi"
)

// backends serves the backend of every API at once.
type backends struct {
	legacy.Backend
	test testapi.Backend
}

func (b backends) IsTestOnlyApiEnabled() bool { return b.test.IsTestOnlyApiEnabled() }

func (b backends) ProposeTransactions(txs types.Transactions) error {
	return b.test.ProposeTransactions(txs)
}

func (b backends) AddTransactions(txs types.Transactions) error {
	return b.test.AddTransactions(txs)
}

// TestGetAPIs_ExposesExactlyTheLegacyMethods pins the methods the node's own
// handler serves: everything else is served by the RPC module, so a method that
// shows up here by accident would be callable on the IPC socket without being
// listed anywhere.
func TestGetAPIs_ExposesExactlyTheLegacyMethods(t *testing.T) {
	ctrl := gomock.NewController(t)
	node := legacy.NewMockBackend(ctrl)
	node.EXPECT().AccountManager().Return(accounts.NewManager(&accounts.Config{})).AnyTimes()
	node.EXPECT().ChainID().Return(big.NewInt(1)).AnyTimes()
	b := backends{node, testapi.NewMockBackend(ctrl)}

	var methods []string
	for _, api := range GetAPIs(b) {
		typ := reflect.TypeOf(api.Service)
		for i := 0; i < typ.NumMethod(); i++ {
			name := typ.Method(i).Name
			methods = append(methods, api.Namespace+"_"+strings.ToLower(name[:1])+name[1:])
		}
	}
	sort.Strings(methods)

	require.Equal(t, []string{
		"abft_getDowntime", "abft_getEpochUptime", "abft_getOriginatedEpochFee", "abft_getValidators",
		"dag_getEvent", "dag_getEventPayload", "dag_getHeads",
		"debug_chaindbCompact", "debug_chaindbProperty", "debug_setHead",
		"eth_accounts", "eth_fillTransaction", "eth_getEpochBlock", "eth_getEpochStats", "eth_resend",
		"eth_sendTransaction", "eth_sign", "eth_signTransaction",
		"personal_deriveAccount", "personal_ecRecover", "personal_importRawKey", "personal_initializeWallet",
		"personal_listAccounts", "personal_listWallets", "personal_lockAccount", "personal_newAccount",
		"personal_openWallet", "personal_sendTransaction", "personal_sign", "personal_signAndSendTransaction",
		"personal_signTransaction", "personal_unlockAccount", "personal_unpair",
		"test_addTransactions", "test_proposeTransactions",
	}, methods)
}
