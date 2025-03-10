package tests

import (
	"cmp"
	"context"
	"fmt"
	"math/big"
	"slices"
	"testing"
	"time"

	"github.com/0xsoniclabs/carmen/go/carmen"
	"github.com/0xsoniclabs/carmen/go/common/immutable"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/gasprice"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/0xsoniclabs/sonic/opera/contracts/evmwriter"
	"github.com/0xsoniclabs/sonic/opera/contracts/netinit"
	"github.com/0xsoniclabs/sonic/opera/contracts/sfc"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter_event_emitter"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"
)

func TestBlockHeader_FakeGenesis_SatisfiesInvariants(t *testing.T) {
	net := StartIntegrationTestNetWithFakeGenesis(t)
	testBlockHeadersOnNetwork(t, net)
}

func TestBlockHeader_JsonGenesis_SatisfiesInvariants(t *testing.T) {
	net := StartIntegrationTestNetWithJsonGenesis(t)
	testBlockHeadersOnNetwork(t, net)
}

func testBlockHeadersOnNetwork(t *testing.T, net *IntegrationTestNet) {
	const numBlocks = 10
	require := require.New(t)

	// Produce a few blocks on the network. We use the counter contract since
	// it is also producing events.
	counter, receipt, err := DeployContract(net, counter_event_emitter.DeployCounterEventEmitter)
	require.NoError(err)
	for range numBlocks {
		_, err := net.Apply(counter.Increment)
		require.NoError(err, "failed to increment counter")
	}
	counterAddress := receipt.ContractAddress

	originalHeaders, err := net.GetHeaders()
	require.NoError(err)
	originalHashes := []common.Hash{}
	for _, header := range originalHeaders {
		originalHashes = append(originalHashes, header.Hash())
	}

	runTests := func(t *testing.T) {
		headers, err := net.GetHeaders()
		require.NoError(err)

		client, err := net.GetClient()
		require.NoError(err)
		defer client.Close()

		t.Run("CompareHeaderHashes", func(t *testing.T) {
			testHeaders_CompareHeaderHashes(t, originalHashes, headers)
		})

		t.Run("BlockNumberEqualsPositionInChain", func(t *testing.T) {
			testHeaders_BlockNumberEqualsPositionInChain(t, headers)
		})

		t.Run("ParentHashCoversParentContent", func(t *testing.T) {
			testHeaders_ParentHashCoversParentContent(t, headers)
		})

		t.Run("GasUsedIsBelowGasLimit", func(t *testing.T) {
			testHeaders_GasUsedIsBelowGasLimit(t, headers)
		})

		t.Run("EncodesDurationAndNanoTimeInExtraData", func(t *testing.T) {
			testHeaders_EncodesDurationAndNanoTimeInExtraData(t, headers)
		})

		t.Run("BaseFeeEvolutionFollowsPricingRules", func(t *testing.T) {
			testHeaders_BaseFeeEvolutionFollowsPricingRules(t, headers)
		})

		t.Run("TransactionRootMatchesHashOfBlockTxs", func(t *testing.T) {
			testHeaders_TransactionRootMatchesBlockTxsHash(t, headers, client)
		})

		t.Run("TransactionReceiptReferencesCorrectContext", func(t *testing.T) {
			testHeaders_TransactionReceiptReferencesCorrectContext(t, headers, client)
		})

		t.Run("ReceiptBlockHashMatchesBlockHash", func(t *testing.T) {
			testHeaders_ReceiptBlockHashMatchesBlockHash(t, headers, client)
		})

		t.Run("ReceiptRootMatchesBlockReceipts", func(t *testing.T) {
			testHeaders_ReceiptRootMatchesBlockReceipts(t, headers, client)
		})

		t.Run("LogsBloomMatchesLogsInReceipts", func(t *testing.T) {
			testHeaders_LogsBloomMatchesLogsInReceipts(t, headers, client)
		})

		t.Run("CoinbaseIsZeroForAllBlocks", func(t *testing.T) {
			testHeaders_CoinbaseIsZeroForAllBlocks(t, headers)
		})

		t.Run("DifficultyIsZeroForAllBlocks", func(t *testing.T) {
			testHeaders_DifficultyIsZeroForAllBlocks(t, headers)
		})

		t.Run("NonceIsZeroForAllBlocks", func(t *testing.T) {
			testHeaders_NonceIsZeroForAllBlocks(t, headers)
		})

		t.Run("TimeProgressesMonotonically", func(t *testing.T) {
			testHeaders_TimeProgressesMonotonically(t, headers)
		})

		t.Run("MixDigestDiffersForAllBlocks", func(t *testing.T) {
			testHeaders_MixDigestDiffersForAllBlocks(t, headers)
		})

		t.Run("InitialBlocksHaveCorrectEpochNumbers", func(t *testing.T) {
			testHeaders_InitialBlocksHaveCorrectEpochNumbers(t, client)
		})

		t.Run("LastBlockOfEpochContainsSealingTransaction", func(t *testing.T) {
			testHeaders_LastBlockOfEpochContainsSealingTransaction(t, headers, client)
		})

		t.Run("StateRootsMatchActualStateRoots", func(t *testing.T) {
			testHeaders_StateRootsMatchActualStateRoots(t, headers, client)
		})

		t.Run("SystemContractsHaveNonZeroNonce", func(t *testing.T) {
			testHeaders_SystemContractsHaveNonZeroNonce(t, headers, client)
		})

		t.Run("LogsReferenceTheirContext", func(t *testing.T) {
			testHeaders_LogsReferenceTheirContext(t, headers, client)
		})

		t.Run("CanRetrieveLogEvents", func(t *testing.T) {
			testHeaders_CanRetrieveLogEvents(t, headers, client)
		})

		t.Run("CounterStateIsVerifiable", func(t *testing.T) {
			testHeaders_CounterStateIsVerifiable(t, headers, client, counterAddress)
		})

		t.Run("HasCommitteeCertificates", func(t *testing.T) {
			testScc_HasCommitteeCertificates(t, client)
		})

		t.Run("HasBlockCertificatesForBlocks", func(t *testing.T) {
			testScc_HasBlockCertificatesForBlocks(t, headers, client)
		})

		t.Run("HasSignedBlockCertificatesForBlocks", func(t *testing.T) {
			testScc_HasSignedBlockCertificatesForBlocks(t, headers, client)
		})
	}

	t.Run("BeforeRestart", runTests)

	require.NoError(net.Restart())
	t.Run("AfterRestart", runTests)

	require.NoError(net.RestartWithExportImport())
	t.Run("AfterImport", runTests)
}

