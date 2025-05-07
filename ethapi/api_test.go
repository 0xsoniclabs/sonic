package ethapi

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	reflect "reflect"
	"testing"
	"time"

	cc "github.com/0xsoniclabs/carmen/go/common"
	"github.com/0xsoniclabs/carmen/go/common/amount"
	"github.com/0xsoniclabs/carmen/go/common/immutable"
	"github.com/0xsoniclabs/carmen/go/common/witness"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

func TestGetBlockReceipts(t *testing.T) {

	tests := []struct {
		name  string
		block rpc.BlockNumberOrHash
	}{
		{
			name:  "number",
			block: rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(42)),
		},
		{
			name:  "latest",
			block: rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
		},
		{
			name:  "pending",
			block: rpc.BlockNumberOrHashWithNumber(rpc.PendingBlockNumber),
		},
		{
			name:  "hash",
			block: rpc.BlockNumberOrHashWithHash(common.Hash{42}, false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receipts, err := testGetBlockReceipts(t, tt.block)
			if err != nil {
				t.Fatal(err)
			}

			if len(receipts) != 1 {
				t.Fatalf("expected 1 receipt, got %d", len(receipts))
			}
		})
	}
}

func TestAPI_GetProof(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Input address and keys for witness proof
	addr := cc.Address{1}
	keys := []string{"0x1"}
	hexKeys := []common.Hash{common.HexToHash("0x1")}

	// Return data
	codeHash := cc.Hash{2}
	storageHash := cc.Hash{3}
	balance := amount.New(4)
	nonce := cc.ToNonce(5)
	headerRoot := common.Hash{6}
	storageElements := []immutable.Bytes{immutable.NewBytes([]byte("stElement"))}
	value := cc.Value{7}
	storageProof := StorageResult{
		Key:   hexKeys[0].Hex(),
		Value: (*hexutil.Big)(new(big.Int).SetBytes(value[:])),
		Proof: toHexSlice(storageElements),
	}
	accountElements := []immutable.Bytes{immutable.NewBytes([]byte("accElement"))}

	// Mocks
	mockBackend := NewMockBackend(ctrl)
	mockState := state.NewMockStateDB(ctrl)
	mockProof := witness.NewMockProof(ctrl)
	mockHeader := &evmcore.EvmHeader{Root: headerRoot}

	blkNr := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)

	mockBackend.EXPECT().StateAndHeaderByNumberOrHash(gomock.Any(), blkNr).Return(mockState, mockHeader, nil)
	mockState.EXPECT().GetProof(common.Address(addr), hexKeys).Return(mockProof, nil)
	mockProof.EXPECT().GetState(cc.Hash(headerRoot), addr, cc.Key(hexKeys[0])).Return(value, true, nil)
	mockProof.EXPECT().GetStorageElements(cc.Hash(headerRoot), addr, cc.Key(hexKeys[0])).Return(storageElements, true)
	mockProof.EXPECT().GetAccountElements(cc.Hash(headerRoot), addr).Return(accountElements, storageHash, true)
	mockProof.EXPECT().GetCodeHash(cc.Hash(headerRoot), addr).Return(codeHash, true, nil)
	mockProof.EXPECT().GetBalance(cc.Hash(headerRoot), addr).Return(balance, true, nil)
	mockProof.EXPECT().GetNonce(cc.Hash(headerRoot), addr).Return(nonce, true, nil)
	mockState.EXPECT().Error().Return(nil)
	mockState.EXPECT().Release()

	api := NewPublicBlockChainAPI(mockBackend)

	accountProof, err := api.GetProof(context.Background(), common.Address(addr), keys, blkNr)
	require.NoError(t, err, "failed to get account")

	u256Balance := balance.Uint256()
	require.Equal(t, common.Address(addr), accountProof.Address)
	require.Equal(t, toHexSlice(accountElements), accountProof.AccountProof)
	require.Equal(t, (*hexutil.U256)(&u256Balance), accountProof.Balance)
	require.Equal(t, common.Hash(codeHash), accountProof.CodeHash)
	require.Equal(t, hexutil.Uint64(nonce.ToUint64()), accountProof.Nonce)
	require.Equal(t, common.Hash(storageHash), accountProof.StorageHash)
	require.Equal(t, []StorageResult{storageProof}, accountProof.StorageProof)
}

