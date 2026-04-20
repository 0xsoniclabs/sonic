// Copyright 2026 Sonic Operations Ltd
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

package geth_compatibility

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// List of all functions in geth that we depend on and want to ensure we don't miss behavior
// changes in:
//
// | Package                 | Functions                                                          |
// |-------------------------|--------------------------------------------------------------------|
// | go-ethereum             | FilterQuery                                                        |
// | accounts                | ParseDerivationPath, TextHash                                      |
// | accounts/abi            | ConvertType, JSON, UnpackRevert                                    |
// | accounts/abi/bind       | DeployContract, NewBoundContract, NewKeyedTransactorWithChainID    |
// | accounts/external       | NewExternalBackend                                                 |
// | accounts/keystore       | Add, DecryptDataV3, EncryptDataV3, Get, NewKeyStore, StoreKey      |
// | accounts/scwallet       | NewHub                                                             |
// | accounts/usbwallet      | NewLedgerHub, NewTrezorHubWithHID, NewTrezorHubWithWebUSB          |
// | common                  | BigToAddress, BigToHash, Bytes2Hex, BytesToAddress, BytesToHash,   |
// |                         | CopyBytes, FromHex, Hex2Bytes, HexToAddress, HexToHash,            |
// |                         | IsHexAddress, LeftPadBytes, PrettyDuration, RightPadBytes,         |
// |                         | StorageSize                                                        |
// | common/fdlimit          | Maximum, Raise                                                     |
// | common/hexutil          | Decode, DecodeBig, DecodeUint64, Encode, EncodeBig,                |
// |                         | EncodeUint64, MustDecode, U256                                     |
// | common/math             | HexOrDecimal64, NewHexOrDecimal256                                 |
// | consensus/ethash        | NewFaker                                                           |
// | consensus/misc/eip1559  | CalcBaseFee                                                        |
// | consensus/misc/eip4844  | CalcBlobFee                                                        |
// | console                 | Evaluate, Interactive, New, Stop, Welcome                          |
// | core                    | ApplyMessage, FloorDataGas, GenerateChain, IntrinsicGas,           |
// |                         | NewEVMBlockContext, TransactionToMessage                           |
// | core/forkid             | NewId                                                              |
// | core/rawdb              | NewDatabase, NewMemoryDatabase, NewTable, ReadCanonicalHash,       |
// |                         | ReadCode, ReadHeadBlockHash, ReadHeader, ReadHeaderNumber,         |
// |                         | ReadReceipts, WriteBlock, WriteCanonicalHash, WriteHeadBlockHash,  |
// |                         | WriteReceipts                                                      |
// | core/state              | NewAccessEvents                                                    |
// | core/types              | CreateBloom, DeriveSha, LatestSigner, LatestSignerForChainID,      |
// |                         | MergeBloom, MustSignNewTx, NewBlock, NewCancunSigner,              |
// |                         | NewContractCreation, NewEIP155Signer, NewEIP2930Signer,            |
// |                         | NewLondonSigner, NewPragueSigner, NewReceipt, NewTransaction,      |
// |                         | NewTx, ParseDelegation, Sender, SignNewTx, SignSetCode, SignTx,    |
// |                         | TxByNonce, TxDifference                                            |
// | core/vm                 | ActivePrecompiledContracts, ActivePrecompiles, CanTransferFunc,    |
// |                         | NewEVM, NewEvmInterpreter, TransferFunc                            |
// | crypto                  | CreateAddress, FromECDSA, FromECDSAPub, GenerateKey, HexToECDSA,   |
// |                         | Keccak256, Keccak256Hash, LoadECDSA, NewKeccakState,               |
// |                         | PubkeyToAddress, S256, SaveECDSA, SigToPub, Sign, ToECDSA,         |
// |                         | VerifySignature                                                    |
// | crypto/kzg4844          | BlobToCommitment, ComputeBlobProof                                 |
// | eth/tracers/logger      | Hooks, NewAccessListTracer, NewStructLogger                        |
// | ethclient               | Dial, NewClient                                                    |
// | ethdb/leveldb           | New                                                                |
// | ethdb/pebble            | New                                                                |
// | event                   | NewSubscription                                                    |
// | log                     | Crit, Debug, Error, FormatSlogValue, FromLegacyLevel, Info,        |
// |                         | LevelAlignedString, New, NewGlogHandler, NewLogger,                |
// |                         | NewTerminalHandler, NewTerminalHandlerWithLevel, Root, SetDefault, |
// |                         | Trace, Warn                                                        |
// | metrics                 | CollectProcessMetrics, Enable, Enabled, GetOrRegisterCounter,      |
// |                         | GetOrRegisterGauge, GetOrRegisterGaugeInfo, GetOrRegisterMeter,    |
// |                         | GetOrRegisterResettingTimer, GetOrRegisterTimer,                   |
// |                         | NewRegisteredGauge, Unregister                                     |
// | metrics/exp             | Exp, Setup                                                         |
// | metrics/influxdb        | InfluxDBV2WithTags, InfluxDBWithTags                               |
// | node                    | Close, HTTPEndpoint, New, Server, Wait, WithHostname               |
// | p2p                     | NewPeer, Send                                                      |
// | p2p/discover/discfilter | Ban, Banned, BannedDynamic, Enable                                 |
// | p2p/dnsdisc             | NewClient                                                          |
// | p2p/enode               | MustParse, NewFairMix, Parse, ParseV4, PubkeyToIDV4, String,       |
// |                         | UnmarshalText                                                      |
// | p2p/nat                 | Any, Parse                                                         |
// | p2p/netutil             | ParseNetlist                                                       |
// | rlp                     | Decode, DecodeBytes, Encode, EncodeToBytes, Get, NewStream, Set,   |
// |                         | Split                                                              |
// | rpc                     | BlockNumberOrHashWithHash, BlockNumberOrHashWithNumber, Dial,      |
// |                         | NewID, NotifierFromContext, SetExecutionTimeLimit                  |
// | rpc/rpc_test_utils      | GetRpcApis                                                         |
// | trie                    | NewStackTrie, NewStateTrie, StateTrieID                            |
// | triedb                  | NewDatabase                                                        |
// |-------------------------|--------------------------------------------------------------------|

