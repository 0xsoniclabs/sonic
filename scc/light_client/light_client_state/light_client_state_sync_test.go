package lc_state

import (
	"crypto/sha256"
	"slices"
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

	// generate history of blocks and committees certificates
	blockHeight := scc.BLOCKS_PER_PERIOD * 50 / 3
	firstCommittee, provider, err := generateCertificatesAndProvider(
		ctrl, idx.Block(blockHeight))
	require.NoError(err)

	// create a new state with the first committee
	state := NewState(firstCommittee)
	headNumber, err := state.Sync(provider)
	require.NoError(err)
	require.Equal(idx.Block(blockHeight), headNumber)
}

// /////////////////////////
// Helper functions
// /////////////////////////

// generateCertificatesAndProvider generates a committee and a provider for testing.
// The amount of committee certificates is generated based on the given
// block height.
// The provider is a mock provider that returns the generated committee and
// blocks certificates.
func generateCertificatesAndProvider(
	ctrl *gomock.Controller,
	blockHeight idx.Block,
) (scc.Committee, provider.Provider, error) {

	// generate first committee with committees and blocks certificates
	firstCommittee, blocks, committees, err := generateHistory(blockHeight)
	if err != nil {
		return scc.Committee{}, nil, err
	}

	// prepare mock provider
	prov := prepareProvider(ctrl, blockHeight, blocks, committees)

	return firstCommittee, prov, nil
}

// generateHistory generates a history of blocks and committees certificates.
// Certificates are signed by 3 committee members and the committee rotates
// every period.
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
	}

	genesis = scc.NewCommittee(
		makeMember(keys[0]),
		makeMember(keys[1]),
		makeMember(keys[2]))

	// generate first block and committee certificates.
	blocks = append(blocks, cert.NewCertificate(cert.BlockStatement{}))
	committees = append(committees, cert.NewCertificate(cert.CommitteeStatement{
		Committee: genesis,
	}))

	// generate certificates up to blockHeight.
	committee := genesis
	head := idx.Block(0)
	headHash := common.Hash{}
	for i := head; i < blockHeight; i++ {

		// Compute next block.
		head += 1
		// the next line is a dummy hash, only for testing purposes.
		headHash = common.Hash(sha256.Sum256(headHash[:]))

		// Add period boundaries, update the committee.
		if scc.IsFirstBlockOfPeriod(head) {
			committee := scc.NewCommittee(rotate(committee.Members())...)

			certificate := cert.NewCertificate(
				cert.NewCommitteeStatement(
					1234,
					scc.GetPeriod(head),
					committee))

			for i, key := range keys {
				err := certificate.Add(
					scc.MemberId(i),
					cert.Sign(certificate.Subject(), key))
				if err != nil {
					return scc.Committee{}, nil, nil, err
				}
			}
			committees = append(committees, certificate)
			keys = rotate(keys)
		}

		// Sign the new block using the current committee.
		block := cert.NewCertificate(
			cert.NewBlockStatement(
				1234,
				head,
				headHash,
				headHash,
			))

		for i, key := range keys {
			err := block.Add(scc.MemberId(i), cert.Sign(block.Subject(), key))
			if err != nil {
				return scc.Committee{}, nil, nil, err
			}
		}
		blocks = append(blocks, block)

	}

	return genesis, blocks, committees, nil
}

// prepareProvider prepares a mock provider that returns the given blocks and
// committees certificates.
// if the block number is LatestBlock, it returns the latest block.
func prepareProvider(
	ctrl *gomock.Controller,
	blockHeight idx.Block,
	blocks []cert.BlockCertificate,
	committees []cert.CommitteeCertificate,
) provider.Provider {

	prov := provider.NewMockProvider(ctrl)
	prov.
		EXPECT().
		GetBlockCertificates(gomock.Any(), gomock.Any()).
		DoAndReturn(func(number idx.Block, max uint64) ([]cert.BlockCertificate, error) {
			if number == provider.LatestBlock {
				return blocks[len(blocks)-1:], nil
			}
			start := uint64(number)
			end := start + max
			end = min(end, uint64(len(committees)))
			start = min(start, end)
			return blocks[start:end], nil
		}).
		AnyTimes()

	prov.EXPECT().
		GetCommitteeCertificates(gomock.Any(), gomock.Any()).
		DoAndReturn(func(from scc.Period, max uint64) ([]cert.CommitteeCertificate, error) {
			start := uint64(from)
			end := start + max
			end = min(end, uint64(len(committees)))
			start = min(start, end)
			return committees[start:end], nil
		}).
		AnyTimes()

	return prov
}

func makeMember(key bls.PrivateKey) scc.Member {
	return scc.Member{
		PublicKey:         key.PublicKey(),
		ProofOfPossession: key.GetProofOfPossession(),
		VotingPower:       1,
	}
}

func rotate[T any](list []T) []T {
	res := slices.Clone(list)
	res = append(res[1:], res[0])
	return res
}
