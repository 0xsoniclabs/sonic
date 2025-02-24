package inter

import (
	"bytes"
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/idx"
)

func emptyEvent(ver uint8) EventPayload {
	empty := MutableEventPayload{}
	empty.SetVersion(ver)
	if ver == 0 {
		empty.SetEpoch(256)
	}
	empty.SetParents(hash.Events{})
	empty.SetExtra([]byte{})
	empty.SetTxs(types.Transactions{})
	empty.SetPayloadHash(EmptyPayloadHash(ver))
	return *empty.Build()
}

func TestEventPayloadSerialization(t *testing.T) {
	max := MutableEventPayload{}
	max.SetVersion(2)
	max.SetEpoch(math.MaxUint32)
	max.SetSeq(idx.Event(math.MaxUint32))
	max.SetLamport(idx.Lamport(math.MaxUint32))
	h := hash.BytesToEvent(bytes.Repeat([]byte{math.MaxUint8}, 32))
	max.SetParents(hash.Events{hash.Event(h), hash.Event(h), hash.Event(h)})
	max.SetPayloadHash(hash.Hash(h))
	max.SetSig(BytesToSignature(bytes.Repeat([]byte{math.MaxUint8}, SigSize)))
	max.SetExtra(bytes.Repeat([]byte{math.MaxUint8}, 100))
	max.SetCreationTime(math.MaxUint64)
	max.SetMedianTime(math.MaxUint64)
	tx1 := types.NewTx(&types.LegacyTx{
		Nonce:    math.MaxUint64,
		GasPrice: h.Big(),
		Gas:      math.MaxUint64,
		To:       nil,
		Value:    h.Big(),
		Data:     []byte{},
		V:        big.NewInt(0xff),
		R:        h.Big(),
		S:        h.Big(),
	})
	tx2 := types.NewTx(&types.LegacyTx{
		Nonce:    math.MaxUint64,
		GasPrice: h.Big(),
		Gas:      math.MaxUint64,
		To:       &common.Address{},
		Value:    h.Big(),
		Data:     max.extra,
		V:        big.NewInt(0xff),
		R:        h.Big(),
		S:        h.Big(),
	})
	txs := types.Transactions{}
	for i := 0; i < 200; i++ {
		txs = append(txs, tx1)
		txs = append(txs, tx2)
	}
	max.SetTxs(txs)

	ee := map[string]EventPayload{
		"empty0":  emptyEvent(0),
		"empty1":  emptyEvent(1),
		"empty2":  emptyEvent(2),
		"max":     *max.Build(),
		"random1": *FakeEvent(1, 12, 1, 1, true),
		"random2": *FakeEvent(2, 12, 0, 0, false),
	}

	t.Run("ok", func(t *testing.T) {
		require := require.New(t)

		for name, header0 := range ee {
			buf, err := rlp.EncodeToBytes(&header0)
			require.NoError(err)

			var header1 EventPayload
			err = rlp.DecodeBytes(buf, &header1)
			require.NoError(err, name)

			require.EqualValues(header0.extEventData, header1.extEventData, name)
			require.EqualValues(header0.sigData, header1.sigData, name)
			for i := range header0.payloadData.txs {
				require.EqualValues(header0.payloadData.txs[i].Hash(), header1.payloadData.txs[i].Hash(), name)
			}
			require.EqualValues(header0.baseEvent, header1.baseEvent, name)
			require.EqualValues(header0.ID(), header1.ID(), name)
			require.EqualValues(header0.HashToSign(), header1.HashToSign(), name)
			require.EqualValues(header0.Size(), header1.Size(), name)
		}
	})

	t.Run("err", func(t *testing.T) {
		require := require.New(t)

		for name, header0 := range ee {
			bin, err := header0.MarshalBinary()
			require.NoError(err, name)

			n := rand.IntN(len(bin) - len(header0.Extra()) - 1)
			bin = bin[0:n]

			buf, err := rlp.EncodeToBytes(bin)
			require.NoError(err, name)

			var header1 Event
			err = rlp.DecodeBytes(buf, &header1)
			require.Error(err, name)
		}
	})
}

