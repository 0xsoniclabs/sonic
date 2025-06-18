package eventcheck

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/eventcheck/basiccheck"
	"github.com/0xsoniclabs/sonic/eventcheck/epochcheck"
	"github.com/0xsoniclabs/sonic/eventcheck/gaspowercheck"
	"github.com/0xsoniclabs/sonic/eventcheck/heavycheck"
	"github.com/0xsoniclabs/sonic/eventcheck/parentscheck"
	"github.com/0xsoniclabs/sonic/eventcheck/proposalcheck"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/opera"
	parentscheckbase "github.com/Fantom-foundation/lachesis-base/eventcheck/parentscheck"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCheckers_Validate_ValidEventPassesValidation(t *testing.T) {

	tests := map[string]struct {
		modify              func(*inter.MutableEventPayload)
		useInvalidSignature bool
		expected            error
	}{
		"valid event": {
			modify: func(event *inter.MutableEventPayload) {},
		},
		"basic check violation": {
			modify: func(event *inter.MutableEventPayload) {
				event.SetCreationTime(0)
			},
			expected: basiccheck.ErrZeroTime,
		},
		"epoch check violation": {
			modify: func(event *inter.MutableEventPayload) {
				event.SetExtra([]byte{1, 2, 3})
			},
			expected: epochcheck.ErrTooBigExtra,
		},
		"parents check violation": {
			modify: func(event *inter.MutableEventPayload) {
				event.SetLamport(2)
			},
			expected: parentscheckbase.ErrWrongLamport,
		},
		"gas power check violation": {
			modify: func(event *inter.MutableEventPayload) {
				event.SetGasPowerLeft(inter.GasPowerLeft{
					Gas: [inter.GasPowerConfigs]uint64{1000, 2000},
				})
			},
			expected: gaspowercheck.ErrWrongGasPowerLeft,
		},
		"proposal check violation": {
			modify: func(event *inter.MutableEventPayload) {
				event.SetVersion(4)
				event.SetPayload(inter.Payload{
					ProposalSyncState: inter.ProposalSyncState{
						LastSeenProposalTurn: 75,
					},
				})
			},
			expected: proposalcheck.ErrSyncStateProgressionWithoutProposal,
		},
		"heavy check violation": {
			modify:              func(event *inter.MutableEventPayload) {},
			useInvalidSignature: true,
			expected:            heavycheck.ErrWrongEventSig,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			epochCheckReader := epochcheck.NewMockReader(ctrl)
			gasPowerCheckReader := gaspowercheck.NewMockReader(ctrl)
			proposalCheckReader := proposalcheck.NewMockReader(ctrl)
			heavyCheckReader := heavycheck.NewMockReader(ctrl)
			checkers := newCheckersForTests(
				epochCheckReader,
				gasPowerCheckReader,
				proposalCheckReader,
				heavyCheckReader,
			)

			epoch := idx.Epoch(12)

			creator := idx.ValidatorID(1)
			builder := pos.ValidatorsBuilder{}
			builder.Set(creator, pos.Weight(100))
			validators := builder.Build()

			// Prepare a private and public key for signing the event.
			privateKey := evmcore.FakeKey(1)

			// Assemble a valid event that passes all checks.
			eventBuilder := &inter.MutableEventPayload{}
			eventBuilder.SetVersion(0)
			eventBuilder.SetEpoch(epoch)
			eventBuilder.SetCreator(creator)
			eventBuilder.SetSeq(1)
			eventBuilder.SetFrame(1)
			eventBuilder.SetLamport(1)
			eventBuilder.SetCreationTime(inter.Timestamp(1000))
			eventBuilder.SetMedianTime(inter.Timestamp(1000))

			// Allow test case to modify the event payload.
			test.modify(eventBuilder)
			eventBuilder.SetPayloadHash(inter.CalcPayloadHash(eventBuilder))

			// Sign and build the final event.
			digest := eventBuilder.HashToSign()
			if test.useInvalidSignature {
				digest[0]++
			}
			signature, err := crypto.Sign(digest[:], privateKey)
			require.NoError(t, err)
			eventBuilder.SetSig(inter.Signature(signature[:64]))

			event := eventBuilder.Build()

			rules := opera.Rules{
				Economy: opera.EconomyRules{
					Gas: opera.GasRules{
						MaxEventGas: 30_000,
					},
				},
			}

			if event.Version() == 4 {
				rules.Upgrades.SingleProposerBlockFormation = true
			}

			// Prepare the mocks for checkers.
			epochCheckReader.EXPECT().GetEpochValidators().Return(validators, epoch).AnyTimes()
			epochCheckReader.EXPECT().GetEpochRules().Return(rules, epoch).AnyTimes()

			validationContext := &gaspowercheck.ValidationContext{
				Epoch:           epoch,
				Validators:      validators,
				ValidatorStates: []gaspowercheck.ValidatorState{{}},
			}
			gasPowerCheckReader.EXPECT().GetValidationContext().Return(validationContext).AnyTimes()

			keys := map[idx.ValidatorID]validatorpk.PubKey{
				creator: {
					Raw:  crypto.FromECDSAPub(&privateKey.PublicKey),
					Type: validatorpk.Types.Secp256k1,
				},
			}
			heavyCheckReader.EXPECT().GetEpochPubKeys().Return(keys, epoch).AnyTimes()

			// Run the checker and verify that the correct issue is detected.
			require.ErrorIs(t, checkers.Validate(event, nil), test.expected)
		})
	}
}

func newCheckersForTests(
	epochCheckReader epochcheck.Reader,
	gasPowerCheckReader gaspowercheck.Reader,
	proposalCheckReader proposalcheck.Reader,
	heavyCheckReader heavycheck.Reader,
) *Checkers {
	signer := types.NewCancunSigner(big.NewInt(12))
	return &Checkers{
		Basiccheck:    basiccheck.New(),
		Epochcheck:    epochcheck.New(epochCheckReader),
		Parentscheck:  parentscheck.New(),
		Gaspowercheck: gaspowercheck.New(gasPowerCheckReader),
		Proposalcheck: proposalcheck.New(proposalCheckReader),
		Heavycheck: heavycheck.New(
			heavycheck.Config{},
			heavyCheckReader,
			signer,
		),
	}
}
