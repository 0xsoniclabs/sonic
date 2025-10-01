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

package evmcore

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testBerlin   = 0x01
	testLondon   = 0x02
	testShanghai = 0x03
	testCancun   = 0x04
	testPrague   = 0x05
)

func addSeeds(f *testing.F) {
	// Seed corpus with a few values to trigger different code paths.
	seeds := []struct {
		// transaction values
		txType   uint8
		nonce    uint64
		gas      uint64
		feeCap   int64
		tip      int64
		value    int64
		data     []byte
		isCreate bool
		// block context
		blockNum int64
		revision int8
		// because the number of elements in access list and authorization list
		// affects intrinsic gas cost, these values are fuzzed.
		accessListSize uint
		authListSize   uint
		// state values
		stateBalance uint64
		stateNonce   uint64
		maxGas       uint64
		baseFee      uint64
		minTip       uint64
	}{
		{ // LegacyTx, london, enough balance, next nonce, little gas
			txType:         0,
			nonce:          1,
			gas:            20,
			feeCap:         1_000,
			tip:            500,
			value:          0,
			data:           []byte("hi"),
			isCreate:       false,
			blockNum:       1_000_000_000,
			revision:       2,
			accessListSize: 0,
			authListSize:   0,
			stateBalance:   1_000_000_000,
			stateNonce:     1,
			maxGas:         50_000,
			baseFee:        1_001,
			minTip:         0,
		},
		{ // access list tx, berlin, negative value, far future nonce, low fee cap
			txType:         1,
			nonce:          1,
			gas:            42_000,
			feeCap:         1_000,
			tip:            500,
			value:          -1,
			data:           []byte("123456789123456789123456789123456789123456789123456789"),
			isCreate:       false,
			blockNum:       123_456_789,
			revision:       2,
			accessListSize: 1,
			authListSize:   0,
			stateBalance:   1_000_000,
			stateNonce:     42,
			maxGas:         5_000,
			baseFee:        5_500,
			minTip:         100,
		},
		{ // dynamic fee tx, shanghai, low balance, right nonce, low tip, create, low max gas
			txType:         2,
			nonce:          1,
			gas:            42_000,
			feeCap:         1_000,
			tip:            500,
			value:          0,
			data:           []byte(""),
			isCreate:       true,
			blockNum:       1_000_000_000,
			revision:       3,
			accessListSize: 2,
			authListSize:   0,
			stateBalance:   1_000,
			stateNonce:     1,
			maxGas:         500,
			baseFee:        1_000,
			minTip:         10_000,
		},
		{ // blob tx, cancun, no balance, good nonce,
			txType:         3,
			nonce:          1,
			gas:            42_000,
			feeCap:         1_000,
			tip:            500,
			value:          0,
			data:           []byte("some"),
			isCreate:       false,
			blockNum:       1_000_000_000,
			revision:       4,
			accessListSize: 3,
			authListSize:   0,
			stateBalance:   0,
			stateNonce:     1,
			maxGas:         50_000,
			baseFee:        500,
			minTip:         0,
		},
		{ // set code tx, prague, auth list not empty, same gas as max gas, high min tip
			txType:         4,
			nonce:          1,
			gas:            50_000,
			feeCap:         1_000,
			tip:            500,
			value:          5_000,
			data:           []byte("code"),
			isCreate:       false,
			blockNum:       1_000_000_000,
			revision:       5,
			accessListSize: 3,
			authListSize:   1,
			stateBalance:   1_000_000,
			stateNonce:     1,
			maxGas:         50_000,
			baseFee:        500,
			minTip:         1_000,
		},
		{ // a healthy transaction
			txType:         2,
			nonce:          1,
			gas:            42_000,
			feeCap:         65_000,
			tip:            500,
			value:          1,
			data:           []byte(""),
			isCreate:       false,
			blockNum:       1_000_000_000,
			revision:       5,
			accessListSize: 0,
			authListSize:   1,
			stateBalance:   1_000_000,
			stateNonce:     1,
			maxGas:         50_000,
			baseFee:        500,
			minTip:         0,
		},
	}

	for _, seed := range seeds {
		f.Add(
			seed.txType, seed.nonce, seed.gas, seed.feeCap, seed.tip, seed.value,
			seed.data, seed.isCreate, seed.blockNum, seed.revision,
			seed.accessListSize, seed.authListSize, seed.stateBalance,
			seed.stateNonce, seed.maxGas, seed.baseFee, seed.minTip,
		)
	}
}

