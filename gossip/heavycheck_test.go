package gossip

import (
	"bytes"
	"math"
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/suite"

	"github.com/0xsoniclabs/sonic/eventcheck/epochcheck"
	"github.com/0xsoniclabs/sonic/eventcheck/heavycheck"
	"github.com/0xsoniclabs/sonic/inter"
)

type LLRHeavyCheckTestSuite struct {
	suite.Suite

	env        *testEnv
	me         *inter.MutableEventPayload
	startEpoch consensus.Epoch
}

func (s *LLRHeavyCheckTestSuite) SetupSuite() {
	s.T().Log("setting up test suite")

	const (
		validatorsNum = 10
		startEpoch    = 1
	)

	env := newTestEnv(startEpoch, validatorsNum, s.T())

	em := env.emitters[0]
	e, err := em.EmitEvent()
	s.Require().NoError(err)
	s.Require().NotNil(e)

	s.env = env
	s.me = mutableEventPayloadFromImmutable(e)
	s.startEpoch = consensus.Epoch(startEpoch)
}

func (s *LLRHeavyCheckTestSuite) TearDownSuite() {
	s.T().Log("tearing down test suite")
	s.Require().NoError(s.env.Close())
}

func mutableEventPayloadFromImmutable(e *inter.EventPayload) *inter.MutableEventPayload {
	me := &inter.MutableEventPayload{}
	me.SetVersion(e.Version())
	me.SetNetForkID(e.NetForkID())
	me.SetCreator(e.Creator())
	me.SetEpoch(e.Epoch())
	me.SetCreationTime(e.CreationTime())
	me.SetMedianTime(e.MedianTime())
	me.SetPrevEpochHash(e.PrevEpochHash())
	me.SetExtra(e.Extra())
	me.SetGasPowerLeft(e.GasPowerLeft())
	me.SetGasPowerUsed(e.GasPowerUsed())
	me.SetPayloadHash(e.PayloadHash())
	me.SetSig(e.Sig())
	me.SetTxs(e.Txs())
	return me
}

func (s *LLRHeavyCheckTestSuite) TestHeavyCheckValidateEvent() {

	testCases := []struct {
		name    string
		errExp  error
		pretest func()
	}{
		{
			"success",
			nil,
			func() {
				s.me.SetVersion(1)
				s.me.SetEpoch(consensus.Epoch(s.startEpoch))
				s.me.SetCreator(3)
				s.me.SetSeq(consensus.Seq(1))
				s.me.SetFrame(consensus.Frame(1))
				s.me.SetLamport(consensus.Lamport(1))
				s.me.SetPayloadHash(inter.CalcPayloadHash(s.me))

				sig, err := s.env.signer.Sign(s.env.pubkeys[2], s.me.HashToSign().Bytes())
				s.Require().NoError(err)
				sSig := inter.Signature{}
				copy(sSig[:], sig)
				s.me.SetSig(sSig)
			},
		},
		{
			"epochcheck.ErrNotRelevant",
			epochcheck.ErrNotRelevant,
			func() {
				s.me.SetVersion(1)
				s.me.SetEpoch(consensus.Epoch(s.startEpoch + 1))
				s.me.SetCreator(3)
				s.me.SetPayloadHash(inter.CalcPayloadHash(s.me))

				sig, err := s.env.signer.Sign(s.env.pubkeys[2], s.me.HashToSign().Bytes())
				s.Require().NoError(err)
				sSig := inter.Signature{}
				copy(sSig[:], sig)
				s.me.SetSig(sSig)
			},
		},
		{
			"epochcheck.ErrAuth",
			epochcheck.ErrAuth,
			func() {
				s.me.SetVersion(1)
				s.me.SetEpoch(consensus.Epoch(s.startEpoch))
				s.me.SetSeq(consensus.Seq(1))
				s.me.SetFrame(consensus.Frame(1))
				s.me.SetLamport(consensus.Lamport(1))
				invalidCreator := consensus.ValidatorID(100)
				s.me.SetCreator(invalidCreator)
				s.me.SetPayloadHash(inter.CalcPayloadHash(s.me))

				sig, err := s.env.signer.Sign(s.env.pubkeys[2], s.me.HashToSign().Bytes())
				s.Require().NoError(err)
				sSig := inter.Signature{}
				copy(sSig[:], sig)
				s.me.SetSig(sSig)
			},
		},
		{
			"ErrWrongEventSig",
			heavycheck.ErrWrongEventSig,
			func() {
				s.me.SetVersion(1)
				s.me.SetEpoch(consensus.Epoch(s.startEpoch))
				s.me.SetCreator(3)
				s.me.SetSeq(consensus.Seq(1))
				s.me.SetFrame(consensus.Frame(1))
				s.me.SetLamport(consensus.Lamport(1))
				s.me.SetPayloadHash(inter.CalcPayloadHash(s.me))

				sig, err := s.env.signer.Sign(s.env.pubkeys[1], s.me.HashToSign().Bytes())
				s.Require().NoError(err)
				sSig := inter.Signature{}
				copy(sSig[:], sig)
				s.me.SetSig(sSig)
			},
		},
		{
			"ErrMalformedTxSig",
			heavycheck.ErrMalformedTxSig,
			func() {
				s.me.SetVersion(1)
				s.me.SetEpoch(consensus.Epoch(s.startEpoch))
				s.me.SetCreator(3)
				s.me.SetSeq(consensus.Seq(1))
				s.me.SetFrame(consensus.Frame(1))
				s.me.SetLamport(consensus.Lamport(1))
				h := consensus.BytesToEvent(bytes.Repeat([]byte{math.MaxUint8}, 32))
				tx1 := types.NewTx(&types.LegacyTx{
					Nonce:    math.MaxUint64,
					GasPrice: h.Big(),
					Gas:      math.MaxUint64,
					To:       nil,
					Value:    h.Big(),
					Data:     []byte{},
				})
				txs := types.Transactions{}
				txs = append(txs, tx1)
				s.me.SetTxs(txs)
				s.me.SetPayloadHash(inter.CalcPayloadHash(s.me))

				sig, err := s.env.signer.Sign(s.env.pubkeys[2], s.me.HashToSign().Bytes())
				s.Require().NoError(err)
				sSig := inter.Signature{}
				copy(sSig[:], sig)
				s.me.SetSig(sSig)
			},
		},
		{
			"ErrWrongPayloadHash",
			heavycheck.ErrWrongPayloadHash,
			func() {
				s.me.SetVersion(1)
				s.me.SetEpoch(consensus.Epoch(s.startEpoch))
				s.me.SetSeq(consensus.Seq(1))
				s.me.SetFrame(consensus.Frame(1))
				s.me.SetLamport(consensus.Lamport(1))
				s.me.SetCreator(3)

				invalidPayloadHash := consensus.Hash{}
				s.me.SetPayloadHash(invalidPayloadHash)

				sig, err := s.env.signer.Sign(s.env.pubkeys[2], s.me.HashToSign().Bytes())
				s.Require().NoError(err)
				sSig := inter.Signature{}
				copy(sSig[:], sig)
				s.me.SetSig(sSig)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupSuite()
			tc.pretest()

			err := s.env.checkers.Heavycheck.ValidateEvent(s.me)

			if tc.errExp != nil {
				s.Require().Error(err)
				s.Require().EqualError(err, tc.errExp.Error())
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func TestLLRHeavyCheckTestSuite(t *testing.T) {
	t.Skip() // skip until fixed
	suite.Run(t, new(LLRHeavyCheckTestSuite))
}
