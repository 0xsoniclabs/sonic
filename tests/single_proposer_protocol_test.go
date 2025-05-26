package tests

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
	// To avoid
	accounts := make([]*Account, NumTxsPerRound)
	for i := range accounts {
		accounts[i] = NewAccount()
		_, err := net.EndowAccount(accounts[i].Address(), big.NewInt(1e18))
		require.NoError(err)
	}

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

	// Advance the epoch and make sure that the network is still able to process
	// transactions after the switch to the single-proposer protocol.
	require.NoError(net.AdvanceEpoch(1))
	for range 5 {
		_, err = net.EndowAccount(address, big.NewInt(50))
		require.NoError(err)
	}

	// TODO: check that the single-proposer protocol can also be disabled once
	// the feature is controlled by its own feature flag.
}