// FuzzValidateTransaction fuzzes the validateTx function with randomly generated transactions.
func FuzzValidateTransaction(f *testing.F) {

	addSeeds(f)

	f.Fuzz(func(t *testing.T,
		// transaction values
		txType uint8, nonce, gas uint64, feeCap, tip, value int64,
		data []byte, isCreate bool,
		// block context
		blockNum int64, revision int8,
		// because the number of elements in access list and authorization list
		// affects intrinsic gas cost, these values are fuzzed,
		// but the actual values are not relevant for this test.
		accessListSize, authListSize uint,
		// state
		stateBalance, stateNonce, maxGas, baseFee, minTip uint64,
	) {

		if txType > types.SetCodeTxType {
			// Skip invalid transaction types
			t.Skip()
		}
		if revision > testPrague {
			// Skip invalid revisions
			t.Skip()
		}
		if isCreate && (txType == types.BlobTxType || txType == types.SetCodeTxType) {
			// Skip invalid transaction types for contract creation
			t.Skip()
		}

		// a full persistent state is not need. ValidateTx needs to see the same state as the processor.
		ctrl := gomock.NewController(t)

		chainId := big.NewInt(84)

		tx := makeTxOfType(
			txType, nonce, gas, feeCap,
			tip, data, value, isCreate,
			chainId, accessListSize,
			authListSize,
		)
		from, signedTx := signTxForTestWithChainId(t, tx, chainId)

		state := state.NewMockStateDB(ctrl)
		state.EXPECT().GetBalance(from).Return(uint256.NewInt(stateBalance)).AnyTimes()
		state.EXPECT().GetNonce(from).Return(stateNonce).AnyTimes()
		if txType == types.SetCodeTxType {
			state.EXPECT().GetCode(gomock.Any()).Return([]byte("some code")).AnyTimes()
		} else {
			// This must return empty because externally owned accounts cannot have
			// code prior to the Prague revision.
			state.EXPECT().GetCode(gomock.Any()).Return([]byte{}).AnyTimes()
		}
		stateExpectCalls(state)

		chain := NewMockStateReader(ctrl)
		chain.EXPECT().GetCurrentBaseFee().Return(big.NewInt(int64(baseFee))).AnyTimes()
		chain.EXPECT().MaxGasLimit().Return(maxGas).AnyTimes()

		opt, netRules := getTestTransactionsOptionFromRevision(revision, int64(minTip))

		signer := types.LatestSignerForChainID(chainId)

		subsidiesChecker := NewMocksubsidiesChecker(ctrl)

		// Validate the transaction
		validateErr := validateTx(signedTx, opt, netRules, chain, state, subsidiesChecker, signer)

		// create evm to check validateTx is consistent with processor.
		evm := makeTestEvm(blockNum, int64(baseFee), uint64(baseFee), state, revision, chainId)

		msg, err := core.TransactionToMessage(signedTx, signer, evm.Context.BaseFee)
		require.NoError(t, err)

		gp := new(core.GasPool).AddGas(maxGas)
		var usedGas uint64
		_, _, processorError := applyTransaction(msg, gp, state, big.NewInt(blockNum),
			signedTx, &usedGas, evm, nil)

		// validateTx should not reject transactions that the processor would accept
		if processorError != nil && validateErr == nil {
			// if the nonce is too high this is also acceptable for the validateTx
			if !errorContains(processorError, fmt.Errorf("nonce too high")) {
				if errors.Is(validateErr, ErrUnderpriced) {
					t.Logf("feeCap: %v, baseFee: %v", feeCap, baseFee)
				}
				t.Fatalf("\n2: validateTx: %v - applyTx: %v\n", validateErr, processorError)
			}
		}
	})
}

// errorContains checks if the error contains the subErr message.
// Returns false if either error is nil.
func errorContains(err error, subErr error) bool {
	if err == nil || subErr == nil {
		return false
	}
	return strings.Contains(err.Error(), subErr.Error())
}

