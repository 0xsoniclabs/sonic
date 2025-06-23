package emitter

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/eventcheck/proposalcheck"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/emitter/scheduler"
	"github.com/0xsoniclabs/sonic/gossip/gasprice"
	"github.com/0xsoniclabs/sonic/gossip/randao"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestEmitter_MaxProposalSize_IsWithinLimitCheckedByEventValidator(t *testing.T) {
	require.LessOrEqual(t,
		maxTotalTransactionsSizeInEventInBytes,
		proposalcheck.MaxSizeOfProposedTransactions,
	)
}

func TestEmitter_CreatePayload_ProducesValidPayload(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	world := NewMockExternal(ctrl)
	event := inter.NewMockEventI(ctrl)

	event.EXPECT().Parents().Return(hash.Events{})
	event.EXPECT().Epoch().Return(idx.Epoch(12)).AnyTimes()
	event.EXPECT().Frame().Return(idx.Frame(0))

	world.EXPECT().GetLatestBlock().Return(
		inter.NewBlockBuilder().WithNumber(61).Build(),
	)

	builder := pos.ValidatorsBuilder{}
	builder.Set(idx.ValidatorID(123), 10) // => different validator
	validators := builder.Build()

	emitter := &Emitter{
		world: World{External: world},
	}
	emitter.validators.Store(validators)

	// It is not this emitter's turn to propose a block, so the payload just
	// contains the proposal sync state but no proposal.
	payload, err := emitter.createPayload(event, nil)
	require.NoError(err)
	want := inter.Payload{
		ProposalSyncState: inter.ProposalSyncState{
			LastSeenProposalTurn:  inter.Turn(0),
			LastSeenProposalFrame: idx.Frame(0),
		},
	}
	require.Equal(want, payload)
}

func TestEmitter_CreatePayload_FailsOnInvalidValidators(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	world := NewMockExternal(ctrl)
	event := inter.NewMockEventI(ctrl)

	event.EXPECT().Parents().Return(hash.Events{})
	event.EXPECT().Epoch().Return(idx.Epoch(12)).AnyTimes()
	event.EXPECT().Frame().Return(idx.Frame(0))

	world.EXPECT().GetLatestBlock().Return(
		inter.NewBlockBuilder().WithNumber(62).Build(),
	)

	validators := pos.ValidatorsBuilder{}.Build() // no validators

	emitter := &Emitter{
		world: World{External: world},
	}
	emitter.validators.Store(validators)

	_, err := emitter.createPayload(event, nil)
	require.ErrorContains(err, "no validators")
}

func TestWorldAdapter_GetEventPayload_ForwardsCallToGetExternalEventPayload(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	world := NewMockExternal(ctrl)

	payload := inter.Payload{
		ProposalSyncState: inter.ProposalSyncState{
			LastSeenProposalTurn:  inter.Turn(1),
			LastSeenProposalFrame: idx.Frame(2),
		},
	}

	builder := &inter.MutableEventPayload{}
	builder.SetPayload(payload)
	eventPayload := builder.Build()

	event := hash.Event{1}
	world.EXPECT().GetEventPayload(event).Return(eventPayload)

	adapter := worldAdapter{world}
	got := adapter.GetEventPayload(event)
	require.Equal(payload, got)
}

func TestWorldAdapter_GetEvmChainConfig_ForwardsCallToGetRulesAndGetUpgradeHeights(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	world := NewMockExternal(ctrl)

	rules := opera.Rules{}
	var updateHeights []opera.UpgradeHeight

	world.EXPECT().GetRules().Return(rules)
	world.EXPECT().GetUpgradeHeights().Return(updateHeights)

	adapter := worldAdapter{world}
	got := adapter.GetEvmChainConfig(idx.Block(1))
	want := opera.CreateTransientEvmChainConfig(rules.NetworkID, updateHeights, 1)
	require.Equal(want, got)
}

