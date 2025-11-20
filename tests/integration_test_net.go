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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sonicd "github.com/0xsoniclabs/sonic/cmd/sonicd/app"
	sonictool "github.com/0xsoniclabs/sonic/cmd/sonictool/app"
	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/contract/driverauth100"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	geth_crypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// IntegrationTestNetSession a collection of methods to run tests against the
// integration test network.
// It provides the methods to launch transactions and queries against the network.
// Additionally, it provides the methods to endow accounts with funds.
type IntegrationTestNetSession interface {
	// GetUpgrades returns the upgrades the network has been started with.
	GetUpgrades() opera.Upgrades

	// EndowAccount sends a requested amount of tokens to the given account. This is
	// mainly intended to provide funds to accounts for testing purposes.
	EndowAccount(address common.Address, value *big.Int) (*types.Receipt, error)

	// EndowAccounts sends the requested amount of tokens to each of the given
	// accounts. This is a faster than calling EndowAccount for each account since
	// multiple endowments may get bundled in the same block.
	EndowAccounts(addresses []common.Address, value *big.Int) ([]*types.Receipt, error)

	// Run sends the given transaction to the network and waits for it to be processed.
	// The resulting receipt is returned.
	Run(tx *types.Transaction) (*types.Receipt, error)

	// RunAll sends the given transactions to the network and waits for them to be processed.
	// The resulting receipts are returned.
	RunAll(tx []*types.Transaction) ([]*types.Receipt, error)

	// GetReceipt waits for the receipt of the given transaction hash to be available.
	GetReceipt(txHash common.Hash) (*types.Receipt, error)

	// GetReceipts waits for the receipts of the given transaction hashes to be available.
	GetReceipts(txHash []common.Hash) ([]*types.Receipt, error)

	// GetTransactOptions provides transaction options to be used to send a transaction
	// from the given account.
	GetTransactOptions(account *Account) (*bind.TransactOpts, error)

	// Apply sends a transaction to the network using the session account.
	// and waits for the transaction to be processed. The resulting receipt is returned.
	Apply(issue func(*bind.TransactOpts) (*types.Transaction, error)) (*types.Receipt, error)

	// GetSessionSponsor returns the default account of the session. This account is used
	// to sign transactions and pay for gas when using the Apply and EndowAccount methods.
	GetSessionSponsor() *Account

	// GetClient provides raw access to a connection to the network.
	// The resulting client must be closed after use.
	GetClient() (*PooledEhtClient, error)

	// GetChainId returns the chain ID of the network.
	GetChainId() *big.Int

	// SpawnSession creates a new test session on the network based from the
	// network's sponsor account. This should be done before entering a new
	// parallel context to prevent conflicting nonces inside.
	SpawnSession(t *testing.T) IntegrationTestNetSession

	// GetWebSocketClient provides raw access to a fresh connection to the network
	// The resulting client must be closed after use.
	// This function does not returned a PooledEthClient, because they need to
	// be kept apart since their behavior is different.
	GetWebSocketClient() (*ethClient, error)

	// NumNodes returns the number of nodes in the test network.
	NumNodes() int

	// GetClientConnectedToNode returns a client connected to the specified node
	// in the test network.
	GetClientConnectedToNode(node int) (*PooledEhtClient, error)

	// GetGenesisJson returns the genesis JSON used to start the network.
	GetGenesisId() common.Hash
}

// AsPointer is a utility function that returns a pointer to the given value.
// Useful to initialize values which nil value is semantically significant. e.g. to
// initialize the `Upgrades` field in `IntegrationTestNetOptions` to a non-nil value.
func AsPointer[T any](v T) *T {
	return &v
}

// IntegrationTestNetOptions are configuration options for the integration test network.
type IntegrationTestNetOptions struct {
	// Upgrades specifies the upgrades to be used for the integration test network.
	// nil value will initialize network using SonicUpgrades.
	Upgrades *opera.Upgrades
	// NumNodes specifies the number of nodes to be started on the integration
	// test network.
	// This setting is only used by the JSON genesis procedure, fake genesis will ignore it
	// and execute a single node network.
	// If NumNodes is not defined, it will be set to the length of ValidatorsStake if that is defined
	// otherwise it will be set to 1.
	NumNodes int
	// ValidatorsStake specifies the stake of each validator in the network in sonics.
	// This setting is only used by the JSON genesis procedure, fake genesis will ignore it
	// and execute a single node network.
	// If NumNodes is defined, ValidatorsStake must have the same length as NumNodes.
	// If ValidatorsStake is not defined, NumNodes validators will be created with equal stake.
	ValidatorsStake []uint64
	// ClientExtraArguments specifies additional arguments to be passed to the client.
	ClientExtraArguments []string
	// ModifyConfig allows the caller to modify the configuration of the nodes
	// on the integration test network. This modified configuration will be saved
	// as a toml file and loaded by the nodes when they are started.
	// Please read carefully the config type declaration, config fields with tag `-`
	// will not be saved into the toml file, modifications will be ignored.
	// Zero value means no modification.
	ModifyConfig func(*config.Config)
	// Accounts to be deployed with the genesis.
	Accounts []makefakegenesis.Account
	// SkipCleanUp indicates whether the network should add its stop function
	// to t.Cleanup or not.
	SkipCleanUp bool
}