// makeTxOfType creates a transaction of the specified type with the given parameters.
func makeTxOfType(txType uint8, nonce, gas uint64, feeCap, tip int64,
	data []byte, value int64, isCreate bool, chainId *big.Int,
	accessListSize, authListSize uint) types.TxData {

	feeCapBig := big.NewInt(feeCap)
	tipBig := big.NewInt(tip)

	to := common.Address{0x42}
	toPtr := &to
	if isCreate {
		// Set to nil
		toPtr = nil
	}

	accessList := make([]types.AccessTuple, accessListSize)
	accessTuple := types.AccessTuple{
		Address:     common.Address{0x42},
		StorageKeys: []common.Hash{{0x42}},
	}
	for i := range accessList {
		accessList[i] = accessTuple
	}
	authList := make([]types.SetCodeAuthorization, authListSize)
	for j := range authListSize {
		authList[j] = types.SetCodeAuthorization{}
	}

	var tx types.TxData
	switch txType {
	case types.LegacyTxType:
		tx = &types.LegacyTx{
			Nonce:    nonce,
			Gas:      gas,
			GasPrice: feeCapBig,
			To:       toPtr,
			Value:    big.NewInt(value),
			Data:     data,
		}
	case types.AccessListTxType:
		tx = &types.AccessListTx{
			ChainID:    chainId,
			Nonce:      nonce,
			Gas:        gas,
			GasPrice:   feeCapBig,
			To:         toPtr,
			Value:      big.NewInt(value),
			Data:       data,
			AccessList: accessList,
		}
	case types.DynamicFeeTxType:
		tx = &types.DynamicFeeTx{
			ChainID:    chainId,
			Nonce:      nonce,
			Gas:        gas,
			GasFeeCap:  feeCapBig,
			GasTipCap:  tipBig,
			To:         toPtr,
			Value:      big.NewInt(value),
			Data:       data,
			AccessList: accessList,
		}
	case types.BlobTxType:
		tx = &types.BlobTx{
			ChainID:    uint256.MustFromBig(chainId),
			Nonce:      nonce,
			Gas:        gas,
			GasFeeCap:  uint256.MustFromBig(feeCapBig),
			GasTipCap:  uint256.MustFromBig(tipBig),
			To:         to, // cannot be create
			Value:      uint256.NewInt(uint64(value)),
			Data:       data,
			AccessList: accessList,
		}
	case types.SetCodeTxType:
		tx = &types.SetCodeTx{
			ChainID:    uint256.MustFromBig(chainId),
			Gas:        gas,
			GasFeeCap:  uint256.MustFromBig(feeCapBig),
			GasTipCap:  uint256.MustFromBig(tipBig),
			To:         to, // cannot be create
			Value:      uint256.NewInt(uint64(value)),
			Data:       data,
			AccessList: accessList,
			AuthList:   authList,
		}
	}
	return tx
}

// stateExpectCalls sets up expected calls to the state mock object.
func stateExpectCalls(state *state.MockStateDB) {
	// expected calls to the state
	any := gomock.Any()
	state.EXPECT().Snapshot().AnyTimes()
	state.EXPECT().RevertToSnapshot(any).AnyTimes()

	// All accounts are preloaded in the state to avoid unintended early
	// exits or implicit account creation.
	state.EXPECT().Exist(any).Return(true).AnyTimes()

	state.EXPECT().CreateAccount(any).AnyTimes()
	state.EXPECT().CreateContract(any).AnyTimes()
	state.EXPECT().EndTransaction().AnyTimes()
	state.EXPECT().TxIndex().Return(4).AnyTimes()

	state.EXPECT().GetCodeHash(any).Return(types.EmptyCodeHash).AnyTimes()
	state.EXPECT().GetCodeSize(any).Return(0).AnyTimes()
	state.EXPECT().GetLogs(any, any).Return([]*types.Log{}).AnyTimes()
	state.EXPECT().GetNonce(any).Return(uint64(1)).AnyTimes()
	state.EXPECT().GetRefund().Return(uint64(0)).AnyTimes()
	state.EXPECT().GetState(any, any).Return(common.Hash{}).AnyTimes()
	state.EXPECT().GetStorageRoot(any).Return(types.EmptyRootHash).AnyTimes()

	state.EXPECT().HasSelfDestructed(any).Return(false).AnyTimes()
	state.EXPECT().SelfDestruct(any).AnyTimes()

	state.EXPECT().AddBalance(any, any, any).AnyTimes()
	state.EXPECT().AddLog(any).AnyTimes()
	state.EXPECT().AddRefund(any).AnyTimes()

	state.EXPECT().SetCode(any, any).Return([]byte{}).AnyTimes()
	state.EXPECT().SetNonce(any, any, any).AnyTimes()
	state.EXPECT().SetState(any, any, any).Return(common.Hash{}).AnyTimes()

	state.EXPECT().AddAddressToAccessList(any).AnyTimes()
	state.EXPECT().Prepare(any, any, any, any, any, any).AnyTimes()
	state.EXPECT().SubBalance(any, any, any).AnyTimes()
	state.EXPECT().Witness().AnyTimes()
}

