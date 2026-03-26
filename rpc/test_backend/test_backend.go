package testbackend

import (
	"context"
	"math/big"
	"time"

	"github.com/0xsoniclabs/carmen/go/common/witness"
	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	rpctypes "github.com/0xsoniclabs/sonic/rpc/types"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	geth_state "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/holiman/uint256"
)

type AccountState struct {
	Nonce   uint64
	Balance *big.Int
}

type Blockchain struct {
	StateDb
}

func NewBlockchain() *Blockchain {
	return &Blockchain{
		StateDb{state: make(map[common.Address]AccountState)},
	}
}

func (b *Blockchain) SetAccount(addr common.Address, account AccountState) {
	b.state[addr] = account
}

func (b *Blockchain) StateAndBlockByNumberOrHash(
	ctx context.Context,
	blockNrOrHash rpc.BlockNumberOrHash,
) (state.StateDB, *evmcore.EvmBlock, error) {

	return b.StateDb, b.CurrentBlock(), nil
}

// ============================================================================

// AccountManager implements [ethapi.BundleApiBackend].
func (b *Blockchain) AccountManager() *accounts.Manager {
	panic("unimplemented")
}

// BlockByHash implements [ethapi.BundleApiBackend].
func (b *Blockchain) BlockByHash(ctx context.Context, hash common.Hash) (*evmcore.EvmBlock, error) {
	panic("unimplemented")
}

// BlockByNumber implements [ethapi.BundleApiBackend].
func (b *Blockchain) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*evmcore.EvmBlock, error) {
	panic("unimplemented")
}

// CalcBlockExtApi implements [ethapi.BundleApiBackend].
func (b *Blockchain) CalcBlockExtApi() bool {
	panic("unimplemented")
}

// ChainConfig implements [ethapi.BundleApiBackend].
func (b *Blockchain) ChainConfig(blockHeight idx.Block) *params.ChainConfig {
	return &params.ChainConfig{}
}

// ChainID implements [ethapi.BundleApiBackend].
func (b *Blockchain) ChainID() *big.Int {
	return big.NewInt(321)
}

// CurrentBlock implements [ethapi.BundleApiBackend].
func (b *Blockchain) CurrentBlock() *evmcore.EvmBlock {

	return &evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number: big.NewInt(1),
		},
	}
}

// ExtRPCEnabled implements [ethapi.BundleApiBackend].
func (b *Blockchain) ExtRPCEnabled() bool {
	panic("unimplemented")
}

// FetchReceiptsForBlock implements [ethapi.BundleApiBackend].
func (b *Blockchain) FetchReceiptsForBlock(block *evmcore.EvmBlock) types.Receipts {
	panic("unimplemented")
}

// GetBundleExecutionInfo implements [ethapi.BundleApiBackend].
func (b *Blockchain) GetBundleExecutionInfo(common.Hash) *bundle.ExecutionInfo {
	panic("unimplemented")
}

// GetEVM implements [ethapi.BundleApiBackend].
func (b *Blockchain) GetEVM(ctx context.Context, state vm.StateDB, header *evmcore.EvmHeader, vmConfig *vm.Config, blockContext *vm.BlockContext) (*vm.EVM, func() error, error) {

	chainConfig := &params.ChainConfig{}

	if blockContext == nil {
		chainCtx := ethapi.ChainContext{
			Ctx: ctx,
			B:   b,
		}
		newCtx := evmcore.NewEVMBlockContext(header, &chainCtx, nil)
		blockContext = &newCtx
	}

	return vm.NewEVM(*blockContext, state, chainConfig, *vmConfig), func() error { return nil }, nil
}

// GetGenesisID implements [ethapi.BundleApiBackend].
func (b *Blockchain) GetGenesisID() common.Hash {
	panic("unimplemented")
}

// GetNetworkRules implements [ethapi.BundleApiBackend].
func (b *Blockchain) GetNetworkRules(ctx context.Context, blockHeight idx.Block) (*opera.Rules, error) {
	return &opera.Rules{}, nil
}

// GetReceiptsByNumber implements [ethapi.BundleApiBackend].
func (b *Blockchain) GetReceiptsByNumber(ctx context.Context, number rpc.BlockNumber) (types.Receipts, error) {
	panic("unimplemented")
}

// GetUpgradeHeights implements [ethapi.BundleApiBackend].
func (b *Blockchain) GetUpgradeHeights() []opera.UpgradeHeight {
	panic("unimplemented")
}

// HeaderByHash implements [ethapi.BundleApiBackend].
func (b *Blockchain) HeaderByHash(ctx context.Context, hash common.Hash) (*evmcore.EvmHeader, error) {
	panic("unimplemented")
}

// HeaderByNumber implements [ethapi.BundleApiBackend].
func (b *Blockchain) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*evmcore.EvmHeader, error) {
	return &evmcore.EvmHeader{
		Number: big.NewInt(1),
	}, nil
}

// HistoryPruningCutoff implements [ethapi.BundleApiBackend].
func (b *Blockchain) HistoryPruningCutoff() uint64 {
	panic("unimplemented")
}

