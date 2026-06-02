package execution_apis

import (
	"flag"
	"fmt"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	rpctest "github.com/0xsoniclabs/sonic/api/rpc_test"
	"github.com/ethereum/go-ethereum/rpc"
)

var executionAPIsDir = flag.String(
	"execution.apis",
	"/home/luis/code/execution-apis",
	"Path to the cloned github.com/ethereum/execution-apis repository. "+
		"If empty, conformance tests are skipped.",
)

// skipMethods lists methods that Sonic does not implement or whose behavior
// is intentionally different from the Ethereum spec. These are skipped entirely.
var skipMethods = map[string]string{
	"eth_getProof":                            "not implemented in Sonic",
	"eth_simulateV1":                          "not implemented in Sonic",
	"eth_getStorageValues":                    "not implemented in Sonic",
	"testing_buildBlockV1":                    "test-only method, not part of Sonic",
	"txpool_content":                          "different pool semantics",
	"txpool_contentFrom":                      "different pool semantics",
	"txpool_status":                           "different pool semantics",
	"eth_getTransactionCount":                 "not implemented in Sonic",
	"eth_getTransactionByBlockNumberAndIndex": "not implemented in Sonic",

	"debug_getRawBlock":    "not implemented in Sonic",
	"debug_getRawReceipts": "not implemented in Sonic",

	"eth_getStorageAt": "not implemented in Sonic",
}

// skipTests lists individual test files to skip (relative to tests/ dir).
// Keys are "method/filename.io".
var skipTests = map[string]string{
	// Add individual test skips here with reasons as values.
	// Example: "eth_getBlockByNumber/get-block-prague-fork.io": "Prague not yet activated",
}

// receiptDependentMethods are methods that require live pool or features beyond
// static chain replay.
var receiptDependentMethods = map[string]string{
	"eth_sendRawTransaction": "requires live transaction pool",
	"eth_feeHistory":         "requires full block history with gas data",
}

func TestConformance(t *testing.T) {
	if *executionAPIsDir == "" {
		t.Skip("Skipping conformance tests: -execution.apis flag not set")
	}

	testsDir := filepath.Join(*executionAPIsDir, "tests")

	// Load genesis
	genesisPath := filepath.Join(testsDir, "genesis.json")
	genesis, err := LoadGenesis(genesisPath)
	if err != nil {
		t.Fatalf("Failed to load genesis: %v", err)
	}

	// Load chain
	chainPath := filepath.Join(testsDir, "chain.rlp")
	rawBlocks, err := LoadChain(chainPath)
	if err != nil {
		t.Fatalf("Failed to load chain: %v", err)
	}

	// Build fake backend
	backend, err := rpctest.NewBackendBuilder(t).
		BuildFromReplay(genesis, rawBlocks)
	if err != nil {
		t.Fatalf("Failed to build backend from replay: %v", err)
	}

	// Set up dispatcher with RPC APIs
	apis := buildAPIs(backend)
	dispatcher, err := NewDispatcher(apis)
	if err != nil {
		t.Fatalf("Failed to create dispatcher: %v", err)
	}
	defer dispatcher.Close()

	// Discover test vectors
	vectors, err := DiscoverTestVectors(testsDir)
	if err != nil {
		t.Fatalf("Failed to discover test vectors: %v", err)
	}

	// Run tests grouped by method
	for method, testVectors := range vectors {
		method := method
		testVectors := testVectors

		t.Run(method, func(t *testing.T) {
			// Check skip lists
			if reason, ok := skipMethods[method]; ok {
				t.Skipf("Skipping method: %s", reason)
			}
			if reason, ok := receiptDependentMethods[method]; ok {
				t.Skipf("Skipping method: %s", reason)
			}

			for _, tv := range testVectors {
				tv := tv
				testName := filepath.Base(tv.File)
				if tv.Index > 0 {
					testName = fmt.Sprintf("%s[%d]", testName, tv.Index)
				}

				t.Run(testName, func(t *testing.T) {
					// Check individual test skip
					relPath := filepath.Join(method, filepath.Base(tv.File))
					if reason, ok := skipTests[relPath]; ok {
						t.Skipf("Skipping test: %s", reason)
					}

					// Execute
					actual, err := dispatcher.Call(t.Context(), tv.Request)
					if err != nil {
						t.Fatalf("Dispatcher call failed: %v", err)
					}

					// Compare
					if err := CompareJSON(tv.Response, actual); err != nil {
						t.Errorf("Response mismatch:\n"+
							"  Request:  %s\n"+
							"  Expected: %s\n"+
							"  Actual:   %s\n"+
							"  Diff:     %v",
							string(tv.Request),
							string(tv.Response),
							string(actual),
							err,
						)
					}
				})
			}
		})
	}
}

// buildAPIs creates the RPC API list using the ethapi handlers.
func buildAPIs(backend ethapi.Backend) []rpc.API {
	nonceLock := new(ethapi.AddrLocker)
	return []rpc.API{
		{
			Namespace: "eth",
			Service:   ethapi.NewPublicEthereumAPI(backend),
		},
		{
			Namespace: "eth",
			Service:   ethapi.NewPublicBlockChainAPI(backend),
		},
		{
			Namespace: "net",
			Service:   &netAPI{chainID: backend.ChainID().Uint64()},
		},
		{
			Namespace: "eth",
			Service:   ethapi.NewPublicTransactionPoolAPI(backend, nonceLock),
		},
	}
}

// netAPI implements the net namespace for conformance tests.
type netAPI struct {
	chainID uint64
}

// Version returns the network ID.
func (api *netAPI) Version() string {
	return new(big.Int).SetUint64(api.chainID).String()
}
