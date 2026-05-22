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

package gossip

import (
	"bytes"
	"cmp"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/lachesis"
	"github.com/Fantom-foundation/lachesis-base/utils/workers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/0xsoniclabs/sonic/gossip/blockproc"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/evmmodule"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/gossip/emitter"
	"github.com/0xsoniclabs/sonic/gossip/randao"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/0xsoniclabs/sonic/valkeystore"
	"github.com/0xsoniclabs/sonic/valkeystore/encryption"
)

func TestConsensusCallback(t *testing.T) {

	for name, feature := range opera.GetAllHardForksInOrder() {
		t.Run(name+"/single_proposer", func(t *testing.T) {
			feature.SingleProposerBlockFormation = true
			testConsensusCallback(t, feature)
		})
		t.Run(name+"/distributed_proposer", func(t *testing.T) {
			feature.SingleProposerBlockFormation = false
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
			fee := uint256.NewInt(0).Mul(new(uint256.Int).SetUint64(r.GasUsed), utils.BigIntToUint256Clamped(txs[i].GasPrice()))
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

func TestConsensusCallback_SingleProposer_HandlesBlockSkippingCorrectly(t *testing.T) {
	t.Parallel()
	MaxEmptyBlockSkipPeriod := inter.Timestamp(10_000)

	tests := map[string]struct {
		lastBlockTime inter.Timestamp
		atroposTime   inter.Timestamp
		proposal      *inter.Proposal
		proposalTime  inter.Timestamp
		producesBlock bool
		blockTime     inter.Timestamp
	}{
		"no proposal, before max empty block skip period": {
			lastBlockTime: inter.Timestamp(1000),
			atroposTime:   inter.Timestamp(2000),
			proposal:      nil,
			producesBlock: false,
		},
		"no proposal, after max empty block skip period": {
			lastBlockTime: inter.Timestamp(1000),
			atroposTime:   inter.Timestamp(1000 + MaxEmptyBlockSkipPeriod + 1),
			proposal:      nil,
			producesBlock: true,
			blockTime:     inter.Timestamp(1000 + MaxEmptyBlockSkipPeriod + 1),
		},
		"empty proposal, before max empty block skip period": {
			lastBlockTime: inter.Timestamp(1000),
			atroposTime:   inter.Timestamp(2000),
			proposal:      &inter.Proposal{},
			producesBlock: false, // empty proposals are ignored
		},
		"empty proposal, after max empty block skip period": {
			lastBlockTime: inter.Timestamp(1000),
			atroposTime:   inter.Timestamp(1000 + MaxEmptyBlockSkipPeriod + 42),
			proposal:      &inter.Proposal{},
			proposalTime:  inter.Timestamp(1000 + MaxEmptyBlockSkipPeriod + 1),
			producesBlock: true, // an empty block is created, with the proposal time
			blockTime:     inter.Timestamp(1000 + MaxEmptyBlockSkipPeriod + 1),
		},
		"non-empty proposal, before max empty block skip period": {
			lastBlockTime: inter.Timestamp(1000),
			atroposTime:   inter.Timestamp(2000),
			proposal: &inter.Proposal{
				Transactions: []*types.Transaction{types.NewTx(&types.LegacyTx{})},
			},
			proposalTime:  inter.Timestamp(1500),
			producesBlock: true,
			blockTime:     inter.Timestamp(1500),
		},
		"non-empty proposal, after max empty block skip period": {
			lastBlockTime: inter.Timestamp(1000),
			atroposTime:   inter.Timestamp(1000 + MaxEmptyBlockSkipPeriod + 42),
			proposal: &inter.Proposal{
				Transactions: []*types.Transaction{types.NewTx(&types.LegacyTx{})},
			},
			proposalTime:  inter.Timestamp(1000 + MaxEmptyBlockSkipPeriod + 1),
			producesBlock: true,
			blockTime:     inter.Timestamp(1000 + MaxEmptyBlockSkipPeriod + 1),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Evaluate a confirmed block with a two events, one carrying the
			// proposal given in the test, the other being the atropos.

			// Create a store with an applied genesis.
			upgrades := opera.GetAllegroUpgrades()
			upgrades.SingleProposerBlockFormation = true
			store := newInMemoryStoreWithGenesisData(t, upgrades, 1, 2)

			// Create the event carrying the proposal, if there is one.
			var events []*inter.EventPayload
			if test.proposal != nil {
				builder := inter.MutableEventPayload{}
				builder.SetVersion(3)
				builder.SetEpoch(2)
				builder.SetMedianTime(test.proposalTime)
				if test.proposal != nil {
					proposal := *test.proposal
					// Fix some required fields in any proposal.
					proposal.Number = 1
					proposal.ParentHash = store.GetBlock(0).Hash()
					builder.SetPayload(inter.Payload{
						Proposal: &proposal,
					})
				}
				events = append(events, builder.Build())
			}

			// Create the atropos event of the current block.
			builder := inter.MutableEventPayload{}
			builder.SetVersion(3)
			builder.SetEpoch(2)
			builder.SetMedianTime(test.atroposTime)
			atropos := builder.Build()
			events = append(events, atropos)

			// Publish the events in the store.
			for _, event := range events {
				store.SetEvent(event)
			}

			// Update the block and epoch state to match the test conditions.
			bs := store.GetBlockState()
			bs.LastBlock = iblockproc.BlockCtx{
				Time: test.lastBlockTime,
			}
			es := store.GetEpochState()
			es.Rules.Blocks.MaxEmptyBlockSkipPeriod = MaxEmptyBlockSkipPeriod
			store.SetBlockEpochState(bs, es)

			// Create the environment for the consensus callback cycle.
			ctrl := gomock.NewController(t)
			_any := gomock.Any()

			confirmedEventProcessor := blockproc.NewMockConfirmedEventsProcessor(ctrl)
			confirmedEventProcessor.EXPECT().Finalize(_any, _any).Return(iblockproc.BlockState{})
			confirmedEventProcessor.EXPECT().ProcessConfirmedEvent(_any).MaxTimes(2)

			eventsModule := blockproc.NewMockConfirmedEventsModule(ctrl)
			eventsModule.EXPECT().Start(_any, _any).Return(confirmedEventProcessor)

			proc := BlockProc{
				EventsModule: eventsModule,
			}

			// If a block is produced, mocks for the block creation process
			// need to be set up. This is implicitly checking that the
			// expectation of whether a block is produced or not is correct.
			if test.producesBlock {
				sealer := blockproc.NewMockSealerProcessor(ctrl)
				sealer.EXPECT().EpochSealing().Return(false)

				sealerModule := blockproc.NewMockSealerModule(ctrl)
				sealerModule.EXPECT().Start(_any, _any, _any).Return(sealer)

				txListener := blockproc.NewMockTxListener(ctrl)
				txListener.EXPECT().Finalize().Return(iblockproc.BlockState{}).AnyTimes()

				txListenerModule := blockproc.NewMockTxListenerModule(ctrl)
				txListenerModule.EXPECT().Start(_any, _any, _any, _any).Return(txListener)

				evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
				evmProcessor.EXPECT().Execute(_any, _any, _any).MinTimes(1)
				evmProcessor.EXPECT().Finalize().Return(&evmcore.EvmBlock{
					EvmHeader: evmcore.EvmHeader{
						BaseFee: big.NewInt(0),
						TxHash:  common.Hash{1, 2, 3},
					},
				}, 0, nil)

				evmModule := blockproc.NewMockEVM(ctrl)
				evmModule.EXPECT().
					Start(_any, _any, _any, _any, _any, _any, _any).
					DoAndReturn(func(block iblockproc.BlockCtx, _, _, _, _, _, _ any) blockproc.EVMProcessor {
						require.Equal(t, test.blockTime, block.Time)
						return evmProcessor
					})

				txTransactor := blockproc.NewMockTxTransactor(ctrl)
				txTransactor.EXPECT().PopInternalTxs(_any, _any, _any, _any, _any).Return(types.Transactions{}).AnyTimes()

				proc = BlockProc{
					EventsModule:     eventsModule,
					SealerModule:     sealerModule,
					TxListenerModule: txListenerModule,
					EVMModule:        evmModule,
					PreTxTransactor:  txTransactor,
					PostTxTransactor: txTransactor,
				}
			}

			// Create the worker group for running the callbacks.
			stop := make(chan struct{})
			var workerWaitGroup sync.WaitGroup
			workers := workers.New(&workerWaitGroup, stop, 1)
			workers.Start(1)
			defer func() {
				close(stop)
				workerWaitGroup.Wait()
			}()

			// Prepare the callback functions.
			var callbackWaitGroup sync.WaitGroup
			bootstrapping := false
			blockBusyFlag := uint32(0)
			emitters := []*emitter.Emitter{}
			beginBlock := consensusCallbackBeginBlockFn(
				workers, &callbackWaitGroup, &blockBusyFlag, store, proc, false, nil, &emitters, nil, &bootstrapping, nil,
			)

			// Run a full consensus callback cycle for this block.
			callbacks := beginBlock(&lachesis.Block{
				Atropos: atropos.ID(),
			})
			for _, event := range events {
				callbacks.ApplyEvent(event)
			}
			callbacks.EndBlock()

			callbackWaitGroup.Wait()
		})
	}
}

func TestExtractProposalForNextBlock_NoEvents_ReturnsNoProposal(t *testing.T) {
	last := &evmcore.EvmHeader{
		Number: big.NewInt(100),
	}
	result, proposer, time := extractProposalForNextBlock(last, nil, nil)
	require.Nil(t, result)
	require.Equal(t, idx.ValidatorID(0), proposer)
	require.Equal(t, inter.Timestamp(0), time)
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
	event.EXPECT().MedianTime().Return(inter.Timestamp(1234)).AnyTimes()
	events := []inter.EventPayloadI{event}

	result, proposer, time := extractProposalForNextBlock(last, events, nil)
	require.NotNil(t, result)
	require.Equal(t, proposal, *result)
	require.Equal(t, idx.ValidatorID(33), proposer)
	require.Equal(t, inter.Timestamp(1234), time)
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

			result, _, _ := extractProposalForNextBlock(last, events, logger)
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
	event1.EXPECT().MedianTime().Return(inter.Timestamp(1))
	event2.EXPECT().Payload().Return(payload2)
	event2.EXPECT().Creator().Return(idx.ValidatorID(2))
	event2.EXPECT().MedianTime().Return(inter.Timestamp(2))

	events := []inter.EventPayloadI{event1, event2}

	logger.EXPECT().Warn(
		gomock.Regex("multiple proposals"),
		"block", proposal.Number, "proposals", len(events),
	)

	result, proposer, time := extractProposalForNextBlock(last, events, logger)
	require.NotNil(t, result)
	require.Equal(t, *proposal, *result)
	require.Equal(t, idx.ValidatorID(1), proposer)
	require.Equal(t, inter.Timestamp(1), time)
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
				Number:       101,
				ParentHash:   last.Hash,
				RandaoReveal: randao.RandaoReveal{1, 2, 3},
			},
		},
		{
			ProposalSyncState: inter.ProposalSyncState{
				LastSeenProposalTurn: 1,
			},
			Proposal: &inter.Proposal{
				Number:       101,
				ParentHash:   last.Hash,
				RandaoReveal: randao.RandaoReveal{4, 5, 6},
			},
		},
		{
			ProposalSyncState: inter.ProposalSyncState{
				LastSeenProposalTurn: 2,
			},
			Proposal: &inter.Proposal{
				Number:       101,
				ParentHash:   last.Hash,
				RandaoReveal: randao.RandaoReveal{7, 8, 9},
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
	event1.EXPECT().MedianTime().Return(inter.Timestamp(1)).AnyTimes()
	event2.EXPECT().Payload().Return(payloads[1]).AnyTimes()
	event2.EXPECT().Creator().Return(idx.ValidatorID(2)).AnyTimes()
	event2.EXPECT().MedianTime().Return(inter.Timestamp(2)).AnyTimes()
	event3.EXPECT().Payload().Return(payloads[2]).AnyTimes()
	event3.EXPECT().Creator().Return(idx.ValidatorID(3)).AnyTimes()
	event3.EXPECT().MedianTime().Return(inter.Timestamp(3)).AnyTimes()
	events := []inter.EventPayloadI{event1, event2, event3}

	any := gomock.Any()
	logger.EXPECT().Warn(any, any, any, any, any).AnyTimes()

	for events := range utils.Permute(events) {
		proposal, proposer, time := extractProposalForNextBlock(last, events, logger)
		require.NotNil(t, proposal)
		require.Equal(t, payloads[0].Proposal, proposal,
			"should pick the best proposal based on turn and hash",
		)
		require.Equal(t, idx.ValidatorID(1), proposer)
		require.Equal(t, inter.Timestamp(1), time)
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

func TestFilterNonPermissibleTransactions_InactiveWithoutAllegro(t *testing.T) {
	require := require.New(t)

	withoutAllegro := opera.Rules{}
	withAllegro := opera.Rules{
		Upgrades: opera.Upgrades{
			Allegro: true,
		},
	}

	valid := types.NewTx(&types.LegacyTx{})
	invalid := types.NewTx(&types.SetCodeTx{})

	require.NoError(isPermissible(valid, &withAllegro, nil))
	require.Error(isPermissible(invalid, &withAllegro, nil))

	txs := []*types.Transaction{valid, invalid}

	require.Equal(txs, filterNonPermissibleTransactions(txs, &withoutAllegro, nil, nil, nil))
	require.Equal([]*types.Transaction{valid}, filterNonPermissibleTransactions(txs, &withAllegro, nil, nil, nil))
}

func TestFilterNonPermissibleTransactions_FiltersNonPermissibleTransactions(t *testing.T) {
	rules := opera.Rules{
		Upgrades: opera.Upgrades{
			Allegro: true,
		},
	}

	valid1 := types.NewTx(&types.LegacyTx{Nonce: 1})
	valid2 := types.NewTx(&types.LegacyTx{Nonce: 2})
	valid3 := types.NewTx(&types.LegacyTx{Nonce: 3})

	invalid := types.NewTx(&types.SetCodeTx{})

	txs := []*types.Transaction{invalid, valid1, invalid, valid2, invalid, invalid, valid3, invalid}
	want := []*types.Transaction{valid1, valid2, valid3}
	require.Equal(t, want, filterNonPermissibleTransactions(txs, &rules, nil, nil, nil))
}

func TestFilterNonPermissibleTransactions_LogsIssuesOfNonPermissibleTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := logger.NewMockLogger(ctrl)

	rules := opera.Rules{
		Upgrades: opera.Upgrades{
			Allegro: true,
		},
	}

	invalid1 := types.NewTx(&types.SetCodeTx{})
	invalid2 := types.NewTx(&types.BlobTx{
		BlobHashes: []common.Hash{{1, 2, 3}},
	})

	log.EXPECT().Warn(
		"Non-permissible transaction in the proposal",
		"tx", gomock.Any(),
		"issue", isPermissible(invalid1, &rules, nil),
	)

	log.EXPECT().Warn(
		"Non-permissible transaction in the proposal",
		"tx", gomock.Any(),
		"issue", isPermissible(invalid2, &rules, nil),
	)

	filterNonPermissibleTransactions(
		[]*types.Transaction{invalid1, invalid2},
		&rules,
		nil,
		log,
		nil,
	)
}

func TestFilterNonPermissibleTransactions_ReportsNonPermissibleTransactionsToMonitoring(t *testing.T) {
	ctrl := gomock.NewController(t)
	counter := NewMockmetricCounter(ctrl)

	rules := opera.Rules{
		Upgrades: opera.Upgrades{
			Allegro: true,
		},
	}

	valid := types.NewTx(&types.LegacyTx{Nonce: 1})
	invalid := types.NewTx(&types.SetCodeTx{})

	// One issue reported per invalid transaction.
	counter.EXPECT().Mark(int64(1))
	counter.EXPECT().Mark(int64(1))

	filterNonPermissibleTransactions(
		[]*types.Transaction{valid, invalid, valid, invalid},
		&rules,
		nil,
		nil,
		counter,
	)
}

func TestIsPermissible_AcceptsPermissibleTransactions(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tests := map[string]*types.Transaction{
		"legacy":      types.NewTx(&types.LegacyTx{}),
		"access list": types.NewTx(&types.AccessListTx{}),
		"dynamic fee": types.NewTx(&types.DynamicFeeTx{}),
		"blob":        types.NewTx(&types.BlobTx{}),
		"set code": types.NewTx(&types.SetCodeTx{
			AuthList: []types.SetCodeAuthorization{{}},
		}),
		"empty all-of bundle": bundle.AllOf().Build(),
		"empty one-of bundle": bundle.OneOf().Build(),
		"non-empty bundle": bundle.AllOf(
			bundle.Step(key, &types.AccessListTx{}),
			bundle.Step(key, &types.AccessListTx{}),
		).Build(),
		"composed bundle": bundle.OneOf(
			bundle.AllOf(
				bundle.Step(key, &types.AccessListTx{}),
			),
		).Build(),
		"nested bundle": bundle.AllOf(
			bundle.Step(key, bundle.AllOf(
				bundle.Step(key, &types.AccessListTx{}),
			).Build()),
		).Build(),
	}

	rules := opera.Rules{
		Upgrades: opera.Upgrades{
			Allegro:            true,
			Brio:               true,
			TransactionBundles: true,
		},
	}
	for name, tx := range tests {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, isPermissible(tx, &rules, signer))
		})
	}
}

func TestIsPermissible_AcceptsSetCodeTransactionsInAllegroAndBeyond(t *testing.T) {
	tx := types.NewTx(&types.SetCodeTx{
		AuthList: []types.SetCodeAuthorization{{}},
	})

	for name, updates := range opera.GetAllHardForksInOrder() {
		t.Run(name, func(t *testing.T) {
			rules := opera.Rules{Upgrades: updates}
			if updates.Allegro {
				require.NoError(t, isPermissible(tx, &rules, nil))
			} else {
				require.ErrorContains(t,
					isPermissible(tx, &rules, nil),
					"unsupported transaction type",
				)
			}
		})
	}
}

func TestIsPermissible_WithBrio_RejectsTransactionsFailingStaticChecks(t *testing.T) {

	invalidTx := types.NewTx(&types.DynamicFeeTx{
		GasFeeCap: big.NewInt(-1), // Invalid gas price
	})

	issue := evmcore.ValidateTxStatic(invalidTx)
	require.Error(t, issue)

	// Before Brio, static checks are ignored for permissibility.
	rules := opera.Rules{Upgrades: opera.Upgrades{Brio: false}}
	require.NoError(t, isPermissible(invalidTx, &rules, nil))

	// With Brio, transactions failing static checks are not permissible.
	rules = opera.Rules{Upgrades: opera.Upgrades{Brio: true}}
	require.ErrorIs(t, isPermissible(invalidTx, &rules, nil), issue)
}

func TestIsPermissible_WithBrio_DetectsInvalidTransactionsInBundles(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	invalidInnerTx := types.NewTx(&types.DynamicFeeTx{
		GasFeeCap: new(big.Int).Lsh(big.NewInt(1), 256), // Invalid fee cap
	})
	issue := evmcore.ValidateTxStatic(invalidInnerTx)
	require.Error(t, issue)

	signer := types.LatestSignerForChainID(big.NewInt(1))

	tests := map[string]*types.Transaction{
		"invalid transaction in bundle": bundle.NewBuilder().AllOf(
			bundle.Step(key, invalidInnerTx),
		).WithSigner(signer).Build(),
		"invalid nested transaction in bundle": bundle.NewBuilder().OneOf(
			bundle.Step(key, bundle.NewBuilder().AllOf(
				bundle.Step(key, invalidInnerTx),
			).WithSigner(signer).Build()),
		).WithSigner(signer).Build(),
	}

	for name, tx := range tests {
		t.Run(name, func(t *testing.T) {
			rules := opera.Rules{Upgrades: opera.Upgrades{
				Brio:               true,
				TransactionBundles: true,
			}}
			got := isPermissible(tx, &rules, signer)
			require.ErrorIs(t, got, issue)
		})
	}
}

func TestIsPermissible_WithBrio_RejectsBundleOnlyTransactions(t *testing.T) {
	require := require.New(t)
	tx := types.NewTx(&types.AccessListTx{
		AccessList: []types.AccessTuple{{
			Address: bundle.BundleOnly,
		}},
	})

	require.True(bundle.IsBundleOnly(tx))

	// Before Brio, this transaction should be permissible.
	rules := opera.Rules{Upgrades: opera.Upgrades{Brio: false}}
	require.NoError(isPermissible(tx, &rules, nil))

	// With Brio, this transaction should not be permissible.
	rules.Upgrades.Brio = true
	require.ErrorContains(isPermissible(tx, &rules, nil), "bundle-only transactions are not supported")
}

func TestIsPermissible_WithBrio_BundlesDisabled_RejectsBundleEnvelopes(t *testing.T) {
	require := require.New(t)
	signer := types.LatestSignerForChainID(big.NewInt(1))
	tx := bundle.AllOf().Build()

	require.True(bundle.IsEnvelope(tx))

	// Before Brio, envelopes are accepted, but not interpreted as such.
	rules := opera.Rules{Upgrades: opera.Upgrades{Brio: false}}
	require.NoError(isPermissible(tx, &rules, signer))

	// With Brio, envelopes are rejected if bundles are disabled.
	rules.Upgrades.Brio = true
	rules.Upgrades.TransactionBundles = false
	require.ErrorContains(isPermissible(tx, &rules, signer), "bundle transactions are disabled")

	// If bundles are enabled, envelopes should be accepted.
	rules.Upgrades.TransactionBundles = true
	require.NoError(isPermissible(tx, &rules, signer))
}

func TestIsPermissible_NonOpenableEnvelope_IsRejectedWithBrio(t *testing.T) {
	require := require.New(t)
	tx := types.NewTx(&types.LegacyTx{
		To:   &bundle.BundleProcessor,
		Data: []byte("not a valid encoding"),
	})

	require.True(bundle.IsEnvelope(tx))

	// Before Brio, this is accepted.
	rules := opera.Rules{Upgrades: opera.Upgrades{Brio: false}}
	require.NoError(isPermissible(tx, &rules, nil))

	// With Brio, and bundles disabled, this is rejected for being an envelope.
	rules.Upgrades.Brio = true
	rules.Upgrades.TransactionBundles = false
	require.ErrorContains(isPermissible(tx, &rules, nil), "bundle transactions are disabled")

	// With Brio and bundles enabled, this should be rejected as it's an invalid envelope.
	rules.Upgrades.Brio = true
	rules.Upgrades.TransactionBundles = true
	require.ErrorContains(isPermissible(tx, &rules, nil), "invalid bundle envelope")
}

func TestIsPermissible_BundlesWithInvalidContent_Rejected(t *testing.T) {
	require := require.New(t)
	signer := types.LatestSignerForChainID(big.NewInt(1))

	key, err := crypto.GenerateKey()
	require.NoError(err)

	tx, txBundle, _ := bundle.NewBuilder().
		With(bundle.Step(key, &types.SetCodeTx{
			// Invalid, since there are no authorizations.
		})).
		BuildEnvelopeBundleAndPlan()

	rules := opera.Rules{Upgrades: opera.Upgrades{
		Allegro:            true,
		Brio:               true,
		TransactionBundles: true,
	}}

	txs := txBundle.GetTransactionsInReferencedOrder()

	// The first transaction in the bundle is not permissible.
	issue := isPermissible(txs[0], &rules, signer)
	require.Error(issue)

	// Thus, the bundle should be rejected.
	got := isPermissible(tx, &rules, signer)
	require.ErrorContains(got, "bundle contains non-permissible transaction")
	require.ErrorContains(got, issue.Error())
}

func TestIsPermissible_BundlesWithInvalidNestedContent_Rejected(t *testing.T) {
	require := require.New(t)
	signer := types.LatestSignerForChainID(big.NewInt(1))

	key, err := crypto.GenerateKey()
	require.NoError(err)

	// Issues in nested bundles are also detected.
	inner, txBundle, _ := bundle.NewBuilder().
		With(bundle.Step(key, &types.SetCodeTx{
			// Invalid, since there are no authorizations.
		})).
		BuildEnvelopeBundleAndPlan()

	outer := bundle.NewBuilder().
		With(bundle.Step(key, inner)).
		Build()

	rules := opera.Rules{Upgrades: opera.Upgrades{
		Allegro:            true,
		Brio:               true,
		TransactionBundles: true,
	}}

	txs := txBundle.GetTransactionsInReferencedOrder()

	// The first transaction in the bundle is not permissible.
	issue := isPermissible(txs[0], &rules, signer)
	require.Error(issue)

	// Thus, the bundle should be rejected.
	got := isPermissible(outer, &rules, signer)
	require.ErrorContains(got, "bundle contains non-permissible transaction")
	require.ErrorContains(got, issue.Error())
}

func TestMergeCheaters_CanMergeLists(t *testing.T) {

	// This test checks the current behavior of merging cheaters lists,
	// it does not check for order or duplicates. Although the function
	// can be improved, any modification risks breaking the history replay.
	//
	// - it will copy verbatim the cheaters from the first argument list
	// and append cheaters from the second list, removing duplicates.
	// - it will not remove duplicates from the first argument list if any.
	// - it will preserve the order of both lists.
	// - it does not modify the original lists.

	tests := map[string]struct {
		a, b     lachesis.Cheaters
		expected lachesis.Cheaters
	}{
		"both empty returns nil": {},
		"a empty returns b": {
			b:        lachesis.Cheaters{1, 2, 3},
			expected: lachesis.Cheaters{1, 2, 3},
		},
		"b empty returns a": {
			a:        lachesis.Cheaters{1, 2, 3},
			expected: lachesis.Cheaters{1, 2, 3},
		},
		"merges both lists": {
			a:        lachesis.Cheaters{1, 2, 3},
			b:        lachesis.Cheaters{4, 5, 6},
			expected: lachesis.Cheaters{1, 2, 3, 4, 5, 6},
		},
		"preserves duplicates from first list": {
			a:        lachesis.Cheaters{1, 2, 3, 1, 2, 3},
			b:        lachesis.Cheaters{7},
			expected: lachesis.Cheaters{1, 2, 3, 1, 2, 3, 7},
		},
		"removes duplicates from second list": {
			a:        lachesis.Cheaters{1, 2, 3},
			b:        lachesis.Cheaters{3, 4, 2},
			expected: lachesis.Cheaters{1, 2, 3, 4},
		},
		"order is preserved": {
			a:        lachesis.Cheaters{1, 3, 5},
			b:        lachesis.Cheaters{2, 4},
			expected: lachesis.Cheaters{1, 3, 5, 2, 4},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			copyA := slices.Clone(test.a)
			copyB := slices.Clone(test.b)

			// merge cheaters
			cheaters := mergeCheaters(test.a, test.b)
			require.Equal(t, test.expected, cheaters)
			require.Equal(t, test.a, copyA, "first argument should not be modified")
			require.Equal(t, test.b, copyB, "second argument should not be modified")
		})
	}
}

