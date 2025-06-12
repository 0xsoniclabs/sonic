package tests

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/0xsoniclabs/sonic/vecmt"
	"github.com/Fantom-foundation/lachesis-base/abft"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/filtermaps"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/stretchr/testify/require"
)

// Known missing APIs which are not implemented in Sonic
var (
	knownMissingAPIs = namespaceMap{
		"eth": {
			"SimulateV1": true,
		},
		"debug": {
			"DbAncient":                   true,
			"DbAncients":                  true,
			"DbGet":                       true,
			"GetRawBlock":                 true,
			"GetRawHeader":                true,
			"GetRawReceipts":              true,
			"GetRawTransaction":           true,
			"IntermediateRoots":           true,
			"StandardTraceBadBlockToFile": true,
			"StandardTraceBlockToFile":    true,
			"TraceBadBlock":               true,
			"TraceBlock":                  true,
			"TraceBlockFromFile":          true,
			"TraceChain":                  true,
		},
	}
)

type namespaceMap map[string]map[string]bool

func TestRPCApis(t *testing.T) {
	ethAPIs := parseAPIs(tests.GetRPCApis(t, newBackendMock()))
	sonicAPIs := parseAPIs(getNodeService(t).APIs())

	// look for missing methods which are in go-ethereum and not in Sonic
	missingInSonic := findMissingMethods(ethAPIs, sonicAPIs)

	// look for missing methods which are in Sonic and are not in known missing
	missing := findMissingMethods(missingInSonic, knownMissingAPIs)
	if len(missing) != 0 {
		t.Errorf("Missing namespaces: %v", missing)
	}

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

	txPool := &dummyTxPool{}

	cacheRatio := cachescale.Ratio{
		Base:   uint64(config.DefaultCacheSize*1 - config.ConstantCacheSize),
		Target: uint64(config.DefaultCacheSize*2 - config.ConstantCacheSize),
	}

	defaultConfig := gossip.DefaultConfig(cacheRatio)
	s, err := gossip.NewService(node, defaultConfig, store, gossip.BlockProc{}, engine, vecClock, func(_ evmcore.StateReader) gossip.TxPool {
		return txPool
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
			if !b[outerKey][innerKey] {
				if missing[outerKey] == nil {
					missing[outerKey] = make(map[string]bool)
				}
				missing[outerKey][innerKey] = value
			}
		}
	}
	return missing
}

