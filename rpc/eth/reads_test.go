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

package eth_test

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/eth"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// newEth starts a harness over the core with the namespace registered.
func newEth(t *testing.T, fake *tests.FakeCore, mutations ...tests.Mutate) *tests.Harness {
	t.Helper()
	return tests.NewHarness(t, fake, func(dispatcher *dispatch.Dispatcher, environment *exec.Environment) {
		eth.Register(dispatcher, environment, fake)
	}, mutations...)
}

// chainWithTransfer returns a core whose served head is block 1, holding a
// signed transfer and a log, and an unpublished block 2.
func chainWithTransfer(t *testing.T) (*tests.FakeCore, *evmcore.EvmBlock, *types.Transaction) {
	t.Helper()
	fake := tests.NewFakeCore(t)
	signer := tests.NewSigner(t)
	recipient := common.HexToAddress("0x2")
	transfer := signer.Sign(t, &types.DynamicFeeTx{
		ChainID: big.NewInt(tests.FakeChainID), Nonce: 0, To: &recipient, Gas: 21000,
		GasTipCap: big.NewInt(5), GasFeeCap: big.NewInt(2_000_000_000), Value: big.NewInt(7),
	})
	log := &types.Log{Address: recipient, Topics: []common.Hash{{1}}, Data: []byte{1}}
	block := fake.AppendBlock(tests.BlockSpec{
		Transactions: types.Transactions{transfer},
		Receipts:     types.Receipts{{Status: types.ReceiptStatusSuccessful, GasUsed: 21000, Logs: []*types.Log{log}}},
	})
	fake.PublishHead(1)
	fake.AppendBlock(tests.BlockSpec{})
	return fake, block, transfer
}

func decodeMap(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	return tests.Decode[map[string]any](t, raw)
}

func TestChainID_IsTheNetworkID(t *testing.T) {
	harness := newEth(t, tests.NewFakeCore(t))
	require.JSONEq(t, `"0xfa3"`, string(harness.Result("eth_chainId")))
}

func TestBlockNumber_IsTheServedHead(t *testing.T) {
	fake, _, _ := chainWithTransfer(t)
	harness := newEth(t, fake)
	require.JSONEq(t, `"0x1"`, string(harness.Result("eth_blockNumber")), "block 2 is committed but not archived")
}

func TestSyncing_IsFalseForAFreshHead(t *testing.T) {
	fake := tests.NewFakeCore(t)
	harness := newEth(t, fake)
	fake.SetProgress(core.Progress{CurrentBlockTime: inter.Timestamp(time.Now().UnixNano())})
	require.JSONEq(t, `false`, string(harness.Result("eth_syncing")))
}

func TestSyncing_ReportsProgressForAnOldHead(t *testing.T) {
	fake := tests.NewFakeCore(t)
	harness := newEth(t, fake)
	old := time.Now().Add(-2 * time.Hour)
	fake.SetProgress(core.Progress{
		CurrentEpoch: 3, CurrentBlock: 10, CurrentBlockHash: common.Hash{9},
		CurrentBlockTime: inter.Timestamp(old.UnixNano()), HighestBlock: 20, HighestEpoch: 4,
	})
	require.JSONEq(t, `{
		"startingBlock": "0x0", "currentEpoch": "0x3", "currentBlock": "0xa",
		"currentBlockHash": "0x0900000000000000000000000000000000000000000000000000000000000000",
		"currentBlockTime": "`+hexutil.Uint64(old.UnixNano()).String()+`",
		"highestBlock": "0x14", "highestEpoch": "0x4", "pulledStates": "0x0", "knownStates": "0x0"
	}`, string(harness.Result("eth_syncing")))
}

func TestCurrentEpoch_IsTheOpenEpoch(t *testing.T) {
	fake := tests.NewFakeCore(t)
	harness := newEth(t, fake)
	fake.SetProgress(core.Progress{CurrentEpoch: 7})
	require.JSONEq(t, `"0x7"`, string(harness.Result("eth_currentEpoch")))
}