func TestIsPermissible_DetectsNonPermissibleTransactions(t *testing.T) {
	tests := map[string]struct {
		transaction *types.Transaction
		issue       string
	}{
		"nil transaction": {
			transaction: nil,
			issue:       "nil transaction",
		},
		"blob with blob hashes": {
			transaction: types.NewTx(&types.BlobTx{
				BlobHashes: []common.Hash{{1, 2, 3}},
			}),
			issue: "blob transaction with blob hashes is not supported, got 1",
		},
		"set code without authorization": {
			transaction: types.NewTx(&types.SetCodeTx{}),
			issue:       "set code transaction without authorizations is not supported",
		},
	}

	rules := opera.Rules{
		Upgrades: opera.Upgrades{
			Allegro: true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := isPermissible(test.transaction, &rules, nil)
			require.ErrorContains(t, err, test.issue)
		})
	}
}

func TestRlpEncodedMaxHeaderSizeInBytes_IsAnUpperBound(t *testing.T) {
	setToMax := func(size int) []byte {
		b := bytes.Repeat([]byte{0xFF}, size)
		return b
	}

	maxHash := common.Hash(setToMax(32))
	maxAddress := common.Address(setToMax(20))
	maxBloom := types.Bloom(setToMax(2048))
	maxBlock := types.BlockNonce(setToMax(8))
	maxUint64 := uint64(math.MaxUint64)

	// The sanity checks for the extra field inside of geth define an upper bound
	// of 100 * 1024 bytes. Sonic uses the extra field to store time and duration.
	extra := inter.EncodeExtraData(
		time.Unix(math.MaxInt64, math.MaxInt64),
		time.Duration(math.MaxInt64)*time.Nanosecond,
	)

	header := &types.Header{
		ParentHash:       maxHash,
		UncleHash:        maxHash,
		Coinbase:         maxAddress,
		Root:             maxHash,
		TxHash:           maxHash,
		ReceiptHash:      maxHash,
		Bloom:            maxBloom,
		Difficulty:       big.NewInt(math.MaxInt64),
		Number:           big.NewInt(math.MaxInt64),
		GasLimit:         math.MaxUint64,
		GasUsed:          math.MaxUint64,
		Time:             math.MaxUint64,
		Extra:            extra,
		MixDigest:        maxHash,
		Nonce:            maxBlock,
		BaseFee:          big.NewInt(math.MaxInt64),
		WithdrawalsHash:  &maxHash,
		BlobGasUsed:      &maxUint64,
		ExcessBlobGas:    &maxUint64,
		ParentBeaconRoot: &maxHash,
		RequestsHash:     &maxHash,
	}

	data, err := rlp.EncodeToBytes(header)
	require.NoError(t, err)
	require.Less(t, len(data), rlpEncodedMaxHeaderSizeInBytes, "header exceeds maximum size")
}

