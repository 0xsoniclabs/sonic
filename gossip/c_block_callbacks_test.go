package gossip

import (
	"bytes"
	"cmp"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
	"slices"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/randao"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/0xsoniclabs/sonic/valkeystore"
	"github.com/0xsoniclabs/sonic/valkeystore/encryption"
)

func TestConsensusCallback(t *testing.T) {

	withSingleProposer := opera.GetAllegroUpgrades()
	withSingleProposer.SingleProposerBlockFormation = true

	features := map[string]opera.Upgrades{
		"sonic":           opera.GetSonicUpgrades(),
		"allegro":         opera.GetAllegroUpgrades(),
		"single proposer": withSingleProposer,
	}

	for name, feature := range features {
		t.Run(name, func(t *testing.T) {
			testConsensusCallback(t, feature)
		})
	}
}

func testConsensusCallback(t *testing.T, upgrades opera.Upgrades) {
	logger.SetTestMode(t)
	require := require.New(t)

	const rounds = 30

	const validatorsNum = 3

	env := newTestEnvWithUpgrades(2, validatorsNum, upgrades, t)
	t.Cleanup(func() {
		err := env.Close()
		require.NoError(err)
	})

	// save start balances
	balances := make([]*uint256.Int, validatorsNum)
	for i := range balances {
		balances[i] = env.State().GetBalance(env.Address(idx.ValidatorID(i + 1)))
	}

	for n := uint64(0); n < rounds; n++ {
		// transfers
		txs := make([]*types.Transaction, validatorsNum)
		for i := idx.Validator(0); i < validatorsNum; i++ {
			from := i % validatorsNum
			to := 0
			txs[i] = env.Transfer(idx.ValidatorID(from+1), idx.ValidatorID(to+1), utils.ToFtm(100))
		}
		tm := sameEpoch
		if n%10 == 0 {
			tm = nextEpoch
		}
		rr, err := env.ApplyTxs(tm, txs...)
		require.NoError(err)
		// subtract fees
		for i, r := range rr {
			fee := uint256.NewInt(0).Mul(new(uint256.Int).SetUint64(r.GasUsed), utils.BigIntToUint256(txs[i].GasPrice()))
			balances[i] = uint256.NewInt(0).Sub(balances[i], fee)
		}
		// balance movements
		balances[0].Add(balances[0], utils.ToFtmU256(200))
		balances[1].Sub(balances[1], utils.ToFtmU256(100))
		balances[2].Sub(balances[2], utils.ToFtmU256(100))
	}

	// check balances
	for i := range balances {
		require.Equal(
			balances[i],
			env.State().GetBalance(env.Address(idx.ValidatorID(i+1))),
			fmt.Sprintf("account%d", i),
		)
	}

}

func TestExtractProposalForNextBlock_NoEvents_ReturnsNoProposal(t *testing.T) {
	last := &evmcore.EvmHeader{
		Number: big.NewInt(100),
	}
	result, proposer := extractProposalForNextBlock(last, nil, nil)
	require.Nil(t, result)
	require.Equal(t, idx.ValidatorID(0), proposer)
}

func TestExtractProposalForNextBlock_OneMatchingProposal_ReturnsTheGivenProposal(t *testing.T) {
	ctrl := gomock.NewController(t)
	event := inter.NewMockEventPayloadI(ctrl)

	lastHash := common.Hash{1, 2, 3}
	last := &evmcore.EvmHeader{
		Number: big.NewInt(100),
		Hash:   lastHash,
	}

	proposal := inter.Proposal{
		Number:     101,
		ParentHash: lastHash,
	}

	event.EXPECT().Payload().Return(&inter.Payload{Proposal: &proposal})
	event.EXPECT().Creator().Return(idx.ValidatorID(33)).AnyTimes()
	events := []inter.EventPayloadI{event}

	result, proposer := extractProposalForNextBlock(last, events, nil)
	require.NotNil(t, result)
	require.Equal(t, proposal, *result)
	require.Equal(t, idx.ValidatorID(33), proposer)
}

