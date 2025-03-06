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
	"syscall"
	"testing"
	"time"

	sonicd "github.com/0xsoniclabs/sonic/cmd/sonicd/app"
	sonictool "github.com/0xsoniclabs/sonic/cmd/sonictool/app"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver/drivercall"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/0xsoniclabs/sonic/opera/contracts/evmwriter"
	"github.com/0xsoniclabs/sonic/opera/contracts/netinit"
	"github.com/0xsoniclabs/sonic/opera/contracts/sfc"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	futils "github.com/0xsoniclabs/sonic/utils"
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
}

// IntegrationTestNet is a in-process test network for integration tests. When
// started, it runs a full Sonic node maintaining a chain within the process
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
	directory            string
	done                 <-chan struct{}
	extraClientArguments []string

	sessionsMutex sync.Mutex
	Session
}

// StartIntegrationTestNet starts a single-node test network for integration tests.
// The node serving the network is started in the same process as the caller. This
// is intended to facilitate debugging of client code in the context of a running
// node.
//
// The network start procedure will create a temporary directory and populate with
// a fake network genesis block. To retrieve the directory path, use the GetDirectory
func StartIntegrationTestNet(
	t *testing.T,
	extraClientArguments ...string,
) *IntegrationTestNet {
	t.Helper()
	net, err := startIntegrationTestNet(t.TempDir(), []string{"genesis", "fake", "1"}, extraClientArguments)
	if err != nil {
		t.Fatal("failed to start integration test network: ", err)
	}
	t.Cleanup(func() {
		net.Stop()
	})
	return net
}

func StartIntegrationTestNetFromJsonGenesis(
	t *testing.T,
	extraArguments ...string,
) *IntegrationTestNet {
	t.Helper()

	jsonGenesis := makefakegenesis.GenesisJson{
		Rules:         opera.FakeNetRules(),
		BlockZeroTime: time.Now(),
	}

	// Create infrastructure contracts.
	jsonGenesis.Accounts = []makefakegenesis.Account{
		{
			Name:    "NetworkInitializer",
			Address: netinit.ContractAddress,
			Code:    netinit.GetContractBin(),
			Nonce:   1,
		},
		{
			Name:    "NodeDriver",
			Address: driver.ContractAddress,
			Code:    driver.GetContractBin(),
			Nonce:   1,
		},
		{
			Name:    "NodeDriverAuth",
			Address: driverauth.ContractAddress,
			Code:    driverauth.GetContractBin(),
			Nonce:   1,
		},
		{
			Name:    "SFC",
			Address: sfc.ContractAddress,
			Code:    sfc.GetContractBin(),
			Nonce:   1,
		},
		{
			Name:    "ContractAddress",
			Address: evmwriter.ContractAddress,
			Code:    []byte{0},
			Nonce:   1,
		},
	}

	// Create the validator account and provide some tokens.
	totalSupply := futils.ToFtm(1000000000)
	validators := makefakegenesis.GetFakeValidators(1)
	for _, validator := range validators {
		jsonGenesis.Accounts = append(jsonGenesis.Accounts, makefakegenesis.Account{
			Address: validator.Address,
			Balance: totalSupply,
		})
	}

	var delegations []drivercall.Delegation
	for _, val := range validators {
		delegations = append(delegations, drivercall.Delegation{
			Address:            val.Address,
			ValidatorID:        val.ID,
			Stake:              futils.ToFtm(5000000),
			LockedStake:        new(big.Int),
			LockupFromEpoch:    0,
			LockupEndTime:      0,
			LockupDuration:     0,
			EarlyUnlockPenalty: new(big.Int),
			Rewards:            new(big.Int),
		})
	}

	// Create the genesis transactions.
	genesisTxs := makefakegenesis.GetGenesisTxs(0, validators, totalSupply, delegations, validators[0].Address)
	for _, tx := range genesisTxs {
		jsonGenesis.Txs = append(jsonGenesis.Txs, makefakegenesis.Transaction{
			To:   *tx.To(),
			Data: tx.Data(),
		})
	}

	// Create the genesis SCC committee.
	key := bls.NewPrivateKeyForTests(0)
	committee := scc.NewCommittee(scc.Member{
		PublicKey:         key.PublicKey(),
		ProofOfPossession: key.GetProofOfPossession(),
		VotingPower:       1,
	})
	if err := committee.Validate(); err != nil {
		t.Fatal("failed to validate the committee:", err)
	}
	jsonGenesis.GenesisCommittee = &committee

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
	net, err := startIntegrationTestNet(directory,
		[]string{"genesis", "json", "--experimental", jsonFile},
		extraArguments,
	)
	if err != nil {
		t.Fatal("failed to start integration test network: ", err)
	}
	t.Cleanup(func() {
		net.Stop()
	})
	return net
}