func TestProcessUserTransactions_ForwardsBlockGasLimitToEVMProcessor(t *testing.T) {
	ctrl := gomock.NewController(t)
	evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
	blockBuilder := inter.NewBlockBuilder()

	// Create some dummy transactions with gas usage
	tx1 := types.NewTx(&types.LegacyTx{Nonce: 1, Gas: 1000})
	tx2 := types.NewTx(&types.LegacyTx{Nonce: 2, Gas: 1000})
	tx3 := types.NewTx(&types.LegacyTx{Nonce: 3, Gas: 1000})

	// Set up receipts for each transaction
	receipt1 := &types.Receipt{GasUsed: 1000}
	receipt2 := &types.Receipt{GasUsed: 1000}
	receipt3 := &types.Receipt{GasUsed: 1000}

	userTransactionGasLimit := uint64(3000)

	orderedTxs := []*types.Transaction{tx1, tx2, tx3}

	// Mock EVMProcessor.Execute to be called once with all txs and the full gas limit
	evmProcessor.EXPECT().
		Execute(orderedTxs, userTransactionGasLimit, gomock.Any()).
		Return(evmcore.ProcessSummary{ProcessedTransactions: []evmcore.ProcessedTransaction{
			{Transaction: tx1, Receipt: receipt1},
			{Transaction: tx2, Receipt: receipt2},
			{Transaction: tx3, Receipt: receipt3},
		}})

	upgrades := opera.Upgrades{Brio: true}
	processUserTransactions(evmProcessor, blockBuilder, orderedTxs, userTransactionGasLimit, upgrades)

	// All transactions should be included
	gotTxs := blockBuilder.GetTransactions()
	require.Equal(t, types.Transactions{tx1, tx2, tx3}, gotTxs)
}