// IntegrationTestNet is a in-process test network for integration tests. When
// started, it runs full Sonic nodes maintaining a chain within the process
// containing this object. The network can be used to run transactions on and
// to perform queries against.
//
// The main purpose of this network is to facilitate end-to-end debugging of
// client code in the controlled scope of individual unit tests. When running
// tests against an integration test network instance, break-points can be set
// in the client code, thereby facilitating debugging.
//
// A typical use case would look as follows:
//
//	func TestMyClientCode(t *testing.T) {
//	  net := StartIntegrationTestNet(t)
//	  <run tests against the network>
//	}
//
// Additionally, by providing support for scripting test traffic on a network,
// integration test networks can also be used for automated integration and
// regression tests for client code.
type IntegrationTestNet struct {
	options   IntegrationTestNetOptions
	genesis   *makefakegenesis.GenesisJson
	genesisId common.Hash
	nodes     []integrationTestNode

	sessionsMutex sync.Mutex
	Session
}

// per-node state for the integration test network
type integrationTestNode struct {
	directory string
	httpPort  int
	shutdown  chan<- struct{}
	done      <-chan struct{}

	clients *sync.Pool
}

////////////////////////////////////////////////////////////////////////////////
// Memory profiler.
// if enabled with the `SONIC_TEST_HEAP_PROFILE` env var, it will write a heap dump to
// the `../build/profile/` directory at the end of the test run.
// The file will be named `mem_<test_name>.pprof` where `<test_name>` is the name
// of the test that started the profiling.
// The environment variable can be set to `1`, `on`, or `true` (regardless of case)
// to enable the profiling.
////////////////////////////////////////////////////////////////////////////////

const heapProfileEnvVar = "SONIC_TEST_HEAP_PROFILE"

// startHeapProfiler starts a goroutine that periodically checks the heap memory
// usage and at the end of the test, writes a heap profile to a file in
// `../build/profile/` directory.
func startHeapProfiler(tb testing.TB) {

	heapProfile := os.Getenv(heapProfileEnvVar)
	if heapProfile != "1" &&
		!strings.EqualFold(heapProfile, "on") &&
		!strings.EqualFold(heapProfile, "true") {
		return
	}

	go func() {

		// highest memory usage seen so far,
		// used to write only the peak consumption to a file
		highestSeen := uint64(0)
		ctx := tb.Context()

		buffer := bytes.NewBuffer(nil)
		memStats := &runtime.MemStats{}

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				runtime.ReadMemStats(memStats)
				if memStats.HeapAlloc <= highestSeen {
					continue
				}
				buffer.Reset()
				highestSeen = memStats.HeapAlloc
				require.NoError(tb, pprof.WriteHeapProfile(buffer))

			case <-ctx.Done():
				// write a file with the name of the test case that started the profiling
				buildProfile := "../build/profile/"
				require.NoError(tb, os.MkdirAll(buildProfile, os.ModeDir|os.ModePerm),
					"Failed to create profile directory")

				fileName := strings.ReplaceAll(tb.Name(), "/", "_")
				fileName = filepath.Join(buildProfile, fmt.Sprintf("mem_%v.pprof", fileName))

				require.NoError(tb, os.WriteFile(fileName, buffer.Bytes(), 0644))
				return
			}
		}
	}()
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

// StartIntegrationTestNet starts a single-node test network for integration tests.
// The node serving the network is started in the same process as the caller. This
// is intended to facilitate debugging of client code in the context of a running
// node.
//
// The network start procedure will create a temporary directory and populate with
// a network genesis block. To retrieve the directory path, use the GetDirectory.
func StartIntegrationTestNet(
	t *testing.T,
	options ...IntegrationTestNetOptions,
) *IntegrationTestNet {
	t.Helper()
	return StartIntegrationTestNetWithJsonGenesis(t, options...)
}

// StartIntegrationTestNetWithFakeGenesis starts a single-node test network for
// integration tests using the Fake-Genesis procedure. The fake genesis procedure
// is mainly intended for demo and small scale test networks used for development
// and integration testing in Norma.
func StartIntegrationTestNetWithFakeGenesis(
	t *testing.T,
	options ...IntegrationTestNetOptions,
) *IntegrationTestNet {
	t.Helper()

	effectiveOptions, err := validateAndSanitizeOptions(options...)
	require.NoError(t, err, "failed to validate and sanitize options")
	require.Equal(t, 0, len(effectiveOptions.Accounts),
		"fake genesis does not support custom accounts")

	var upgrades string
	if *effectiveOptions.Upgrades == opera.GetSonicUpgrades() {
		upgrades = "sonic"
	} else if *effectiveOptions.Upgrades == opera.GetAllegroUpgrades() {
		upgrades = "allegro"
	} else {
		t.Fatal("fake genesis only supports sonic and allegro feature sets")
	}

	numNodesString := fmt.Sprintf("%d", effectiveOptions.NumNodes)

	net, err := startIntegrationTestNet(
		t,
		t.TempDir(),
		[]string{"genesis", "fake", numNodesString, "--upgrades", upgrades},
		effectiveOptions,
	)
	require.NoError(t, err, "failed to start integration test network with fake genesis")

	return net
}