func TestCreatePayload_PendingProposal_CreatesPayloadWithoutProposal(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	world := NewMockworldReader(ctrl)
	event := inter.NewMockEventI(ctrl)

	world.EXPECT().GetLatestBlock().Return(
		inter.NewBlockBuilder().WithNumber(4).Build(),
	)

	event.EXPECT().Parents().Return(hash.Events{})
	event.EXPECT().Frame().Return(idx.Frame(2))

	proposalTracker := NewMockproposalTracker(ctrl)
	proposalTracker.EXPECT().IsPending(idx.Frame(2), idx.Block(5)).Return(true)

	// This call fails since it tries to propose block 5 while according to the
	// proposal tracker, a proposal for block 5 has already been made.
	payload, err := createPayload(
		world, 0, nil, event, proposalTracker, nil, nil, nil, nil, nil,
	)

	want := inter.Payload{
		ProposalSyncState: inter.ProposalSyncState{},
	}

	require.NoError(err)
	require.Equal(want, payload)
}

func TestCreatePayload_UnableToCreateProposalDueToLackOfTimeProgress_CreatesPayloadWithoutProposal(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	world := NewMockworldReader(ctrl)
	event := inter.NewMockEventI(ctrl)

	p1 := hash.Event{1}
	p2 := hash.Event{2}
	payloads := map[hash.Event]inter.Payload{
		p1: {ProposalSyncState: inter.ProposalSyncState{
			LastSeenProposalTurn:  inter.Turn(0x01),
			LastSeenProposalFrame: idx.Frame(0x12),
		}},
		p2: {ProposalSyncState: inter.ProposalSyncState{
			LastSeenProposalTurn:  inter.Turn(0x03),
			LastSeenProposalFrame: idx.Frame(0x11),
		}},
	}

	world.EXPECT().GetEventPayload(p1).Return(payloads[p1])
	world.EXPECT().GetEventPayload(p2).Return(payloads[p2])

	lastBlockTime := inter.Timestamp(1234)
	world.EXPECT().GetLatestBlock().Return(
		inter.NewBlockBuilder().
			WithNumber(0x23).
			WithTime(lastBlockTime).
			Build(),
	)
	world.EXPECT().GetRules().Return(opera.Rules{})

	event.EXPECT().Parents().Return(hash.Events{p1, p2})
	event.EXPECT().Epoch().Return(idx.Epoch(0x12))
	event.EXPECT().Frame().Return(idx.Frame(0x14))
	event.EXPECT().MedianTime().Return(lastBlockTime)

	validator := idx.ValidatorID(1)
	builder := pos.ValidatorsBuilder{}
	builder.Set(validator, 10)
	validators := builder.Build()

	tracker := NewMockproposalTracker(ctrl)
	tracker.EXPECT().IsPending(idx.Frame(0x14), idx.Block(0x24)).Return(false)

	// This attempt to create a proposal should result in an empty payload since
	// no time has passed since the last proposal.
	payload, err := createPayload(
		world, validator, validators, event, tracker, nil, nil, nil, nil, nil,
	)

	want := inter.Payload{
		ProposalSyncState: inter.ProposalSyncState{
			LastSeenProposalTurn:  inter.Turn(0x03),
			LastSeenProposalFrame: idx.Frame(0x12),
		},
	}

	require.NoError(err)
	require.Equal(want, payload)
}

func TestCreatePayload_InvalidValidators_ForwardsError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	event := inter.NewMockEventI(ctrl)
	event.EXPECT().Parents().Return(hash.Events{})
	event.EXPECT().Epoch().Return(idx.Epoch(12)).AnyTimes()
	event.EXPECT().Frame().Return(idx.Frame(0))

	world := NewMockworldReader(ctrl)
	world.EXPECT().GetLatestBlock().Return(
		inter.NewBlockBuilder().WithNumber(62).Build(),
	)

	validators := pos.ValidatorsBuilder{}.Build()
	tracker := NewMockproposalTracker(ctrl)
	tracker.EXPECT().IsPending(idx.Frame(0), idx.Block(63)).Return(false)

	_, err := createPayload(
		world, 0, validators, event, tracker, nil, nil, nil, nil, nil,
	)
	require.ErrorContains(err, "no validators")
}