func TestProcessUserTransactions_TransactionsWithNoReceiptAreNotIncluded(t *testing.T) {
	ctrl := gomock.NewController(t)
	evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
	blockBuilder := inter.NewBlockBuilder()

	tx := types.NewTx(&types.LegacyTx{Nonce: 1})

	// Simulate skipped transaction (no receipt)
	evmProcessor.EXPECT().
		Execute([]*types.Transaction{tx}, gomock.Any(), gomock.Any()).
		Return(evmcore.ProcessSummary{ProcessedTransactions: []evmcore.ProcessedTransaction{{Transaction: tx, Receipt: nil}}})

	upgrades := opera.Upgrades{Brio: true}
	processUserTransactions(evmProcessor, blockBuilder, []*types.Transaction{tx}, 10000, upgrades)

	// Should not be added to blockBuilder
	gotTxs := blockBuilder.GetTransactions()
	require.Empty(t, gotTxs)
}

func TestProcessUserTransactions_DeductsInternalTxsSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
	blockBuilder := inter.NewBlockBuilder()

	// Add an internal tx to blockBuilder
	internalTx := types.NewTx(&types.LegacyTx{Data: make([]byte, 1000)})
	blockBuilder.AddTransaction(internalTx, &types.Receipt{})

	// Create a user tx
	userTx := types.NewTx(&types.LegacyTx{Data: make([]byte, 1000)})

	// Ensure the evmProcessor is called with the correct remaining size after accounting for the internal transaction.
	orderedTxs := []*types.Transaction{userTx}
	evmProcessor.EXPECT().
		Execute(orderedTxs, uint64(10000), params.MaxBlockSize-rlpEncodedMaxHeaderSizeInBytes-internalTx.Size()).
		Return(evmcore.ProcessSummary{ProcessedTransactions: []evmcore.ProcessedTransaction{
			{Transaction: userTx, Receipt: &types.Receipt{}},
		}})

	upgrades := opera.Upgrades{Brio: true}
	processUserTransactions(evmProcessor, blockBuilder, orderedTxs, 10000, upgrades)

	// Ensure both internal and user transactions are included.
	gotTxs := blockBuilder.GetTransactions()
	require.Equal(t, types.Transactions{internalTx, userTx}, gotTxs)
}