// StartIntegrationTestNetWithJsonGenesis starts a single-node test network for
// integration tests using the JSON-Genesis procedure. The JSON genesis procedure
// is the genesis procedure used in long-running production networks like the
// Sonic mainnet and the testnet.
func StartIntegrationTestNetWithJsonGenesis(
	t *testing.T,
	options ...IntegrationTestNetOptions,
) *IntegrationTestNet {
	t.Helper()

	effectiveOptions, err := validateAndSanitizeOptions(options...)
	require.NoError(t, err, "failed to validate and sanitize options")

	jsonGenesis := makefakegenesis.GenerateFakeJsonGenesis(
		*effectiveOptions.Upgrades,
		effectiveOptions.ValidatorsStake,
	)

	jsonGenesis.Accounts = append(jsonGenesis.Accounts, effectiveOptions.Accounts...)

	// Speed up the block generation time to reduce test time.
	jsonGenesis.Rules.Emitter.Interval = inter.Timestamp(time.Millisecond)

	encoded, err := json.MarshalIndent(jsonGenesis, "", "  ")
	require.NoError(t, err, "failed to marshal genesis json")

	directory, err := os.MkdirTemp("", "TestNet")
	require.NoError(t, err, "failed to create test directory")

	jsonFile := filepath.Join(directory, "genesis.json")
	err = os.WriteFile(jsonFile, encoded, 0644)
	require.NoError(t, err, "failed to write genesis json file")

	net, err := startIntegrationTestNet(
		t,
		directory,
		[]string{"genesis", "json", "--experimental", jsonFile},
		effectiveOptions,
	)
	require.NoError(t, err, "failed to start integration test network with json genesis")

	net.genesis = jsonGenesis
	net.genesisId, err = makefakegenesis.GetGenesisIdFromJson(jsonGenesis)
	require.NoError(t, err, "failed to get genesis ID from json genesis")

	return net
}

func startIntegrationTestNet(
	t *testing.T,
	directory string,
	sonicToolArguments []string,
	options IntegrationTestNetOptions,
) (*IntegrationTestNet, error) {
	net := &IntegrationTestNet{
		options: options,
		Session: Session{
			account: Account{evmcore.FakeKey(1)},
		},
		nodes: make([]integrationTestNode, len(options.ValidatorsStake)),
	}
	// the network's session needs to know about the network itself
	net.net = net

	startHeapProfiler(t)

	if verbosityVariable := os.Getenv("SONIC_VERBOSITY"); verbosityVariable == "" {
		require.NoError(t, os.Setenv("SONIC_VERBOSITY", "1"), "failed to set verbosity")
	}

	// start the integration test nodes
	for i := range net.nodes {
		net.nodes[i].directory = filepath.Join(directory, fmt.Sprintf("node%d", i))

		// initialize the data directory for the single node on the test network
		// using the configuration arguments provided by the caller
		args := append([]string{
			"sonictool",
			"--datadir", net.nodes[i].getStateDir(),
			"--statedb.livecache", "1",
			"--statedb.archivecache", "1",
			"--statedb.cache", "1024",
		}, sonicToolArguments...)
		require.NoError(t, sonictool.RunWithArgs(args), "failed to initialize the test network")
	}

	require.NoError(t, net.start(), "failed to start the integration test network")

	if !options.SkipCleanUp {
		t.Cleanup(net.Stop)
	}
	return net, nil
}

func (n *integrationTestNode) getStateDir() string {
	return filepath.Join(n.directory, "state")
}

