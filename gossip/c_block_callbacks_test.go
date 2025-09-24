// Copyright 2025 Sonic Operations Ltd
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
	"math/big"
	"slices"
	"sync"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/lachesis"
	"github.com/Fantom-foundation/lachesis-base/utils/workers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc"
	"github.com/0xsoniclabs/sonic/gossip/emitter"
	"github.com/0xsoniclabs/sonic/gossip/randao"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
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
				evmProcessor.EXPECT().Execute(_any, _any).Return(types.Receipts{}).MinTimes(1)
				evmProcessor.EXPECT().Finalize().Return(&evmcore.EvmBlock{
					EvmHeader: evmcore.EvmHeader{
						BaseFee: big.NewInt(0),
						TxHash:  common.Hash{1, 2, 3},
					},
				}, nil, nil)

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

	require.NoError(isPermissible(valid, &withAllegro))
	require.Error(isPermissible(invalid, &withAllegro))

	txs := []*types.Transaction{valid, invalid}

	require.Equal(txs, filterNonPermissibleTransactions(txs, &withoutAllegro, nil, nil))
	require.Equal([]*types.Transaction{valid}, filterNonPermissibleTransactions(txs, &withAllegro, nil, nil))
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
	require.Equal(t, want, filterNonPermissibleTransactions(txs, &rules, nil, nil))
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
		"issue", isPermissible(invalid1, &rules),
	)

	log.EXPECT().Warn(
		"Non-permissible transaction in the proposal",
		"tx", gomock.Any(),
		"issue", isPermissible(invalid2, &rules),
	)

	filterNonPermissibleTransactions(
		[]*types.Transaction{invalid1, invalid2},
		&rules,
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
		counter,
	)
}

func TestIsPermissible_AcceptsPermissibleTransactions(t *testing.T) {
	tests := map[string]*types.Transaction{
		"legacy":      types.NewTx(&types.LegacyTx{}),
		"access list": types.NewTx(&types.AccessListTx{}),
		"dynamic fee": types.NewTx(&types.DynamicFeeTx{}),
		"blob":        types.NewTx(&types.BlobTx{}),
		"set code": types.NewTx(&types.SetCodeTx{
			AuthList: []types.SetCodeAuthorization{{}},
		}),
	}

	rules := opera.Rules{
		Upgrades: opera.Upgrades{
			Allegro: true,
		},
	}
	for name, tx := range tests {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, isPermissible(tx, &rules))
		})
	}
}

func TestIsPermissible_AcceptsSetCodeTransactionsOnlyInAllegro(t *testing.T) {
	tx := types.NewTx(&types.SetCodeTx{
		AuthList: []types.SetCodeAuthorization{{}},
	})

	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("allegro=%t", enabled), func(t *testing.T) {
			rules := opera.Rules{
				Upgrades: opera.Upgrades{
					Allegro: enabled,
				},
			}
			if enabled {
				require.NoError(t, isPermissible(tx, &rules))
			} else {
				require.ErrorContains(t,
					isPermissible(tx, &rules),
					"unsupported transaction type",
				)
			}
		})
	}
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
			err := isPermissible(test.transaction, &rules)
			require.ErrorContains(t, err, test.issue)
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
