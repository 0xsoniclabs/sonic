package emitter

import (
	"math/big"
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils/txtime"
	"github.com/0xsoniclabs/sonic/vecmt"
)

//go:generate mockgen -package=mock -destination=mock/world.go github.com/0xsoniclabs/sonic/gossip/emitter External,TxPool,TxSigner,Signer

func TestEmitter(t *testing.T) {
	cfg := DefaultConfig()
	gValidators := makefakegenesis.GetFakeValidators(3)
	vv := pos.NewBuilder()
	for _, v := range gValidators {
		vv.Set(v.ID, pos.Weight(1))
	}
	validators := vv.Build()
	cfg.Validator.ID = gValidators[0].ID

	ctrl := gomock.NewController(t)
	external := NewMockExternal(ctrl)
	txPool := NewMockTxPool(ctrl)
	signer := NewMockSigner(ctrl)
	txSigner := NewMockTxSigner(ctrl)

	external.EXPECT().Lock().
		AnyTimes()
	external.EXPECT().Unlock().
		AnyTimes()
	external.EXPECT().DagIndex().
		Return((*vecmt.Index)(nil)).
		AnyTimes()
	external.EXPECT().IsSynced().
		Return(true).
		AnyTimes()
	external.EXPECT().PeersNum().
		Return(int(3)).
		AnyTimes()
	external.EXPECT().StateDB().
		Return(nil).
		AnyTimes()

	em := NewEmitter(cfg, World{
		External: external,
		TxPool:   txPool,
		Signer:   signer,
		TxSigner: txSigner,
	}, fixedPriceBaseFeeSource{})

	t.Run("init", func(t *testing.T) {
		external.EXPECT().GetRules().
			Return(opera.FakeNetRules(opera.GetSonicUpgrades())).
			AnyTimes()

		external.EXPECT().GetEpochValidators().
			Return(validators, idx.Epoch(1)).
			AnyTimes()

		external.EXPECT().GetLastEvent(idx.Epoch(1), cfg.Validator.ID).
			Return((*hash.Event)(nil)).
			AnyTimes()

		external.EXPECT().GetGenesisTime().
			Return(inter.Timestamp(uint64(time.Now().UnixNano()))).
			AnyTimes()

		em.init()
	})

	t.Run("memorizeTxTimes", func(t *testing.T) {
		txtime.Enabled = true
		require := require.New(t)
		tx1 := types.NewTransaction(1, common.Address{}, big.NewInt(1), 1, big.NewInt(1), nil)
		tx2 := types.NewTransaction(2, common.Address{}, big.NewInt(2), 2, big.NewInt(2), nil)

		external.EXPECT().IsBusy().
			Return(true).
			AnyTimes()

		txtime.Saw(tx1.Hash(), time.Unix(1, 0))

		require.Equal(time.Unix(1, 0), txtime.Of(tx1.Hash()))
		txtime.Saw(tx1.Hash(), time.Unix(2, 0))
		require.Equal(time.Unix(1, 0), txtime.Of(tx1.Hash()))
		txtime.Validated(tx1.Hash(), time.Unix(2, 0))
		require.Equal(time.Unix(1, 0), txtime.Of(tx1.Hash()))

		// reversed order
		txtime.Validated(tx2.Hash(), time.Unix(3, 0))
		txtime.Saw(tx2.Hash(), time.Unix(2, 0))

		require.Equal(time.Unix(3, 0), txtime.Of(tx2.Hash()))
		txtime.Saw(tx2.Hash(), time.Unix(3, 0))
		require.Equal(time.Unix(3, 0), txtime.Of(tx2.Hash()))
		txtime.Validated(tx2.Hash(), time.Unix(3, 0))
		require.Equal(time.Unix(3, 0), txtime.Of(tx2.Hash()))
	})

	t.Run("tick", func(t *testing.T) {
		em.tick()
	})
}

type fixedPriceBaseFeeSource struct{}

func (fixedPriceBaseFeeSource) GetCurrentBaseFee() *big.Int {
	return big.NewInt(1e6)
}

func TestEmitter_CreateEvent_CreatesCorrectEventVersion(t *testing.T) {

	tests := map[string]struct {
		upgrades opera.Upgrades
		version  uint8
	}{
		"sonic": {
			upgrades: opera.Upgrades{
				Sonic:   true,
				Allegro: false,
			},
			version: 2,
		},
		"allegro": {
			upgrades: opera.Upgrades{
				Sonic:   true,
				Allegro: true,
			},
			version: 3,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			world := NewMockExternal(ctrl)
			signer := NewMockSigner(ctrl)

			validator := idx.ValidatorID(1)
			builder := pos.NewBuilder()
			builder.Set(validator, pos.Weight(1))
			validators := builder.Build()

			rules := opera.Rules{
				Upgrades: test.upgrades,
			}

			em := &Emitter{
				config: Config{
					Validator: ValidatorConfig{
						ID: validator,
					},
				},
				world: World{
					External: world,
					Signer:   signer,
				},
				validators: validators,
			}

			any := gomock.Any()
			world.EXPECT().GetRules().Return(rules).AnyTimes()
			world.EXPECT().GetLastEvent(any, any).AnyTimes()
			world.EXPECT().Build(any, any).AnyTimes()
			world.EXPECT().Check(any, any).Return(nil).AnyTimes()
			world.EXPECT().GetLatestBlock().Return(&inter.Block{}).AnyTimes()

			signer.EXPECT().Sign(any, any).AnyTimes()

			event, err := em.createEvent(nil)
			require.NoError(t, err)
			require.Equal(t, test.version, event.Version())
		})
	}
}

func TestEmitter_CreateEvent_InvalidValidatorSetIsDetected(t *testing.T) {

	ctrl := gomock.NewController(t)
	world := NewMockExternal(ctrl)
	signer := NewMockSigner(ctrl)
	log := logger.NewMockLogger(ctrl)

	validator := idx.ValidatorID(1)
	validators := pos.NewBuilder().Build() // invalid empty validator set

	rules := opera.Rules{
		Upgrades: opera.Upgrades{
			Sonic:   true,
			Allegro: true,
		},
	}

	em := &Emitter{
		Periodic: logger.Periodic{
			Instance: logger.Instance{
				Log: log,
			},
		},
		config: Config{
			Validator: ValidatorConfig{
				ID: validator,
			},
		},
		world: World{
			External: world,
			Signer:   signer,
		},
		validators: validators,
	}

	any := gomock.Any()
	world.EXPECT().GetRules().Return(rules).AnyTimes()
	world.EXPECT().GetLastEvent(any, any).AnyTimes()
	world.EXPECT().Build(any, any).AnyTimes()
	world.EXPECT().Check(any, any).Return(nil).AnyTimes()
	world.EXPECT().GetLatestBlock().Return(&inter.Block{}).AnyTimes()

	signer.EXPECT().Sign(any, any).AnyTimes()

	log.EXPECT().Error("Failed to create payload", "err", any)

	_, err := em.createEvent(nil)
	require.ErrorContains(t, err, "no validators")
}