func (n *IntegrationTestNet) start() error {
	if n.nodes[0].done != nil {
		return errors.New("network already started")
	}

	nodeIds := make([]chan string, len(n.nodes))
	httpPorts := make([]chan string, len(n.nodes))
	for i := range nodeIds {
		nodeIds[i] = make(chan string, 1)
		httpPorts[i] = make(chan string, 1)
	}

	for i := range n.nodes {
		stop := make(chan struct{})
		done := make(chan struct{})
		n.nodes[i].shutdown = stop
		n.nodes[i].done = done
		go func() {
			defer close(done)

			// MacOS uses other temporary directories than Linux, which is a too long name for the Unix domain socket.
			// Since /tmp is also available on MacOS, we can use it as a short temporary directory.
			tmp, err := os.MkdirTemp("/tmp", "sonic_integration_test_*")
			if err != nil {
				panic(fmt.Sprintf("Failed to create temporary directory: %v", err))
			}
			defer func() {
				if err := os.RemoveAll(tmp); err != nil {
					fmt.Printf("Failed to remove temporary directory: %v\n", err)
				}
			}()

			// start the fakenet sonic node
			// equivalent to running `sonicd ...` but in this local process
			args := append([]string{
				"sonicd",

				// data storage options
				"--datadir", n.nodes[i].getStateDir(),
				"--datadir.minfreedisk", "0",

				// fake network options
				"--fakenet", fmt.Sprintf("%d/%d", i+1, len(n.nodes)),

				// http-client option
				"--http", "--http.addr", "127.0.0.1", "--http.port", "0",
				"--http.api", "admin,eth,dag,web3,net,txpool,trace,debug,sonic",

				// websocket-client options
				"--ws", "--ws.addr", "127.0.0.1", "--ws.port", "0",
				"--ws.api", "admin,eth",

				//  net options
				"--port", "0",
				"--nat", "none",
				"--nodiscover",

				// database memory usage options
				"--statedb.livecache", "1",
				"--statedb.archivecache", "1",
				"--statedb.cache", "1024",

				"--ipcpath", fmt.Sprintf("%s/sonic.ipc", tmp),
			},
				// append extra arguments
				n.options.ClientExtraArguments...,
			)

			if n.options.ModifyConfig != nil {
				configFile := filepath.Join(tmp, "config.toml")
				if err := sonicd.RunWithArgs(append(args, "--dump-config", configFile), nil); err != nil {
					panic(fmt.Sprint("Failed to dump config file:", err))
				}
				var loadedConfig config.Config
				if err := config.LoadAllConfigs(configFile, &loadedConfig); err != nil {
					panic(fmt.Sprint("Failed to load default config file:", err))
				}
				n.options.ModifyConfig(&loadedConfig)
				if err := config.SaveAllConfigs(configFile, &loadedConfig); err != nil {
					panic(fmt.Sprint("Failed to save modified config file:", err))
				}
				args = append(args, "--config", configFile)
			}

			control := &sonicd.AppControl{
				NodeIdAnnouncement:   nodeIds[i],
				HttpPortAnnouncement: httpPorts[i],
				Shutdown:             stop,
			}

			if err := sonicd.RunWithArgs(args, control); err != nil {
				panic(fmt.Sprint("Failed to start the fake network:", err))
			}
		}()
	}

	// Collect all enode IDs and HTTP ports.
	endPointPattern := regexp.MustCompile(`^http://.*:(\d+)$`)
	enodes := make([]string, len(n.nodes))
	for i := range n.nodes {
		id, ok := <-nodeIds[i]
		if !ok {
			return fmt.Errorf("failed to start the network, no ID announced for node %d", i)
		}
		enodes[i] = id
		endpoint, ok := <-httpPorts[i]
		if !ok {
			return fmt.Errorf("failed to start the network, no HTTP port announced for node %d", i)
		}

		// Extract the HTTP port form the endpoint string.
		match := endPointPattern.FindStringSubmatch(endpoint)
		if len(match) != 2 {
			return fmt.Errorf("failed to parse the HTTP endpoint: %s", endpoint)
		}
		httpPort, err := strconv.Atoi(match[1])
		if err != nil {
			return fmt.Errorf("failed to parse the HTTP port %s: %w", endpoint, err)
		}
		n.nodes[i].httpPort = httpPort

		n.nodes[i].clients = &sync.Pool{
			New: func() any {
				client, err := ethclient.Dial(fmt.Sprintf("http://localhost:%d", n.nodes[i].httpPort))
				if err != nil {
					return nil
				}
				sharedClient := PooledEhtClient{*client, n.nodes[i].clients}
				return &sharedClient
			},
		}
	}

	// Wait for all nodes to be ready to serve requests
	for i := range n.nodes {
		err := WaitFor(context.Background(), func(ctx context.Context) (bool, error) {
			client, err := n.GetClientConnectedToNode(i)
			if err != nil {
				return false, fmt.Errorf("failed to connect to the Ethereum client: %w", err)
			}
			defer client.Close()

			_, err = client.ChainID(ctx)
			return err == nil, nil
		})
		if err != nil {
			return fmt.Errorf("failed to connect to the Ethereum client: %w", err)
		}
	}

	// Connect the nodes P2P network together
	if err := n.connectP2PNetwork(enodes); err != nil {
		return fmt.Errorf("failed to connect P2P network: %w", err)
	}

	return nil
}