func testHeaders_CompareHeaderHashes(t *testing.T, hashes []common.Hash, newHeaders []*types.Header) {
	require := require.New(t)

	require.GreaterOrEqual(len(newHeaders), len(hashes), "length mismatch")
	for i, hash := range hashes {
		require.Equal(hash, newHeaders[i].Hash(), "hash mismatch for block %d", i)
	}
}

func testHeaders_BlockNumberEqualsPositionInChain(t *testing.T, headers []*types.Header) {
	require := require.New(t)
	for i, header := range headers {
		require.Equal(header.Number.Uint64(), uint64(i))
	}
}

func testHeaders_ParentHashCoversParentContent(t *testing.T, headers []*types.Header) {
	require := require.New(t)

	// The parent hash of block 0 is expected to be zero.
	require.Equal(
		headers[0].ParentHash, common.Hash{},
		"invalid parent hash for block 0",
	)

	// All other blocks have a parent hash that matches the previous block's hash.
	for i := 1; i < len(headers); i++ {
		require.Equal(
			headers[i].ParentHash,
			headers[i-1].Hash(),
			"invalid hash stored in block %d for block %d", i, i-1,
		)
	}
}

func testHeaders_GasUsedIsBelowGasLimit(t *testing.T, headers []*types.Header) {
	require := require.New(t)
	for i, header := range headers {
		require.LessOrEqual(header.GasUsed, header.GasLimit, "block %d", i)
	}
}