func TestExtractProposalForNextBlock_WrongProposals_ReturnsNoProposal(t *testing.T) {
	last := &evmcore.EvmHeader{
		Number: big.NewInt(100),
		Hash:   common.Hash{1, 2, 3},
	}

	tests := map[string]struct {
		proposal  inter.Proposal
		loggerMsg string
	}{
		"too high block number": {
			proposal: inter.Proposal{
				Number:     idx.Block(last.Number.Int64() + 2), // +1 is expected
				ParentHash: last.Hash,
			},
			loggerMsg: "wrong block number",
		},
		"block number matching current block": {
			proposal: inter.Proposal{
				Number:     idx.Block(last.Number.Int64()),
				ParentHash: last.Hash,
			},
			loggerMsg: "wrong block number",
		},
		"too low block number": {
			proposal: inter.Proposal{
				Number:     idx.Block(last.Number.Int64() - 1),
				ParentHash: last.Hash,
			},
			loggerMsg: "wrong block number",
		},
		"wrong parent hash": {
			proposal: inter.Proposal{
				Number:     idx.Block(last.Number.Int64() + 1),
				ParentHash: common.Hash{4, 5, 6},
			},
			loggerMsg: "wrong parent hash",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			event := inter.NewMockEventPayloadI(ctrl)
			logger := logger.NewMockLogger(ctrl)

			payload := &inter.Payload{Proposal: &test.proposal}
			event.EXPECT().Payload().Return(payload)
			creator := idx.ValidatorID(1)
			event.EXPECT().Creator().Return(creator).AnyTimes()

			events := []inter.EventPayloadI{event}

			any := gomock.Any()
			logger.EXPECT().Warn(
				gomock.Regex(test.loggerMsg),
				any, any, any, any, "creator", creator,
			)

			result, _ := extractProposalForNextBlock(last, events, logger)
			require.Nil(t, result)
		})
	}
}

func TestExtractProposalForNextBlock_MultipleValidProposals_EmitsWarning(t *testing.T) {
	ctrl := gomock.NewController(t)
	event1 := inter.NewMockEventPayloadI(ctrl)
	event2 := inter.NewMockEventPayloadI(ctrl)
	logger := logger.NewMockLogger(ctrl)

	last := &evmcore.EvmHeader{
		Number: big.NewInt(100),
		Hash:   common.Hash{1, 2, 3},
	}

	proposal := &inter.Proposal{
		Number:     idx.Block(last.Number.Int64() + 1),
		ParentHash: last.Hash,
	}

	payload1 := &inter.Payload{Proposal: proposal}
	payload2 := &inter.Payload{Proposal: proposal}
	event1.EXPECT().Payload().Return(payload1)
	event1.EXPECT().Creator().Return(idx.ValidatorID(1))
	event2.EXPECT().Payload().Return(payload2)
	event2.EXPECT().Creator().Return(idx.ValidatorID(2))

	events := []inter.EventPayloadI{event1, event2}

	logger.EXPECT().Warn(
		gomock.Regex("multiple proposals"),
		"block", proposal.Number, "proposals", len(events),
	)

	result, proposer := extractProposalForNextBlock(last, events, logger)
	require.NotNil(t, result)
	require.Equal(t, *proposal, *result)
	require.Equal(t, idx.ValidatorID(1), proposer)
}