func startIntegrationTestNet(directory string, sonicToolArguments []string, extraClientArguments []string) (*IntegrationTestNet, error) {
	// start the fakenet sonic node
	result := &IntegrationTestNet{
		directory:            directory,
		extraClientArguments: extraClientArguments,
		Session: Session{
			account: Account{evmcore.FakeKey(1)},
		},
	}

	// initialize the data directory for the single node on the test network
	// equivalent to running `sonictool --datadir <dataDir> genesis fake 1`
	originalArgs := os.Args
	os.Args = append([]string{
		"sonictool",
		"--datadir", result.stateDir(),
		"--statedb.livecache", "1",
		"--statedb.archivecache", "1",
		"--statedb.cache", "1024",
	}, sonicToolArguments...)
	if err := sonictool.Run(); err != nil {
		os.Args = originalArgs
		return nil, fmt.Errorf("failed to initialize the test network: %w", err)
	}
	os.Args = originalArgs

	if err := result.start(); err != nil {
		return nil, fmt.Errorf("failed to start the test network: %w", err)
	}
	return result, nil
}

func (n *IntegrationTestNet) stateDir() string {
	return filepath.Join(n.directory, "state")
}

func (n *IntegrationTestNet) start() error {
	if n.done != nil {
		return errors.New("network already started")
	}

	httpEndpointAnnouncement := make(chan string)
	done := make(chan struct{})
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
			"--datadir", n.stateDir(),
			"--datadir.minfreedisk", "0",

			// fake network options
			"--fakenet", "1/1",

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
			n.extraClientArguments...,
		)

		if err := sonicd.RunWithArgs(args, httpEndpointAnnouncement); err != nil {
			panic(fmt.Sprint("Failed to start the fake network:", err))
		}
	}()

	n.done = done

	// wait for the HTTP endpoint announcement
	endpoint, ok := <-httpEndpointAnnouncement
	if !ok {
		return errors.New("failed to start the network, no HTTP endpoint announced")
	}

	// Extract the HTTP port form the endpoint string.
	pattern := regexp.MustCompile(`^http://.*:(\d+)$`)
	match := pattern.FindStringSubmatch(endpoint)
	if len(match) != 2 {
		return fmt.Errorf("failed to parse the HTTP endpoint: %s", endpoint)
	}
	httpPort, err := strconv.Atoi(match[1])
	if err != nil {
		return fmt.Errorf("failed to parse the HTTP port %s: %w", endpoint, err)
	}
	n.httpPort = httpPort

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
	for time.Since(start) < timeout {
		_, err := client.ChainID(context.Background())
		if err != nil {
			time.Sleep(delay)
			delay = 2 * delay
			if delay > maxDelay {
				delay = maxDelay
			}
			continue
		}
		return nil
	}

	return fmt.Errorf("failed to successfully start up a test network within %v", timeout)
}

// Stop shuts the underlying network down.
func (n *IntegrationTestNet) Stop() {
	if n.done == nil {
		return
	}
	// best effort to stop the test environment, ignore error
	_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	<-n.done
	n.done = nil
}

// Stops and restarts the single node on the test network.
func (n *IntegrationTestNet) Restart() error {
	n.Stop()
	return n.start()
}

// GetWebSocketClient provides raw access to a fresh connection to the network
// using the WebSocket protocol. The resulting client must be closed after use.
func (n *IntegrationTestNet) GetWebSocketClient() (*ethclient.Client, error) {
	return ethclient.Dial(fmt.Sprintf("ws://localhost:%d", n.httpPort))
}
func (n *IntegrationTestNet) GetDirectory() string {
	return n.directory
}

// RestartWithExportImport stops the network, exports the genesis file, cleans the
// temporary directory, imports the genesis file, and starts the network again.
func (n *IntegrationTestNet) RestartWithExportImport() error {
	n.Stop()
	fmt.Println("Network stopped. Exporting genesis file...")

	// save original args
	originalArgs := os.Args

	// export
	genesisFile := filepath.Join(n.directory, "testGenesis.g")
	os.Args = []string{
		"sonictool",
		"--datadir", n.stateDir(),
		"genesis", "export", genesisFile,
	}
	err := sonictool.Run()
	if err != nil {
		return err
	}

	// clean client state
	err = os.RemoveAll(n.stateDir())
	if err != nil {
		return err
	}

	fmt.Println("State directory cleaned. Importing genesis file...")

	// import genesis file
	os.Args = []string{
		"sonictool",
		"--datadir", n.stateDir(),
		"genesis", "--experimental", genesisFile,
	}
	err = sonictool.Run()
	if err != nil {
		return err
	}

	// restore original args
	os.Args = originalArgs

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
//	net, err := StartIntegrationTestNet(t.TempDir())
//	if err != nil {
//	    ...
//	}
//	t.Cleanup(func(){net.Stop()})
//	t.Run("test_case",, func(t *testing.T) {
//			t.Parallel()
//			session := net.SpawnSession(t)
//	        < use session instead of net of the rest of the test >
//	})
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
		account:  nextSessionAccount,
		httpPort: n.httpPort,
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
	account  Account
	httpPort int
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
	nonce, err := client.NonceAt(context.Background(), s.account.Address(), nil)
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

	nonce, err := client.NonceAt(ctxt, account.Address(), nil)
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
	return ethclient.Dial(fmt.Sprintf("http://localhost:%d", s.httpPort))
}