func testHeaders_EncodesDurationAndNanoTimeInExtraData(t *testing.T, headers []*types.Header) {
	require := require.New(t)

	getUnixTime := func(header *types.Header) time.Time {
		t.Helper()
		nanos, _, err := inter.DecodeExtraData(header.Extra)
		require.NoError(err)
		return time.Unix(int64(header.Time), int64(nanos))
	}

	// Check the nano-time and duration encoded in the extra data field.
	for i := 1; i < len(headers); i++ {
		require.Equal(len(headers[i].Extra), 12, "extra data length of block %d", i)
		lastTime := getUnixTime(headers[i-1])
		currentTime := getUnixTime(headers[i])
		wantedDuration := currentTime.Sub(lastTime)
		_, gotDuration, err := inter.DecodeExtraData(headers[i].Extra)
		require.NoError(err, "decoding extra data of block %d", i)
		require.Equal(wantedDuration, gotDuration, "duration of block %d", i)
	}
}

func testHeaders_BaseFeeEvolutionFollowsPricingRules(t *testing.T, headers []*types.Header) {
	require := require.New(t)

	// The genesis block must use the initial base fee.
	rules := opera.FakeEconomyRules()
	require.Equal(
		gasprice.GetInitialBaseFee(rules),
		headers[0].BaseFee,
	)

	// All other blocks compute the base-fee based on the previous block.
	for i := 1; i < len(headers); i++ {
		_, duration, err := inter.DecodeExtraData(headers[i-1].Extra)
		require.NoError(err, "decoding extra data of block %d", i-1)
		last := &evmcore.EvmHeader{
			BaseFee:  headers[i-1].BaseFee,
			GasUsed:  headers[i-1].GasUsed,
			Duration: duration,
		}
		require.Equal(
			gasprice.GetBaseFeeForNextBlock(last, rules),
			headers[i].BaseFee,
			"base fee of block %d", i,
		)
	}
}

func testHeaders_TransactionRootMatchesBlockTxsHash(t *testing.T, headers []*types.Header, client *ethclient.Client) {
	require := require.New(t)

	for i, header := range headers {
		block, err := client.BlockByNumber(context.Background(), header.Number)
		require.NoError(err, "failed to get block %d", i)

		txsHash := types.DeriveSha(block.Transactions(), trie.NewStackTrie(nil))
		require.Equal(header.TxHash, txsHash, "transaction root hash mismatch")
	}
}

func testHeaders_TransactionReceiptReferencesCorrectContext(
	t *testing.T, headers []*types.Header, client *ethclient.Client,
) {
	require := require.New(t)

	for i, header := range headers {
		block, err := client.BlockByNumber(context.Background(), header.Number)
		require.NoError(err, "failed to get block %d", i)

		for j, tx := range block.Transactions() {
			receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
			require.NoError(err, "failed to get transaction receipt")

			require.Equal(tx.Hash(), receipt.TxHash, "transaction hash mismatch")
			require.Equal(i, int(receipt.BlockNumber.Uint64()), "block number mismatch")
			require.Equal(j, int(receipt.TransactionIndex), "transaction index mismatch")
			require.Equal(header.Hash(), receipt.BlockHash, "block hash mismatch")
		}
	}
}

func testHeaders_ReceiptBlockHashMatchesBlockHash(t *testing.T, headers []*types.Header, client *ethclient.Client) {
	require := require.New(t)

	for _, header := range headers {
		receipts, err := client.BlockReceipts(context.Background(),
			rpc.BlockNumberOrHashWithHash(header.Hash(), false))
		require.NoError(err, "failed to get block receipts")

		for _, receipt := range receipts {
			require.Equal(header.Hash(), receipt.BlockHash, "receipt block hash mismatch")
		}
	}
}

func testHeaders_ReceiptRootMatchesBlockReceipts(t *testing.T, headers []*types.Header, client *ethclient.Client) {
	require := require.New(t)

	for _, header := range headers {
		receipts, err := client.BlockReceipts(context.Background(),
			rpc.BlockNumberOrHashWithHash(header.Hash(), false))
		require.NoError(err, "failed to get block receipts")

		receiptsHash := types.DeriveSha(types.Receipts(receipts), trie.NewStackTrie(nil))
		require.Equal(header.ReceiptHash, receiptsHash, "receipt root hash mismatch")
	}
}