func TestCreatePayload_ValidTurn_ProducesExpectedPayload(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	world := NewMockworldReader(ctrl)
	event := inter.NewMockEventI(ctrl)
	durationMetric := NewMocktimerMetric(ctrl)
	timeoutMetric := NewMockcounterMetric(ctrl)

	p1 := hash.Event{1}
	p2 := hash.Event{2}
	payloads := map[hash.Event]inter.Payload{
		p1: {ProposalSyncState: inter.ProposalSyncState{
			LastSeenProposalTurn:  inter.Turn(1),
			LastSeenProposalFrame: idx.Frame(2),
		}},
		p2: {ProposalSyncState: inter.ProposalSyncState{
			LastSeenProposalTurn:  inter.Turn(1),
			LastSeenProposalFrame: idx.Frame(2),
		}},
	}

	world.EXPECT().GetEventPayload(p1).Return(payloads[p1])
	world.EXPECT().GetEventPayload(p2).Return(payloads[p2])

	world.EXPECT().GetLatestBlock().Return(
		inter.NewBlockBuilder().
			WithNumber(5).
			WithBaseFee(big.NewInt(100)).
			WithDuration(500 * time.Millisecond).
			Build(),
	)

	world.EXPECT().GetRules().Return(opera.Rules{})

	event.EXPECT().Parents().Return(hash.Events{p1, p2})
	event.EXPECT().Epoch().Return(idx.Epoch(3)).AnyTimes()
	event.EXPECT().Frame().Return(idx.Frame(4)).AnyTimes()
	event.EXPECT().MedianTime().Return(inter.Timestamp(1234))

	tracker := NewMockproposalTracker(ctrl)
	tracker.EXPECT().IsPending(idx.Frame(4), idx.Block(6)).Return(false)

	validator := idx.ValidatorID(1)
	builder := pos.ValidatorsBuilder{}
	builder.Set(validator, 10)
	validators := builder.Build()

	txs := []*types.Transaction{
		types.NewTx(&types.LegacyTx{Nonce: 1}),
		types.NewTx(&types.LegacyTx{Nonce: 2}),
	}

	any := gomock.Any()
	scheduler := NewMocktxScheduler(ctrl)
	scheduler.EXPECT().Schedule(any, any, any, any).Return(txs)

	durationMetric.EXPECT().Update(any).AnyTimes()
	timeoutMetric.EXPECT().Inc(any).AnyTimes()
	randaoMixer := randao.NewMockRandaoMixer(ctrl)
	someRandaoReveal := randao.RandaoReveal{0x42}
	randaoMixer.EXPECT().MixRandao(any).Return(
		someRandaoReveal, common.Hash{}, nil,
	)

	payload, err := createPayload(
		world, validator, validators, event, tracker, nil,
		scheduler, randaoMixer, durationMetric, timeoutMetric,
	)
	require.NoError(err)

	require.Equal(inter.Turn(2), payload.LastSeenProposalTurn)
	require.Equal(idx.Frame(4), payload.LastSeenProposalFrame)
	require.Equal(idx.Block(6), payload.Proposal.Number)
	require.Equal(txs, payload.Proposal.Transactions)
	require.Equal(someRandaoReveal, payload.Proposal.RandaoReveal)
}

