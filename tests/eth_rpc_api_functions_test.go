// Copyright 2025 Sonic Operations Ltd
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
	"strings"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/0xsoniclabs/sonic/vecmt"
	"github.com/Fantom-foundation/lachesis-base/abft"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/rpc/rpc_test_utils"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// Known missing APIs which are not implemented in Sonic
var (
	knownMissingAPIs = namespaceMap{
		"eth": {
			"SimulateV1": struct{}{},
		},
		"debug": {
			"DbAncient":                   struct{}{},
			"DbAncients":                  struct{}{},
			"DbGet":                       struct{}{},
			"GetRawBlock":                 struct{}{},
			"GetRawHeader":                struct{}{},
			"GetRawReceipts":              struct{}{},
			"GetRawTransaction":           struct{}{},
			"IntermediateRoots":           struct{}{},
			"StandardTraceBadBlockToFile": struct{}{},
			"StandardTraceBlockToFile":    struct{}{},
			"TraceBadBlock":               struct{}{},
			"TraceBlock":                  struct{}{},
			"TraceBlockFromFile":          struct{}{},
			"TraceChain":                  struct{}{},
		},
	}
)

type namespaceMap map[string]map[string]interface{}

// TestRPCApis checks if all go-ethereum RPC APIs are implemented in Sonic
func TestRPCApis(t *testing.T) {
	ethAPIs := parseAPIs(rpc_test_utils.GetRpcApis())
	sonicAPIs := parseAPIs(getNodeService(t).APIs())

	// look for missing methods which are in go-ethereum and not in Sonic
	missingInSonic := findMissingMethods(ethAPIs, sonicAPIs)

	// look for missing methods which are in Sonic and are not in known missing
	missing := findMissingMethods(missingInSonic, knownMissingAPIs)
	require.Zero(t, len(missing), "missing namespaces %v", missing)
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

	genStore := makefakegenesis.FakeGenesisStoreWithRulesAndStart(
		1,
		utils.ToFtm(genesisBalance),
		utils.ToFtm(genesisStake),
		rules,
		1,
		2,
	)
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

// findMissingMethods returns a map of namespaces and missing methods
// all methods in `a` are present in `b` otherwise they are returned
func findMissingMethods(a, b namespaceMap) namespaceMap {
	missing := make(namespaceMap)

	for outerKey, innerMap := range a {
		for innerKey, value := range innerMap {
			if _, exists := b[outerKey][innerKey]; !exists {
				if missing[outerKey] == nil {
					missing[outerKey] = make(map[string]interface{})
				}
				missing[outerKey][innerKey] = value
			}
		}
	}
	return missing
}

// parseAPIs returns a map of namespaces and methods
func parseAPIs(apis []rpc.API) namespaceMap {
	namespaces := make(map[string]map[string]interface{})

	for _, api := range apis {
		if _, exists := namespaces[api.Namespace]; !exists {
			namespaces[api.Namespace] = make(map[string]interface{})
		}
		pt := reflect.TypeOf(api.Service)
		for i := range pt.NumMethod() {
			method := pt.Method(i)
			namespaces[api.Namespace][method.Name] = true
		}
	}
	return namespaces
}

func (nm namespaceMap) String() string {
	var sb strings.Builder
	sb.WriteString("{\n")
	for key, innerMap := range nm {
		sb.WriteString(fmt.Sprintf("  \"%s\": [", key))
		funcs := []string{}
		for innerKey := range innerMap {
			funcs = append(funcs, fmt.Sprintf("\"%s\"", innerKey))
		}
		sb.WriteString(strings.Join(funcs, ", "))
		sb.WriteString("],\n")
	}
	sb.WriteString("}")
	return sb.String()
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
	activationTime := block1.Header().Time

	upgradeHeight := opera.UpgradeHeight{
		Upgrades: opera.GetBrioUpgrades(),
		Height:   1,
	}
	genesisId := net.GetGenesisId()
	expectedForkId, err := ethapi.MakeForkId(upgradeHeight, genesisId)
	require.NoError(t, err, "could not make expected fork ID")

	want := map[string]map[string]any{
		"current": {
			"activationTime": activationTime,
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
	var rules opera.Rules
	err = client.Client().Call(&rules, "eth_getRules", "latest")
	require.NoError(t, err, "eth_getRules failed")

	rules.Upgrades.GasSubsidies = true
	UpdateNetworkRules(t, net, rules)
	AdvanceEpochAndWaitForBlocks(t, net)

	// get current block to confirm epoch advancement
	currentBlock, err := client.BlockByNumber(t.Context(), nil)
	require.NoError(t, err, "could not get current block after epoch advancement")
	require.Greater(t, currentBlock.NumberU64(), block1.NumberU64(), "block number did not advance after epoch advancement")

	WaitForProofOf(t, client, int(currentBlock.NumberU64()))

	// get new config
	err = client.Client().Call(&response, "eth_config")
	require.NoError(t, err, "eth_config failed")

	// again, activation time needs to be reinterpreted as a uint64
	response["last"]["activationTime"] = uint64(response["last"]["activationTime"].(float64))
	require.Equal(t, want["current"], response["last"],
		"original config should be in 'last' field")

	require.Contains(t, response["current"], "systemContracts")
	require.Contains(t, response["current"]["systemContracts"], "GAS_SUBSIDY_REGISTRY_ADDRESS")

	foundActivationBlock := false
	activationBlockNumber := idx.Block(0)
	for i := currentBlock.NumberU64(); i > 0 || !foundActivationBlock; i-- {
		// get previous block to determine expected activation time
		previousBlock, err := client.BlockByNumber(t.Context(), big.NewInt(int64(i)))
		require.NoError(t, err, "could not get previous block after epoch advancement")

		if previousBlock.Header().Time == uint64(response["current"]["activationTime"].(float64)) {
			foundActivationBlock = true
			activationBlockNumber = idx.Block(previousBlock.NumberU64())
			break
		}
	}

	upgradeHeight = opera.UpgradeHeight{
		Upgrades: rules.Upgrades,
		Height:   activationBlockNumber,
	}
	expectedForkId, err = ethapi.MakeForkId(upgradeHeight, genesisId)
	require.NoError(t, err, "could not make expected fork ID after upgrade")

	require.Equal(t, fmt.Sprintf("0x%x", expectedForkId), response["current"]["forkId"],
		"fork ID after upgrade is incorrect")
	require.True(t, foundActivationBlock, "should have found block with the same timestamp as the activation time")
}