// connectP2PNetwork connects all nodes in the network to each other.
// The current implementation aims to keep the arity of the network low,
// by connecting each node to the next one in the list, and the last one to the first.
// This reduces the amount of duplicated messages generated and improves test stability.
// Regarding latencies, the net is small enough and the local loop is fast enough
// to have latency not be a concern.
func (n *IntegrationTestNet) connectP2PNetwork(enodes []string) error {
	if len(n.nodes) == 1 {
		return nil
	}

	for i := range n.nodes {
		client, err := n.GetClientConnectedToNode(i)
		if err != nil {
			return fmt.Errorf("failed to connect to the Ethereum client: %w", err)
		}
		defer client.Close()

		// Wait until connection is established
		err = WaitFor(context.Background(), func(ctx context.Context) (bool, error) {

			// Connect each node to the next one, and the last one to the first.
			enode := enodes[(i+1)%len(n.nodes)]
			if err := client.Client().Call(nil, "admin_addPeer", enode); err != nil {
				return false, fmt.Errorf("failed to connect to node %d: %v", i, err)
			}

			// Fetch the list of connected peers
			var res []map[string]any
			if err := client.Client().Call(&res, "admin_peers"); err != nil {
				return false, fmt.Errorf("failed to connect to node %d: %v", i, err)
			}

			// Expect each node to be connected to the previous and next nodes,
			// except for the first node which will only be connected to the
			// next at this point in time, and each node in a 2-nodes
			// network which can only have one connection each.
			expectedConnections := 1
			if i > 0 {
				// min is for the 2-nodes network special case
				expectedConnections = min(len(n.nodes)-1, 2)
			}
			return len(res) >= expectedConnections, nil
		})
		if err != nil {
			return fmt.Errorf("failed to wait for node %d to be connected: %v", i, err)
		}
	}
	return nil
}

// Stop shuts the underlying network down.
func (n *IntegrationTestNet) Stop() {
	if n.nodes[0].done == nil {
		return
	}

	// send the stop signal to all nodes
	for i := range n.nodes {
		close(n.nodes[i].shutdown)
		n.nodes[i].shutdown = nil
	}

	// wait for all nodes to be stopped
	for i := range n.nodes {
		<-n.nodes[i].done
		n.nodes[i].done = nil
	}

	// release clients pools
	for i := range n.nodes {
		n.nodes[i].clients = nil
	}

}

// Restart stops and restarts the single node on the test network.
func (n *IntegrationTestNet) Restart() error {
	n.Stop()
	return n.start()
}

// GetJsonGenesis returns the JSON genesis used to start the network, if it was
// started with JSON genesis. If the network was started with fake genesis,
// this method will return nil.
func (n *IntegrationTestNet) GetJsonGenesis() *makefakegenesis.GenesisJson {
	return n.genesis
}

// NumNodes returns the number of nodes on the network.
func (n *IntegrationTestNet) NumNodes() int {
	return len(n.nodes)
}

// GetClient provides raw access to a fresh connection to the network.
// The resulting client must be closed after use.
func (n *IntegrationTestNet) GetClient() (*PooledEhtClient, error) {
	return n.GetClientConnectedToNode(0)
}

// GetChainId returns the chain ID of the network.
func (n *IntegrationTestNet) GetChainId() *big.Int {
	return big.NewInt(int64(opera.FakeNetworkID))
}

// GetClientConnectedToNode provides raw access to a fresh connection to a selected node on
// the network. The resulting client must be closed after use.
func (n *IntegrationTestNet) GetClientConnectedToNode(i int) (*PooledEhtClient, error) {
	if i < 0 || i >= len(n.nodes) {
		return nil, fmt.Errorf("node index out of bounds: %d", i)
	}
	client := n.nodes[i].clients.Get().(*PooledEhtClient)
	if client != nil {
		return client, nil
	}
	ethclient, err := ethclient.Dial(fmt.Sprintf("http://localhost:%d", n.nodes[i].httpPort))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the Ethereum client: %w", err)
	}
	return &PooledEhtClient{*ethclient, n.nodes[i].clients}, nil
}

// GetWebSocketClient provides raw access to a fresh connection to the network
// using the WebSocket protocol. The resulting client must be closed after use.
func (n *IntegrationTestNet) GetWebSocketClient() (*ethclient.Client, error) {
	return ethclient.Dial(fmt.Sprintf("ws://localhost:%d", n.nodes[0].httpPort))
}

func (n *IntegrationTestNet) GetDirectory() string {
	return n.nodes[0].directory
}

// GetJsonRpcPort returns the JSON-RPC port of the first node in the network.
func (n *IntegrationTestNet) GetJsonRpcPort() int {
	return n.nodes[0].httpPort
}

// RestartWithExportImport stops the network, exports the genesis file, cleans the
// temporary directory, imports the genesis file, and starts the network again.
func (n *IntegrationTestNet) RestartWithExportImport() error {
	n.Stop()
	fmt.Println("Network stopped. Exporting genesis file...")

	for _, node := range n.nodes {
		// export
		genesisFile := filepath.Join(node.directory, "testGenesis.g")
		err := sonictool.RunWithArgs([]string{
			"sonictool",
			"--datadir", node.getStateDir(),
			"genesis", "export", genesisFile,
		})
		if err != nil {
			return err
		}

		// clean client state
		err = os.RemoveAll(node.getStateDir())
		if err != nil {
			return err
		}

		fmt.Println("State directory cleaned. Importing genesis file...")

		// import genesis file
		err = sonictool.RunWithArgs([]string{
			"sonictool",
			"--datadir", node.getStateDir(),
			"genesis", "--experimental", genesisFile,
		})
		if err != nil {
			return err
		}
	}

	fmt.Println("Genesis file imported. Restarting network...")

	// start network again
	return n.start()
}