func testHeaders_LogsBloomMatchesLogsInReceipts(t *testing.T, headers []*types.Header, client *ethclient.Client) {
	require := require.New(t)

	for _, header := range headers {
		receipts, err := client.BlockReceipts(context.Background(),
			rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(header.Number.Uint64())))
		require.NoError(err, "failed to get block receipts")

		logsBloom := types.CreateBloom(receipts)
		require.Equal(header.Bloom, logsBloom, "logs bloom mismatch")
	}
}

func testHeaders_CoinbaseIsZeroForAllBlocks(t *testing.T, headers []*types.Header) {
	require := require.New(t)

	for _, header := range headers {
		require.Zero(header.Coinbase, "coinbase is not zero")
	}
}

func testHeaders_DifficultyIsZeroForAllBlocks(t *testing.T, headers []*types.Header) {
	require := require.New(t)

	for _, header := range headers {
		// Cmp returns 0 when the values are equal
		require.Zero(big.NewInt(0).Cmp(header.Difficulty), "difficulty is not zero")
	}
}

func testHeaders_NonceIsZeroForAllBlocks(t *testing.T, headers []*types.Header) {
	require := require.New(t)

	for _, header := range headers {
		require.Zero(header.Nonce.Uint64(), "nonce is not zero")
	}
}

func testHeaders_TimeProgressesMonotonically(t *testing.T, headers []*types.Header) {
	require := require.New(t)

	getTimeFrom := func(header *types.Header) time.Time {
		currentNano, _, err := inter.DecodeExtraData(header.Extra)
		require.NoError(err)
		return time.Unix(int64(header.Time), int64(currentNano))
	}

	for i := 1; i < len(headers); i++ {
		currentTime := getTimeFrom(headers[i])
		previousTime := getTimeFrom(headers[i-1])
		require.Greater(currentTime, previousTime, "time is not monotonically increasing. block %d", i)
	}
}

func testHeaders_MixDigestDiffersForAllBlocks(t *testing.T, headers []*types.Header) {
	require := require.New(t)

	seen := map[common.Hash]struct{}{}

	for i := 1; i < len(headers); i++ {
		// We skip empty blocks, since in those cases the MixDigest value is not
		// consumed by any transaction. For those cases, values may be reused.
		// Since the prev-randao value filling this field is computed based on
		// the hash of non-empty lachesis events, the value used for empty blocks
		// is always the same.
		header := headers[i]
		if header.GasUsed == 0 {
			continue
		}
		digest := header.MixDigest
		_, found := seen[digest]
		require.False(found, "mix digest is not unique, block %d, value %x", i, digest)
		seen[digest] = struct{}{}
	}
	require.NotZero(len(seen), "no non-empty blocks in the chain")
}

func testHeaders_InitialBlocksHaveCorrectEpochNumbers(t *testing.T, client *ethclient.Client) {
	require := require.New(t)
	for block, want := range []int{1, 1, 2} {
		got, err := getEpochOfBlock(client, block)
		require.NoError(err, "failed to get epoch of block %d", block)
		require.Equal(want, got, "block %d", block)
	}
}

func testHeaders_LastBlockOfEpochContainsSealingTransaction(t *testing.T, headers []*types.Header, client *ethclient.Client) {
	require := require.New(t)

	maxEpoch := 0
	for i := 0; i < len(headers)-1; i++ {

		block, err := client.BlockByNumber(context.Background(), big.NewInt(int64(i)))
		require.NoError(err, "failed to get block body")

		currentBlockEpoch, err := getEpochOfBlock(client, i)
		require.NoError(err, "failed to get epoch of block %d", i)

		nextBlockEpoch, err := getEpochOfBlock(client, i+1)
		require.NoError(err, "failed to get epoch of block %d", i+1)

		shouldContainSealingTx := currentBlockEpoch != nextBlockEpoch

		containsSealingTx := false
		for _, tx := range block.Transactions() {
			// Any call to the driver is considered a sealing transaction.
			if tx.To() != nil && *tx.To() == driver.ContractAddress {
				containsSealingTx = true
				break
			}
		}

		require.Equal(
			shouldContainSealingTx,
			containsSealingTx,
			"block %d, current epoch %d, next epoch %d",
			i, currentBlockEpoch, nextBlockEpoch,
		)

		if currentBlockEpoch > maxEpoch {
			maxEpoch = currentBlockEpoch
		}
	}
}

