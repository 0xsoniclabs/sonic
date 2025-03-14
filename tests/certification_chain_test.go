package tests

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter_event_emitter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func TestCertificationChain_FakeGenesis_SatisfiesInvariants(t *testing.T) {
	net := StartIntegrationTestNetWithFakeGenesis(t, IntegrationTestNetOptions{
		FeatureSet: opera.AllegroFeatures,
		NumNodes:   2,
	})
	testCertificationChainInvariants(t, net)
}

func TestCertificationChain_JsonGenesis_SatisfiesInvariants(t *testing.T) {
	net := StartIntegrationTestNetWithJsonGenesis(t, IntegrationTestNetOptions{
		FeatureSet: opera.AllegroFeatures,
		NumNodes:   3,
	})
	testCertificationChainInvariants(t, net)
}

func testCertificationChainInvariants(t *testing.T, net *IntegrationTestNet) {
	const numBlocks = 10
	require := require.New(t)

	// Produce a few blocks on the network. We use the counter contract since
	// it is also producing events.
	counter, _, err := DeployContract(net, counter_event_emitter.DeployCounterEventEmitter)
	require.NoError(err)
	for range numBlocks {
		_, err := net.Apply(counter.Increment)
		require.NoError(err, "failed to increment counter")
	}

	runTests := func(t *testing.T) {
		t.Run("AllBlockHaveCertificates", func(t *testing.T) {
			testScc_AllBlocksHaveCertificates(t, net)
		})
	}

	t.Run("BeforeRestart", runTests)

	require.NoError(net.Restart())
	t.Run("AfterRestart", runTests)

	require.NoError(net.RestartWithExportImport())
	t.Run("AfterImport", runTests)
}

func testForEachNode(
	t *testing.T,
	net *IntegrationTestNet,
	test func(t *testing.T, client *ethclient.Client),
) {
	for id := range net.NumNodes() {
		t.Run(fmt.Sprintf("Node%d", id), func(t *testing.T) {
			client, err := net.GetClientConnectedToNode(id)
			require.NoError(t, err)
			defer client.Close()
			test(t, client)
		})
	}
}

func testScc_AllBlocksHaveCertificates(t *testing.T, net *IntegrationTestNet) {
	testForEachNode(t, net, testScc_OnNode_AllBlocksHaveCertificates)
}

func testScc_OnNode_AllBlocksHaveCertificates(t *testing.T, client *ethclient.Client) {
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