// GetHeaders returns the headers of all blocks on the network from block 0 to the latest block.
func (n *IntegrationTestNet) GetHeaders() ([]*types.Header, error) {
	client, err := n.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the Ethereum client: %w", err)
	}
	defer client.Close()

	lastBlock, err := client.BlockByNumber(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get last block: %w", err)
	}

	headers := []*types.Header{}
	for i := int64(0); i < int64(lastBlock.NumberU64()); i++ {
		header, err := client.HeaderByNumber(context.Background(), big.NewInt(i))
		if err != nil {
			return nil, fmt.Errorf("failed to get header: %w", err)
		}
		headers = append(headers, header)
	}

	return headers, nil
}

// SpawnSession creates a new test session on the network.
// The session is backed by an account which will be used to sign and pay for
// transactions. By using this function, multiple test sessions can be run in
// parallel on the same network, without conflicting nonce issues, since the
// accounts are isolated.
//
// A typical use case would look as follows:
//
//	 net := StartIntegrationTestNet(t)
//		t.Run("test_case",, func(t *testing.T) {
//				t.Parallel()
//				session := net.SpawnSession(t)
//		        < use session instead of net of the rest of the test >
//		})
func (n *IntegrationTestNet) SpawnSession(t *testing.T) IntegrationTestNetSession {
	t.Helper()
	n.sessionsMutex.Lock()
	defer n.sessionsMutex.Unlock()

	key, _ := geth_crypto.GenerateKey()
	nextSessionAccount := Account{
		PrivateKey: key,
	}
	receipt, err := n.EndowAccount(nextSessionAccount.Address(), new(big.Int).SetUint64(math.MaxUint64))
	require.NoError(t, err, "Failed to endow account")
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "Failed to endow account")

	return &Session{
		net:     n,
		account: nextSessionAccount,
	}
}

// AdvanceEpoch trigger the sealing of an epoch and the epoch number to progress by the given number.
// The function blocks until the final epoch has been reached. This method can only be called
// on a validator account.
func (n *IntegrationTestNet) AdvanceEpoch(t testing.TB, epochs int) {
	t.Helper()
	client, err := n.GetClient()
	require.NoError(t, err, "failed to connect to the Ethereum client")
	defer client.Close()

	var currentEpoch hexutil.Uint64
	err = client.Client().Call(&currentEpoch, "eth_currentEpoch")
	require.NoError(t, err, "failed to get current epoch")

	contract, err := driverauth100.NewContract(driverauth.ContractAddress, client)
	require.NoError(t, err, "failed to create contract instance")

	receipt, err := n.Apply(func(ops *bind.TransactOpts) (*types.Transaction, error) {
		return contract.AdvanceEpochs(ops, big.NewInt(int64(epochs)))
	})
	require.NoError(t, err, "failed to advance epoch")
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// wait until the epoch is advanced
	err = WaitFor(t.Context(), func(ctx context.Context) (bool, error) {
		var newEpoch hexutil.Uint64
		if err := client.Client().Call(&newEpoch, "eth_currentEpoch"); err != nil {
			return false, fmt.Errorf("failed to get current epoch: %w", err)
		}
		return newEpoch >= currentEpoch+hexutil.Uint64(epochs), nil
	})
	require.NoError(t, err, "failed to wait for epoch to advance")
}

// DeployContract is a utility function handling the deployment of a contract on the network.
// The contract is deployed with by the network's validator account. The function returns the
// deployed contract instance and the transaction receipt.
func DeployContract[T any](n IntegrationTestNetSession, deploy contractDeployer[T]) (*T, *types.Receipt, error) {
	client, err := n.GetClient()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to the Ethereum client: %w", err)
	}
	defer client.Close()

	transactOptions, err := n.GetTransactOptions(n.GetSessionSponsor())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get transaction options: %w", err)
	}

	// Deployments may comprise more than one transaction, so nonces must be
	// set to nil to allow the client to determine the correct nonce for each
	// transaction.
	transactOptions.Nonce = nil

	// Deployments may also be more expensive than the default transaction.
	transactOptions.GasLimit = 10_000_000

	_, transaction, contract, err := deploy(transactOptions, client)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to deploy contract: %w", err)
	}

	receipt, err := n.GetReceipt(transaction.Hash())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get receipt: %w", err)
	}
	return contract, receipt, nil
}

// contractDeployer is the type of the deployment functions generated by abigen.
type contractDeployer[T any] func(*bind.TransactOpts, bind.ContractBackend) (common.Address, *types.Transaction, *T, error)

// Session is a test session on the network. It is backed by an account which
// will be used to sign and pay for transactions.
// Its purpose is to isolate transaction issuing accounts, so that multiple test
// sessions can be run in parallel on the same network without conflicting nonce issues.
type Session struct {
	net     *IntegrationTestNet
	account Account
}