func getEpochOfBlock(client *ethclient.Client, blockNumber int) (int, error) {
	var result struct {
		Epoch hexutil.Uint64
	}
	err := client.Client().Call(
		&result,
		"eth_getBlockByNumber",
		fmt.Sprintf("0x%x", blockNumber),
		false,
	)
	if err != nil {
		return 0, err
	}
	return int(result.Epoch), nil
}

func testHeaders_StateRootsMatchActualStateRoots(t *testing.T, headers []*types.Header, client *ethclient.Client) {
	require := require.New(t)
	for i, header := range headers {
		// The direct way to get the state root of a block would be to request the
		// block header and extract the state root from it. However, we would like
		// to verify that the state root is correct by comparing it to the state
		// root we see in the database. To get access to the database, we request
		// a witness proof for an account at the given block. From this proof we
		// we have a list of state-root candidates which we can test for.
		want, err := getStateRoot(client, int(header.Number.Int64()))
		require.NoError(err, "failed to get witness proof for block %d", i)
		got := header.Root
		require.Equal(want, got, "state root mismatch for block %d", i)
	}
}

func getStateRoot(client *ethclient.Client, blockNumber int) (common.Hash, error) {
	var result struct {
		AccountProof []string
	}
	err := client.Client().Call(
		&result,
		"eth_getProof",
		fmt.Sprintf("%v", common.Address{}),
		[]string{},
		fmt.Sprintf("0x%x", blockNumber),
	)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get witness proof; %v", err)
	}

	// The hash of the first element of the account proof is the state root.
	if len(result.AccountProof) == 0 {
		return common.Hash{}, fmt.Errorf("no account proof found")
	}

	data, err := hexutil.Decode(result.AccountProof[0])
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(crypto.Keccak256(data)), nil
}

func testHeaders_SystemContractsHaveNonZeroNonce(t *testing.T, headers []*types.Header, client *ethclient.Client) {
	require := require.New(t)
	ctxt := context.Background()
	for i := range headers {
		block := big.NewInt(int64(i))
		for _, addr := range []common.Address{
			netinit.ContractAddress,
			driver.ContractAddress,
			driverauth.ContractAddress,
			sfc.ContractAddress,
			evmwriter.ContractAddress,
		} {
			nonce, err := client.NonceAt(ctxt, addr, block)
			require.NoError(err, "failed to get nonce for %s at block %d", addr, i)

			want := uint64(1)
			if addr == netinit.ContractAddress && i > 0 {
				want = 2
			}
			require.Equal(want, nonce, "nonce for %s at block %d is not one", addr, i)
		}
	}
}

func testHeaders_LogsReferenceTheirContext(t *testing.T, headers []*types.Header, client *ethclient.Client) {
	require := require.New(t)

	numLogs := 0
	for _, header := range headers {
		blockHash := header.Hash()
		blockNumber := header.Number.Uint64()

		receipts, err := client.BlockReceipts(context.Background(),
			rpc.BlockNumberOrHashWithHash(blockHash, false))
		require.NoError(err, "failed to get block receipts")

		index := 0
		for txIndex, receipt := range receipts {
			for _, log := range receipt.Logs {
				numLogs++
				require.Equal(blockHash, log.BlockHash, "block hash mismatch")
				require.Equal(blockNumber, log.BlockNumber, "block number mismatch")
				require.Equal(receipt.TxHash, log.TxHash, "transaction hash mismatch")
				require.Equal(uint(txIndex), log.TxIndex, "transaction index mismatch")
				require.Equal(uint(index), log.Index, "log index mismatch")
				require.False(log.Removed, "log was removed")
				index++
			}
		}
	}

	require.NotZero(numLogs, "no logs found in the chain")
}