func BenchmarkEventPayload_EncodeRLP_empty(b *testing.B) {
	e := emptyEvent(0)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf, err := rlp.EncodeToBytes(&e)
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(buf)), "size")
	}
}

func BenchmarkEventPayload_EncodeRLP_NoPayload(b *testing.B) {
	e := FakeEvent(2, 0, 0, 0, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf, err := rlp.EncodeToBytes(&e)
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(buf)), "size")
	}
}

func BenchmarkEventPayload_EncodeRLP(b *testing.B) {
	e := FakeEvent(2, 1000, 0, 0, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf, err := rlp.EncodeToBytes(&e)
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(buf)), "size")
	}
}

func BenchmarkEventPayload_DecodeRLP_empty(b *testing.B) {
	e := emptyEvent(0)
	me := MutableEventPayload{}

	buf, err := rlp.EncodeToBytes(&e)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err = rlp.DecodeBytes(buf, &me)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEventPayload_DecodeRLP_NoPayload(b *testing.B) {
	e := FakeEvent(2, 0, 0, 0, false)
	me := MutableEventPayload{}

	buf, err := rlp.EncodeToBytes(&e)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err = rlp.DecodeBytes(buf, &me)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEventPayload_DecodeRLP(b *testing.B) {
	e := FakeEvent(2, 22, 0, 0, false)
	me := MutableEventPayload{}

	buf, err := rlp.EncodeToBytes(&e)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err = rlp.DecodeBytes(buf, &me)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func randBig(rand *rand.Rand) *big.Int {
	b := make([]byte, rand.IntN(8))
	for i := range b {
		b[i] = byte(rand.IntN(256))
	}
	if len(b) == 0 {
		b = []byte{0}
	}
	return new(big.Int).SetBytes(b)
}

func randAddr(rand *rand.Rand) common.Address {
	addr := common.Address{}
	for i := 0; i < len(addr); i++ {
		addr[i] = byte(rand.IntN(256))
	}
	return addr
}

func randBytes(rand *rand.Rand, size int) []byte {
	b := make([]byte, size)
	for i := 0; i < size; i++ {
		b[i] = byte(rand.IntN(256))
	}
	return b
}

func randHash(rand *rand.Rand) hash.Hash {
	return hash.BytesToHash(randBytes(rand, 32))
}

func randAddrPtr(rand *rand.Rand) *common.Address {
	addr := randAddr(rand)
	return &addr
}

func randAccessList(rand *rand.Rand, maxAddrs, maxKeys int) types.AccessList {
	accessList := make(types.AccessList, rand.IntN(maxAddrs))
	for i := range accessList {
		accessList[i].Address = randAddr(rand)
		accessList[i].StorageKeys = make([]common.Hash, rand.IntN(maxKeys))
		for j := range accessList[i].StorageKeys {
			for k := 0; k < len(accessList[i].StorageKeys[j]); k++ {
				accessList[i].StorageKeys[j][k] = byte(rand.IntN(256))
			}
		}
	}
	return accessList
}

// FakeEvent generates random event for testing purpose.
func FakeEvent(version uint8, txsNum, mpsNum, bvsNum int, ersNum bool) *EventPayload {
	r := rand.New(rand.NewPCG(0, 0))
	random := &MutableEventPayload{}
	random.SetVersion(version)
	random.SetNetForkID(uint16(r.Uint32() >> 16))
	random.SetLamport(1000)
	random.SetExtra([]byte{byte(r.Uint32())})
	random.SetSeq(idx.Event(r.Uint32() >> 8))
	random.SetEpoch(idx.Epoch(1234))
	random.SetCreator(idx.ValidatorID(r.Uint32()))
	random.SetFrame(idx.Frame(r.Uint32() >> 16))
	random.SetCreationTime(Timestamp(r.Uint64()))
	random.SetMedianTime(Timestamp(r.Uint64()))
	random.SetGasPowerUsed(r.Uint64())
	random.SetGasPowerLeft(GasPowerLeft{[2]uint64{r.Uint64(), r.Uint64()}})
	txs := types.Transactions{}
	for i := 0; i < txsNum; i++ {
		h := hash.Hash{}
		for i := 0; i < len(h); i++ {
			h[i] = byte(r.Uint32())
		}
		if i%3 == 0 {
			tx := types.NewTx(&types.LegacyTx{
				Nonce:    r.Uint64(),
				GasPrice: randBig(r),
				Gas:      257 + r.Uint64(),
				To:       nil,
				Value:    randBig(r),
				Data:     randBytes(r, rand.IntN(300)),
				V:        big.NewInt(int64(rand.IntN(0xffffffff))),
				R:        h.Big(),
				S:        h.Big(),
			})
			txs = append(txs, tx)
		} else if i%3 == 1 {
			tx := types.NewTx(&types.AccessListTx{
				ChainID:    randBig(r),
				Nonce:      r.Uint64(),
				GasPrice:   randBig(r),
				Gas:        r.Uint64(),
				To:         randAddrPtr(r),
				Value:      randBig(r),
				Data:       randBytes(r, rand.IntN(300)),
				AccessList: randAccessList(r, 300, 300),
				V:          big.NewInt(int64(rand.IntN(0xffffffff))),
				R:          h.Big(),
				S:          h.Big(),
			})
			txs = append(txs, tx)
		} else {
			tx := types.NewTx(&types.DynamicFeeTx{
				ChainID:    randBig(r),
				Nonce:      r.Uint64(),
				GasTipCap:  randBig(r),
				GasFeeCap:  randBig(r),
				Gas:        r.Uint64(),
				To:         randAddrPtr(r),
				Value:      randBig(r),
				Data:       randBytes(r, rand.IntN(300)),
				AccessList: randAccessList(r, 300, 300),
				V:          big.NewInt(int64(rand.IntN(0xffffffff))),
				R:          h.Big(),
				S:          h.Big(),
			})
			txs = append(txs, tx)
		}
	}
	random.SetTxs(txs)

	if version == 1 {
		mps := []MisbehaviourProof{}
		for i := 0; i < mpsNum; i++ {
			// MPs are serialized with RLP, so no need to test extensively
			mps = append(mps, MisbehaviourProof{
				EventsDoublesign: &EventsDoublesign{
					Pair: [2]SignedEventLocator{SignedEventLocator{}, SignedEventLocator{}},
				},
				BlockVoteDoublesign: nil,
				WrongBlockVote:      nil,
				EpochVoteDoublesign: nil,
				WrongEpochVote:      nil,
			})
		}
		random.SetMisbehaviourProofs(mps)

		bvs := LlrBlockVotes{}
		if bvsNum > 0 {
			bvs.Start = 1 + idx.Block(rand.IntN(1000))
			bvs.Epoch = 1 + idx.Epoch(rand.IntN(1000))
		}
		for i := 0; i < bvsNum; i++ {
			bvs.Votes = append(bvs.Votes, randHash(r))
		}
		random.SetBlockVotes(bvs)

		ers := LlrEpochVote{}
		if ersNum {
			ers.Epoch = 1 + idx.Epoch(rand.IntN(1000))
			ers.Vote = randHash(r)
		}
		random.SetEpochVote(ers)
	}

	random.SetPayloadHash(CalcPayloadHash(random))

	parent := MutableEventPayload{}
	parent.SetVersion(1)
	parent.SetLamport(random.Lamport() - 500)
	parent.SetEpoch(random.Epoch())
	random.SetParents(hash.Events{parent.Build().ID()})

	return random.Build()
}
