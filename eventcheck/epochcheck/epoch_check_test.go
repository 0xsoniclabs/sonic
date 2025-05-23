package epochcheck

import (
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	base "github.com/Fantom-foundation/lachesis-base/eventcheck/epochcheck"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	pos "github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestChecker_Validate_SonicAndAllegroRequireDifferentVersions(t *testing.T) {

	test := map[string]struct {
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

	for name, test := range test {
		t.Run(name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			reader := NewMockReader(ctrl)
			event := inter.NewMockEventPayloadI(ctrl)

			creator := idx.ValidatorID(1)
			event.EXPECT().Epoch().AnyTimes()
			event.EXPECT().Parents().AnyTimes()
			event.EXPECT().Extra().AnyTimes()
			event.EXPECT().GasPowerUsed().AnyTimes()
			event.EXPECT().Txs().AnyTimes()
			event.EXPECT().MisbehaviourProofs().AnyTimes()
			event.EXPECT().BlockVotes().AnyTimes()
			event.EXPECT().EpochVote().AnyTimes()
			event.EXPECT().Creator().Return(creator).AnyTimes()

			builder := pos.NewBuilder()
			builder.Set(creator, 10)
			validators := builder.Build()
			reader.EXPECT().GetEpochValidators().Return(validators, idx.Epoch(0)).AnyTimes()

			rules := opera.Rules{Upgrades: test.upgrades}
			reader.EXPECT().GetEpochRules().Return(rules, idx.Epoch(0)).AnyTimes()

			checker := Checker{
				Base:   base.New(reader),
				reader: reader,
			}

			// Check that the correct version is fine.
			event.EXPECT().Version().Return(test.version)
			require.NoError(t, checker.Validate(event))

			// Check that the wrong version fails.
			event.EXPECT().Version().Return(test.version + 1)
			require.ErrorIs(t, checker.Validate(event), ErrWrongVersion)
		})
	}
}
