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
	mockBackend.EXPECT().GetEVM(any, any, any, any, any, any).DoAndReturn(getEvmFunc(mockState)).AnyTimes()
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
	mockBackend.EXPECT().GetEVM(any, any, any, any, any, any).DoAndReturn(getEvmFunc(mockState)).AnyTimes()
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
	mockBackend.EXPECT().GetEVM(any, any, any, any, noBaseFeeMatcher{expected: true}, any).DoAndReturn(getEvmFuncWithParameters(mockState, chainConfig, &blockCtx, vmConfig)).AnyTimes()
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
	mockBackend.EXPECT().GetEVM(any, any, any, any, any, BlockContextMatcher{expectedBlockCtx}).DoAndReturn(getEvmFunc(mockState)).AnyTimes()

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

func getEvmFunc(mockState *state.MockStateDB) func(interface{}, interface{}, interface{}, interface{}, interface{}, interface{}) (*vm.EVM, func() error, error) {
	return func(interface{}, interface{}, interface{}, interface{}, interface{}, interface{}) (*vm.EVM, func() error, error) {
		blockCtx := vm.BlockContext{
			Transfer: vm.TransferFunc(func(sd vm.StateDB, a1, a2 common.Address, i *uint256.Int) {}),
		}
		return vm.NewEVM(blockCtx, mockState, &opera.BaseChainConfig, opera.DefaultVMConfig), func() error { return nil }, nil
	}
}

func getEvmFuncWithParameters(mockState *state.MockStateDB, chainConfig *params.ChainConfig, blockContext *vm.BlockContext, vmConfig vm.Config) func(interface{}, interface{}, interface{}, interface{}, interface{}, interface{}) (*vm.EVM, func() error, error) {
	return func(interface{}, interface{}, interface{}, interface{}, interface{}, interface{}) (*vm.EVM, func() error, error) {
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

	authorization := types.SetCodeAuthorization{
		ChainID: *uint256.NewInt(17),
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
			Nonce:    1,
			To:       &common.Address{1},
			Gas:      1e6,
			GasPrice: big.NewInt(500e9),
		},
		"accessList": &types.AccessListTx{
			Nonce:    1,
			To:       &common.Address{1},
			Gas:      1e6,
			GasPrice: big.NewInt(500e9),
			AccessList: types.AccessList{
				{Address: common.Address{1}, StorageKeys: []common.Hash{{0x01}}},
			},
		},
		"dynamicFee": &types.DynamicFeeTx{
			Nonce:     2,
			To:        &common.Address{1},
			Gas:       1e6,
			GasFeeCap: big.NewInt(500e9),
			GasTipCap: big.NewInt(500e9),
		},
		"blob empty list": &types.BlobTx{
			Nonce:      3,
			Gas:        1e6,
			GasFeeCap:  uint256.NewInt(500e9),
			BlobFeeCap: uint256.NewInt(500e9),
		},
		"blob": &types.BlobTx{
			Nonce:      3,
			Gas:        1e6,
			GasFeeCap:  uint256.NewInt(500e9),
			BlobFeeCap: uint256.NewInt(500e9),
			BlobHashes: []common.Hash{{0x01}},
		},
		"setCode empty list": &types.SetCodeTx{
			Nonce: 4,
			To:    common.Address{42},
			Gas:   1e6,
		},
		"setCode": &types.SetCodeTx{
			Nonce: 4,
			To:    common.Address{42},
			Gas:   1e6,
			AuthList: []types.SetCodeAuthorization{
				authorization,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			signed := signTransaction(t, big.NewInt(1), test, key)

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
		})
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
