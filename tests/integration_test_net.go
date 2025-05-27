package tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/gossip/contract/driverauth100"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/ethereum/go-ethereum/common/hexutil"

	sonicd "github.com/0xsoniclabs/sonic/cmd/sonicd/app"
	sonictool "github.com/0xsoniclabs/sonic/cmd/sonictool/app"
	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	geth_crypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
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

	// Run sends the given transaction to the network and waits for it to be processed.
	// The resulting receipt is returned.
	Run(tx *types.Transaction) (*types.Receipt, error)
	// GetReceipt waits for the receipt of the given transaction hash to be available.
	GetReceipt(txHash common.Hash) (*types.Receipt, error)

	// GetTransactOptions provides transaction options to be used to send a transaction
	// from the given account.
	GetTransactOptions(account *Account) (*bind.TransactOpts, error)
	// Apply sends a transaction to the network using the session account.
	// and waits for the transaction to be processed. The resulting receipt is returned.
	Apply(issue func(*bind.TransactOpts) (*types.Transaction, error)) (*types.Receipt, error)

	// GetSessionSponsor returns the default account of the session. This account is used
	// to sign transactions and pay for gas when using the Apply and EndowAccount methods.
	GetSessionSponsor() *Account
	// GetClient provides raw access to a fresh connection to the network.
	// The resulting client must be closed after use.
	GetClient() (*ethclient.Client, error)

	// AdvanceEpoch sends a transaction to advance to the next epoch.
	// It also waits until the new epoch is really reached.
	AdvanceEpoch(epochs int) error
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
	// test network. A value of 0 is interpreted as 1.
	NumNodes int
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
	options IntegrationTestNetOptions
	nodes   []integrationTestNode

	sessionsMutex sync.Mutex
	Session
}

// per-node state for the integration test network
type integrationTestNode struct {
	directory string
	httpPort  int
	shutdown  chan<- struct{}
	done      <-chan struct{}
}

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
	if err != nil {
		t.Fatal("failed to validate and sanitize options: ", err)
	}

	if len(effectiveOptions.Accounts) != 0 {
		t.Fatal("fake genesis does not support custom accounts")
	}

	var upgrades string
	if *effectiveOptions.Upgrades == opera.GetSonicUpgrades() {
		upgrades = "sonic"
	} else if *effectiveOptions.Upgrades == opera.GetAllegroUpgrades() {
		upgrades = "allegro"
	} else {
		t.Fatal("fake genesis only supports sonic and allegro feature sets")
	}

	net, err := startIntegrationTestNet(
		t,
		t.TempDir(),
		[]string{"genesis", "fake", "1", "--upgrades", upgrades},
		effectiveOptions,
	)
	if err != nil {
		t.Fatal("failed to start integration test network: ", err)
	}
	return net
}

