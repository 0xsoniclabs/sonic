package tests

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestSingleProposerProtocol_CanProcessTransactions(t *testing.T) {
	for _, numNodes := range []int{1, 3} {
		t.Run(fmt.Sprintf("numNodes=%d", numNodes), func(t *testing.T) {
			testSingleProposerProtocol_CanProcessTransactions(t, numNodes)
		})
	}
}

func testSingleProposerProtocol_CanProcessTransactions(t *testing.T, numNodes int) {
	// This test is a general smoke test for the single-proposer protocol. It
	// checks that transactions can be processed and that the network is not
	// producing (excessive) empty blocks.
	const NumRounds = 30
	const EpochLength = 7
	const NumTxsPerRound = 5

	require := require.New(t)
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		Upgrades: AsPointer(opera.GetAllegroUpgrades()),
		NumNodes: numNodes,
	})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	// --- setup network accounts ---

	// Create NumTxsPerRound accounts and send them each 1e18 wei to allow each
	// of them to send independent transactions in each round.
	accounts := make([]*Account, NumTxsPerRound)
	for i := range accounts {
		accounts[i] = NewAccount()
		_, err := net.EndowAccount(accounts[i].Address(), big.NewInt(1e18))
		require.NoError(err)
	}

	// Check that the network is using the single-proposer protocol.
	require.Equal(3, getUsedEventVersion(t, client))

	// --- check processing of transactions ---

	chainId, err := client.ChainID(t.Context())
	require.NoError(err)
	signer := types.NewPragueSigner(chainId)
	target := common.Address{0x42}

	startBlock, err := client.BlockNumber(t.Context())
	require.NoError(err)

	// Send a sequence of transactions to the network, in several rounds,
	// across multiple epochs, and check that all get processed.
	for round := range uint64(NumRounds) {
		transactionHashes := []common.Hash{}
		for sender := range NumTxsPerRound {
			transaction := types.MustSignNewTx(
				accounts[sender].PrivateKey,
				signer,
				&types.DynamicFeeTx{
					ChainID:   chainId,
					Nonce:     round,
					To:        &target,
					Value:     big.NewInt(1),
					Gas:       21000,
					GasFeeCap: big.NewInt(1e11),
					GasTipCap: big.NewInt(int64(sender) + 1),
				},
			)
			transactionHashes = append(transactionHashes, transaction.Hash())
			require.NoError(client.SendTransaction(t.Context(), transaction))
		}

		for _, hash := range transactionHashes {
			receipt, err := net.GetReceipt(hash)
			require.NoError(err)
			require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
		}

		// Start a new epoch every EpochLength rounds, but not as part of the
		// first round to avoid mixing up issued introduced by the transaction
		// processing and the epoch change. Thus, the first epoch will run for
		// EpochLength/2 rounds, and the rest for EpochLength rounds.
		if round%EpochLength == EpochLength/2 {
			require.NoError(net.AdvanceEpoch(1))
		}
	}

	// Check that rounds have been processed fairly efficient, without the use
	// of a large number of blocks. This is a mere smoke test to check that the
	// validators are not spamming unnecessary empty proposals.
	endBlock, err := client.BlockNumber(t.Context())
	require.NoError(err)

	duration := endBlock - startBlock
	require.Less(duration, uint64(2*NumRounds))
}

func TestSingleProposerProtocol_CanBeEnabled(t *testing.T) {
	// Test with different numbers of nodes
	for _, numNodes := range []int{1, 3} {
		t.Run(fmt.Sprintf("numNodes=%d", numNodes), func(t *testing.T) {
			testSingleProposerProtocol_CanBeEnabled(t, numNodes)
		})
	}
}

func testSingleProposerProtocol_CanBeEnabled(t *testing.T, numNodes int) {
	require := require.New(t)

	// The network is initially started using the distributed protocol.
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		NumNodes: numNodes,
	})

	// Test that before the switch transactions can be processed.
	address := common.Address{0x42}
	_, err := net.EndowAccount(address, big.NewInt(50))
	require.NoError(err)

	// Initially, Version 2 of the event protocol should be used.
	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()
	require.Equal(2, getUsedEventVersion(t, client))

	// Send the network rule update.
	type rulesType struct {
		Upgrades struct{ Allegro bool }
	}
	rulesDiff := rulesType{
		Upgrades: struct{ Allegro bool }{Allegro: true},
	}
	updateNetworkRules(t, net, rulesDiff)

	// The rules only take effect after the epoch change. Make sure that until
	// then, transactions can be processed.
	_, err = net.EndowAccount(address, big.NewInt(50))
	require.NoError(err)

	// At this point, still version 2 should be used.
	require.Equal(2, getUsedEventVersion(t, client))

	// Advance the epoch by one, enabling the single-proposer protocol.
	require.NoError(net.AdvanceEpoch(1))

	// Check that transactions can still be processed after the epoch change.
	for range 5 {
		_, err = net.EndowAccount(address, big.NewInt(50))
		require.NoError(err)
	}

	// Check that in this epoch the single-proposer protocol is used.
	require.Equal(3, getUsedEventVersion(t, client))

	// TODO(#193): check that the single-proposer protocol can also be disabled
	// once the feature is controlled by its own feature flag.
}

// getUsedEventVersion retrieves the current event version used by the network.
func getUsedEventVersion(
	t *testing.T,
	client *ethclient.Client,
) int {
	t.Helper()
	require := require.New(t)

	// Get the current epoch.
	block := struct {
		Epoch hexutil.Uint64
	}{}
	err := client.Client().Call(&block, "eth_getBlockByNumber", rpc.BlockNumber(-1), false)
	require.NoError(err)

	// Get the head events of the current epoch.
	heads := []hexutil.Bytes{}
	err = client.Client().Call(&heads, "dag_getHeads", rpc.BlockNumber(block.Epoch))
	require.NoError(err)

	// Download one of the head events and fetch the version.
	event := struct {
		Version hexutil.Uint64
	}{}
	err = client.Client().Call(&event, "dag_getEvent", heads[0].String())
	require.NoError(err)

	return int(event.Version)
}