func testHeaders_CanRetrieveLogEvents(t *testing.T, headers []*types.Header, client *ethclient.Client) {
	require := require.New(t)

	allLogs := []types.Log{}
	for _, header := range headers {
		blockHash := header.Hash()
		receipts, err := client.BlockReceipts(context.Background(),
			rpc.BlockNumberOrHashWithHash(blockHash, false))
		require.NoError(err, "failed to get block receipts")

		for _, receipt := range receipts {
			require.Equal(blockHash, receipt.BlockHash, "block hash mismatch")
			for _, log := range receipt.Logs {
				allLogs = append(allLogs, *log)

				// Check that logs can be retrieved by specifically filtering
				// for each individual log entry.
				topicFilter := [][]common.Hash{}
				for _, topic := range log.Topics {
					topicFilter = append(topicFilter, []common.Hash{topic})
				}
				logs, err := client.FilterLogs(
					context.Background(),
					ethereum.FilterQuery{
						BlockHash: &blockHash,
						Addresses: []common.Address{log.Address},
						Topics:    topicFilter,
					},
				)
				require.NoError(err, "failed to get logs")
				require.Equal([]types.Log{*log}, logs, "log mismatch")
			}
		}
	}

	require.NotZero(len(allLogs), "no logs found in the chain")

	// Fetch all logs from the chain in a single query and see that no extra
	// log entries are returned.
	logs, err := client.FilterLogs(
		context.Background(),
		ethereum.FilterQuery{
			FromBlock: big.NewInt(0),
			ToBlock:   big.NewInt(int64(len(headers) - 1)),
		},
	)
	require.NoError(err, "failed to get logs")

	// Sort logs in chronological order to have a deterministic comparison.
	logCompare := func(a, b types.Log) int {
		if res := cmp.Compare(a.BlockNumber, b.BlockNumber); res != 0 {
			return res
		}
		return cmp.Compare(a.Index, b.Index)
	}

	slices.SortFunc(logs, logCompare)
	slices.SortFunc(allLogs, logCompare)
	require.Equal(allLogs, logs, "log mismatch")

	// Fetch by topic index -- this uses a different table in the database.
	topicZeroOptions := []common.Hash{}
	for _, log := range allLogs {
		topicZeroOptions = append(topicZeroOptions, log.Topics[0])
	}
	logs, err = client.FilterLogs(
		context.Background(),
		ethereum.FilterQuery{
			ToBlock: big.NewInt(int64(len(headers) - 1)),
			Topics:  [][]common.Hash{topicZeroOptions},
		},
	)
	require.NoError(err, "failed to get logs")

	slices.SortFunc(logs, logCompare)
	slices.SortFunc(allLogs, logCompare)
	require.Equal(allLogs, logs, "log mismatch")
}

func testHeaders_CounterStateIsVerifiable(
	t *testing.T,
	headers []*types.Header,
	client *ethclient.Client,
	counterAddress common.Address,
) {
	require := require.New(t)

	counter, err := counter_event_emitter.NewCounterEventEmitter(counterAddress, client)
	require.NoError(err, "failed to instantiate contract")

	fromLogs := 0
	for i, header := range headers {

		// Get counter value according to the reported logs.
		receipts, err := client.BlockReceipts(context.Background(), rpc.BlockNumberOrHashWithHash(header.Hash(), false))
		require.NoError(err, "failed to get block receipts")
		for _, receipt := range receipts {
			require.Equal(header.Hash(), receipt.BlockHash, "block hash mismatch")
			for _, log := range receipt.Logs {
				event, err := counter.ParseCount(*log)
				if err != nil {
					continue
				}
				fromLogs = int(event.TotalCount.Int64())
			}
		}

		// Get the counter value from the archive.
		fromArchive := 0
		fromArchiveAsBig, err := counter.GetTotalCount(&bind.CallOpts{
			BlockNumber: new(big.Int).SetUint64(header.Number.Uint64()),
		})
		if err == nil {
			fromArchive = int(fromArchiveAsBig.Int64())
		}

		// Get the counter value from the state directly.
		fromStateAsHash, err := client.StorageAt(context.Background(), counterAddress, common.Hash{}, big.NewInt(int64(i)))
		require.NoError(err)
		fromState := int(new(big.Int).SetBytes(fromStateAsHash).Uint64())

		// Get the counter value from a witness proof.
		fromProof := getVerifiedCounterState(t, client, header.Root, counterAddress, i)

		require.Equal(fromLogs, fromArchive, "block %d", i)
		require.Equal(fromLogs, fromState, "block %d", i)
		require.Equal(fromLogs, fromProof, "block %d", i)
	}
}