func TestAPI_GetAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	addr := cc.Address{1}
	codeHash := cc.Hash{2}
	storageRoot := cc.Hash{3}
	balance := amount.New(4)
	nonce := cc.ToNonce(5)
	headerRoot := common.Hash{123}

	mockBackend := NewMockBackend(ctrl)
	mockState := state.NewMockStateDB(ctrl)
	mockProof := witness.NewMockProof(ctrl)
	mockHeader := &evmcore.EvmHeader{Root: headerRoot}

	blkNr := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)

	mockBackend.EXPECT().StateAndHeaderByNumberOrHash(gomock.Any(), blkNr).Return(mockState, mockHeader, nil)
	mockState.EXPECT().GetProof(common.Address(addr), nil).Return(mockProof, nil)
	mockProof.EXPECT().GetCodeHash(cc.Hash(headerRoot), addr).Return(codeHash, true, nil)
	mockProof.EXPECT().GetAccountElements(cc.Hash(headerRoot), addr).Return(nil, storageRoot, true)
	mockProof.EXPECT().GetBalance(cc.Hash(headerRoot), addr).Return(balance, true, nil)
	mockProof.EXPECT().GetNonce(cc.Hash(headerRoot), addr).Return(nonce, true, nil)
	mockState.EXPECT().Error().Return(nil)
	mockState.EXPECT().Release()

	api := NewPublicBlockChainAPI(mockBackend)

	account, err := api.GetAccount(context.Background(), common.Address(addr), blkNr)
	require.NoError(t, err, "failed to get account")

	u256Balance := balance.Uint256()
	require.Equal(t, common.Hash(codeHash), account.CodeHash)
	require.Equal(t, common.Hash(storageRoot), account.StorageRoot)
	require.Equal(t, (*hexutil.U256)(&u256Balance), account.Balance)
	require.Equal(t, hexutil.Uint64(nonce.ToUint64()), account.Nonce)
}

func testGetBlockReceipts(t *testing.T, blockParam rpc.BlockNumberOrHash) ([]map[string]interface{}, error) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockObj := NewMockBackend(ctrl)

	header, transaction, receipts, err := getTestData()
	if err != nil {
		return nil, err
	}

	if blockParam.BlockNumber != nil {
		mockObj.EXPECT().HeaderByNumber(gomock.Any(), *blockParam.BlockNumber).Return(header, nil)
	}

	if blockParam.BlockHash != nil {
		mockObj.EXPECT().HeaderByHash(gomock.Any(), *blockParam.BlockHash).Return(header, nil)
	}

	mockObj.EXPECT().GetReceiptsByNumber(gomock.Any(), gomock.Any()).Return(receipts, nil)
	mockObj.EXPECT().GetTransaction(gomock.Any(), transaction.Hash()).Return(transaction, uint64(0), uint64(0), nil)
	mockObj.EXPECT().ChainConfig().Return(&params.ChainConfig{}).AnyTimes()

	api := NewPublicTransactionPoolAPI(
		mockObj,
		&AddrLocker{},
	)

	receiptsRes, err := api.GetBlockReceipts(context.Background(), blockParam)
	if err != nil {
		return nil, err
	}

	return receiptsRes, nil
}

func getTestData() (*evmcore.EvmHeader, *types.Transaction, types.Receipts, error) {

	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, nil, err
	}

	address := crypto.PubkeyToAddress(key.PublicKey)
	chainId := big.NewInt(1)

	transaction, err := types.SignTx(types.NewTx(&types.AccessListTx{
		ChainID:  chainId,
		Gas:      21000,
		GasPrice: big.NewInt(1),
		To:       &address,
		Nonce:    0,
	}), types.NewLondonSigner(chainId), key)
	if err != nil {
		return nil, nil, nil, err
	}

	header := &evmcore.EvmHeader{
		Number: big.NewInt(1),
	}

	receipt := types.Receipt{
		Status:  1,
		TxHash:  transaction.Hash(),
		GasUsed: 0,
	}

	receipts := types.Receipts{
		&receipt,
	}
	return header, transaction, receipts, nil
}

func TestEstimateGas(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	headerRoot := common.Hash{123}

	mockBackend := NewMockBackend(ctrl)
	mockState := state.NewMockStateDB(ctrl)
	mockHeader := &evmcore.EvmHeader{Root: headerRoot}

	blkNr := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)

	any := gomock.Any()
	mockBackend.EXPECT().StateAndHeaderByNumberOrHash(any, blkNr).Return(mockState, mockHeader, nil).AnyTimes()
	mockBackend.EXPECT().RPCGasCap().Return(uint64(10000000))
	mockBackend.EXPECT().MaxGasLimit().Return(uint64(10000000))
	mockBackend.EXPECT().GetEVM(any, any, any, any, any).DoAndReturn(getEvmFunc(mockState)).AnyTimes()
	setExpectedStateCalls(mockState)

	api := NewPublicBlockChainAPI(mockBackend)

	gas, err := api.EstimateGas(context.Background(), getTxArgs(t), &blkNr)
	require.NoError(t, err, "failed to estimate gas")
	require.Greater(t, gas, uint64(0))
}

func TestReplayTransactionOnEmptyBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBackend := NewMockBackend(ctrl)
	mockState := state.NewMockStateDB(ctrl)

	block := &evmcore.EvmBlock{}
	block.Number = big.NewInt(5)
	any := gomock.Any()
	mockBackend.EXPECT().BlockByNumber(any, any).Return(block, nil)
	mockBackend.EXPECT().StateAndHeaderByNumberOrHash(any, any).Return(mockState, nil, nil).AnyTimes()
	mockBackend.EXPECT().RPCGasCap().Return(uint64(10000000)).AnyTimes()
	mockBackend.EXPECT().GetEVM(any, any, any, any, any).DoAndReturn(getEvmFunc(mockState)).AnyTimes()
	mockBackend.EXPECT().ChainConfig().Return(&params.ChainConfig{}).AnyTimes()
	setExpectedStateCalls(mockState)

	api := NewPublicDebugAPI(mockBackend, 10000, 10000)

	_, err := api.TraceCall(context.Background(), getTxArgs(t), rpc.BlockNumberOrHashWithNumber(5), &TraceCallConfig{})
	require.NoError(t, err, "must be possible to replay tx on empty block")
}