func TestGethDependency_EffectiveGasTipProducesUnchangedResults(t *testing.T) {
	tests := map[string]struct {
		baseFee     int64
		gasTipCap   int64
		gasFeeCap   int64
		expectedTip int64
	}{
		"all zero": {
			baseFee:     0,
			gasTipCap:   0,
			gasFeeCap:   0,
			expectedTip: 0,
		},
		"all equal": {
			baseFee:     50,
			gasTipCap:   50,
			gasFeeCap:   50,
			expectedTip: 0,
		},
		"tip limited by tip cap": {
			baseFee:     50,
			gasTipCap:   20,
			gasFeeCap:   100,
			expectedTip: 20,
		},
		"tip limited by fee cap": {
			baseFee:     50,
			gasTipCap:   100,
			gasFeeCap:   70,
			expectedTip: 20,
		},
		"tip cap equal to fee cap minus base fee": {
			baseFee:     50,
			gasTipCap:   50,
			gasFeeCap:   100,
			expectedTip: 50,
		},
		"fee cap equal to base fee": {
			baseFee:     50,
			gasTipCap:   10,
			gasFeeCap:   50,
			expectedTip: 0,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			txData := &types.DynamicFeeTx{
				GasTipCap: big.NewInt(test.gasTipCap),
				GasFeeCap: big.NewInt(test.gasFeeCap),
			}
			tx := types.NewTx(txData)
			effectiveTip, err := tx.EffectiveGasTip(big.NewInt(test.baseFee))
			require.NoError(t, err)
			require.Zero(t, big.NewInt(test.expectedTip).Cmp(effectiveTip))
		})
	}
}

