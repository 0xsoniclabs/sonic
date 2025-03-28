package inter

import (
	"bytes"
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func emptyEvent(ver uint8) EventPayload {
	empty := MutableEventPayload{}
	empty.SetVersion(ver)
	if ver == 0 {
		empty.SetEpoch(256)
	}
	empty.SetParents(consensus.EventHashes{})
	empty.SetExtra([]byte{})
	empty.SetTxs(types.Transactions{})
	empty.SetPayloadHash(EmptyPayloadHash(ver))
	return *empty.Build()
}

func TestEventPayloadSerialization(t *testing.T) {
	event := MutableEventPayload{}
	event.SetVersion(2)
	event.SetEpoch(math.MaxUint32)
	event.SetSeq(consensus.Seq(math.MaxUint32))
	event.SetLamport(consensus.Lamport(math.MaxUint32))
	h := consensus.BytesToEvent(bytes.Repeat([]byte{math.MaxUint8}, 32))
	event.SetParents(consensus.EventHashes{consensus.EventHash(h), consensus.EventHash(h), consensus.EventHash(h)})
	event.SetPayloadHash(consensus.Hash(h))
	event.SetSig(BytesToSignature(bytes.Repeat([]byte{math.MaxUint8}, SigSize)))
	event.SetExtra(bytes.Repeat([]byte{math.MaxUint8}, 100))
	event.SetCreationTime(math.MaxUint64)
	event.SetMedianTime(math.MaxUint64)

	allTransactionTypes := makeAllTransactionTypes()
	txs := types.Transactions{}
	for i := 0; i < 50; i++ {
		txs = append(txs, allTransactionTypes...)
	}
	event.SetTxs(txs)
	require.Len(t, event.txs, len(allTransactionTypes)*50)

	tests := map[string]EventPayload{
		"empty0":  emptyEvent(0),
		"empty1":  emptyEvent(1),
		"empty2":  emptyEvent(2),
		"event":   *event.Build(),
		"random1": *FakeEvent(1, 12, 1, 1, true),
		"random2": *FakeEvent(2, 12, 0, 0, false),
	}

	t.Run("ok", func(t *testing.T) {
		for name, toEncode := range tests {
			t.Run(name, func(t *testing.T) {
				buf, err := rlp.EncodeToBytes(&toEncode)
				require.NoError(t, err)

				var decoded EventPayload
				err = rlp.DecodeBytes(buf, &decoded)
				require.NoError(t, err)

				require.EqualValues(t, toEncode.extEventData, decoded.extEventData)
				require.EqualValues(t, toEncode.sigData, decoded.sigData)
				require.Equal(t, len(toEncode.txs), len(decoded.txs))
				for i := range toEncode.payloadData.txs {
					require.EqualValues(t, toEncode.payloadData.txs[i].Hash(), decoded.payloadData.txs[i].Hash())
				}
				require.EqualValues(t, toEncode.baseEvent, decoded.baseEvent)
				require.EqualValues(t, toEncode.ID(), decoded.ID())
				require.EqualValues(t, toEncode.HashToSign(), decoded.HashToSign())
				require.EqualValues(t, toEncode.Size(), decoded.Size())
			})
		}
	})

	t.Run("err", func(t *testing.T) {
		for name, toEncode := range tests {
			t.Run(name, func(t *testing.T) {
				bin, err := toEncode.MarshalBinary()
				require.NoError(t, err)

				n := rand.IntN(len(bin) - len(toEncode.Extra()) - 1)
				bin = bin[0:n]

				buf, err := rlp.EncodeToBytes(bin)
				require.NoError(t, err)

				var decoded Event
				err = rlp.DecodeBytes(buf, &decoded)
				require.Error(t, err)
			})
		}
	})
}

func makeAllTransactionTypes() []*types.Transaction {
	chainId := big.NewInt(1)

	return []*types.Transaction{
		types.NewTx(&types.LegacyTx{
			Nonce:    1,
			GasPrice: big.NewInt(1),
			Gas:      1,
			To:       nil,
			Value:    big.NewInt(1),
			Data:     []byte{1},
			V:        big.NewInt(1),
			R:        big.NewInt(123),
			S:        big.NewInt(123),
		}),
		types.NewTx(&types.AccessListTx{
			ChainID:  chainId,
			Nonce:    1,
			GasPrice: big.NewInt(1),
			Gas:      1,
			To:       nil,
			Value:    big.NewInt(1),
			Data:     []byte{1},
			AccessList: types.AccessList{
				types.AccessTuple{
					Address: common.HexToAddress("0x1"),
					StorageKeys: []common.Hash{
						common.HexToHash("0x1"),
					},
				},
			},
			V: big.NewInt(1),
			R: big.NewInt(123),
			S: big.NewInt(123),
		}),
		types.NewTx(&types.DynamicFeeTx{
			ChainID:   chainId,
			Nonce:     1,
			Gas:       1,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(1),
			To:        nil,
			Value:     big.NewInt(1),
			Data:      []byte{1},
			AccessList: types.AccessList{
				types.AccessTuple{
					Address: common.HexToAddress("0x1"),
					StorageKeys: []common.Hash{
						common.HexToHash("0x1"),
					},
				},
			},

			V: big.NewInt(1),
			R: big.NewInt(123),
			S: big.NewInt(123),
		}),
		types.NewTx(&types.BlobTx{
			ChainID:   uint256.MustFromBig(chainId),
			Nonce:     1,
			Gas:       1,
			GasFeeCap: uint256.NewInt(1),
			GasTipCap: uint256.NewInt(1),
			To:        common.HexToAddress("0x1"),
			Value:     uint256.NewInt(1),
			Data:      []byte{1},
			AccessList: types.AccessList{
				types.AccessTuple{
					Address: common.HexToAddress("0x1"),
					StorageKeys: []common.Hash{
						common.HexToHash("0x1"),
					},
				},
			},
			BlobFeeCap: uint256.NewInt(1),
			BlobHashes: []common.Hash{
				common.HexToHash("0x1"),
			},
			V: uint256.NewInt(1),
			R: uint256.NewInt(123),
			S: uint256.NewInt(123),
		}),
		types.NewTx(&types.SetCodeTx{
			ChainID:   uint256.MustFromBig(chainId),
			Nonce:     1,
			Gas:       1,
			GasFeeCap: uint256.NewInt(1),
			GasTipCap: uint256.NewInt(1),
			To:        common.HexToAddress("0x1"),
			Value:     uint256.NewInt(1),
			Data:      []byte{1},
			AccessList: types.AccessList{
				types.AccessTuple{
					Address: common.HexToAddress("0x1"),
					StorageKeys: []common.Hash{
						common.HexToHash("0x1"),
					},
				},
			},
			AuthList: []types.SetCodeAuthorization{
				{
					ChainID: *uint256.MustFromBig(chainId),
					Address: common.HexToAddress("0x1"),
					Nonce:   1,
					V:       1,
					R:       *uint256.NewInt(123),
					S:       *uint256.NewInt(123),
				},
			},
		}),
	}
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

func randHash(rand *rand.Rand) consensus.Hash {
	return consensus.BytesToHash(randBytes(rand, 32))
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
	random.SetSeq(consensus.Seq(r.Uint32() >> 8))
	random.SetEpoch(consensus.Epoch(1234))
	random.SetCreator(consensus.ValidatorID(r.Uint32()))
	random.SetFrame(consensus.Frame(r.Uint32() >> 16))
	random.SetCreationTime(Timestamp(r.Uint64()))
	random.SetMedianTime(Timestamp(r.Uint64()))
	random.SetGasPowerUsed(r.Uint64())
	random.SetGasPowerLeft(GasPowerLeft{[2]uint64{r.Uint64(), r.Uint64()}})
	txs := types.Transactions{}
	for i := 0; i < txsNum; i++ {
		h := consensus.Hash{}
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
			bvs.Start = 1 + consensus.BlockID(rand.IntN(1000))
			bvs.Epoch = 1 + consensus.Epoch(rand.IntN(1000))
		}
		for i := 0; i < bvsNum; i++ {
			bvs.Votes = append(bvs.Votes, randHash(r))
		}
		random.SetBlockVotes(bvs)

		ers := LlrEpochVote{}
		if ersNum {
			ers.Epoch = 1 + consensus.Epoch(rand.IntN(1000))
			ers.Vote = randHash(r)
		}
		random.SetEpochVote(ers)
	}

	random.SetPayloadHash(CalcPayloadHash(random))

	parent := MutableEventPayload{}
	parent.SetVersion(1)
	parent.SetLamport(random.Lamport() - 500)
	parent.SetEpoch(random.Epoch())
	random.SetParents(consensus.EventHashes{parent.Build().ID()})

	return random.Build()
}