type noBaseFeeMatcher struct {
	expected bool
}

func (m noBaseFeeMatcher) String() string {
	return fmt.Sprintf("Expected NoBaseFee in vm config for replaying transaction is %v", m.expected)
}

func (m noBaseFeeMatcher) Matches(x interface{}) bool {
	obj, ok := x.(*vm.Config)
	if !ok {
		return false
	}
	return obj.NoBaseFee == m.expected
}

func TestReplayInternalTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBackend := NewMockBackend(ctrl)
	mockState := state.NewMockStateDB(ctrl)

	// internal transaction with gas price 0
	internalTx := &types.LegacyTx{
		Nonce:    0,
		GasPrice: big.NewInt(0),
		Value:    big.NewInt(1),
		Gas:      21000,
		To:       &common.Address{0x1},
	}

	block := &evmcore.EvmBlock{}
	block.Number = big.NewInt(5)

	// Put transactions in the block
	block.Transactions = append(block.Transactions, types.NewTx(internalTx))
	block.Transactions = append(block.Transactions, types.NewTx(internalTx))

	// transaction index is 1 for obtaining state after transaction 0
	txIndex := uint64(1)

	chainConfig := &params.ChainConfig{
		ChainID:     big.NewInt(1),
		LondonBlock: big.NewInt(0),
	}

	blockCtx := vm.BlockContext{
		BlockNumber: block.Number,
		BaseFee:     big.NewInt(1_000_000_000),
		Transfer:    vm.TransferFunc(func(sd vm.StateDB, a1, a2 common.Address, i *uint256.Int) {}),
		CanTransfer: vm.CanTransferFunc(func(sd vm.StateDB, a1 common.Address, i *uint256.Int) bool { return true }),
	}

	vmConfig := opera.DefaultVMConfig
	vmConfig.NoBaseFee = true

	any := gomock.Any()
	mockBackend.EXPECT().BlockByNumber(any, any).Return(block, nil)
	mockBackend.EXPECT().StateAndHeaderByNumberOrHash(any, any).Return(mockState, nil, nil).AnyTimes()
	mockBackend.EXPECT().RPCGasCap().Return(uint64(10000000)).AnyTimes()
	mockBackend.EXPECT().GetTransaction(any, any).Return(types.NewTx(internalTx), block.NumberU64(), txIndex, nil).AnyTimes()
	mockBackend.EXPECT().GetEVM(any, any, any, noBaseFeeMatcher{expected: true}, any).DoAndReturn(getEvmFuncWithParameters(mockState, chainConfig, &blockCtx, vmConfig)).AnyTimes()
	mockBackend.EXPECT().ChainConfig().Return(chainConfig).AnyTimes()
	setExpectedStateCalls(mockState)

	// Replay transaction
	api := NewPublicDebugAPI(mockBackend, 10000, 10000)
	_, err := api.TraceTransaction(context.Background(), common.Hash{}, &tracers.TraceConfig{})
	require.NoError(t, err, "must be possible to trace internal transaction on index 0 and 1 with zero gas price")
}

func TestBlockOverrides(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBackend := NewMockBackend(ctrl)
	mockState := state.NewMockStateDB(ctrl)

	blockNr := 10
	block := &evmcore.EvmBlock{}
	block.Number = big.NewInt(int64(blockNr))

	any := gomock.Any()
	mockBackend.EXPECT().BlockByNumber(any, any).Return(block, nil).AnyTimes()
	mockBackend.EXPECT().StateAndHeaderByNumberOrHash(any, any).Return(mockState, &evmcore.EvmHeader{Number: big.NewInt(int64(blockNr))}, nil).AnyTimes()
	mockBackend.EXPECT().RPCGasCap().Return(uint64(10000000)).AnyTimes()
	mockBackend.EXPECT().ChainConfig().Return(&params.ChainConfig{}).AnyTimes()
	mockBackend.EXPECT().RPCEVMTimeout().Return(time.Duration(0)).AnyTimes()
	setExpectedStateCalls(mockState)

	expectedBlockCtx := &vm.BlockContext{
		BlockNumber: big.NewInt(5),
		Time:        0,
		Difficulty:  big.NewInt(1),
		BaseFee:     big.NewInt(1234),
		BlobBaseFee: big.NewInt(1),
	}

	// Check that the correct block context is used when creating EVM instance
	mockBackend.EXPECT().GetEVM(any, any, any, any, BlockContextMatcher{expectedBlockCtx}).DoAndReturn(getEvmFunc(mockState)).AnyTimes()

	blockOverrides := &BlockOverrides{
		Number:  (*hexutil.Big)(big.NewInt(5)),
		BaseFee: (*hexutil.Big)(big.NewInt(1234)),
	}

	// Check block overrides on debug api with debug_traceCall rpc function
	apiDebug := NewPublicDebugAPI(mockBackend, 10000, 10000)
	traceConfig := &TraceCallConfig{
		BlockOverrides: blockOverrides,
	}

	_, err := apiDebug.TraceCall(context.Background(), getTxArgs(t), rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(blockNr)), traceConfig)
	require.NoError(t, err, "debug api must be able to override block number and base fee")

	// Check block overrides on eth api with eth_call rpc function
	apiEth := NewPublicBlockChainAPI(mockBackend)

	_, err = apiEth.Call(context.Background(), getTxArgs(t), rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(blockNr)), nil, blockOverrides)
	require.NoError(t, err, "debug api must be able to override block number and base fee")

}

