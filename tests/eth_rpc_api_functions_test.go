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

package tests

import (
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	rpceth "github.com/0xsoniclabs/sonic/rpc/eth"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/0xsoniclabs/sonic/vecmt"
	"github.com/Fantom-foundation/lachesis-base/abft"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/node"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// nodeHandlerMethods is the exact method set of the node's own handler: the
// legacy namespaces X1 and X2 of rpc/PLAN.md, mirrored as ftm where they are
// eth methods, the test namespace and the p2p facts. Everything else is served
// by the RPC module; nothing reaches IPC unlisted.
var nodeHandlerMethods = []string{
	"abft_getDowntime", "abft_getEpochUptime", "abft_getOriginatedEpochFee", "abft_getValidators",
	"dag_getEvent", "dag_getEventPayload", "dag_getHeads",
	"debug_chaindbCompact", "debug_chaindbProperty", "debug_setHead",
	"eth_accounts", "eth_fillTransaction", "eth_getEpochBlock", "eth_getEpochStats", "eth_resend",
	"eth_sendTransaction", "eth_sign", "eth_signTransaction",
	"ftm_accounts", "ftm_fillTransaction", "ftm_getEpochBlock", "ftm_getEpochStats", "ftm_resend",
	"ftm_sendTransaction", "ftm_sign", "ftm_signTransaction",
	"net_listening", "net_peerCount", "net_version",
	"personal_deriveAccount", "personal_ecRecover", "personal_importRawKey", "personal_initializeWallet",
	"personal_listAccounts", "personal_listWallets", "personal_lockAccount", "personal_newAccount",
	"personal_openWallet", "personal_sendTransaction", "personal_sign", "personal_signAndSendTransaction",
	"personal_signTransaction", "personal_unlockAccount", "personal_unpair",
	"test_addTransactions", "test_proposeTransactions",
}

// TestNodeHandler_ServesExactlyTheLegacyMethods pins the method set of the
// node's own handler (X3).
func TestNodeHandler_ServesExactlyTheLegacyMethods(t *testing.T) {
	var methods []string
	for _, api := range getNodeService(t).APIs() {
		pt := reflect.TypeOf(api.Service)
		for i := range pt.NumMethod() {
			name := pt.Method(i).Name
			methods = append(methods, api.Namespace+"_"+strings.ToLower(name[:1])+name[1:])
		}
	}
	slices.Sort(methods)
	require.Equal(t, nodeHandlerMethods, methods)
}

// getNodeService returns a gossip service
// which includes initialization of RPC APIs for Sonic
func getNodeService(t *testing.T) *gossip.Service {
	node, err := node.New(&node.Config{})
	require.NoError(t, err)

	store, err := gossip.NewMemStore(&testing.B{})
	require.NoError(t, err)

	rules := opera.FakeNetRules(opera.GetSonicUpgrades())
	rules.Epochs.MaxEpochDuration = inter.Timestamp(maxEpochDuration)
	rules.Emitter.Interval = 0

	genStore, err := makefakegenesis.FakeGenesisStoreWithRulesAndStart(
		1,
		utils.ToFtmU256(genesisBalance),
		utils.ToFtmU256(genesisStake),
		rules,
		1,
		2,
		t.TempDir(),
	)
	require.NoError(t, err)
	genesis := genStore.Genesis()

	err = store.ApplyGenesis(genesis)
	require.NoError(t, err)

	engine, vecClock := makeTestEngine(store)

	feed := event.Feed{}
	mockCtrl := gomock.NewController(t)
	txPoolMock := gossip.NewMockTxPool(mockCtrl) //prompt.NewMockUserPrompter(mockCtrl)
	txPoolMock.EXPECT().SubscribeNewTxsNotify(gomock.Any()).AnyTimes().Return(feed.Subscribe(make(chan evmcore.NewTxsNotify)))

	cacheRatio := cachescale.Ratio{
		Base:   uint64(config.DefaultCacheSize*1 - config.ConstantCacheSize),
		Target: uint64(config.DefaultCacheSize*2 - config.ConstantCacheSize),
	}

	defaultConfig := gossip.DefaultConfig(cacheRatio)
	s, err := gossip.NewService(node, defaultConfig, store, gossip.BlockProc{}, engine, vecClock, func(_ evmcore.StateReader) gossip.TxPool {
		return txPoolMock
	}, nil)
	require.NoError(t, err)
	return s
}

const (
	genesisBalance   = 1e18
	genesisStake     = 2 * 4e6
	maxEpochDuration = time.Hour
)

// makeTestEngine creates test engine
func makeTestEngine(gdb *gossip.Store) (*abft.Lachesis, *vecmt.Index) {
	cdb := abft.NewMemStore()
	_ = cdb.ApplyGenesis(&abft.Genesis{
		Epoch:      gdb.GetEpoch(),
		Validators: gdb.GetValidators(),
	})
	vecClock := vecmt.NewIndex(nil, vecmt.LiteConfig())
	engine := abft.NewLachesis(cdb, nil, nil, nil, abft.LiteConfig())
	return engine, vecClock
}

func TestEthConfig_ProducesReadableConfig(t *testing.T) {

	net := StartIntegrationTestNet(t,
		IntegrationTestNetOptions{
			Upgrades: AsPointer(opera.GetBrioUpgrades()),
		})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	response := map[string]map[string]any{}
	err = client.Client().Call(&response, "eth_config")
	require.NoError(t, err, "eth_config failed")
	response["current"]["activationTime"] = uint64(response["current"]["activationTime"].(float64))

	// get time from block one because genesis block has time zero
	// and upgrades gets updated in block 1
	block1, err := client.BlockByNumber(t.Context(), big.NewInt(1))
	require.NoError(t, err, "could not get block 1 to determine expected ActivationTime")

	upgradeHeight := opera.UpgradeHeight{
		Upgrades: opera.GetBrioUpgrades(),
		Height:   1,
	}
	genesisId := net.GetGenesisId()
	expectedForkId, err := rpceth.MakeForkId(upgradeHeight, genesisId)
	require.NoError(t, err, "could not make expected fork ID")

	want := map[string]map[string]any{
		"current": {
			"activationTime": block1.Time(),
			"blockHeight":    fmt.Sprintf("0x%x", block1.Number().Uint64()),
			"blobSchedule":   nil,
			"chainId":        "0xfa3",
			"forkId":         fmt.Sprintf("0x%x", expectedForkId),
			"precompiles": map[string]any{
				"BLAKE2F":              "0x0000000000000000000000000000000000000009",
				"BN254_ADD":            "0x0000000000000000000000000000000000000006",
				"BN254_MUL":            "0x0000000000000000000000000000000000000007",
				"BN254_PAIRING":        "0x0000000000000000000000000000000000000008",
				"ECREC":                "0x0000000000000000000000000000000000000001",
				"ID":                   "0x0000000000000000000000000000000000000004",
				"KZG_POINT_EVALUATION": "0x000000000000000000000000000000000000000a",
				"MODEXP":               "0x0000000000000000000000000000000000000005",
				"RIPEMD160":            "0x0000000000000000000000000000000000000003",
				"SHA256":               "0x0000000000000000000000000000000000000002",
				"BLS12_G1ADD":          "0x000000000000000000000000000000000000000b",
				"BLS12_G1MSM":          "0x000000000000000000000000000000000000000c",
				"BLS12_G2ADD":          "0x000000000000000000000000000000000000000d",
				"BLS12_G2MSM":          "0x000000000000000000000000000000000000000e",
				"BLS12_MAP_FP2_TO_G2":  "0x0000000000000000000000000000000000000011",
				"BLS12_MAP_FP_TO_G1":   "0x0000000000000000000000000000000000000010",
				"BLS12_PAIRING_CHECK":  "0x000000000000000000000000000000000000000f",
				"P256VERIFY":           "0x0000000000000000000000000000000000000100",
			},
			"systemContracts": map[string]any{"HISTORY_STORAGE_ADDRESS": "0x0000f90827f1c53a10cb7a02335b175320002935"},
		},
		"next": nil,
		"last": nil,
	}
	require.Equal(t, want, response, "eth_config returned unexpected result")

	originalForkId := response["current"]["forkId"]
	require.NotZero(t, originalForkId, "forkId should not be zero")

	// get current rules and change to single proposer mode
	var originalRules opera.Rules
	err = client.Client().Call(&originalRules, "eth_getRules", "latest")
	require.NoError(t, err, "eth_getRules failed")

	originalRules.Upgrades.GasSubsidies = true
	UpdateNetworkRules(t, net, originalRules)
	net.AdvanceEpoch(t, 1)

	// get new config
	err = client.Client().Call(&response, "eth_config")
	require.NoError(t, err, "eth_config failed")

	// again, activation time needs to be reinterpreted as a uint64
	response["last"]["activationTime"] = uint64(response["last"]["activationTime"].(float64))
	require.Equal(t, want["current"], response["last"],
		"original config should be in 'last' field")

	require.Contains(t, response["current"], "systemContracts")
	require.Contains(t, response["current"]["systemContracts"], "GAS_SUBSIDY_REGISTRY_ADDRESS")

	// reinterpret block height from string to uint64
	heightString := response["current"]["blockHeight"].(string)
	heightUint, err := hexutil.DecodeUint64(heightString)
	require.NoError(t, err, "could not decode block height from hex string")

	// get current block to confirm epoch advancement
	activationBlock, err := client.HeaderByNumber(t.Context(), big.NewInt(int64(heightUint)))
	require.NoError(t, err)
	require.Equal(t, activationBlock.Time, uint64(response["current"]["activationTime"].(float64)),
		"activation block time should be greater than the activation time in config")

	upgradeHeight = opera.UpgradeHeight{
		Upgrades: originalRules.Upgrades,
		Height:   idx.Block(heightUint),
	}
	expectedForkId, err = rpceth.MakeForkId(upgradeHeight, genesisId)
	require.NoError(t, err, "could not make expected fork ID after upgrade")

	require.Equal(t, fmt.Sprintf("0x%x", expectedForkId), response["current"]["forkId"],
		"fork ID after upgrade is incorrect")
}