// makeTestEvm creates a new EVM instance for testing purposes.
// It sets up the block context, chain configuration, and state database.
func makeTestEvm(blockNum, basefee int64, evmGasPrice uint64, state vm.StateDB, revision int8, chainId *big.Int) *vm.EVM {

	chainConfig := &params.ChainConfig{
		ChainID:             chainId,
		HomesteadBlock:      new(big.Int),
		EIP150Block:         new(big.Int),
		EIP155Block:         new(big.Int),
		EIP158Block:         new(big.Int),
		ByzantiumBlock:      new(big.Int),
		ConstantinopleBlock: new(big.Int),
		PetersburgBlock:     new(big.Int),
		IstanbulBlock:       new(big.Int),
		MuirGlacierBlock:    new(big.Int),
	}

	u64One := uint64(1)
	blockTime := uint64(0)

	switch revision {
	case testPrague:
		chainConfig.PragueTime = &u64One
		fallthrough
	case testCancun:
		chainConfig.CancunTime = &u64One
		fallthrough
	case testShanghai:
		chainConfig.ShanghaiTime = &u64One
		blockTime = 1
		fallthrough
	case testLondon:
		chainConfig.LondonBlock = common.Big0
		fallthrough
	case testBerlin:
		chainConfig.BerlinBlock = common.Big0
		fallthrough
	default:
		chainConfig.IstanbulBlock = common.Big0
	}
	random := common.Hash([32]byte{0x42})
	evm := vm.NewEVM(
		vm.BlockContext{
			BlockNumber: big.NewInt(blockNum),
			Difficulty:  big.NewInt(1),
			BaseFee:     big.NewInt(basefee),
			BlobBaseFee: big.NewInt(0),
			Random:      &random,
			Time:        blockTime,

			Transfer:    vm.TransferFunc(func(sd vm.StateDB, a1, a2 common.Address, i *uint256.Int) {}),
			CanTransfer: vm.CanTransferFunc(func(sd vm.StateDB, a1 common.Address, i *uint256.Int) bool { return true }),
			GetHash:     func(i uint64) common.Hash { return common.Hash{} },
		},
		state,
		chainConfig,
		vm.Config{},
	)
	evm.GasPrice = big.NewInt(int64(evmGasPrice))
	return evm
}

// signTxForTest generates a new key, signs the transaction with it, and returns
// the signer, address, and signed transaction.
func signTxForTestWithChainId(t *testing.T, tx types.TxData, chainId *big.Int) (common.Address, *types.Transaction) {
	t.Helper()
	key, err := crypto.GenerateKey()
	address := crypto.PubkeyToAddress(key.PublicKey)
	require.NoError(t, err)
	signer := types.NewPragueSigner(chainId)
	signedTx, err := types.SignTx(types.NewTx(tx), signer, key)
	require.NoError(t, err)
	return address, signedTx
}

// getTestTransactionsOptionFromRevision creates a validationOptions struct
// with the specified revision and chain ID.
func getTestTransactionsOptionFromRevision(revision int8, MinTip int64) (poolOptions, NetworkRules) {
	opt := poolOptions{
		minTip: big.NewInt(MinTip),
		// locally submitted transactions have the more relaxed validation version. Therefore we test local true.
		isLocal: true,
	}

	netRules := NetworkRules{}

	switch revision {
	case testPrague:
		netRules.eip7702 = true
		netRules.eip7623 = true
		fallthrough
	case testCancun:
		netRules.eip4844 = true
		fallthrough
	case testShanghai:
		netRules.shanghai = true
		fallthrough
	case testLondon:
		netRules.eip1559 = true
		fallthrough
	case testBerlin:
		netRules.eip2718 = true
		fallthrough
	default:
		netRules.istanbul = true
	}

	return opt, netRules
}