func TestGetRules_AnswersTheRulesInForceAtABlock(t *testing.T) {
	fake, _, _ := chainWithTransfer(t)
	harness := newEth(t, fake)
	blockRules := opera.FakeNetRules(opera.GetAllegroUpgrades())
	blockRules.Name = "of the block"
	openRules := opera.FakeNetRules(opera.GetAllegroUpgrades())
	openRules.Name = "of the open epoch"
	fake.SetEpochRules(1, blockRules)
	fake.SetOpenRules(openRules)

	name := func(raw json.RawMessage) any { return decodeMap(t, raw)["Name"] }
	require.Equal(t, "of the block", name(harness.Result("eth_getRules", "0x1")))
	require.JSONEq(t, `null`, string(harness.Result("eth_getRules", "0x2")), "a block above the served head")
	for _, tag := range []string{"latest", "pending", "safe", "finalized"} {
		require.Equal(t, "of the block", name(harness.Result("eth_getRules", tag)), tag)
	}
	fake.AppendAndPublish(tests.BlockSpec{})
	harness.WaitForHead(3)
	require.Equal(t, "of the open epoch", name(harness.Result("eth_getRules", "latest")))
}

func TestGetRules_ADecodingErrorNamesTheParameter(t *testing.T) {
	harness := newEth(t, tests.NewFakeCore(t))
	err := harness.Failure("eth_getRules", "0x01")
	require.Equal(t, dispatch.CodeInvalidParameters, err.Code)
	require.Equal(t, "parameter 0 (block): hex number with leading zero digits", err.Message)
	err = harness.Failure("eth_getRules")
	require.Equal(t, "missing value for required argument 0", err.Message)
}

func TestConfig_DescribesTheCurrentAndTheLastUpgrade(t *testing.T) {
	fake, _, _ := chainWithTransfer(t)
	allegro := opera.GetAllegroUpgrades()
	brio := opera.GetBrioUpgrades()
	fake.SetChainInfo(core.ChainInfo{
		ChainID:   tests.FakeChainID,
		GenesisID: common.HexToHash("0x6e6573"),
		UpgradeHeights: []opera.UpgradeHeight{
			{Upgrades: allegro, Height: 0},
			{Upgrades: brio, Height: 1},
		},
	})
	harness := newEth(t, fake)
	answer := decodeMap(t, harness.Result("eth_config"))
	require.Nil(t, answer["next"])
	current := answer["current"].(map[string]any)
	last := answer["last"].(map[string]any)
	require.Equal(t, "0x1", current["blockHeight"])
	require.Equal(t, "0x0", last["blockHeight"])
	require.Equal(t, "0xfa3", current["chainId"])
	require.Contains(t, current["systemContracts"], "HISTORY_STORAGE_ADDRESS")
	require.Contains(t, current["precompiles"], "ECREC")
	require.NotEqual(t, current["forkId"], last["forkId"])
	require.Equal(t, map[string]any{"baseFeeUpdateFraction": float64(0), "max": float64(0), "target": float64(0)}, current["blobSchedule"])
}

func TestConfig_FailsWithoutUpgrades(t *testing.T) {
	fake := tests.NewFakeCore(t)
	fake.SetChainInfo(core.ChainInfo{ChainID: tests.FakeChainID})
	harness := newEth(t, fake)
	require.Equal(t, "no configs found", harness.Failure("eth_config").Message)
}

func TestConfig_FailsForAnUpgradeAboveTheHead(t *testing.T) {
	fake := tests.NewFakeCore(t)
	fake.SetChainInfo(core.ChainInfo{ChainID: tests.FakeChainID, UpgradeHeights: []opera.UpgradeHeight{{Height: 5}}})
	harness := newEth(t, fake)
	require.Equal(t, "failed to get current config, block 5 not found to determine activation time", harness.Failure("eth_config").Message)
}

func TestConstants_AnswerAsTheReference(t *testing.T) {
	harness := newEth(t, tests.NewFakeCore(t))
	require.JSONEq(t, `"0x0000000000000000000000000000000000000000"`, string(harness.Result("eth_etherbase")))
	require.JSONEq(t, `"0x0000000000000000000000000000000000000000"`, string(harness.Result("eth_coinbase")))
	require.JSONEq(t, `"0x0"`, string(harness.Result("eth_hashrate")))
}