func getVerifiedCounterState(
	t *testing.T,
	client *ethclient.Client,
	stateRoot common.Hash,
	counterAddress common.Address,
	blockNumber int,
) int {
	require := require.New(t)
	var result struct {
		AccountProof []string
		StorageHash  common.Hash
		StorageProof []struct {
			Value string
			Proof []string
		}
	}
	err := client.Client().Call(
		&result,
		"eth_getProof",
		fmt.Sprintf("%v", counterAddress),
		[]string{fmt.Sprintf("%v", common.Hash{})},
		fmt.Sprintf("0x%x", blockNumber),
	)
	require.NoError(err, "failed to get witness proof")
	require.Equal(1, len(result.StorageProof), "expected exactly one storage proof")
	require.GreaterOrEqual(len(result.StorageProof[0].Proof), 1, "expected at least one proof element")

	// Verify the proof.
	elements := []carmen.Bytes{}
	for _, proof := range [][]string{result.AccountProof, result.StorageProof[0].Proof} {
		for _, element := range proof {
			data, err := hexutil.Decode(element)
			require.NoError(err)
			elements = append(elements, immutable.NewBytes(data))
		}
	}
	proof := carmen.CreateWitnessProofFromNodes(elements...)
	require.True(proof.IsValid())

	// Extract the storage value from the proof.
	value, present, err := proof.GetState(carmen.Hash(stateRoot), carmen.Address(counterAddress), carmen.Key{})
	require.NoError(err, "failed to get state from proof")
	require.True(present, "slot not found in proof")

	// Check that the storage root hash is consistent.
	_, storageRoot, complete := proof.GetAccountElements(carmen.Hash(stateRoot), carmen.Address(counterAddress))
	require.True(complete, "proof is not complete")
	require.Equal(common.Hash(storageRoot), result.StorageHash, "storage root mismatch")

	// Check that the storage proof starts with an element that corresponds to
	// the storage root.
	hash := crypto.Keccak256(hexutil.MustDecode(result.StorageProof[0].Proof[0]))
	firstElementHash := carmen.Hash{}
	copy(firstElementHash[:], hash)
	require.Equal(storageRoot, firstElementHash, "storage proof does not start with the storage root")

	// Compare the proof value with the value in the RPC result.
	fromProof := int(new(big.Int).SetBytes(value[:]).Uint64())
	fromResult, err := hexutil.DecodeUint64(result.StorageProof[0].Value)
	require.NoError(err, "failed to decode counter value")
	require.Equal(int(fromResult), fromProof, "proof value mismatch")

	return fromProof
}

func testScc_HasCommitteeCertificates(
	t *testing.T,
	client *ethclient.Client,
) {
	require := require.New(t)
	results := []struct {
		ChainId uint64
		Period  scc.Period
		Members []scc.Member
	}{}
	err := client.Client().Call(&results, "sonic_getCommitteeCertificates", "0x0", "max")
	require.NoError(err)
	require.NotEmpty(results, "no committee certificates found")

	chainId, err := client.ChainID(context.Background())
	require.NoError(err)
	for i, result := range results {
		require.Equal(chainId.Uint64(), result.ChainId)
		require.Equal(scc.Period(i), result.Period)
		require.NotEmpty(result.Members, "period %d", i)
	}
}