func TestProcessUserTransactions_SkipsTxsExceedingSizeLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
	blockBuilder := inter.NewBlockBuilder()

	// Create a tx that exceeds the size limit
	largeTx := types.NewTx(&types.LegacyTx{Data: make([]byte, params.MaxBlockSize)})

	// Following txs are not skipped
	tx0 := types.NewTx(&types.LegacyTx{Data: []byte{0x01}})
	tx1 := types.NewTx(&types.LegacyTx{Data: []byte{0x01}})

	orderedTxs := []*types.Transaction{largeTx, tx0, tx1}

	// All txs are passed to Execute; the EVMProcessor handles size-based skipping
	evmProcessor.EXPECT().
		Execute(orderedTxs, uint64(10000), gomock.Any()).
		Return(evmcore.ProcessSummary{ProcessedTransactions: []evmcore.ProcessedTransaction{
			{Transaction: largeTx, Receipt: nil}, // skipped due to size
			{Transaction: tx0, Receipt: &types.Receipt{}},
			{Transaction: tx1, Receipt: &types.Receipt{}},
		}})

	upgrades := opera.Upgrades{Brio: true}
	processUserTransactions(evmProcessor, blockBuilder, orderedTxs, 10000, upgrades)

	// Huge transactions should not be added
	gotTxs := blockBuilder.GetTransactions()
	require.Equal(t, types.Transactions{tx0, tx1}, gotTxs)
}