func TestGetTransactionReceiptReturnsNilNotError(t *testing.T) {

	txHash := common.Hash{1}

	ctrl := gomock.NewController(t)
	mockBackend := NewMockBackend(ctrl)
	mockBackend.EXPECT().GetTransaction(gomock.Any(), txHash).Return(&types.Transaction{}, uint64(0), uint64(0), nil)
	mockBackend.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).Return(nil, nil)
	mockBackend.EXPECT().ChainConfig().Return(&params.ChainConfig{}).AnyTimes()

	api := NewPublicTransactionPoolAPI(
		mockBackend,
		&AddrLocker{},
	)
	receiptsRes, err := api.GetTransactionReceipt(context.Background(), txHash)
	require.NoError(t, err)
	require.Nil(t, receiptsRes)
}

// Custom matcher to compare vm.BlockContext values
type BlockContextMatcher struct {
	expected *vm.BlockContext
}

func (m BlockContextMatcher) Matches(x interface{}) bool {
	if bc, ok := x.(*vm.BlockContext); ok {
		bcCopy := *bc
		bcCopy.Transfer = nil
		bcCopy.CanTransfer = nil
		bcCopy.GetHash = nil
		return reflect.DeepEqual(bcCopy, *m.expected)
	}
	return false
}

func (m BlockContextMatcher) String() string {
	return fmt.Sprintf("%v", m.expected)
}

func getTxArgs(t *testing.T) TransactionArgs {
	dataBytes, err := hexutil.Decode("0xe9ae5c53")
	require.NoError(t, err)

	addr := common.Address{1}

	data := hexutil.Bytes(dataBytes)
	return TransactionArgs{
		From: &addr,
		To:   &addr,
		Data: &data,
	}
}

func getEvmFunc(mockState *state.MockStateDB) func(any, any, any, any, any) (*vm.EVM, func() error, error) {
	return func(any, any, any, any, any) (*vm.EVM, func() error, error) {
		blockCtx := vm.BlockContext{
			Transfer: vm.TransferFunc(func(sd vm.StateDB, a1, a2 common.Address, i *uint256.Int) {}),
		}
		return vm.NewEVM(blockCtx, mockState, &opera.BaseChainConfig, opera.DefaultVMConfig), func() error { return nil }, nil
	}
}

func getEvmFuncWithParameters(mockState *state.MockStateDB, chainConfig *params.ChainConfig, blockContext *vm.BlockContext, vmConfig vm.Config) func(any, any, any, any, any) (*vm.EVM, func() error, error) {
	return func(any, any, any, any, any) (*vm.EVM, func() error, error) {
		return vm.NewEVM(*blockContext, mockState, chainConfig, vmConfig), func() error { return nil }, nil
	}
}

func setExpectedStateCalls(mockState *state.MockStateDB) {
	any := gomock.Any()
	mockState.EXPECT().GetBalance(any).Return(uint256.NewInt(0)).AnyTimes()
	mockState.EXPECT().SubBalance(any, any, any).AnyTimes()
	mockState.EXPECT().AddBalance(any, any, any).AnyTimes()
	mockState.EXPECT().Prepare(any, any, any, any, any, any).AnyTimes()
	mockState.EXPECT().GetNonce(any).Return(uint64(0)).AnyTimes()
	mockState.EXPECT().SetNonce(any, any, any).AnyTimes()
	mockState.EXPECT().Snapshot().AnyTimes()
	mockState.EXPECT().Exist(any).Return(true).AnyTimes()
	mockState.EXPECT().SetTxContext(any, any).AnyTimes()
	mockState.EXPECT().Release().AnyTimes()
	mockState.EXPECT().GetCode(any).Return(nil).AnyTimes()
	mockState.EXPECT().Witness().AnyTimes()
	mockState.EXPECT().GetRefund().AnyTimes()
	mockState.EXPECT().EndTransaction().AnyTimes()
	mockState.EXPECT().GetLogs(any, any).AnyTimes()
	mockState.EXPECT().TxIndex().AnyTimes()
}