func TestMakeProposal_ValidArguments_CreatesValidProposal(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	mockScheduler := NewMocktxScheduler(ctrl)
	durationMetric := NewMocktimerMetric(ctrl)
	timeoutMetric := NewMockcounterMetric(ctrl)

	rules := opera.Rules{}
	state := inter.ProposalSyncState{
		LastSeenProposalTurn:  inter.Turn(5),
		LastSeenProposalFrame: idx.Frame(12),
	}
	latestBlock := inter.NewBlockBuilder().
		WithNumber(5).
		WithTime(1234).
		WithBaseFee(big.NewInt(100)).
		WithDuration(500 * time.Millisecond).
		Build()

	delta := 20 * time.Millisecond
	newBlockTime := inter.Timestamp(1234) + inter.Timestamp(delta)
	currentFrame := idx.Frame(17)

	transactions := []*types.Transaction{
		types.NewTx(&types.LegacyTx{Nonce: 1}),
		types.NewTx(&types.LegacyTx{Nonce: 2}),
	}

	// Check that parameters are correctly forwarded to the scheduler.
	any := gomock.Any()
	someRandaoReveal := randao.RandaoReveal{0x42}
	someRandao := common.Hash{0x43}
	mockScheduler.EXPECT().Schedule(
		any,
		&scheduler.BlockInfo{
			Number:      idx.Block(latestBlock.Number) + 1,
			Time:        newBlockTime,
			GasLimit:    rules.Blocks.MaxBlockGas,
			MixHash:     someRandao,
			BaseFee:     *uint256.NewInt(100),
			BlobBaseFee: *uint256.NewInt(0),
		},
		nil,
		scheduler.Limits{
			Gas:  inter.GetEffectiveGasLimit(delta, rules.Economy.ShortGasPower.AllocPerSec),
			Size: maxTotalTransactionsSizeInEventInBytes,
		},
	).Return(transactions)

	// Scheduling time should be monitored.
	durationMetric.EXPECT().Update(any).Do(func(duration time.Duration) {
		require.True(duration > 0)
	})

	randaoMixer := randao.NewMockRandaoMixer(ctrl)
	randaoMixer.EXPECT().MixRandao(any).Return(someRandaoReveal, someRandao, nil)

	// Run the proposal creation.
	proposal, err := makeProposal(
		rules,
		state,
		latestBlock,
		newBlockTime,
		currentFrame,
		mockScheduler,
		nil,
		randaoMixer,
		durationMetric,
		timeoutMetric,
	)
	require.NoError(err)

	require.Equal(idx.Block(latestBlock.Number)+1, proposal.Number)
	require.Equal(latestBlock.Hash(), proposal.ParentHash)
	require.Equal(transactions, proposal.Transactions)
	require.Equal(someRandaoReveal, proposal.RandaoReveal)
}

func TestMakeProposal_InvalidBlockTime_ReturnsNil(t *testing.T) {
	state := inter.ProposalSyncState{
		LastSeenProposalTurn:  inter.Turn(5),
		LastSeenProposalFrame: idx.Frame(12),
	}
	latestBlock := inter.NewBlockBuilder().WithTime(1234).Build()
	for _, delta := range []time.Duration{-1 * time.Nanosecond, 0} {
		newTime := inter.Timestamp(1234) + inter.Timestamp(delta)
		payload, err := makeProposal(
			opera.Rules{}, state, latestBlock, newTime, 0, nil, nil, nil, nil, nil,
		)
		require.NoError(t, err, "not error but no-proposal expected")
		require.Nil(t, payload)
	}
}

func TestMakeProposal_IfSchedulerTimesOut_SignalTimeoutToMonitor(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockScheduler := NewMocktxScheduler(ctrl)
	durationMetric := NewMocktimerMetric(ctrl)
	timeoutMetric := NewMockcounterMetric(ctrl)

	any := gomock.Any()
	mockScheduler.EXPECT().Schedule(any, any, any, any).Do(
		func(
			ctx context.Context, _ *scheduler.BlockInfo,
			_ scheduler.PrioritizedTransactions, _ scheduler.Limits,
		) {
			deadline, ok := ctx.Deadline()
			require.True(t, ok, "scheduler call should have a deadline")
			for {
				delay := time.Until(deadline)
				if delay > 0 {
					<-time.After(delay)
				}
				if err := ctx.Err(); err != nil {
					require.ErrorIs(t, err, context.DeadlineExceeded)
					break
				}
			}
		})

	durationMetric.EXPECT().Update(any)
	timeoutMetric.EXPECT().Inc(int64(1))

	randaoMixer := randao.NewMockRandaoMixer(ctrl)
	randaoMixer.EXPECT().MixRandao(any)

	_, err := makeProposal(
		opera.Rules{},
		inter.ProposalSyncState{},
		inter.NewBlockBuilder().
			WithBaseFee(big.NewInt(100)).
			WithDuration(500*time.Millisecond).
			Build(),
		inter.Timestamp(1),
		0,
		mockScheduler,
		nil,
		randaoMixer,
		durationMetric,
		timeoutMetric,
	)
	require.NoError(t, err)
}

