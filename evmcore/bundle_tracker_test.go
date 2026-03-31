package evmcore

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestBundleTracker_BundleHasNotBeenRetired(t *testing.T) {

	ctrl := gomock.NewController(t)
	stateReader := NewMockStateReader(ctrl)

	tracker := NewBundleTracker(stateReader)
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	envelopeTx, plan := bundle.NewBuilder().With(
		bundle.Step(key, types.AccessListTx{
			Value: big.NewInt(1),
		}),
	).BuildEnvelopeAndPlan()

	tracker.TrackTransaction(envelopeTx)

	stateReader.EXPECT().CurrentBlock().Return(&EvmBlock{
		EvmHeader: EvmHeader{
			Number: big.NewInt(100),
		},
	})
	require.True(t, tracker.IsBundlePending(plan.Hash()))
}

func TestBundleTracker_BundleHasBeenRetired_ButMayStillBeExecuted(t *testing.T) {

	ctrl := gomock.NewController(t)
	stateReader := NewMockStateReader(ctrl)

	tracker := NewBundleTracker(stateReader)
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	envelopeTx, plan := bundle.NewBuilder().With(
		bundle.Step(key, types.AccessListTx{
			Value: big.NewInt(1),
		}),
	).BuildEnvelopeAndPlan()

	tracker.TrackTransaction(envelopeTx)

	stateReader.EXPECT().CurrentBlock().Return(&EvmBlock{
		EvmHeader: EvmHeader{
			Number: big.NewInt(100),
		},
	})
	tracker.SunsetTransaction(envelopeTx)

	stateReader.EXPECT().CurrentBlock().Return(&EvmBlock{
		EvmHeader: EvmHeader{
			Number: big.NewInt(101),
		},
	})
	stateReader.EXPECT().HasBundleBeenProcessed(plan.Hash()).Return(false)
	require.True(t, tracker.IsBundlePending(plan.Hash()))
}

func TestBundleTracker_BundleHasBeenRetired_ButExecutedAfterSunset(t *testing.T) {

	ctrl := gomock.NewController(t)
	stateReader := NewMockStateReader(ctrl)

	tracker := NewBundleTracker(stateReader)
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	envelopeTx, plan := bundle.NewBuilder().With(
		bundle.Step(key, types.AccessListTx{
			Value: big.NewInt(1),
		}),
	).BuildEnvelopeAndPlan()

	tracker.TrackTransaction(envelopeTx)

	stateReader.EXPECT().CurrentBlock().Return(&EvmBlock{
		EvmHeader: EvmHeader{
			Number: big.NewInt(100),
		},
	})
	tracker.SunsetTransaction(envelopeTx)

	stateReader.EXPECT().CurrentBlock().Return(&EvmBlock{
		EvmHeader: EvmHeader{
			Number: big.NewInt(101),
		},
	})
	stateReader.EXPECT().HasBundleBeenProcessed(plan.Hash()).Return(true)
	require.False(t, tracker.IsBundlePending(plan.Hash()))
}

func TestBundleTracker_BundleHasBeenRetired_LongEnoughToNotBeExecuted(t *testing.T) {

	ctrl := gomock.NewController(t)
	stateReader := NewMockStateReader(ctrl)

	tracker := NewBundleTracker(stateReader)
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	envelopeTx, plan := bundle.NewBuilder().With(
		bundle.Step(key, types.AccessListTx{
			Value: big.NewInt(1),
		}),
	).BuildEnvelopeAndPlan()

	tracker.TrackTransaction(envelopeTx)

	stateReader.EXPECT().CurrentBlock().Return(&EvmBlock{
		EvmHeader: EvmHeader{
			Number: big.NewInt(100),
		},
	})
	tracker.SunsetTransaction(envelopeTx)

	stateReader.EXPECT().CurrentBlock().Return(&EvmBlock{
		EvmHeader: EvmHeader{
			Number: big.NewInt(104),
		},
	})
	require.False(t, tracker.IsBundlePending(plan.Hash()))
}