func TestGetBlockByNumber_AnswersTheBlockWithItsTransactions(t *testing.T) {
	fake, block, transfer := chainWithTransfer(t)
	harness := newEth(t, fake)

	hashes := decodeMap(t, harness.Result("eth_getBlockByNumber", "0x1", false))
	require.Equal(t, block.Hash.Hex(), hashes["hash"])
	require.Equal(t, []any{transfer.Hash().Hex()}, hashes["transactions"])
	require.Equal(t, block.TxHash.Hex(), hashes["transactionsRoot"])
	require.NotEqual(t, types.EmptyRootHash.Hex(), hashes["receiptsRoot"])
	require.Equal(t, []any{}, hashes["uncles"])

	full := decodeMap(t, harness.Result("eth_getBlockByNumber", "latest", true))
	transaction := full["transactions"].([]any)[0].(map[string]any)
	require.Equal(t, transfer.Hash().Hex(), transaction["hash"])
	require.Equal(t, "0x1", transaction["blockNumber"])
	require.Equal(t, "0x3b9aca05", transaction["gasPrice"], "the base fee of 1 gwei plus the tip of 5")
}

func TestGetBlockByNumber_IsNullForABlockTheModuleDoesNotServe(t *testing.T) {
	fake, _, _ := chainWithTransfer(t)
	harness := newEth(t, fake)
	require.JSONEq(t, `null`, string(harness.Result("eth_getBlockByNumber", "0x2", false)), "committed but not archived")
	require.JSONEq(t, `null`, string(harness.Result("eth_getBlockByNumber", "0x99", false)))
	require.JSONEq(t, `null`, string(harness.Result("eth_getBlockByHash", common.Hash{1}, false)))
}

func TestGetBlockByNumber_HandsOnAReceiptFailure(t *testing.T) {
	fake, _, _ := chainWithTransfer(t)
	harness := newEth(t, fake)
	fake.SetMode("Receipts", tests.Fail)
	require.Equal(t, tests.ErrInjected.Error(), harness.Failure("eth_getBlockByNumber", "0x1", false).Message)
	fake.SetMode("Receipts", tests.NotFound)
	answer := decodeMap(t, harness.Result("eth_getBlockByNumber", "0x1", false))
	require.Equal(t, common.Hash{}.Hex(), answer["receiptsRoot"], "without receipts the root stays unset")
}

func TestGetBlockByHash_AnswersTheBlock(t *testing.T) {
	fake, block, _ := chainWithTransfer(t)
	harness := newEth(t, fake)
	answer := decodeMap(t, harness.Result("eth_getBlockByHash", block.Hash, false))
	require.Equal(t, "0x1", answer["number"])
}

func TestGetHeaderByNumber_CarriesTheTransactionsRoot(t *testing.T) {
	fake, block, _ := chainWithTransfer(t)
	harness := newEth(t, fake)
	answer := decodeMap(t, harness.Result("eth_getHeaderByNumber", "0x1"))
	require.Equal(t, block.TxHash.Hex(), answer["transactionsRoot"])
	require.NotContains(t, answer, "transactions")
	require.JSONEq(t, `null`, string(harness.Result("eth_getHeaderByNumber", "0x2")))
	fake.SetMode("Receipts", tests.NotFound)
	require.JSONEq(t, `null`, string(harness.Result("eth_getHeaderByNumber", "0x1")), "a header without receipts is null")
}

func TestGetHeaderByHash_AnUnknownHashIsAnError(t *testing.T) {
	fake, block, _ := chainWithTransfer(t)
	harness := newEth(t, fake)
	require.Equal(t, "0x1", decodeMap(t, harness.Result("eth_getHeaderByHash", block.Hash))["number"])
	require.Equal(t, "block with hash 0x0100000000000000000000000000000000000000000000000000000000000000 not found",
		harness.Failure("eth_getHeaderByHash", common.Hash{1}).Message)
}