func TestExtractProposalForNextBlock_MultipleValidProposals_UsesTurnAndHashAsTieBreaker(t *testing.T) {
	ctrl := gomock.NewController(t)
	event1 := inter.NewMockEventPayloadI(ctrl)
	event2 := inter.NewMockEventPayloadI(ctrl)
	event3 := inter.NewMockEventPayloadI(ctrl)
	logger := logger.NewMockLogger(ctrl)

	last := &evmcore.EvmHeader{
		Number: big.NewInt(100),
		Hash:   common.Hash{1, 2, 3},
	}

	payloads := []*inter.Payload{
		{
			ProposalSyncState: inter.ProposalSyncState{
				LastSeenProposalTurn: 1,
			},
			Proposal: &inter.Proposal{
				Number:     101,
				ParentHash: last.Hash,
				Time:       123,
			},
		},
		{
			ProposalSyncState: inter.ProposalSyncState{
				LastSeenProposalTurn: 1,
			},
			Proposal: &inter.Proposal{
				Number:     101,
				ParentHash: last.Hash,
				Time:       456,
			},
		},
		{
			ProposalSyncState: inter.ProposalSyncState{
				LastSeenProposalTurn: 2,
			},
			Proposal: &inter.Proposal{
				Number:     101,
				ParentHash: last.Hash,
				Time:       789,
			},
		},
	}

	slices.SortFunc(payloads, func(a, b *inter.Payload) int {
		turnA := a.LastSeenProposalTurn
		turnB := b.LastSeenProposalTurn
		if res := cmp.Compare(turnA, turnB); res != 0 {
			return res
		}
		hashA := a.Proposal.Hash()
		hashB := b.Proposal.Hash()
		return bytes.Compare(hashA[:], hashB[:])
	})

	event1.EXPECT().Payload().Return(payloads[0]).AnyTimes()
	event1.EXPECT().Creator().Return(idx.ValidatorID(1)).AnyTimes()
	event2.EXPECT().Payload().Return(payloads[1]).AnyTimes()
	event2.EXPECT().Creator().Return(idx.ValidatorID(2)).AnyTimes()
	event3.EXPECT().Payload().Return(payloads[2]).AnyTimes()
	event3.EXPECT().Creator().Return(idx.ValidatorID(3)).AnyTimes()
	events := []inter.EventPayloadI{event1, event2, event3}

	any := gomock.Any()
	logger.EXPECT().Warn(any, any, any, any, any).AnyTimes()

	for events := range utils.Permute(events) {
		proposal, proposer := extractProposalForNextBlock(last, events, logger)
		require.NotNil(t, proposal)
		require.Equal(t, payloads[0].Proposal, proposal,
			"should pick the best proposal based on turn and hash",
		)
		require.Equal(t, idx.ValidatorID(1), proposer)
	}
}

func TestResolveRandaoMix_ComputesRandaoMixFromReveal(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logger.NewMockLogger(ctrl)
	mockBackend := valkeystore.NewMockKeystoreI(ctrl)
	privateKey, publicKey := generateKeyPair(t)
	mockBackend.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil).AnyTimes()
	signer := valkeystore.NewSignerAuthority(mockBackend, publicKey)

	lastRandao := common.Hash{}
	reveal, expectedMix, err := randao.NewRandaoMixerAdapter(signer).MixRandao(lastRandao)
	require.NoError(t, err)

	proposer := idx.ValidatorID(1)
	dagRandao := common.Hash{}
	validatorKeys := map[idx.ValidatorID]validatorpk.PubKey{
		proposer: publicKey,
	}

	mix := resolveRandaoMix(reveal, proposer, validatorKeys, lastRandao, dagRandao, logger)
	require.Equal(t, expectedMix, mix, "should compute the correct Randao mix")
}

func TestResolveRandaoMix_FallsBackToDAGRandaoWhenVerificationFails(t *testing.T) {

	ctrl := gomock.NewController(t)
	mockBackend := valkeystore.NewMockKeystoreI(ctrl)
	privateKey, publicKey := generateKeyPair(t)
	mockBackend.EXPECT().GetUnlocked(publicKey).Return(privateKey, nil).AnyTimes()
	signer := valkeystore.NewSignerAuthority(mockBackend, publicKey)

	lastRandao := common.Hash{}
	reveal, _, err := randao.NewRandaoMixerAdapter(signer).MixRandao(lastRandao)
	require.NoError(t, err)

	proposer := idx.ValidatorID(1)
	dagRandao := common.Hash{1, 2, 3}

	logger := logger.NewMockLogger(ctrl)
	logger.EXPECT().Warn("Failed to verify randao reveal, using DAG randomization", "proposer validator", proposer)

	_, wrongKey := generateKeyPair(t)
	validatorKeys := map[idx.ValidatorID]validatorpk.PubKey{
		proposer: wrongKey,
	}

	mix := resolveRandaoMix(reveal, proposer, validatorKeys, lastRandao, dagRandao, logger)
	require.Equal(t, dagRandao, mix, "should compute the correct Randao mix")
}

// generateKeyPair is a helper function that creates a new ECDSA key pair
// and packs it in the data structures used by the gossip package.
func generateKeyPair(t testing.TB) (*encryption.PrivateKey, validatorpk.PubKey) {
	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	require.NoError(t, err)

	publicKey := validatorpk.PubKey{
		Raw:  crypto.FromECDSAPub(&privateKeyECDSA.PublicKey),
		Type: validatorpk.Types.Secp256k1,
	}
	privateKey := &encryption.PrivateKey{
		Type:    validatorpk.Types.Secp256k1,
		Decoded: privateKeyECDSA,
	}

	return privateKey, publicKey
}