// MaxGasLimit implements [ethapi.BundleApiBackend].
func (b *Blockchain) MaxGasLimit() uint64 {
	return 30_000_000
}

// MinGasPrice implements [ethapi.BundleApiBackend].
func (b *Blockchain) MinGasPrice() *big.Int {
	return big.NewInt(1_000)
}

// Progress implements [ethapi.BundleApiBackend].
func (b *Blockchain) Progress() rpctypes.PeerProgress {
	panic("unimplemented")
}

// RPCEVMTimeout implements [ethapi.BundleApiBackend].
func (b *Blockchain) RPCEVMTimeout() time.Duration {
	return time.Minute
}

// RPCGasCap implements [ethapi.BundleApiBackend].
func (b *Blockchain) RPCGasCap() uint64 {
	return 30_000_000
}

// RPCTxFeeCap implements [ethapi.BundleApiBackend].
func (b *Blockchain) RPCTxFeeCap() float64 {
	panic("unimplemented")
}

// ResolveRpcBlockNumberOrHash implements [ethapi.BundleApiBackend].
func (b *Blockchain) ResolveRpcBlockNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (idx.Block, error) {
	panic("unimplemented")
}

// SendTx implements [ethapi.BundleApiBackend].
func (b *Blockchain) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	panic("unimplemented")
}

// SuggestGasTipCap implements [ethapi.BundleApiBackend].
func (b *Blockchain) SuggestGasTipCap(ctx context.Context, certainty uint64) *big.Int {
	panic("unimplemented")
}

// UnprotectedAllowed implements [ethapi.BundleApiBackend].
func (b *Blockchain) UnprotectedAllowed() bool {
	panic("unimplemented")
}

// ============================================================================

type StateDb struct {
	state map[common.Address]AccountState
}

// AccessEvents implements [state.StateDB].
func (s StateDb) AccessEvents() *geth_state.AccessEvents {
	panic("unimplemented")
}

// AddAddressToAccessList implements [state.StateDB].
func (s StateDb) AddAddressToAccessList(addr common.Address) {
	panic("unimplemented")
}

// AddBalance implements [state.StateDB].
func (s StateDb) AddBalance(addr common.Address, balance *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	account := s.state[addr]
	if account.Balance == nil {
		account.Balance = big.NewInt(0)
	}
	account.Balance = new(big.Int).Add(account.Balance, balance.ToBig())
	s.state[addr] = account
	return *uint256.NewInt(0).SetBytes(account.Balance.Bytes())
}

// AddLog implements [state.StateDB].
func (s StateDb) AddLog(*types.Log) {
	panic("unimplemented")
}

// AddPreimage implements [state.StateDB].
func (s StateDb) AddPreimage(common.Hash, []byte) {
	panic("unimplemented")
}

// AddRefund implements [state.StateDB].
func (s StateDb) AddRefund(uint64) {
	panic("unimplemented")
}

// AddSlotToAccessList implements [state.StateDB].
func (s StateDb) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	panic("unimplemented")
}

// AddressInAccessList implements [state.StateDB].
func (s StateDb) AddressInAccessList(addr common.Address) bool {
	panic("unimplemented")
}

// BeginBlock implements [state.StateDB].
func (s StateDb) BeginBlock(number uint64) {
	panic("unimplemented")
}

// Copy implements [state.StateDB].
func (s StateDb) Copy() state.StateDB {
	panic("unimplemented")
}

// CreateAccount implements [state.StateDB].
func (s StateDb) CreateAccount(addr common.Address) {
	if _, found := s.state[addr]; !found {
		s.state[addr] = AccountState{}
	}
}

// CreateContract implements [state.StateDB].
func (s StateDb) CreateContract(common.Address) {
	panic("unimplemented")
}

// Empty implements [state.StateDB].
func (s StateDb) Empty(common.Address) bool {
	panic("unimplemented")
}

// EndBlock implements [state.StateDB].
func (s StateDb) EndBlock(number uint64) <-chan error {
	panic("unimplemented")
}

// EndTransaction implements [state.StateDB].
func (s StateDb) EndTransaction() {
	panic("unimplemented")
}

// Error implements [state.StateDB].
func (s StateDb) Error() error {
	panic("unimplemented")
}

// Exist implements [state.StateDB].
func (s StateDb) Exist(addr common.Address) bool {
	_, found := s.state[addr]
	return found
}

// Finalise implements [state.StateDB].
func (s StateDb) Finalise(bool) {
	panic("unimplemented")
}

// GetBalance implements [state.StateDB].
func (s StateDb) GetBalance(addr common.Address) *uint256.Int {
	if account, ok := s.state[addr]; ok {
		return uint256.NewInt(0).SetBytes(account.Balance.Bytes())
	}
	return uint256.NewInt(0)
}

// GetCode implements [state.StateDB].
func (s StateDb) GetCode(common.Address) []byte {
	return nil
}

// GetCodeHash implements [state.StateDB].
func (s StateDb) GetCodeHash(common.Address) common.Hash {
	panic("unimplemented")
}

// GetCodeSize implements [state.StateDB].
func (s StateDb) GetCodeSize(common.Address) int {
	panic("unimplemented")
}

