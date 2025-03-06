package lc_state

import (
	"math"
	"testing"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/0xsoniclabs/sonic/scc/light_client/provider"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestLightClientState_CanSyncWithProvider(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	blockHeight := scc.BLOCKS_PER_PERIOD * 50 / 3
	firstCommittee, provider, err := generateCommitteeAndProvider(ctrl, idx.Block(blockHeight))
	require.NoError(err)

	// In this test case the light client sync to the network period-by-period.
	state := NewState(firstCommittee)
	headNumber, err := state.Sync(provider)
	require.NoError(err)
	require.Equal(idx.Block(blockHeight), headNumber)
}

///////////////////////////
// Helper functions
///////////////////////////

// generateCommitteeAndProvider generates a committee and a provider for testing.
// The committee is generated based on the given block height.
// The provider is a mock provider that returns the generated committee and blocks.
func generateCommitteeAndProvider(ctrl *gomock.Controller, blockHeight idx.Block) (scc.Committee, provider.Provider, error) {

	// generate committee, blocks and certificates
	firstCommittee, blocks, committees, err := generateHistory(blockHeight)
	if err != nil {
		return scc.Committee{}, nil, err
	}

	provider := provider.NewMockProvider(ctrl)
	provider.
		EXPECT().
		GetBlockCertificates(gomock.Any(), gomock.Any()).
		DoAndReturn(func(number idx.Block, max int) (cert.BlockCertificate, error) {
			if number == idx.Block(math.MaxUint64) {
				number = idx.Block(len(blocks) - 1)
			}
			return blocks[number], nil
		}).
		AnyTimes()

	provider.
		EXPECT().
		GetCommitteeCertificates(gomock.Any(), gomock.Any()).
		DoAndReturn(func(from scc.Period, max int) []cert.CommitteeCertificate {
			start := int(from)
			end := start + max
			end = min(end, len(committees))
			start = min(start, end)
			return committees[start:end]
		}).
		AnyTimes()

	return firstCommittee, provider, nil
}

// generateHistory generates a history of blocks and committees.
// certificate is signed by 3 committee members out of 4, and the committee rotates every period.
func generateHistory(blockHeight idx.Block) (
	genesis scc.Committee,
	blocks []cert.BlockCertificate,
	committees []cert.CommitteeCertificate,
	err error,
) {

	keys := []bls.PrivateKey{
		bls.NewPrivateKey(),
		bls.NewPrivateKey(),
		bls.NewPrivateKey(),
		bls.NewPrivateKey(),
	}

	genesis = scc.NewCommittee(
		makeMember(keys[0]),
		makeMember(keys[1]),
		makeMember(keys[2]),
		makeMember(keys[3]))

	// generate blocks and certificates for block and period 0.
	blocks = append(blocks, cert.NewCertificate(cert.BlockStatement{}))
	committees = append(committees, cert.NewCertificate(cert.CommitteeStatement{
		Committee: genesis,
	}))

	// generate blocks and certificates

	committee := genesis
	head := idx.Block(0)
	for i := head; i < blockHeight; i++ {

		// Compute next block.
		head += 1
		// Hash:   util.Sha256(head.Hash[:]), // < dummy step

		// Add period boundaries, update the committee.
		if scc.IsFirstBlockOfPeriod(head) {
			committee := scc.NewCommittee(rotate(committee.Members())...)

			certificate := cert.NewCertificate(cert.NewCommitteeStatement(1234,
				scc.GetPeriod(head),
				committee))
			for i, key := range keys {
				certificate.Add(scc.MemberId(i), cert.Sign(certificate.Subject(), key))
			}
			committees = append(committees, certificate)
			committee = scc.NewCommittee(rotate(committee.Members())...)
		}

		// Sign the new block using the current committee.
		block := cert.NewCertificate(
			cert.NewBlockStatement(
				1234,
				head,
				common.Hash([]byte{byte(head)}),
				common.Hash([]byte{byte(head)}),
			))

		for i, key := range keys {
			block.Add(scc.MemberId(i), cert.Sign(block.Subject(), key))
		}
		blocks = append(blocks, block)

	}

	return genesis, blocks, committees, nil
}

func makeMember(key bls.PrivateKey) scc.Member {
	return scc.Member{
		PublicKey:         key.PublicKey(),
		ProofOfPossession: key.GetProofOfPossession(),
		VotingPower:       1,
	}
}

func rotate(arr []scc.Member) []scc.Member {
	return append(arr[1:], arr[0])
}