func TestTransactionPriorityAdapter_ForwardsCallToWrappedType(t *testing.T) {

	t.Run("Current", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		index := NewMocktransactionIndex(ctrl)

		tx := types.NewTx(&types.LegacyTx{Nonce: 1})
		index.EXPECT().Peek().Return(&txpool.LazyTransaction{Tx: tx}, nil)

		adapter := transactionPriorityAdapter{index}
		got := adapter.Current()
		require.Equal(t, tx, got)
	})

	t.Run("Current_Empty", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		index := NewMocktransactionIndex(ctrl)
		index.EXPECT().Peek().Return(nil, nil)
		adapter := transactionPriorityAdapter{index}
		got := adapter.Current()
		require.Nil(t, got)
	})

	t.Run("Accept", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		index := NewMocktransactionIndex(ctrl)
		index.EXPECT().Shift()
		adapter := transactionPriorityAdapter{index}
		adapter.Accept()
	})

	t.Run("Skip", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		index := NewMocktransactionIndex(ctrl)
		index.EXPECT().Pop()
		adapter := transactionPriorityAdapter{index}
		adapter.Skip()
	})
}

func TestMakeProposal_SkipsProposalOnRandaoRevealError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	rules := opera.Rules{}
	state := inter.ProposalSyncState{
		LastSeenProposalTurn:  inter.Turn(5),
		LastSeenProposalFrame: idx.Frame(12),
	}
	latestBlock := inter.NewBlockBuilder().
		WithNumber(5).
		WithTime(1234).
		Build()

	newBlockTime := latestBlock.Time + 10
	currentFrame := idx.Frame(17)

	randaoMixer := randao.NewMockRandaoMixer(ctrl)
	randaoMixer.EXPECT().MixRandao(gomock.Any()).Return(
		randao.RandaoReveal{}, common.Hash{}, errors.New("randao error"))

	// Run the proposal creation.
	_, err := makeProposal(
		rules,
		state,
		latestBlock,
		newBlockTime,
		currentFrame,
		nil,
		nil,
		randaoMixer,
		nil,
		nil,
	)
	require.ErrorContains(err, "randao reveal generation failed")
}

func TestMakeProposal_SkipsProposalIfBaseFeeIsGettingTooHeigh(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	targetRate := uint64(50_000_000) // 50M gas/sec

	rules := opera.Rules{
		Economy: opera.EconomyRules{
			ShortGasPower: opera.GasPowerRules{
				AllocPerSec: targetRate,
			},
		},
	}

	previousBaseFee := new(big.Int).Lsh(big.NewInt(1), 256)
	latestBlock := inter.NewBlockBuilder().
		WithBaseFee(previousBaseFee).
		WithGasLimit(2 * targetRate).
		WithGasUsed(2 * targetRate).
		WithDuration(500 * time.Millisecond).
		Build()

	newBlockTime := latestBlock.Time + 10

	randaoMixer := randao.NewMockRandaoMixer(ctrl)
	randaoMixer.EXPECT().MixRandao(gomock.Any())

	// Run the proposal creation.
	_, err := makeProposal(
		rules,
		inter.ProposalSyncState{},
		latestBlock,
		newBlockTime,
		0,
		nil,
		nil,
		randaoMixer,
		nil,
		nil,
	)
	require.ErrorContains(err, "overflows uint256")
}