// GetLogs implements [state.StateDB].
func (s StateDb) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	panic("unimplemented")
}

// GetNonce implements [state.StateDB].
func (s StateDb) GetNonce(addr common.Address) uint64 {
	if account, ok := s.state[addr]; ok {
		return account.Nonce
	}
	return 0
}

// GetProof implements [state.StateDB].
func (s StateDb) GetProof(addr common.Address, keys []common.Hash) (witness.Proof, error) {
	panic("unimplemented")
}

// GetRefund implements [state.StateDB].
func (s StateDb) GetRefund() uint64 {
	return 0
}

// GetState implements [state.StateDB].
func (s StateDb) GetState(common.Address, common.Hash) common.Hash {
	panic("unimplemented")
}

// GetStateAndCommittedState implements [state.StateDB].
func (s StateDb) GetStateAndCommittedState(common.Address, common.Hash) (common.Hash, common.Hash) {
	panic("unimplemented")
}

// GetStateHash implements [state.StateDB].
func (s StateDb) GetStateHash() common.Hash {
	panic("unimplemented")
}

// GetStorageRoot implements [state.StateDB].
func (s StateDb) GetStorageRoot(addr common.Address) common.Hash {
	panic("unimplemented")
}

// GetTransientState implements [state.StateDB].
func (s StateDb) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	panic("unimplemented")
}

// HasSelfDestructed implements [state.StateDB].
func (s StateDb) HasSelfDestructed(common.Address) bool {
	panic("unimplemented")
}

// InterTxSnapshot implements [state.StateDB].
func (s StateDb) InterTxSnapshot() int {
	panic("unimplemented")
}

// PointCache implements [state.StateDB].
func (s StateDb) PointCache() *utils.PointCache {
	panic("unimplemented")
}

// Prepare implements [state.StateDB].
func (s StateDb) Prepare(
	rules params.Rules,
	sender common.Address,
	coinbase common.Address,
	dest *common.Address,
	precompiles []common.Address,
	txAccesses types.AccessList,
) {
	// so far tests do not need this
}

// Release implements [state.StateDB].
func (s StateDb) Release() {
}

// RevertToInterTxSnapshot implements [state.StateDB].
func (s StateDb) RevertToInterTxSnapshot(id int) {
	panic("unimplemented")
}

// RevertToSnapshot implements [state.StateDB].
func (s StateDb) RevertToSnapshot(int) {
	panic("unimplemented")
}

// SelfDestruct implements [state.StateDB].
func (s StateDb) SelfDestruct(common.Address) uint256.Int {
	panic("unimplemented")
}

// SelfDestruct6780 implements [state.StateDB].
func (s StateDb) SelfDestruct6780(common.Address) (uint256.Int, bool) {
	panic("unimplemented")
}

// SetBalance implements [state.StateDB].
func (s StateDb) SetBalance(addr common.Address, amount *uint256.Int) {
	panic("unimplemented")
}

// SetCode implements [state.StateDB].
func (s StateDb) SetCode(common.Address, []byte, tracing.CodeChangeReason) []byte {
	panic("unimplemented")
}

// SetNonce implements [state.StateDB].
func (s StateDb) SetNonce(addr common.Address, nonce uint64, reason tracing.NonceChangeReason) {
	if account, ok := s.state[addr]; ok {
		account.Nonce = nonce
		s.state[addr] = account
	}
}

// SetState implements [state.StateDB].
func (s StateDb) SetState(common.Address, common.Hash, common.Hash) common.Hash {
	panic("unimplemented")
}

// SetStorage implements [state.StateDB].
func (s StateDb) SetStorage(addr common.Address, storage map[common.Hash]common.Hash) {
	panic("unimplemented")
}

// SetTransientState implements [state.StateDB].
func (s StateDb) SetTransientState(addr common.Address, key common.Hash, value common.Hash) {
	panic("unimplemented")
}

// SetTxContext implements [state.StateDB].
func (s StateDb) SetTxContext(thash common.Hash, ti int) {
	panic("unimplemented")
}

// SlotInAccessList implements [state.StateDB].
func (s StateDb) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	panic("unimplemented")
}

// Snapshot implements [state.StateDB].
func (s StateDb) Snapshot() int {
	return 0
}

// SubBalance implements [state.StateDB].
func (s StateDb) SubBalance(addr common.Address, slot *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	res := uint256.NewInt(0)
	if account, ok := s.state[addr]; ok {
		previous := account.Balance
		account.Balance = new(big.Int).Sub(previous, slot.ToBig())
		s.state[addr] = account
		res = uint256.NewInt(0).SetBytes(account.Balance.Bytes())
	}
	return *res
}

// SubRefund implements [state.StateDB].
func (s StateDb) SubRefund(uint64) {
	panic("unimplemented")
}

// TxIndex implements [state.StateDB].
func (s StateDb) TxIndex() int {
	panic("unimplemented")
}

// Witness implements [state.StateDB].
func (s StateDb) Witness() *stateless.Witness {
	panic("unimplemented")
}

// ============================================================================