func testScc_HasBlockCertificatesForBlocks(
	t *testing.T,
	headers []*types.Header,
	client *ethclient.Client,
) {
	require := require.New(t)

	results := []struct {
		ChainId   uint64
		Number    uint64
		Hash      common.Hash
		StateRoot common.Hash
	}{}
	err := client.Client().Call(&results, "sonic_getBlockCertificates", "0x0", "max")
	require.NoError(err)

	// Check that all certificates starting with block 0 are present.
	require.Equal(uint64(0), results[0].Number)
	require.LessOrEqual(len(headers), len(results), "too few block certificates")

	// Check that certificates are listed in order.
	for i := range len(results) - 1 {
		require.Equal(results[i].Number+1, results[i+1].Number, "block certificate order mismatch")
	}

	// Check the certificate content.
	chainId, err := client.ChainID(context.Background())
	require.NoError(err)
	for _, cert := range results {
		if cert.Number >= uint64(len(headers)) {
			continue
		}
		header := headers[cert.Number]
		require.Equal(chainId.Uint64(), cert.ChainId, "chain ID mismatch")
		require.Equal(header.Hash(), cert.Hash, "block hash mismatch")
		require.Equal(header.Root, cert.StateRoot, "state root mismatch")
	}
}

// testScc_HasSignedBlockCertificatesForBlocks checks that the block certificates
// produced on by the chain are valid. To be valid, sufficient signatures must
// be present. Since in this context there is only ever a single validator,
// this test only provides a sanity check. It can not demonstrate that signatures
// are successfully distributed among network nodes.
func testScc_HasSignedBlockCertificatesForBlocks(
	t *testing.T,
	headers []*types.Header,
	client *ethclient.Client,
) {
	require := require.New(t)

	// fetch all block certificates available on the server
	results := []struct {
		ChainId   uint64
		Number    idx.Block
		Hash      common.Hash
		StateRoot common.Hash
		Signers   cert.BitSet[scc.MemberId]
		Signature bls.Signature
	}{}
	err := client.Client().Call(&results, "sonic_getBlockCertificates", "0x0", "max")
	require.NoError(err)
	require.NotEmpty(results, "no block certificates found")

	// The test harness only gives enough time to settle the first few blocks
	// so we only check the first 10 blocks, which is enough to verify that
	// valid certificates are generated.
	if len(results) > 10 {
		results = results[:10]
	}

	for _, result := range results {
		// Convert the raw data into a certificate.
		certificate := cert.NewCertificateWithSignature(
			cert.NewBlockStatement(
				result.ChainId,
				result.Number,
				result.Hash,
				result.StateRoot,
			),
			cert.NewAggregatedSignature[cert.BlockStatement](
				result.Signers,
				result.Signature,
			),
		)

		// Fetch the committee for the block.
		committee, err := getCommitteeForBlock(result.Number, client)
		require.NoError(err, "failed to get committee for block", "block", result.Number)

		// Check that the signature on the certificate is valid.
		require.NoError(
			certificate.Verify(committee),
			"failed to verify certificate of block %d",
			result.Number,
		)
	}
}

// getCommitteeForBlock fetches the committee for the given block without
// checking its validity. The service connected through the client is implicitly
// trusted.
func getCommitteeForBlock(
	block idx.Block,
	client *ethclient.Client,
) (scc.Committee, error) {
	period := scc.GetPeriod(block)
	var result []struct {
		Members []scc.Member
	}
	err := client.Client().Call(
		&result,
		"sonic_getCommitteeCertificates",
		period,
		1,
	)
	if err != nil {
		return scc.Committee{}, err
	}
	if len(result) != 1 {
		return scc.Committee{}, fmt.Errorf("unexpected number of results: %d", len(result))
	}
	members := result[0].Members
	if len(members) == 0 {
		return scc.Committee{}, fmt.Errorf("no members in the committee")
	}
	return scc.NewCommittee(members...), nil
}