func TestMakeProposal_SchedulerIsRunWithCorrectBaseFee(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	targetRate := uint64(50_000_000) // 50M gas/sec

	rules := opera.Rules{
		Economy: opera.EconomyRules{
			ShortGasPower: opera.GasPowerRules{
				AllocPerSec: targetRate,
			},
		},
	}

	previousBaseFee := big.NewInt(1000)
	latestBlock := inter.NewBlockBuilder().
		WithBaseFee(previousBaseFee).
		WithGasUsed(2 * targetRate). // high gas usage, should increase base fee
		WithDuration(500 * time.Millisecond).
		Build()

	expectedBaseFee := gasprice.GetBaseFeeForNextBlock(gasprice.ParentBlockInfo{
		BaseFee:  latestBlock.BaseFee,
		Duration: time.Duration(latestBlock.Duration),
		GasUsed:  latestBlock.GasUsed,
	}, rules.Economy)
	require.True(expectedBaseFee.Cmp(previousBaseFee) > 0, "expected base fee to increase")

	randaoMix := common.Hash{0x42}
	randaoMixer := randao.NewMockRandaoMixer(ctrl)
	randaoMixer.EXPECT().MixRandao(gomock.Any()).Return(
		randao.RandaoReveal{0x43}, randaoMix, nil,
	)

	newBlockTime := latestBlock.Time + 10

	txScheduler := NewMocktxScheduler(ctrl)
	txScheduler.EXPECT().Schedule(
		gomock.Any(),
		&scheduler.BlockInfo{
			Number:      idx.Block(latestBlock.Number + 1),
			Time:        newBlockTime,
			GasLimit:    rules.Blocks.MaxBlockGas,
			MixHash:     randaoMix,
			Coinbase:    evmcore.GetCoinbase(),
			BaseFee:     *uint256.MustFromBig(expectedBaseFee),
			BlobBaseFee: evmcore.GetBlobBaseFee(),
		},
		gomock.Any(),
		gomock.Any(),
	)

	durationMetric := NewMocktimerMetric(ctrl)
	durationMetric.EXPECT().Update(gomock.Any()).AnyTimes()
	timeoutMetric := NewMockcounterMetric(ctrl)
	timeoutMetric.EXPECT().Inc(gomock.Any()).AnyTimes()

	// Run the proposal creation.
	_, err := makeProposal(
		rules,
		inter.ProposalSyncState{},
		latestBlock,
		newBlockTime,
		0,
		txScheduler,
		nil,
		randaoMixer,
		durationMetric,
		timeoutMetric,
	)
	require.NoError(err)
}

func TestCreatePayload_ReturnsErrorOnRandaoGenerationFailure(t *testing.T) {

	require := require.New(t)
	ctrl := gomock.NewController(t)
	world := NewMockworldReader(ctrl)

	world.EXPECT().GetLatestBlock().Return(
		inter.NewBlockBuilder().WithNumber(4).Build(), // next expected block number is 5
	)
	world.EXPECT().GetRules().Return(opera.Rules{})

	event := inter.NewMockEventI(ctrl)
	event.EXPECT().Parents().Return(hash.Events{})
	event.EXPECT().Epoch().Return(idx.Epoch(1))
	event.EXPECT().Frame().Return(idx.Frame(2)) // tracker should expect frame 2
	event.EXPECT().MedianTime().Return(inter.Timestamp(1234))

	validator := idx.ValidatorID(1)
	builder := pos.ValidatorsBuilder{}
	builder.Set(validator, 10)
	validators := builder.Build()

	tracker := NewMockproposalTracker(ctrl)
	tracker.EXPECT().IsPending(idx.Frame(2), idx.Block(5)).Return(false)

	randaoMixer := randao.NewMockRandaoMixer(ctrl)
	randaoMixer.EXPECT().MixRandao(gomock.Any()).Return(
		randao.RandaoReveal{}, common.Hash{}, errors.New("randao error"),
	)

	_, err := createPayload(world, validator, validators, event, tracker, nil, nil, randaoMixer, nil, nil)
	require.ErrorContains(err, "randao reveal generation failed")
}