func TestTransactionJSONSerialization(t *testing.T) {

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	ChainId := big.NewInt(17)

	authorization := types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(ChainId),
		Address: common.Address{42},
		Nonce:   5,
		V:       1,
		R:       *uint256.NewInt(2),
		S:       *uint256.NewInt(3),
	}

	tests := map[string]types.TxData{
		"legacy": &types.LegacyTx{
			Nonce:    0,
			To:       &common.Address{1},
			Gas:      1e6,
			GasPrice: big.NewInt(500e9),
		},
		"accessList empty list": &types.AccessListTx{
			ChainID:  ChainId,
			Nonce:    1,
			To:       &common.Address{1},
			Gas:      1e6,
			GasPrice: big.NewInt(500e9),
		},
		"accessList": &types.AccessListTx{
			ChainID:  ChainId,
			Nonce:    1,
			To:       &common.Address{1},
			Gas:      1e6,
			GasPrice: big.NewInt(500e9),
			AccessList: types.AccessList{
				{Address: common.Address{1}, StorageKeys: []common.Hash{{0x01}}},
			},
		},
		"dynamicFee": &types.DynamicFeeTx{
			ChainID:   ChainId,
			Nonce:     2,
			To:        &common.Address{1},
			Gas:       1e6,
			GasFeeCap: big.NewInt(500e9),
			GasTipCap: big.NewInt(500e9),
		},
		"blob empty list": &types.BlobTx{
			ChainID:    uint256.MustFromBig(ChainId),
			Nonce:      3,
			Gas:        1e6,
			GasFeeCap:  uint256.NewInt(500e9),
			BlobFeeCap: uint256.NewInt(500e9),
		},
		"blob": &types.BlobTx{
			ChainID:    uint256.MustFromBig(ChainId),
			Nonce:      3,
			Gas:        1e6,
			GasFeeCap:  uint256.NewInt(500e9),
			BlobFeeCap: uint256.NewInt(500e9),
			BlobHashes: []common.Hash{{0x01}},
		},
		"setCode empty list": &types.SetCodeTx{
			ChainID: uint256.MustFromBig(ChainId),
			Nonce:   4,
			To:      common.Address{42},
			Gas:     1e6,
		},
		"setCode": &types.SetCodeTx{
			ChainID: uint256.MustFromBig(ChainId),
			Nonce:   4,
			To:      common.Address{42},
			Gas:     1e6,
			AuthList: []types.SetCodeAuthorization{
				authorization,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			signed := signTransaction(t, ChainId, test, key)

			blockHash := common.Hash{1, 2, 3, 4}
			blockNumber := uint64(4321)
			index := uint64(0)
			baseFee := big.NewInt(1234)

			rpcTx := newRPCTransaction(signed, blockHash, blockNumber, index, baseFee)
			require.Equal(t, signed.Hash(), rpcTransactionToTransaction(t, rpcTx).Hash())

			encoded, err := json.Marshal(rpcTx)
			require.NoError(t, err)

			decoded := new(RPCTransaction)
			err = json.Unmarshal(encoded, decoded)
			require.NoError(t, err)
			require.Equal(t, blockHash, *decoded.BlockHash)
			require.Equal(t, int64(blockNumber), decoded.BlockNumber.ToInt().Int64())
			require.Equal(t, index, uint64(*decoded.TransactionIndex))
			require.Equal(t, signed.Hash(), rpcTransactionToTransaction(t, decoded).Hash())
			hexBigChainId := hexutil.Big(*ChainId)
			require.Equal(t, &hexBigChainId, decoded.ChainID)
		})
	}
}