func (s *Session) SpawnSession(t *testing.T) IntegrationTestNetSession {
	return s.net.SpawnSession(t)
}

func (s *Session) GetUpgrades() opera.Upgrades {
	return *s.net.options.Upgrades
}

// EndowAccount sends a requested amount of tokens to the given account. This is
// mainly intended to provide funds to accounts for testing purposes.
func (s *Session) EndowAccount(
	address common.Address,
	value *big.Int,
) (*types.Receipt, error) {
	receipts, err := s.EndowAccounts([]common.Address{address}, value)
	if err != nil {
		return nil, err
	}
	return receipts[0], nil
}

// EndowAccounts sends the requested amount of tokens to each of the given
// accounts. This is a faster than calling EndowAccount for each account since
// multiple endowments may get bundled in the same block.
func (s *Session) EndowAccounts(
	addresses []common.Address,
	value *big.Int,
) ([]*types.Receipt, error) {
	client, err := s.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the network: %w", err)
	}
	defer client.Close()

	chainId, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	// The requested funds are moved from the validator account to the target account.
	nonce, err := client.PendingNonceAt(context.Background(), s.account.Address())
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	price, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	transactions := make([]*types.Transaction, len(addresses))
	for i, address := range addresses {
		transaction, err := types.SignTx(types.NewTx(&types.AccessListTx{
			ChainID:  chainId,
			Gas:      21000,
			GasPrice: price,
			To:       &address,
			Value:    value,
			Nonce:    nonce,
		}), types.NewLondonSigner(chainId), s.account.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to sign transaction: %w", err)
		}
		transactions[i] = transaction
		nonce++
	}
	return s.RunAll(transactions)
}

// Run sends the given transaction to the network and waits for it to be processed.
// The resulting receipt is returned. This function times out after 10 seconds.
func (s *Session) Run(tx *types.Transaction) (*types.Receipt, error) {
	receipts, err := s.RunAll([]*types.Transaction{tx})
	if err != nil {
		return nil, fmt.Errorf("failed to run transaction: %w", err)
	}
	return receipts[0], nil
}

func (s *Session) RunAll(tx []*types.Transaction) ([]*types.Receipt, error) {
	hashes := make([]common.Hash, len(tx))
	err := runParallelWithClient(s.net, len(tx), func(client *PooledEhtClient, i int) error {
		err := client.SendTransaction(context.Background(), tx[i])
		if err != nil {
			return fmt.Errorf("failed to send transaction %d: %w", i, err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send transactions: %w", err)
	}
	for i, t := range tx {
		hashes[i] = t.Hash()
	}
	return s.GetReceipts(hashes)
}

// GetReceipt waits for the receipt of the given transaction hash to be available.
// The function times out after 10 seconds.
func (s *Session) GetReceipt(txHash common.Hash) (*types.Receipt, error) {
	receipts, err := s.GetReceipts([]common.Hash{txHash})
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt: %w", err)

	}
	return receipts[0], nil
}

func (s *Session) GetReceipts(txHash []common.Hash) ([]*types.Receipt, error) {
	res := make([]*types.Receipt, len(txHash))
	err := runParallelWithClient(
		s.net,
		len(txHash),
		func(client *PooledEhtClient, i int) error {
			hash := txHash[i]

			err := WaitFor(context.Background(), func(ctx context.Context) (bool, error) {
				receipt, err := client.TransactionReceipt(ctx, hash)
				if errors.Is(err, ethereum.NotFound) {
					return false, nil // receipt not yet available, keep waiting
				}
				if err != nil {
					return false, fmt.Errorf("failed to get transaction receipt: %w", err)
				}
				res[i] = receipt
				return true, nil // receipt available, stop waiting
			})
			if err != nil {
				return fmt.Errorf("failed to get transaction receipt: %w", err)
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// runParallelWithClient as a helper function to run a number of jobs in parallel
// where each job requires access to the network through a client.
func runParallelWithClient(
	net IntegrationTestNetSession,
	numJobs int,
	job func(*PooledEhtClient, int) error,
) error {
	numWorkers := max(min(numJobs, 16), 1)
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	workerErrors := make([]error, numWorkers)
	jobErrors := make([]error, numJobs)
	var jobCounter atomic.Int32
	for worker := range numWorkers {
		go func() {
			defer wg.Done()

			client, err := net.GetClient()
			if err != nil {
				workerErrors[worker] = fmt.Errorf("failed to connect to the network: %w", err)
				return
			}
			defer client.Close()

			for {
				i := int(jobCounter.Add(1) - 1)
				if i >= numJobs {
					return // all jobs are done
				}
				if err := job(client, i); err != nil {
					jobErrors[i] = err
					return
				}
			}
		}()
	}
	wg.Wait()
	return errors.Join(
		errors.Join(workerErrors...),
		errors.Join(jobErrors...),
	)
}

// Apply sends a transaction to the network using the session account
// and waits for the transaction to be processed. The resulting receipt is returned.
func (s *Session) Apply(
	issue func(*bind.TransactOpts) (*types.Transaction, error),
) (*types.Receipt, error) {
	txOpts, err := s.GetTransactOptions(&s.account)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction options: %w", err)
	}
	transaction, err := issue(txOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}
	return s.GetReceipt(transaction.Hash())
}

// GetTransactOptions provides transaction options to be used to send a transaction
// with the given account. The options include the chain ID, a suggested gas price,
// the next free nonce of the given account, and a hard-coded gas limit of 1e6.
// The main purpose of this function is to provide a convenient way to collect all
// the necessary information required to create a transaction in one place.
func (s *Session) GetTransactOptions(account *Account) (*bind.TransactOpts, error) {
	client, err := s.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the Ethereum client: %w", err)
	}
	defer client.Close()

	ctxt := context.Background()
	chainId, err := client.ChainID(ctxt)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	gasPrice, err := client.SuggestGasPrice(ctxt)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price suggestion: %w", err)
	}

	nonce, err := client.PendingNonceAt(ctxt, account.Address())
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	txOpts, err := bind.NewKeyedTransactorWithChainID(account.PrivateKey, chainId)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction options: %w", err)
	}
	txOpts.GasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2))
	txOpts.Nonce = big.NewInt(int64(nonce))
	txOpts.GasLimit = 1e6
	return txOpts, nil
}