func TestGetBlockReceipts_AnswersTheReceiptsOfABlock(t *testing.T) {
	fake, block, transfer := chainWithTransfer(t)
	harness := newEth(t, fake)
	for _, reference := range []any{"0x1", "latest", block.Hash, map[string]any{"blockHash": block.Hash}} {
		receipts := tests.Decode[[]map[string]any](t, harness.Result("eth_getBlockReceipts", reference))
		require.Len(t, receipts, 1)
		require.Equal(t, transfer.Hash().Hex(), receipts[0]["transactionHash"])
		require.Equal(t, "0x3b9aca05", receipts[0]["effectiveGasPrice"])
		logs := receipts[0]["logs"].([]any)
		require.Equal(t, "0x0", logs[0].(map[string]any)["transactionIndex"])
	}
	require.JSONEq(t, `[]`, string(harness.Result("eth_getBlockReceipts", map[string]any{})), "no selector names block 0")
	require.JSONEq(t, `null`, string(harness.Result("eth_getBlockReceipts", "0x2")))
	require.Equal(t, "block with hash 0x0100000000000000000000000000000000000000000000000000000000000000 not found",
		harness.Failure("eth_getBlockReceipts", common.Hash{1}).Message)
}

func TestTransactionCounts_AreNullForAnUnknownBlock(t *testing.T) {
	fake, block, _ := chainWithTransfer(t)
	harness := newEth(t, fake)
	require.JSONEq(t, `"0x1"`, string(harness.Result("eth_getBlockTransactionCountByNumber", "0x1")))
	require.JSONEq(t, `"0x1"`, string(harness.Result("eth_getBlockTransactionCountByHash", block.Hash)))
	require.JSONEq(t, `null`, string(harness.Result("eth_getBlockTransactionCountByNumber", "0x2")))
	fake.SetMode("Block", tests.Fail)
	require.JSONEq(t, `null`, string(harness.Result("eth_getBlockTransactionCountByNumber", "0x1")), "the reference answers null whatever the reason")
}

func TestUncles_AreNeverThere(t *testing.T) {
	fake, block, _ := chainWithTransfer(t)
	harness := newEth(t, fake)
	require.JSONEq(t, `"0x0"`, string(harness.Result("eth_getUncleCountByBlockNumber", "0x1")))
	require.JSONEq(t, `"0x0"`, string(harness.Result("eth_getUncleCountByBlockHash", block.Hash)))
	require.JSONEq(t, `null`, string(harness.Result("eth_getUncleCountByBlockNumber", "0x2")))
	require.JSONEq(t, `null`, string(harness.Result("eth_getUncleByBlockNumberAndIndex", "0x1", "0x0")))
	require.JSONEq(t, `null`, string(harness.Result("eth_getUncleByBlockHashAndIndex", block.Hash, "0x0")))
}

func TestTransactionsByBlockAndIndex(t *testing.T) {
	fake, block, transfer := chainWithTransfer(t)
	harness := newEth(t, fake)
	byNumber := decodeMap(t, harness.Result("eth_getTransactionByBlockNumberAndIndex", "0x1", "0x0"))
	require.Equal(t, transfer.Hash().Hex(), byNumber["hash"])
	byHash := decodeMap(t, harness.Result("eth_getTransactionByBlockHashAndIndex", block.Hash, "0x0"))
	require.Equal(t, byNumber, byHash)
	require.JSONEq(t, `null`, string(harness.Result("eth_getTransactionByBlockNumberAndIndex", "0x1", "0x1")))
	require.JSONEq(t, `null`, string(harness.Result("eth_getTransactionByBlockNumberAndIndex", "0x2", "0x0")))

	raw := tests.Decode[hexutil.Bytes](t, harness.Result("eth_getRawTransactionByBlockNumberAndIndex", "0x1", "0x0"))
	require.Equal(t, tests.Encode(t, transfer), raw)
	require.Equal(t, raw, tests.Decode[hexutil.Bytes](t, harness.Result("eth_getRawTransactionByBlockHashAndIndex", block.Hash, "0x0")))
	require.JSONEq(t, `"0x"`, string(harness.Result("eth_getRawTransactionByBlockNumberAndIndex", "0x1", "0x5")), "the reference answers empty bytes")
	require.JSONEq(t, `"0x"`, string(harness.Result("eth_getRawTransactionByBlockNumberAndIndex", "0x2", "0x0")))
}

