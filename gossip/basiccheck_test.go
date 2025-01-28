package gossip

import (
	"bytes"
	"math"
	"math/big"
	"testing"

	lbasiccheck "github.com/Fantom-foundation/lachesis-base/eventcheck/basiccheck"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/suite"

	"github.com/0xsoniclabs/sonic/eventcheck/basiccheck"
	"github.com/0xsoniclabs/sonic/inter"
)

type LLRBasicCheckTestSuite struct {
	suite.Suite

	env        *testEnv
	me         *inter.MutableEventPayload
	startEpoch idx.Epoch
}

func (s *LLRBasicCheckTestSuite) SetupSuite() {
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
	s.startEpoch = idx.Epoch(startEpoch)
}

func (s *LLRBasicCheckTestSuite) TearDownSuite() {
	s.T().Log("tearing down test suite")
	s.env.Close()
}

func (s *LLRBasicCheckTestSuite) TestBasicCheckValidate() {

	testCases := []struct {
		name    string
		pretest func()
		errExp  error
	}{

		{
			"ErrWrongNetForkID",
			func() {
				s.me.SetNetForkID(1)
			},
			basiccheck.ErrWrongNetForkID,
		},

		{
			"Validate checkLimits ErrHugeValue",
			func() {
				s.me.SetEpoch(math.MaxInt32 - 1)
			},
			lbasiccheck.ErrHugeValue,
		},
		{
			"Validate checkInited checkInited ErrNotInited ",
			func() {
				s.me.SetSeq(0)
			},
			lbasiccheck.ErrNotInited,
		},
		{
			"Validate checkInited ErrNoParents",
			func() {
				s.me.SetEpoch(idx.Epoch(1))
				s.me.SetFrame(idx.Frame(1))
				s.me.SetLamport(idx.Lamport(1))

				s.me.SetSeq(idx.Event(2))
				parents := hash.Events{}
				s.me.SetParents(parents)
			},
			lbasiccheck.ErrNoParents,
		},
		{
			"Validate ErrHugeValue-1",
			func() {
				s.me.SetSeq(idx.Event(1))
				s.me.SetEpoch(idx.Epoch(1))
				s.me.SetFrame(idx.Frame(1))
				s.me.SetLamport(idx.Lamport(1))

				s.me.SetGasPowerUsed(math.MaxInt64 - 1)
			},
			lbasiccheck.ErrHugeValue,
		},
		{
			"Validate ErrHugeValue-2",
			func() {
				s.me.SetSeq(idx.Event(1))
				s.me.SetEpoch(idx.Epoch(1))
				s.me.SetFrame(idx.Frame(1))
				s.me.SetLamport(idx.Lamport(1))

				s.me.SetGasPowerLeft(inter.GasPowerLeft{Gas: [2]uint64{math.MaxInt64 - 1, math.MaxInt64}})
			},
			lbasiccheck.ErrHugeValue,
		},
		{
			"Validate ErrZeroTime-1",
			func() {
				s.me.SetSeq(idx.Event(1))
				s.me.SetEpoch(idx.Epoch(1))
				s.me.SetFrame(idx.Frame(1))
				s.me.SetLamport(idx.Lamport(1))

				s.me.SetCreationTime(0)
			},
			basiccheck.ErrZeroTime,
		},
		{
			"Validate ErrZeroTime-2",
			func() {
				s.me.SetSeq(idx.Event(1))
				s.me.SetEpoch(idx.Epoch(1))
				s.me.SetFrame(idx.Frame(1))
				s.me.SetLamport(idx.Lamport(1))

				s.me.SetMedianTime(0)
			},
			basiccheck.ErrZeroTime,
		},
		{
			"Validate checkTxs validateTx ErrNegativeValue-1",
			func() {
				s.me.SetSeq(idx.Event(1))
				s.me.SetEpoch(idx.Epoch(1))
				s.me.SetFrame(idx.Frame(1))
				s.me.SetLamport(idx.Lamport(1))

				h := hash.BytesToEvent(bytes.Repeat([]byte{math.MaxUint8}, 32))
				tx1 := types.NewTx(&types.LegacyTx{
					Nonce:    math.MaxUint64,
					GasPrice: h.Big(),
					Gas:      math.MaxUint64,
					To:       nil,
					Value:    big.NewInt(-1000),
					Data:     []byte{},
					V:        big.NewInt(0xff),
					R:        h.Big(),
					S:        h.Big(),
				})
				txs := types.Transactions{}
				txs = append(txs, tx1)
				s.me.SetTxs(txs)
			},
			basiccheck.ErrNegativeValue,
		},
		{
			"Validate checkTxs validateTx ErrNegativeValue-2",
			func() {
				s.me.SetSeq(idx.Event(1))
				s.me.SetEpoch(idx.Epoch(1))
				s.me.SetFrame(idx.Frame(1))
				s.me.SetLamport(idx.Lamport(1))

				h := hash.BytesToEvent(bytes.Repeat([]byte{math.MaxUint8}, 32))
				tx1 := types.NewTx(&types.LegacyTx{
					Nonce:    math.MaxUint64,
					GasPrice: big.NewInt(-1000),
					Gas:      math.MaxUint64,
					To:       nil,
					Value:    h.Big(),
					Data:     []byte{},
					V:        big.NewInt(0xff),
					R:        h.Big(),
					S:        h.Big(),
				})
				txs := types.Transactions{}
				txs = append(txs, tx1)
				s.me.SetTxs(txs)
			},
			basiccheck.ErrNegativeValue,
		},
		{
			"Validate checkTxs validateTx ErrIntrinsicGas",
			func() {
				s.me.SetSeq(idx.Event(1))
				s.me.SetEpoch(idx.Epoch(1))
				s.me.SetFrame(idx.Frame(1))
				s.me.SetLamport(idx.Lamport(1))

				h := hash.BytesToEvent(bytes.Repeat([]byte{math.MaxUint8}, 32))
				tx1 := types.NewTx(&types.LegacyTx{
					Nonce:    math.MaxUint64,
					GasPrice: h.Big(),
					Gas:      0,
					To:       nil,
					Value:    h.Big(),
					Data:     []byte{},
					V:        big.NewInt(0xff),
					R:        h.Big(),
					S:        h.Big(),
				})
				txs := types.Transactions{}
				txs = append(txs, tx1)
				s.me.SetTxs(txs)
			},
			basiccheck.ErrIntrinsicGas,
		},

		{
			"Validate checkTxs validateTx ErrTipAboveFeeCap",
			func() {
				s.me.SetSeq(idx.Event(1))
				s.me.SetEpoch(idx.Epoch(1))
				s.me.SetFrame(idx.Frame(1))
				s.me.SetLamport(idx.Lamport(1))

				h := hash.BytesToEvent(bytes.Repeat([]byte{math.MaxUint8}, 32))

				tx1 := types.NewTx(&types.DynamicFeeTx{
					Nonce:     math.MaxUint64,
					To:        nil,
					Data:      []byte{},
					Gas:       math.MaxUint64,
					Value:     h.Big(),
					ChainID:   new(big.Int),
					GasTipCap: big.NewInt(1000),
					GasFeeCap: new(big.Int),
					V:         big.NewInt(0xff),
					R:         h.Big(),
					S:         h.Big(),
				})

				txs := types.Transactions{}
				txs = append(txs, tx1)
				s.me.SetTxs(txs)
			},
			basiccheck.ErrTipAboveFeeCap,
		},
		{
			"Validate returns nil",
			func() {
				s.me.SetSeq(idx.Event(1))
				s.me.SetEpoch(idx.Epoch(1))
				s.me.SetFrame(idx.Frame(1))
				s.me.SetLamport(idx.Lamport(1))

				h := hash.BytesToEvent(bytes.Repeat([]byte{math.MaxUint8}, 32))

				tx1 := types.NewTx(&types.DynamicFeeTx{
					Nonce:     math.MaxUint64,
					To:        nil,
					Data:      []byte{},
					Gas:       math.MaxUint64,
					Value:     h.Big(),
					ChainID:   new(big.Int),
					GasTipCap: new(big.Int),
					GasFeeCap: big.NewInt(1000),
					V:         big.NewInt(0xff),
					R:         h.Big(),
					S:         h.Big(),
				})

				txs := types.Transactions{}
				txs = append(txs, tx1)
				s.me.SetTxs(txs)
			},
			nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupSuite()
			tc.pretest()

			err := s.env.checkers.Basiccheck.Validate(s.me)

			if tc.errExp != nil {
				s.Require().Error(err)
				s.Require().EqualError(err, tc.errExp.Error())
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func TestBasicCheckIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(LLRBasicCheckTestSuite))
}