func TestAPI_EIP2935_InvokesHistoryStorageContract(t *testing.T) {

	senderKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	sender := crypto.PubkeyToAddress(senderKey.PublicKey)
	recipient := common.Address{0x2}

	executeDoCall := func(t *testing.T, backend Backend, txArgs TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash) {
		var stateOverrides *StateOverride
		var blockOverrides *BlockOverrides
		timeout := time.Duration(time.Second)
		gasCap := uint64(10000000)

		result, err := DoCall(t.Context(), backend, txArgs, blockNrOrHash, stateOverrides, blockOverrides, timeout, gasCap)
		require.NoError(t, err)
		require.False(t, result.Failed())
	}

	executeStateAtTransaction := func(t *testing.T, backend Backend, txArgs TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash) {
		block := backend.CurrentBlock()

		// modify block for test: add historical transactions
		block.Transactions = []*types.Transaction{

			// first transaction will be replayed
			signTransaction(t,
				big.NewInt(250),
				&types.LegacyTx{
					To:       &recipient,
					Gas:      150_000,
					GasPrice: big.NewInt(10000000),
				},
				senderKey),

			// second transaction does not matter, we are querying state before
			// it being executed
			types.NewTx(&types.LegacyTx{}),
		}

		// querying state at tx 1 requires executing tx 0
		_, _, err := stateAtTransaction(t.Context(), block, 1, backend)
		require.NoError(t, err)
	}

	executeTraceReplayBlock := func(t *testing.T, backend Backend, txArgs TransactionArgs, blockOrHash rpc.BlockNumberOrHash) {
		api := PublicTxTraceAPI{b: backend}
		block := backend.CurrentBlock()

		tx1 := signTransaction(t,
			big.NewInt(250),
			&types.LegacyTx{
				To:       &recipient,
				Gas:      150_000,
				GasPrice: big.NewInt(10000000),
				Nonce:    0,
			},
			senderKey)

		tx2 := signTransaction(t,
			big.NewInt(250),
			&types.LegacyTx{
				To:       &recipient,
				Gas:      150_000,
				GasPrice: big.NewInt(10000000),
				Nonce:    1,
			},
			senderKey)

		block.Transactions = []*types.Transaction{tx1, tx2}
		txHash := tx2.Hash()
		_, err := api.replayBlock(t.Context(), block, &txHash, &[]hexutil.Uint{hexutil.Uint(0)})
		require.NoError(t, err)
	}

	expectedCallsFromTxCall := func(mockState *state.MockStateDB) {
		mockState.EXPECT().Prepare(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
		mockState.EXPECT().Release().AnyTimes()
		mockState.EXPECT().Snapshot()
		mockState.EXPECT().GetBalance(sender).Return(uint256.NewInt(1e18))
		mockState.EXPECT().GetNonce(sender).Return(uint64(0))
		mockState.EXPECT().SetNonce(sender, uint64(1), gomock.Any())
		mockState.EXPECT().Exist(recipient).Return(true)
		mockState.EXPECT().GetCode(recipient).Return([]byte{}).Times(2)
		mockState.EXPECT().AddBalance(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		mockState.EXPECT().SubBalance(sender, gomock.Any(), gomock.Any()).Times(2)
		mockState.EXPECT().GetRefund().Return(uint64(0)).Times(2)
	}

	expectedCallsFromHistoryStorageContract := func(mockState *state.MockStateDB) {
		mockState.EXPECT().Snapshot()
		mockState.EXPECT().AddAddressToAccessList(params.HistoryStorageAddress)
		mockState.EXPECT().GetCode(params.HistoryStorageAddress).Return(params.HistoryStorageCode).Times(2)
		mockState.EXPECT().GetCodeHash(params.HistoryStorageAddress).Return(common.Hash{})
		mockState.EXPECT().AddRefund(gomock.Any()).Times(2)
		mockState.EXPECT().SlotInAccessList(params.HistoryStorageAddress, gomock.Any())
		mockState.EXPECT().AddSlotToAccessList(params.HistoryStorageAddress, gomock.Any())
		mockState.EXPECT().GetState(params.HistoryStorageAddress, gomock.Any())
		mockState.EXPECT().SubRefund(gomock.Any())
		mockState.EXPECT().Exist(params.HistoryStorageAddress).Return(true)
		mockState.EXPECT().SubBalance(params.SystemAddress, uint256.NewInt(0), gomock.Any())
		mockState.EXPECT().GetRefund().Return(uint64(0))
		mockState.EXPECT().Finalise(true)
	}

	expectedTraceReplayBlock := func(mockState *state.MockStateDB) {
		mockState.EXPECT().SetTxContext(gomock.Any(), gomock.Any())
		mockState.EXPECT().GetCode(sender).Return([]byte{})
		mockState.EXPECT().GetNonce(sender).Return(uint64(0))
		mockState.EXPECT().EndTransaction()

		mockState.EXPECT().SetTxContext(gomock.Any(), gomock.Any())
		mockState.EXPECT().GetNonce(sender).Return(uint64(1)).Times(2)
		mockState.EXPECT().GetCode(sender).Return([]byte{})
		mockState.EXPECT().GetBalance(sender).Return(uint256.NewInt(1e18))
		mockState.EXPECT().SubBalance(sender, gomock.Any(), gomock.Any())

		mockState.EXPECT().Prepare(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
		mockState.EXPECT().SetNonce(sender, uint64(2), gomock.Any())
		mockState.EXPECT().GetCode(recipient).Return([]byte{})
		mockState.EXPECT().Snapshot()
		mockState.EXPECT().Exist(recipient)
		mockState.EXPECT().GetRefund().Times(2)
		mockState.EXPECT().EndTransaction().Times(2)
		mockState.EXPECT().TxIndex()
	}

	tests := map[string]struct {
		features          opera.FeatureSet
		extraSetupBackend func(*MockBackend)
		setupStateDb      func(*state.MockStateDB)
		call              func(*testing.T, Backend, TransactionArgs, rpc.BlockNumberOrHash)
	}{
		"DoCall sonic": {
			features:     opera.SonicFeatures,
			setupStateDb: expectedCallsFromTxCall,
			call:         executeDoCall,
		},
		"DoCall allegro": {
			features: opera.AllegroFeatures,
			setupStateDb: func(mockState *state.MockStateDB) {
				expectedCallsFromHistoryStorageContract(mockState)
				expectedCallsFromTxCall(mockState)
			},
			call: executeDoCall,
		},
		"StateAtTransaction sonic": {
			features: opera.SonicFeatures,
			setupStateDb: func(mockState *state.MockStateDB) {
				mockState.EXPECT().SetTxContext(gomock.Any(), gomock.Any())
				mockState.EXPECT().GetCode(sender).Return([]byte{})
				mockState.EXPECT().GetNonce(sender).Return(uint64(0))
				mockState.EXPECT().EndTransaction()

				expectedCallsFromTxCall(mockState)
			},
			call: executeStateAtTransaction,
		},
		"StateAtTransaction allegro": {
			features: opera.AllegroFeatures,
			setupStateDb: func(mockState *state.MockStateDB) {
				mockState.EXPECT().SetTxContext(gomock.Any(), gomock.Any())
				mockState.EXPECT().GetCode(sender).Return([]byte{})
				mockState.EXPECT().GetNonce(sender).Return(uint64(0))
				mockState.EXPECT().EndTransaction()

				expectedCallsFromHistoryStorageContract(mockState)
				expectedCallsFromTxCall(mockState)
			},
			call: executeStateAtTransaction,
		},
		"trace_replayBlock sonic": {
			features: opera.SonicFeatures,
			extraSetupBackend: func(mockBackend *MockBackend) {
				mockBackend.EXPECT().GetReceiptsByNumber(gomock.Any(), gomock.Any()).
					Return(types.Receipts{
						{Status: types.ReceiptStatusSuccessful},
						{Status: types.ReceiptStatusSuccessful},
					}, nil)
				mockBackend.EXPECT().RPCEVMTimeout()
			},
			setupStateDb: func(mockState *state.MockStateDB) {
				expectedCallsFromTxCall(mockState)

				expectedTraceReplayBlock(mockState)
			},
			call: executeTraceReplayBlock,
		},
		"trace_replayBlock allegro": {
			features: opera.AllegroFeatures,
			extraSetupBackend: func(mockBackend *MockBackend) {
				mockBackend.EXPECT().GetReceiptsByNumber(gomock.Any(), gomock.Any()).
					Return(types.Receipts{
						{Status: types.ReceiptStatusSuccessful},
						{Status: types.ReceiptStatusSuccessful},
					}, nil)
				mockBackend.EXPECT().RPCEVMTimeout()
			},
			setupStateDb: func(mockState *state.MockStateDB) {
				expectedCallsFromHistoryStorageContract(mockState)
				expectedCallsFromTxCall(mockState)
				expectedTraceReplayBlock(mockState)
			},
			call: executeTraceReplayBlock,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctrl := gomock.NewController(t)

			blockOrHash := rpc.BlockNumberOrHashWithNumber(1)

			header := evmcore.EvmHeader{
				// return any block but 0, 0 is genesis and has special semantics
				Number:  big.NewInt(1),
				BaseFee: big.NewInt(10000000),
			}

			mockState := state.NewMockStateDB(ctrl)
			require.NotNil(t, test.setupStateDb, "setupStateDb must be defined")
			test.setupStateDb(mockState)

			backend := NewMockBackend(ctrl)
			backend.EXPECT().StateAndHeaderByNumberOrHash(gomock.Any(), blockOrHash).
				Return(mockState, &header, nil).AnyTimes()
			backend.EXPECT().GetEVM(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(makeTestEVM(test.features)).AnyTimes()
			backend.EXPECT().CurrentBlock().AnyTimes().Return(&evmcore.EvmBlock{EvmHeader: header})
			backend.EXPECT().ChainConfig().AnyTimes().Return(makeChainConfig(test.features))
			backend.EXPECT().SuggestGasTipCap(gomock.Any(), gomock.Any()).AnyTimes().Return(big.NewInt(1))
			backend.EXPECT().MinGasPrice().AnyTimes().Return(big.NewInt(1))
			backend.EXPECT().RPCGasCap().AnyTimes().Return(uint64(10000000))
			backend.EXPECT().MaxGasLimit().AnyTimes().Return(uint64(10000000))
			backend.EXPECT().StateAndHeaderByNumberOrHash(gomock.Any(), gomock.Any()).
				Return(mockState, &header, nil).AnyTimes()
			if test.extraSetupBackend != nil {
				test.extraSetupBackend(backend)
			}

			// every test uses the same transaction
			nonce := hexutil.Uint64(0)
			gas := hexutil.Uint64(150_000)
			gasFeeCap := hexutil.Big(*big.NewInt(10000000))
			txArgs := TransactionArgs{
				From:         &sender,
				To:           &recipient,
				Nonce:        &nonce,
				Gas:          &gas,
				MaxFeePerGas: &gasFeeCap,
			}

			test.call(t, backend, txArgs, blockOrHash)
		})
	}
}

// makeChainConfig allows to create a chain config with a given set of features
func makeChainConfig(features opera.FeatureSet) *params.ChainConfig {
	switch features {
	case opera.SonicFeatures:
		return opera.MainNetRules().EvmChainConfig(
			[]opera.UpgradeHeight{
				{Upgrades: opera.SonicFeatures.ToUpgrades(), Height: 0},
			})
	case opera.AllegroFeatures:
		return opera.MainNetRules().EvmChainConfig(
			[]opera.UpgradeHeight{
				{Upgrades: opera.SonicFeatures.ToUpgrades(), Height: 0},
				{Upgrades: opera.AllegroFeatures.ToUpgrades(), Height: 1},
			})
	default:
		panic(fmt.Errorf("unsupported featureSet %v", features))
	}
}

// makeTestEVM allows to create an evm instance to use in tests with a given set of features
func makeTestEVM(features opera.FeatureSet) func(
	ctx context.Context,
	statedb vm.StateDB,
	header *evmcore.EvmHeader,
	vmConfig *vm.Config,
	blockContext *vm.BlockContext,
) (*vm.EVM, func() error, error) {

	return func(ctx context.Context, statedb vm.StateDB, header *evmcore.EvmHeader, vmConfig *vm.Config, blockContext *vm.BlockContext) (*vm.EVM, func() error, error) {

		chainConfig := makeChainConfig(features)

		if blockContext == nil {
			ethHeader := header.EthHeader()
			chainContext := &FakeChainContext{
				header:      ethHeader,
				chainConfig: chainConfig,
			}
			author := common.Address{}
			tmp := core.NewEVMBlockContext(ethHeader, chainContext, &author)
			blockContext = &tmp
		}

		evm := vm.NewEVM(*blockContext, statedb, chainConfig, *vmConfig)
		return evm, func() error { return nil }, nil
	}
}

func signTransaction(
	t *testing.T,
	chainId *big.Int,
	payload types.TxData,
	key *ecdsa.PrivateKey,
) *types.Transaction {
	t.Helper()
	res, err := types.SignTx(
		types.NewTx(payload),
		types.NewPragueSigner(chainId),
		key)
	require.NoError(t, err)
	return res
}

func rpcTransactionToTransaction(t *testing.T, tx *RPCTransaction) *types.Transaction {
	t.Helper()

	switch tx.Type {
	case types.LegacyTxType:
		return types.NewTx(&types.LegacyTx{
			Nonce:    uint64(tx.Nonce),
			Gas:      uint64(tx.Gas),
			GasPrice: tx.GasPrice.ToInt(),
			To:       tx.To,
			Value:    tx.Value.ToInt(),
			Data:     tx.Input,
			V:        tx.V.ToInt(),
			R:        tx.R.ToInt(),
			S:        tx.S.ToInt(),
		})
	case types.AccessListTxType:
		return types.NewTx(&types.AccessListTx{
			ChainID:    tx.ChainID.ToInt(),
			Nonce:      uint64(tx.Nonce),
			Gas:        uint64(tx.Gas),
			GasPrice:   tx.GasPrice.ToInt(),
			To:         tx.To,
			Value:      tx.Value.ToInt(),
			Data:       tx.Input,
			AccessList: *tx.Accesses,
			V:          tx.V.ToInt(),
			R:          tx.R.ToInt(),
			S:          tx.S.ToInt(),
		})
	case types.DynamicFeeTxType:
		return types.NewTx(&types.DynamicFeeTx{
			ChainID:    tx.ChainID.ToInt(),
			Nonce:      uint64(tx.Nonce),
			Gas:        uint64(tx.Gas),
			GasFeeCap:  tx.GasFeeCap.ToInt(),
			GasTipCap:  tx.GasTipCap.ToInt(),
			To:         tx.To,
			Value:      tx.Value.ToInt(),
			Data:       tx.Input,
			AccessList: *tx.Accesses,
			V:          tx.V.ToInt(),
			R:          tx.R.ToInt(),
			S:          tx.S.ToInt(),
		})
	case types.BlobTxType:
		return types.NewTx(&types.BlobTx{
			ChainID:    uint256.MustFromBig(tx.ChainID.ToInt()),
			Nonce:      uint64(tx.Nonce),
			Gas:        uint64(tx.Gas),
			GasFeeCap:  uint256.MustFromBig(tx.GasFeeCap.ToInt()),
			GasTipCap:  uint256.MustFromBig(tx.GasTipCap.ToInt()),
			To:         *tx.To,
			Value:      uint256.MustFromBig(tx.Value.ToInt()),
			Data:       tx.Input,
			AccessList: *tx.Accesses,
			BlobFeeCap: uint256.MustFromBig(tx.MaxFeePerBlobGas.ToInt()),
			BlobHashes: tx.BlobVersionedHashes,
			V:          uint256.MustFromBig(tx.V.ToInt()),
			R:          uint256.MustFromBig(tx.R.ToInt()),
			S:          uint256.MustFromBig(tx.S.ToInt()),
		})

	case types.SetCodeTxType:
		return types.NewTx(&types.SetCodeTx{
			ChainID:    uint256.MustFromBig(tx.ChainID.ToInt()),
			Nonce:      uint64(tx.Nonce),
			Gas:        uint64(tx.Gas),
			GasFeeCap:  uint256.MustFromBig(tx.GasFeeCap.ToInt()),
			GasTipCap:  uint256.MustFromBig(tx.GasTipCap.ToInt()),
			To:         *tx.To,
			Value:      uint256.MustFromBig(tx.Value.ToInt()),
			Data:       tx.Input,
			AccessList: *tx.Accesses,
			AuthList:   tx.AuthorizationList,
			V:          uint256.MustFromBig(tx.V.ToInt()),
			R:          uint256.MustFromBig(tx.R.ToInt()),
			S:          uint256.MustFromBig(tx.S.ToInt()),
		})
	default:
		t.Error("unsupported transaction type ", tx.Type)
		return nil
	}
}

// fakeChainContext is a fake implementation of evm.ChainContext
// to use in tests
type FakeChainContext struct {
	header      *types.Header
	chainConfig *params.ChainConfig
}

func (fcc *FakeChainContext) Engine() consensus.Engine {
	// currently not used in tests, if needed: engine will have to be faked
	return nil
}

func (fcc *FakeChainContext) GetHeader(common.Hash, uint64) *types.Header {
	return fcc.header
}

func (fcc *FakeChainContext) Config() *params.ChainConfig {
	return fcc.chainConfig
}