// parseAPIs returns a map of namespaces and methods
func parseAPIs(apis []rpc.API) namespaceMap {

	namespaces := make(map[string]map[string]bool)

	for _, api := range apis {
		if _, exists := namespaces[api.Namespace]; !exists {
			namespaces[api.Namespace] = make(map[string]bool)
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
		for innerKey, ok := range innerMap {
			if ok {
				funcs = append(funcs, fmt.Sprintf("\"%s\"", innerKey))
			}
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

// dummyTxPool is a dummy implementation of evmcore.TxPool
type dummyTxPool struct {
	txFeed event.Feed
}

func (p *dummyTxPool) AddRemotes(txs []*types.Transaction) []error {
	return nil
}

func (p *dummyTxPool) AddLocals(txs []*types.Transaction) []error {
	return nil
}

func (p *dummyTxPool) AddLocal(tx *types.Transaction) error {
	return nil
}

func (p *dummyTxPool) Nonce(addr common.Address) uint64 {
	return 0
}

func (p *dummyTxPool) MinTip() *big.Int {
	return nil
}

func (p *dummyTxPool) Stats() (int, int) {
	return p.Count(), 0
}

func (p *dummyTxPool) Content() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return nil, nil
}

func (p *dummyTxPool) ContentFrom(addr common.Address) (types.Transactions, types.Transactions) {
	return nil, nil
}

func (p *dummyTxPool) Pending(enforceTips bool) (map[common.Address]types.Transactions, error) {
	return nil, nil
}

func (p *dummyTxPool) GasPrice() *big.Int {
	return big.NewInt(0)
}

func (p *dummyTxPool) SubscribeNewTxsNotify(ch chan<- evmcore.NewTxsNotify) event.Subscription {
	return p.txFeed.Subscribe(ch)
}

func (p *dummyTxPool) Map() map[common.Hash]*types.Transaction {
	return nil
}

func (p *dummyTxPool) Get(txid common.Hash) *types.Transaction {
	return nil
}

func (p *dummyTxPool) Has(txid common.Hash) bool {
	return false
}

func (p *dummyTxPool) OnlyNotExisting(txids []common.Hash) []common.Hash {
	return nil
}

func (p *dummyTxPool) SampleHashes(max int) []common.Hash {
	return nil
}

func (p *dummyTxPool) Count() int {
	return 0
}

func (p *dummyTxPool) Clear() {
}

func (p *dummyTxPool) Delete(needle common.Hash) {
}

func (p *dummyTxPool) Stop() {}

// backendMock is a dummy implementation of ethapi.Backend
type backendMock struct {
	config *params.ChainConfig
}

func newBackendMock() *backendMock {
	return &backendMock{
		config: &params.ChainConfig{},
	}
}

func (b *backendMock) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return big.NewInt(42), nil
}
func (b *backendMock) BlobBaseFee(ctx context.Context) *big.Int { return big.NewInt(42) }

func (b *backendMock) CurrentHeader() *types.Header     { return nil }
func (b *backendMock) ChainConfig() *params.ChainConfig { return b.config }

// Other methods needed to implement Backend interface.
func (b *backendMock) SyncProgress() ethereum.SyncProgress { return ethereum.SyncProgress{} }
func (b *backendMock) FeeHistory(ctx context.Context, blockCount uint64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*big.Int, [][]*big.Int, []*big.Int, []float64, []*big.Int, []float64, error) {
	return nil, nil, nil, nil, nil, nil, nil
}
func (b *backendMock) ChainDb() ethdb.Database { return nil }

func (b *backendMock) AccountManager() *accounts.Manager { return nil }
func (b *backendMock) ExtRPCEnabled() bool               { return false }
func (b *backendMock) RPCGasCap() uint64                 { return 0 }
func (b *backendMock) RPCEVMTimeout() time.Duration      { return time.Second }
func (b *backendMock) RPCTxFeeCap() float64              { return 0 }
func (b *backendMock) UnprotectedAllowed() bool          { return false }
func (b *backendMock) SetHead(number uint64)             {}
func (b *backendMock) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	return nil, nil
}
func (b *backendMock) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return nil, nil
}
func (b *backendMock) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	return nil, nil
}
func (b *backendMock) CurrentBlock() *types.Header { return nil }
func (b *backendMock) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	return nil, nil
}
func (b *backendMock) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return nil, nil
}
func (b *backendMock) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	return nil, nil
}
func (b *backendMock) GetBody(ctx context.Context, hash common.Hash, number rpc.BlockNumber) (*types.Body, error) {
	return nil, nil
}
func (b *backendMock) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	return nil, nil, nil
}
func (b *backendMock) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	return nil, nil, nil
}
func (b *backendMock) Pending() (*types.Block, types.Receipts, *state.StateDB) { return nil, nil, nil }
func (b *backendMock) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return nil, nil
}
func (b *backendMock) GetLogs(ctx context.Context, blockHash common.Hash, number uint64) ([][]*types.Log, error) {
	return nil, nil
}
func (b *backendMock) GetEVM(ctx context.Context, state *state.StateDB, header *types.Header, vmConfig *vm.Config, blockCtx *vm.BlockContext) *vm.EVM {
	return nil
}
func (b *backendMock) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription { return nil }
func (b *backendMock) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return nil
}
func (b *backendMock) SendTx(ctx context.Context, signedTx *types.Transaction) error { return nil }
func (b *backendMock) GetTransaction(ctx context.Context, txHash common.Hash) (bool, *types.Transaction, common.Hash, uint64, uint64, error) {
	return false, nil, [32]byte{}, 0, 0, nil
}
func (b *backendMock) GetPoolTransactions() (types.Transactions, error)         { return nil, nil }
func (b *backendMock) GetPoolTransaction(txHash common.Hash) *types.Transaction { return nil }
func (b *backendMock) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return 0, nil
}
func (b *backendMock) Stats() (pending int, queued int) { return 0, 0 }
func (b *backendMock) TxPoolContent() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction) {
	return nil, nil
}
func (b *backendMock) TxPoolContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction) {
	return nil, nil
}
func (b *backendMock) SubscribeNewTxsEvent(chan<- core.NewTxsEvent) event.Subscription { return nil }
func (b *backendMock) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription    { return nil }
func (b *backendMock) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return nil
}

func (b *backendMock) Engine() consensus.Engine { return nil }

func (b *backendMock) NewMatcherBackend() filtermaps.MatcherBackend { return nil }