func TestProcessUserTransactions_InternalTransactionsHaveNoImpactOnTheUserTransactionGas(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	statedb.EXPECT().BeginBlock(gomock.Any())
	statedb.EXPECT().SetTxContext(gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(0)).AnyTimes()
	statedb.EXPECT().SubBalance(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().Prepare(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)).AnyTimes()
	statedb.EXPECT().SetNonce(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().GetCode(gomock.Any()).AnyTimes()
	statedb.EXPECT().Snapshot().AnyTimes()
	statedb.EXPECT().Exist(gomock.Any()).Return(true).AnyTimes()
	statedb.EXPECT().SubBalance(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().AddBalance(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().GetCode(gomock.Any()).AnyTimes()
	statedb.EXPECT().GetRefund().AnyTimes()
	statedb.EXPECT().GetRefund().AnyTimes()
	statedb.EXPECT().AddBalance(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().GetLogs(gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().EndTransaction().AnyTimes()
	statedb.EXPECT().TxIndex().AnyTimes()

	evmModule := evmmodule.New()
	evmProcessor := evmModule.Start(
		iblockproc.BlockCtx{},
		statedb,
		&EvmStateReader{},
		func(l *core_types.Log) {},
		opera.Rules{},
		&params.ChainConfig{},
		common.Hash{},
	)
	blockBuilder := inter.NewBlockBuilder()

	internalTx := types.NewTx(&types.LegacyTx{To: &common.Address{0x41}, Gas: 21_000})
	internalTxGasLimit := uint64(30_000)
	internalProcessedTxs := evmProcessor.Execute([]*types.Transaction{internalTx}, internalTxGasLimit, math.MaxUint64).ProcessedTransactions
	require.Len(t, internalProcessedTxs, 1)

	userTx0 := types.NewTx(&types.LegacyTx{To: &common.Address{0x42}, Gas: 21_000})
	skippedTx := types.NewTx(&types.LegacyTx{To: &common.Address{0x44}, Gas: 21_000})

	// the user transaction only fits into the user transaction gas limit if
	// the internal transaction gas is not counted towards it
	userTransactionGasLimit := uint64(30_000)
	upgrades := opera.Upgrades{Brio: true}
	processUserTransactions(evmProcessor, blockBuilder, []*types.Transaction{userTx0, skippedTx}, userTransactionGasLimit, upgrades)

	gotTxs := blockBuilder.GetTransactions()
	require.Equal(t, types.Transactions{userTx0}, gotTxs)
}

func TestProcessUserTransactions_SponsoredTxSizeIsAccountedCorrectly(t *testing.T) {
	tests := map[string]struct {
		gasPrice      int64
		isSponsoredTx bool
	}{
		"normal tx": {
			gasPrice:      10,
			isSponsoredTx: false,
		},
		"sponsored tx": {
			gasPrice:      0,
			isSponsoredTx: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
			blockBuilder := inter.NewBlockBuilder()

			tx0 := types.NewTx(&types.LegacyTx{
				Data: make([]byte, params.MaxBlockSize-5000),
			})
			tx1 := types.NewTx(&types.LegacyTx{
				To:       &common.Address{0x42},
				GasPrice: big.NewInt(test.gasPrice),
				V:        big.NewInt(1),
			})
			remainingSize := params.MaxBlockSize - tx0.Size() - tx1.Size() - rlpEncodedMaxHeaderSizeInBytes
			tx2 := types.NewTx(&types.LegacyTx{
				Data: make([]byte, remainingSize-100), // leave some room for other fields in tx
			})

			var processedTxs []evmcore.ProcessedTransaction
			var feeChargingTransaction *types.Transaction
			if test.gasPrice == 0 {
				feeChargingTransaction = types.NewTx(&types.LegacyTx{
					// Fill the tx to simulate the size of a fee charging tx
					Data: make([]byte, subsidies.RlpEncodedFeeChargingTxSizeInBytes),
				})
				processedTxs = []evmcore.ProcessedTransaction{
					{Transaction: tx0, Receipt: &types.Receipt{}},
					{Transaction: tx1, Receipt: &types.Receipt{}},
					{Transaction: feeChargingTransaction, Receipt: &types.Receipt{}},
					{Transaction: tx2, Receipt: nil}, // skipped due to size
				}
			} else {
				processedTxs = []evmcore.ProcessedTransaction{
					{Transaction: tx0, Receipt: &types.Receipt{}},
					{Transaction: tx1, Receipt: &types.Receipt{}},
					{Transaction: tx2, Receipt: &types.Receipt{}},
				}
			}

			orderedTxs := []*types.Transaction{tx0, tx1, tx2}
			evmProcessor.EXPECT().
				Execute(orderedTxs, uint64(10000), gomock.Any()).
				Return(evmcore.ProcessSummary{
					ProcessedTransactions: processedTxs,
				})

			upgrades := opera.Upgrades{Brio: true}
			processUserTransactions(evmProcessor, blockBuilder, orderedTxs, 10000, upgrades)

			gotTxs := blockBuilder.GetTransactions()
			require.Contains(t, gotTxs, tx0)
			require.Contains(t, gotTxs, tx1)

			if test.isSponsoredTx {
				// ensure the sponsored transaction is followed by the gas paying transaction
				require.Equal(t, tx1, gotTxs[1])
				require.Equal(t, feeChargingTransaction, gotTxs[2])

				require.NotContains(t, gotTxs, tx2)
			} else {
				require.Contains(t, gotTxs, tx2)
			}
		})
	}
}

func TestProcessUserTransactions_SkipUserTransactionIfInternalTransactionsExceedBlockSizeLimit(t *testing.T) {
	tests := map[string][]*types.Transaction{
		"single huge internal tx": {
			types.NewTx(&types.LegacyTx{Data: make([]byte, params.MaxBlockSize)}),
		},
		"multiple internal txs exceeding block size": {
			types.NewTx(&types.LegacyTx{Data: make([]byte, params.MaxBlockSize/4)}),
			types.NewTx(&types.LegacyTx{Data: make([]byte, params.MaxBlockSize/4)}),
			types.NewTx(&types.LegacyTx{Data: make([]byte, params.MaxBlockSize/4)}),
			types.NewTx(&types.LegacyTx{Data: make([]byte, params.MaxBlockSize/4)}),
		},
	}

	for name, internalTxs := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
			blockBuilder := inter.NewBlockBuilder()

			for _, internalTx := range internalTxs {
				// Add internal txs to blockBuilder
				blockBuilder.AddTransaction(internalTx, &types.Receipt{})
			}

			// Create a user tx that would only fit without the internal tx
			userTx := types.NewTx(&types.LegacyTx{})
			upgrades := opera.Upgrades{Brio: true}

			evmProcessor.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any()).Return(evmcore.ProcessSummary{})
			processUserTransactions(evmProcessor, blockBuilder, []*types.Transaction{userTx}, 10000, upgrades)

			// Only internal tx should be present
			gotTxs := blockBuilder.GetTransactions()
			require.Equal(t, types.Transactions(internalTxs), gotTxs)
		})
	}
}

func TestProcessUserTransactions_InternalTxsExceedingBlockSizePassesZeroRemainingSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
	blockBuilder := inter.NewBlockBuilder()

	// Add two internal transactions: one that nearly fills the block, and one that exceeds the remaining size.
	smallTx := types.NewTx(&types.LegacyTx{Data: make([]byte, params.MaxBlockSize-rlpEncodedMaxHeaderSizeInBytes-100)})
	overflowTx := types.NewTx(&types.LegacyTx{Data: make([]byte, 200)})
	blockBuilder.AddTransaction(smallTx, &types.Receipt{})
	blockBuilder.AddTransaction(overflowTx, &types.Receipt{})

	userTx := types.NewTx(&types.LegacyTx{})
	orderedTxs := []*types.Transaction{userTx}
	upgrades := opera.Upgrades{Brio: true}

	// After the overflow is detected, remainingSize is set to 0
	evmProcessor.EXPECT().
		Execute(orderedTxs, uint64(10000), gomock.Any()).
		DoAndReturn(func(txs []*types.Transaction, gas uint64, size uint64) evmcore.ProcessSummary {
			require.Equal(t, uint64(0), size) // Ensure the user transactions are processed with zero remaining size
			return evmcore.ProcessSummary{}
		})

	processUserTransactions(evmProcessor, blockBuilder, orderedTxs, 10000, upgrades)
}