func (s *Session) GetSessionSponsor() *Account {
	return &s.account
}

// GetChainId returns the chain ID of the network.
func (s *Session) GetChainId() *big.Int {
	return s.net.GetChainId()
}

// GetClient provides raw access to a fresh connection to node zero on the network.
// The resulting client must be closed after use.
func (s *Session) GetClient() (*PooledEhtClient, error) {
	return s.net.GetClientConnectedToNode(0)
}

// GetClientConnectedToNode provides raw access to a fresh connection to a selected node on
// the network. The resulting client must be closed after use.
func (s *Session) GetClientConnectedToNode(i int) (*PooledEhtClient, error) {
	return s.net.GetClientConnectedToNode(i)
}

// GetWebSocketClient provides raw access to a fresh connection to the network
// using the WebSocket protocol. The resulting client must be closed after use.
func (s *Session) GetWebSocketClient() (*ethClient, error) {
	return s.net.GetWebSocketClient()
}

func (s *Session) NumNodes() int {
	return s.net.NumNodes()
}

func (s *Session) GetGenesisId() common.Hash {
	return s.net.genesisId
}

// validateAndSanitizeOptions ensures that the options are valid and sets the default values.
func validateAndSanitizeOptions(options ...IntegrationTestNetOptions) (IntegrationTestNetOptions, error) {

	if len(options) > 1 {
		return IntegrationTestNetOptions{}, fmt.Errorf("expected at most one option, got %d", len(options))
	}

	if len(options) == 0 {
		return IntegrationTestNetOptions{
			Upgrades:        AsPointer(opera.GetSonicUpgrades()),
			NumNodes:        1,
			ValidatorsStake: makefakegenesis.CreateEqualValidatorStake(1),
		}, nil
	}

	if options[0].NumNodes <= 0 {
		options[0].NumNodes = max(1, len(options[0].ValidatorsStake))
	}

	if len(options[0].ValidatorsStake) == 0 {
		options[0].ValidatorsStake =
			makefakegenesis.CreateEqualValidatorStake(options[0].NumNodes)
	}

	if options[0].NumNodes != len(options[0].ValidatorsStake) {
		return IntegrationTestNetOptions{}, fmt.Errorf("number of nodes (%d) does not match number of validator stakes (%d)",
			options[0].NumNodes, len(options[0].ValidatorsStake))
	}

	if options[0].Upgrades == nil {
		options[0].Upgrades = AsPointer(opera.GetSonicUpgrades())
	}

	return options[0], nil
}

func IsDataRaceDetectionEnabled() bool {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return false
	}
	for _, setting := range info.Settings {
		if setting.Key == "-race" && setting.Value == "true" {
			return true
		}
	}
	return false
}

// ethClient is an alias for ethclient.Client to avoid name conflicts with
// the method `.Client()` of the type PooledEhtClient.
type ethClient = ethclient.Client

// PooledEhtClient is a wrapper around ethClient that provides a Close method
// that returns the client to the shared client pool.
type PooledEhtClient struct {
	ethClient
	// Each shared client needs to know to which pool it has to return.
	// Keeping a reference to the pool allows the shared client to be compliant
	// with the ethclient.Client close signature.
	pool *sync.Pool
}

// Close returns the shared client to the pool it was generated from.
func (s *PooledEhtClient) Close() {
	if s.pool == nil {
		return
	}
	s.pool.Put(s)
}

// Client provides access to the underlying RPC Client.
func (s *PooledEhtClient) Client() *rpc.Client {
	return s.ethClient.Client()
}