func TestGetTransactionByHash_ReadsTheBlockOrThePool(t *testing.T) {
	fake, _, transfer := chainWithTransfer(t)
	harness := newEth(t, fake)
	located := decodeMap(t, harness.Result("eth_getTransactionByHash", transfer.Hash()))
	require.Equal(t, "0x1", located["blockNumber"])
	require.Equal(t, "0x0", located["transactionIndex"])

	signer := tests.NewSigner(t)
	pending := signer.Sign(t, &types.LegacyTx{Nonce: 3, Gas: 21000, GasPrice: big.NewInt(9)})
	fake.SetPool(map[common.Address]types.Transactions{signer.Address: {pending}}, nil)
	pooled := decodeMap(t, harness.Result("eth_getTransactionByHash", pending.Hash()))
	require.Nil(t, pooled["blockNumber"])
	require.Equal(t, "0x9", pooled["gasPrice"])
	require.JSONEq(t, `null`, string(harness.Result("eth_getTransactionByHash", common.Hash{1})))

	before := fake.Calls("Block")
	harness.Result("eth_getTransactionByHash", transfer.Hash())
	require.Equal(t, before, fake.Calls("Block"), "the header answers the location; the block is not read")
}

func TestGetTransactionByHash_HandsOnAFailureOfTheIndex(t *testing.T) {
	fake, _, transfer := chainWithTransfer(t)
	harness := newEth(t, fake)
	fake.SetMode("TransactionByHash", tests.Fail)
	require.Equal(t, tests.ErrInjected.Error(), harness.Failure("eth_getTransactionByHash", transfer.Hash()).Message)
	require.Equal(t, tests.ErrInjected.Error(), harness.Failure("eth_getRawTransactionByHash", transfer.Hash()).Message)
	require.Equal(t, tests.ErrInjected.Error(), harness.Failure("eth_getTransactionReceipt", transfer.Hash()).Message)
}

func TestGetRawTransactionByHash_ReadsTheBlockOrThePool(t *testing.T) {
	fake, _, transfer := chainWithTransfer(t)
	harness := newEth(t, fake)
	require.Equal(t, tests.Encode(t, transfer), tests.Decode[hexutil.Bytes](t, harness.Result("eth_getRawTransactionByHash", transfer.Hash())))
	signer := tests.NewSigner(t)
	pending := signer.Sign(t, &types.LegacyTx{Nonce: 3, Gas: 21000, GasPrice: big.NewInt(9)})
	fake.SetPool(nil, map[common.Address]types.Transactions{signer.Address: {pending}})
	require.Equal(t, tests.Encode(t, pending), tests.Decode[hexutil.Bytes](t, harness.Result("eth_getRawTransactionByHash", pending.Hash())))
	require.JSONEq(t, `"0x"`, string(harness.Result("eth_getRawTransactionByHash", common.Hash{1})))
}

func TestGetTransactionReceipt_FormatsTheReceipt(t *testing.T) {
	fake, block, transfer := chainWithTransfer(t)
	harness := newEth(t, fake)
	receipt := decodeMap(t, harness.Result("eth_getTransactionReceipt", transfer.Hash()))
	require.Equal(t, block.Hash.Hex(), receipt["blockHash"])
	require.Equal(t, "0x1", receipt["status"])
	require.Equal(t, "0x5208", receipt["gasUsed"])
	require.Equal(t, "0x2", receipt["type"])
	require.Nil(t, receipt["contractAddress"])
	require.JSONEq(t, `null`, string(harness.Result("eth_getTransactionReceipt", common.Hash{1})))
	fake.SetMode("Receipts", tests.NotFound)
	require.JSONEq(t, `null`, string(harness.Result("eth_getTransactionReceipt", transfer.Hash())))
}