func TestProcessUserTransactions_RecordsOriginTransactionForAcceptedProcessedTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	evmProcessor := blockproc.NewMockEVMProcessor(ctrl)

	transactions := []*types.Transaction{
		types.NewTx(&types.LegacyTx{Nonce: 0}),
		types.NewTx(&types.LegacyTx{Nonce: 1}),
		types.NewTx(&types.LegacyTx{Nonce: 2}),
	}

	extras := []*types.Transaction{
		types.NewTx(&types.LegacyTx{Nonce: 11}),
		types.NewTx(&types.LegacyTx{Nonce: 21}),
		types.NewTx(&types.LegacyTx{Nonce: 22}),
		types.NewTx(&types.LegacyTx{Nonce: 23}),
	}

	wantCausedBy := map[common.Hash]common.Hash{
		transactions[0].Hash(): transactions[0].Hash(),
		transactions[1].Hash(): transactions[1].Hash(),
		extras[0].Hash():       transactions[1].Hash(),
		extras[1].Hash():       transactions[2].Hash(),
		extras[2].Hash():       transactions[2].Hash(),
		// no entry for extras[3] since it has no receipt
	}

	evmProcessor.EXPECT().
		Execute(transactions, gomock.Any(), gomock.Any()).
		Return(evmcore.ProcessSummary{
			ProcessedTransactions: []evmcore.ProcessedTransaction{
				{Transaction: transactions[0], Receipt: &types.Receipt{}},
				{Transaction: transactions[1], Receipt: &types.Receipt{}},
				{Transaction: extras[0], Receipt: &types.Receipt{}},
				{Transaction: extras[1], Receipt: &types.Receipt{}},
				{Transaction: extras[2], Receipt: &types.Receipt{}},
				{Transaction: extras[3], Receipt: nil},
			},
			CausedBy: wantCausedBy,
		})

	gotCausedBy := processUserTransactions(
		evmProcessor,
		inter.NewBlockBuilder(),
		transactions,
		math.MaxUint64,
		opera.Upgrades{Brio: true},
	)
	require.Equal(t, wantCausedBy, gotCausedBy)
}

func TestProcessUserTransactions_BlockSizeLimitIsEnforcedStartingFromBrio(t *testing.T) {
	tests := map[string]struct {
		upgrades          opera.Upgrades
		expectedSizeLimit uint64
	}{
		"before Brio, no size limit is enforced": {
			upgrades:          opera.Upgrades{Brio: false},
			expectedSizeLimit: math.MaxUint64,
		},
		"with Brio, block size limit is enforced": {
			upgrades:          opera.Upgrades{Brio: true},
			expectedSizeLimit: uint64(params.MaxBlockSize - rlpEncodedMaxHeaderSizeInBytes),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
			blockBuilder := inter.NewBlockBuilder()

			// Ensure the evmProcessor is called with the expected size limit
			evmProcessor.EXPECT().Execute(gomock.Any(), gomock.Any(), test.expectedSizeLimit)

			processUserTransactions(
				evmProcessor, blockBuilder, []*types.Transaction{}, math.MaxUint64, test.upgrades)
		})
	}
}

func TestSpillBlockEvents(t *testing.T) {

	makeEventPayload :=
		func(gasUsed uint64, sig inter.Signature) fakePayload {
			return fakePayload{gasUsed: gasUsed, signature: sig}
		}

	tests := map[string]struct {
		maxBlockGas uint64
		events      map[hash.Event]fakePayload
		// The test uses mocks for payloads, use the signatures to uniquely identify
		// events in the result.
		expectedSignatures []inter.Signature
	}{
		"empty input returns empty set": {
			expectedSignatures: []inter.Signature{},
		},
		"single event with gas usage below limit is included": {
			maxBlockGas: 10,
			events: map[hash.Event]fakePayload{
				{0x42}: makeEventPayload(5, inter.Signature{0x42}),
			},
			expectedSignatures: []inter.Signature{{0x42}},
		},
		"single event with gas usage exceeding limit is spilled": {
			maxBlockGas: 10,
			events: map[hash.Event]fakePayload{
				{0x42}: makeEventPayload(11, inter.Signature{0x42}),
			},
			expectedSignatures: []inter.Signature{},
		},
		"multiple events with gas usage below limit are included": {
			maxBlockGas: 30,
			events: map[hash.Event]fakePayload{
				{0x42}: makeEventPayload(10, inter.Signature{0x42}),
				{0x43}: makeEventPayload(10, inter.Signature{0x43}),
				{0x44}: makeEventPayload(10, inter.Signature{0x44}),
			},
			expectedSignatures: []inter.Signature{{0x42}, {0x43}, {0x44}},
		},
		"multiple events with last gas usage exceeding limit are spilled": {
			maxBlockGas: 20,
			events: map[hash.Event]fakePayload{
				{0x42}: makeEventPayload(1, inter.Signature{0x42}),
				{0x43}: makeEventPayload(1, inter.Signature{0x43}),
				{0x44}: makeEventPayload(21, inter.Signature{0x44}), // last event checked first
			},
			expectedSignatures: []inter.Signature{},
		},
		"multiple events are included until gas limit is reached, rest is spilled": {
			maxBlockGas: 20,
			events: map[hash.Event]fakePayload{
				{0x42}: makeEventPayload(1, inter.Signature{0x42}),
				{0x43}: makeEventPayload(10, inter.Signature{0x43}),
				{0x44}: makeEventPayload(10, inter.Signature{0x44}),
				{0x45}: makeEventPayload(10, inter.Signature{0x45}), // last event checked first
			},
			expectedSignatures: []inter.Signature{{0x44}, {0x45}},
		},
		"multiple events are included until gas limit is exceeded, rest is spilled even if they would fit independently": {
			maxBlockGas: 20,
			events: map[hash.Event]fakePayload{
				{0x42}: makeEventPayload(1, inter.Signature{0x42}),
				{0x43}: makeEventPayload(11, inter.Signature{0x43}),
				{0x44}: makeEventPayload(10, inter.Signature{0x44}),
			},
			expectedSignatures: []inter.Signature{{0x44}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			events := make([]hash.Event, 0, len(test.events))
			for e := range test.events {
				events = append(events, e)
			}
			// tests are order-dependent, so sort inputs
			slices.SortFunc(events, func(a, b hash.Event) int {
				return bytes.Compare(a[:], b[:])
			})

			getEventPayload := func(id hash.Event) inter.EventPayloadI {
				if payload, ok := test.events[id]; ok {
					return &payload
				}
				return nil
			}

			computed := spillBlockEvents(events, test.maxBlockGas, getEventPayload)
			foundSignatures := make([]inter.Signature, 0, len(computed))
			for _, event := range computed {
				foundSignatures = append(foundSignatures, event.Sig())
			}
			require.Equal(t, test.expectedSignatures, foundSignatures)
		})
	}
}

type fakePayload struct {
	inter.EventPayloadI // just here to satisfy the interface
	signature           inter.Signature
	gasUsed             uint64
}

func (p *fakePayload) Sig() inter.Signature {
	return p.signature
}
func (p *fakePayload) GasPowerUsed() uint64 {
	return p.gasUsed
}

func TestUpdateTransactionMetrics_CountsBundleAndSponsoredTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)

	bundleAddr := bundle.BundleProcessor
	envelopeTx := types.NewTx(&types.LegacyTx{To: &bundleAddr, Nonce: 1})
	sponsoredTx := types.NewTx(&types.LegacyTx{Nonce: 2, GasPrice: big.NewInt(0)})
	regularTx := types.NewTx(&types.LegacyTx{Nonce: 3, GasPrice: big.NewInt(1)})

	inputTransactions := []*types.Transaction{envelopeTx, sponsoredTx, regularTx}

	summary := evmcore.ProcessSummary{
		ProcessedTransactions: []evmcore.ProcessedTransaction{
			{Transaction: envelopeTx, Receipt: &types.Receipt{}},
			{Transaction: sponsoredTx, Receipt: &types.Receipt{}},
			{Transaction: regularTx, Receipt: &types.Receipt{}},
		},
	}

	upgrades := opera.Upgrades{Allegro: true, Brio: true, GasSubsidies: true, TransactionBundles: true}

	sponsoredCounter := utils.NewMockMetricsCounter(ctrl)
	skippedSponsoredCounter := utils.NewMockMetricsCounter(ctrl)
	bundleCounter := utils.NewMockMetricsCounter(ctrl)

	// Expect bundle and sponsored counters to be incremented
	bundleCounter.EXPECT().Inc(int64(1))
	sponsoredCounter.EXPECT().Inc(int64(1))
	// No skipped counters should be called since all have successful receipts

	updateTransactionMetrics(inputTransactions, summary, upgrades, sponsoredCounter, skippedSponsoredCounter, bundleCounter)
}