func TestGethDependency_EffectiveGasTipReturnsUnchangedErrors(t *testing.T) {
	overflow := big.NewInt(0).Lsh(big.NewInt(1), 256) // 2^256 overflows uint256
	tests := map[string]struct {
		baseFee   *big.Int
		gasTipCap *big.Int
		gasFeeCap *big.Int

		// Although returned values should be ignored when an error is expected,
		// we want to catch all behavior changes.
		expectedTip *big.Int
	}{
		"fee cap less than base fee": {
			baseFee:     big.NewInt(50),
			gasTipCap:   big.NewInt(10),
			gasFeeCap:   big.NewInt(40),
			expectedTip: big.NewInt(10),
		},
		"base fee overflow": {
			baseFee:     overflow,
			expectedTip: nil,
		},
		"tip cap overflow": {
			gasTipCap:   overflow,
			expectedTip: big.NewInt(0),
		},
		"fee cap overflow": {
			gasFeeCap:   overflow,
			expectedTip: big.NewInt(0),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			txData := &types.DynamicFeeTx{
				GasTipCap: test.gasTipCap,
				GasFeeCap: test.gasFeeCap,
			}
			tx := types.NewTx(txData)
			tip, err := tx.EffectiveGasTip(test.baseFee)
			require.Error(t, err)

			if test.expectedTip == nil {
				require.Nil(t, tip)
			} else {
				require.Zero(t, test.expectedTip.Cmp(tip))
			}
		})
	}
}

func TestGethDependency_IntrinsicGasProducesUnchangedResults(t *testing.T) {
	tests := map[string]struct {
		data               []byte
		accessList         types.AccessList
		authList           []types.SetCodeAuthorization
		isContractCreation bool
		expectedGas        uint64
	}{
		"empty": {
			expectedGas: 21_000,
		},
		"data": {
			data:        []byte{0x01, 0x00, 0x02, 0x00, 0x03},
			expectedGas: 21_000 + 2*4 + 3*16,
		},
		"access list": {
			accessList: []types.AccessTuple{
				{Address: common.Address{0x01}, StorageKeys: []common.Hash{{0x02}}},
				{Address: common.Address{0x03}},
			},
			expectedGas: 21_000 + 2_400 + 1_900 + 2_400,
		},
		"auth list": {
			authList:    []types.SetCodeAuthorization{{}, {}, {}},
			expectedGas: 21_000 + 25_000*3,
		},
		"contract creation": {
			isContractCreation: true,
			expectedGas:        53_000,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gas, err := core.IntrinsicGas(test.data, test.accessList, test.authList,
				test.isContractCreation, true, true, true)

			// Although IntrinsicGas returns an error for inputs that overflow gas,
			// these are not reachable in practice. For the worst case scenario,
			// the input data needs to be (2^64 - 1 - 53_000) / (16 * 2) which is
			// greater than 5e17 bytes.
			require.NoError(t, err)

			require.Equal(t, test.expectedGas, gas)
		})
	}
}

func TestGethDependency_FloorDataGasProducesUnchangedResults(t *testing.T) {
	nonZeroMultiplier := uint64(4)
	gasPerToken := uint64(10)
	tests := map[string]struct {
		data        []byte
		expectedGas uint64
	}{
		"empty": {
			data:        []byte{},
			expectedGas: 21_000,
		},
		"all zero": {
			data:        []byte{0x00, 0x00, 0x00},
			expectedGas: 21_000 + 3*gasPerToken,
		},
		"all non-zero": {
			data:        []byte{0x01, 0x02, 0x03},
			expectedGas: 21_000 + 3*nonZeroMultiplier*gasPerToken,
		},
		"mixed": {
			data:        []byte{0x00, 0x01, 0x00, 0x02, 0x03},
			expectedGas: 21_000 + 2*gasPerToken + 3*nonZeroMultiplier*gasPerToken,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gas, err := core.FloorDataGas(test.data)

			// Similar to IntrinsicGas, FloorDataGas returns an error for
			// gas overflow inputs, but these are not reachable in practice.
			require.NoError(t, err)

			require.Equal(t, test.expectedGas, gas)
		})
	}
}