// StartIntegrationTestNetWithFakeGenesis starts a single-node test network for
// integration tests using the JSON-Genesis procedure. The JSON genesis procedure
// is the genesis procedure used in long-running production networks like the
// Sonic mainnet and the testnet.
func StartIntegrationTestNetWithJsonGenesis(
	t *testing.T,
	options ...IntegrationTestNetOptions,
) *IntegrationTestNet {
	t.Helper()

	effectiveOptions, err := validateAndSanitizeOptions(options...)
	if err != nil {
		t.Fatal("failed to validate and sanitize options: ", err)
	}

	jsonGenesis := makefakegenesis.GenerateFakeJsonGenesis(
		effectiveOptions.NumNodes,
		*effectiveOptions.Upgrades,
	)

	jsonGenesis.Accounts = append(jsonGenesis.Accounts, effectiveOptions.Accounts...)

	// Speed up the block generation time to reduce test time.
	jsonGenesis.Rules.Emitter.Interval = inter.Timestamp(time.Millisecond)

	encoded, err := json.MarshalIndent(jsonGenesis, "", "  ")
	if err != nil {
		t.Fatal("failed to marshal genesis json:", err)
	}

	directory := t.TempDir()
	jsonFile := filepath.Join(directory, "genesis.json")
	err = os.WriteFile(jsonFile, encoded, 0644)
	if err != nil {
		t.Fatal("failed to write genesis json file: ", err)
	}

	net, err := startIntegrationTestNet(
		t,
		directory,
		[]string{"genesis", "json", "--experimental", jsonFile},
		effectiveOptions,
	)
	if err != nil {
		t.Fatal("failed to start integration test network: ", err)
	}
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
		nodes: make([]integrationTestNode, options.NumNodes),
	}
	net.Session.net = net

	if verbosityVariable := os.Getenv("SONIC_VERBOSITY"); verbosityVariable == "" {
		if err := os.Setenv("SONIC_VERBOSITY", "0"); err != nil {
			t.Fatal("failed to set verbosity: ", err)
		}
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
		if err := sonictool.RunWithArgs(args); err != nil {
			return nil, fmt.Errorf("failed to initialize the test network: %w", err)
		}
	}

	if err := net.start(); err != nil {
		return nil, fmt.Errorf("failed to start the test network: %w", err)
	}
	t.Cleanup(net.Stop)
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
				"--http.api", "admin,eth,web3,net,txpool,ftm,trace,debug,sonic",

				// websocket-client options
				"--ws", "--ws.addr", "127.0.0.1", "--ws.port", "0",
				"--ws.api", "admin,eth,ftm",

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
	nodeEnodes := make([]string, len(n.nodes))
	for i := range n.nodes {
		id, ok := <-nodeIds[i]
		if !ok {
			return fmt.Errorf("failed to start the network, no ID announced for node %d", i)
		}
		nodeEnodes[i] = id
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
	}

	// connect to blockchain network
	client, err := n.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to the Ethereum client: %w", err)
	}
	defer client.Close()

	const timeout = 300 * time.Second
	start := time.Now()

	// wait for the node to be ready to serve requests
	const maxDelay = 100 * time.Millisecond
	delay := time.Millisecond
	for {
		if time.Since(start) > timeout {
			return fmt.Errorf("failed to successfully start up a test network within %v", timeout)
		}
		_, err := client.ChainID(context.Background())
		if err != nil {
			time.Sleep(delay)
			delay = 2 * delay
			if delay > maxDelay {
				delay = maxDelay
			}
			continue
		}
		break
	}

	// Connect the nodes with each other.
	for i, enode := range nodeEnodes {
		if err := client.Client().Call(nil, "admin_addPeer", enode); err != nil {
			return fmt.Errorf("failed to connect to node %d: %v", i, err)
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
}

// Stops and restarts the single node on the test network.
func (n *IntegrationTestNet) Restart() error {
	n.Stop()
	return n.start()
}

// NumNodes returns the number of nodes on the network.
func (n *IntegrationTestNet) NumNodes() int {
	return len(n.nodes)
}

// GetClient provides raw access to a fresh connection to the network.
// The resulting client must be closed after use.
func (n *IntegrationTestNet) GetClient() (*ethclient.Client, error) {
	return n.GetClientConnectedToNode(0)
}

// GetClient provides raw access to a fresh connection to a selected node on
// the network. The resulting client must be closed after use.
func (n *IntegrationTestNet) GetClientConnectedToNode(i int) (*ethclient.Client, error) {
	if i < 0 || i >= len(n.nodes) {
		return nil, fmt.Errorf("node index out of bounds: %d", i)
	}
	return ethclient.Dial(fmt.Sprintf("http://localhost:%d", n.nodes[i].httpPort))
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

// SpawnSession(t) creates a new test session on the network.
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
	if err != nil {
		t.Fatalf("Failed to endow account: %v", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("Failed to endow account: %v", receipt.Status)
	}

	return &Session{
		net:     n,
		account: nextSessionAccount,
	}
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

func (s *Session) GetUpgrades() opera.Upgrades {
	return *s.net.options.Upgrades
}

// EndowAccount sends a requested amount of tokens to the given account. This is
// mainly intended to provide funds to accounts for testing purposes.
func (s *Session) EndowAccount(
	address common.Address,
	value *big.Int,
) (*types.Receipt, error) {
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
	return s.Run(transaction)
}

// Run sends the given transaction to the network and waits for it to be processed.
// The resulting receipt is returned. This function times out after 10 seconds.
func (s *Session) Run(tx *types.Transaction) (*types.Receipt, error) {
	client, err := s.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the Ethereum client: %w", err)
	}
	defer client.Close()
	err = client.SendTransaction(context.Background(), tx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}
	return s.GetReceipt(tx.Hash())
}

// GetReceipt waits for the receipt of the given transaction hash to be available.
// The function times out after 10 seconds.
func (s *Session) GetReceipt(txHash common.Hash) (*types.Receipt, error) {
	client, err := s.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the Ethereum client: %w", err)
	}
	defer client.Close()

	// Wait for the response with some exponential backoff.
	const maxDelay = 100 * time.Millisecond
	now := time.Now()
	delay := time.Millisecond
	for time.Since(now) < 100*time.Second {
		receipt, err := client.TransactionReceipt(context.Background(), txHash)
		if errors.Is(err, ethereum.NotFound) {
			time.Sleep(delay)
			delay = 2 * delay
			if delay > maxDelay {
				delay = maxDelay
			}
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
		}
		return receipt, nil
	}
	return nil, fmt.Errorf("failed to get transaction receipt: timeout")
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

// GetClient provides raw access to a fresh connection to the network.
// The resulting client must be closed after use.
func (s *Session) GetClient() (*ethclient.Client, error) {
	return s.GetClientConnectedToNode(0)
}

// GetClient provides raw access to a fresh connection to a selected node on
// the network. The resulting client must be closed after use.
func (s *Session) GetClientConnectedToNode(i int) (*ethclient.Client, error) {
	return s.net.GetClientConnectedToNode(i)
}

// AdvanceEpoch trigger the sealing of an epoch and the epoch number to progress by the given number.
// The function blocks until the final epoch has been reached.
func (s *Session) AdvanceEpoch(epochs int) error {
	client, err := s.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to the client: %w", err)
	}
	defer client.Close()

	var currentEpoch hexutil.Uint64
	if err := client.Client().Call(&currentEpoch, "eth_currentEpoch"); err != nil {
		return fmt.Errorf("failed to get current epoch: %w", err)
	}

	contract, err := driverauth100.NewContract(driverauth.ContractAddress, client)
	if err != nil {
		return fmt.Errorf("failed to create contract: %w", err)
	}

	receipt, err := s.Apply(func(ops *bind.TransactOpts) (*types.Transaction, error) {
		return contract.AdvanceEpochs(ops, big.NewInt(int64(epochs)))
	})
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	if got, want := receipt.Status, types.ReceiptStatusSuccessful; got != want {
		return fmt.Errorf("expected status %d, got %d", want, got)
	}

	// wait until the epoch is advanced
	for {
		var newEpoch hexutil.Uint64
		if err := client.Client().Call(&newEpoch, "eth_currentEpoch"); err != nil {
			return fmt.Errorf("failed to get current epoch: %w", err)
		}

		if newEpoch >= currentEpoch+hexutil.Uint64(epochs) {
			break
		}
	}

	return nil
}

// validateAndSanitizeOptions ensures that the options are valid and sets the default values.
func validateAndSanitizeOptions(options ...IntegrationTestNetOptions) (IntegrationTestNetOptions, error) {

	if len(options) > 1 {
		return IntegrationTestNetOptions{}, fmt.Errorf("expected at most one option, got %d", len(options))
	}

	if len(options) == 0 {
		return IntegrationTestNetOptions{
			Upgrades: AsPointer(opera.GetSonicUpgrades()),
			NumNodes: 1,
		}, nil
	}
	if options[0].NumNodes <= 0 {
		options[0].NumNodes = 1
	}
	if options[0].Upgrades == nil {
		options[0].Upgrades = AsPointer(opera.GetSonicUpgrades())
	}

	return options[0], nil
}