func TestUpdateTransactionMetrics_CountsSkippedSponsoredTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)

	sponsoredTx := types.NewTx(&types.LegacyTx{Nonce: 1, GasPrice: big.NewInt(0)})

	inputTransactions := []*types.Transaction{sponsoredTx}

	summary := evmcore.ProcessSummary{
		ProcessedTransactions: []evmcore.ProcessedTransaction{
			{Transaction: sponsoredTx, Receipt: nil}, // skipped
		},
	}

	upgrades := opera.Upgrades{Allegro: true, Brio: true, GasSubsidies: true, TransactionBundles: true}

	sponsoredCounter := utils.NewMockMetricsCounter(ctrl)
	skippedSponsoredCounter := utils.NewMockMetricsCounter(ctrl)
	bundleCounter := utils.NewMockMetricsCounter(ctrl)

	sponsoredCounter.EXPECT().Inc(int64(1))
	skippedSponsoredCounter.EXPECT().Inc(int64(1))

	updateTransactionMetrics(inputTransactions, summary, upgrades, sponsoredCounter, skippedSponsoredCounter, bundleCounter)
}

func TestUpdateTransactionMetrics_DoesNotCountWithoutUpgrades(t *testing.T) {
	ctrl := gomock.NewController(t)

	bundleAddr := bundle.BundleProcessor
	envelopeTx := types.NewTx(&types.LegacyTx{To: &bundleAddr, Nonce: 1})
	sponsoredTx := types.NewTx(&types.LegacyTx{Nonce: 2, GasPrice: big.NewInt(0)})

	inputTransactions := []*types.Transaction{envelopeTx, sponsoredTx}

	summary := evmcore.ProcessSummary{
		ProcessedTransactions: []evmcore.ProcessedTransaction{
			{Transaction: envelopeTx, Receipt: nil},
			{Transaction: sponsoredTx, Receipt: nil},
		},
	}

	// Without upgrades, no counters should be touched
	upgrades := opera.Upgrades{}

	sponsoredCounter := utils.NewMockMetricsCounter(ctrl)
	skippedSponsoredCounter := utils.NewMockMetricsCounter(ctrl)
	bundleCounter := utils.NewMockMetricsCounter(ctrl)

	// No expectations set — any call would fail the test

	updateTransactionMetrics(inputTransactions, summary, upgrades, sponsoredCounter, skippedSponsoredCounter, bundleCounter)
}

func TestUpdateBundleEfficiencyHistogram_ComputesEfficiencyForBundles(t *testing.T) {
	ctrl := gomock.NewController(t)
	histogram := utils.NewMockMetricsHistogramWrapper(ctrl)

	bundleAddr := bundle.BundleProcessor
	envelopeTx := types.NewTx(&types.LegacyTx{To: &bundleAddr, Nonce: 1})
	innerTx1 := types.NewTx(&types.LegacyTx{Nonce: 10})
	innerTx2 := types.NewTx(&types.LegacyTx{Nonce: 11})

	summary := evmcore.ProcessSummary{
		ProcessedTransactions: []evmcore.ProcessedTransaction{
			{Transaction: innerTx1, Receipt: &types.Receipt{GasUsed: 50_000}},
			{Transaction: innerTx2, Receipt: &types.Receipt{GasUsed: 30_000}},
		},
		ExecutionCost: map[common.Hash]core_types.ExecutionCost{
			envelopeTx.Hash(): 200_000,
		},
		CausedBy: map[common.Hash]common.Hash{
			innerTx1.Hash(): envelopeTx.Hash(),
			innerTx2.Hash(): envelopeTx.Hash(),
		},
	}

	// efficiency = (50_000 + 30_000) / 200_000 = 0.4
	histogram.EXPECT().Update(0.4)

	updateBundleEfficiencyHistogram(summary, []common.Hash{envelopeTx.Hash()}, histogram)
}

func TestUpdateBundleEfficiencyHistogram_HandlesMultipleBundles(t *testing.T) {
	ctrl := gomock.NewController(t)
	histogram := utils.NewMockMetricsHistogramWrapper(ctrl)

	bundleAddr := bundle.BundleProcessor
	envelope1 := types.NewTx(&types.LegacyTx{To: &bundleAddr, Nonce: 1})
	envelope2 := types.NewTx(&types.LegacyTx{To: &bundleAddr, Nonce: 2})
	innerTx1 := types.NewTx(&types.LegacyTx{Nonce: 10})
	innerTx2 := types.NewTx(&types.LegacyTx{Nonce: 20})

	summary := evmcore.ProcessSummary{
		ProcessedTransactions: []evmcore.ProcessedTransaction{
			{Transaction: innerTx1, Receipt: &types.Receipt{GasUsed: 80_000}},
			{Transaction: innerTx2, Receipt: &types.Receipt{GasUsed: 20_000}},
		},
		ExecutionCost: map[common.Hash]core_types.ExecutionCost{
			envelope1.Hash(): 100_000,
			envelope2.Hash(): 100_000,
		},
		CausedBy: map[common.Hash]common.Hash{
			innerTx1.Hash(): envelope1.Hash(),
			innerTx2.Hash(): envelope2.Hash(),
		},
	}

	// Bundle 1: 80_000 / 100_000 = 0.8
	// Bundle 2: 20_000 / 100_000 = 0.2
	histogram.EXPECT().Update(0.8)
	histogram.EXPECT().Update(0.2)

	updateBundleEfficiencyHistogram(summary, []common.Hash{envelope1.Hash(), envelope2.Hash()}, histogram)
}

func TestUpdateBundleEfficiencyHistogram_SkippedInnerTransactionsContributeOnlyToExecutionCost(t *testing.T) {
	ctrl := gomock.NewController(t)
	histogram := utils.NewMockMetricsHistogramWrapper(ctrl)

	bundleAddr := bundle.BundleProcessor
	envelopeTx := types.NewTx(&types.LegacyTx{To: &bundleAddr, Nonce: 1})
	innerTx1 := types.NewTx(&types.LegacyTx{Nonce: 10})
	innerTx2 := types.NewTx(&types.LegacyTx{Nonce: 11})

	summary := evmcore.ProcessSummary{
		ProcessedTransactions: []evmcore.ProcessedTransaction{
			{Transaction: innerTx1, Receipt: &types.Receipt{GasUsed: 50_000}},
			{Transaction: innerTx2, Receipt: nil}, // skipped, no gas used but has exec cost
		},
		ExecutionCost: map[common.Hash]core_types.ExecutionCost{
			envelopeTx.Hash(): 200_000,
		},
		CausedBy: map[common.Hash]common.Hash{
			innerTx1.Hash(): envelopeTx.Hash(),
			innerTx2.Hash(): envelopeTx.Hash(),
		},
	}

	// efficiency = 50_000 / 200_000 = 0.25
	histogram.EXPECT().Update(0.25)

	updateBundleEfficiencyHistogram(summary, []common.Hash{envelopeTx.Hash()}, histogram)
}
