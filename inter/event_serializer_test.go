package inter

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/randao"
	"github.com/0xsoniclabs/sonic/utils/cser"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
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
	event := MutableEventPayload{}
	event.SetVersion(2)
	event.SetEpoch(math.MaxUint32)
	event.SetSeq(idx.Event(math.MaxUint32))
	event.SetLamport(idx.Lamport(math.MaxUint32))
	h := hash.BytesToEvent(bytes.Repeat([]byte{math.MaxUint8}, 32))
	event.SetParents(hash.Events{hash.Event(h), hash.Event(h), hash.Event(h)})
	event.SetPayloadHash(hash.Hash(h))
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
		"empty3":  emptyEvent(3),
		"event":   *event.Build(),
		"random1": *FakeEvent(1, 12, 1, 1, true),
		"random2": *FakeEvent(2, 12, 0, 0, false),
		"random3": *FakeEvent(3, 12, 0, 0, false),
	}

	t.Run("ok", func(t *testing.T) {
		for name, toEncode := range tests {
			t.Run(name, func(t *testing.T) {
				buf, err := rlp.EncodeToBytes(&toEncode)
				require.NoError(t, err)

				var decoded EventPayload
				err = rlp.DecodeBytes(buf, &decoded)
				require.NoError(t, err, "failed to decode encoding of %+v", toEncode)

				require.EqualValues(t, toEncode.extEventData, decoded.extEventData)
				require.EqualValues(t, toEncode.sigData, decoded.sigData)
				require.Equal(t, len(toEncode.txs), len(decoded.txs))
				require.Equal(t, toEncode.payload.Hash(), decoded.payload.Hash())
				for i := range toEncode.txs {
					require.EqualValues(t, toEncode.txs[i].Hash(), decoded.txs[i].Hash())
				}
				require.EqualValues(t, toEncode.baseEvent, decoded.baseEvent)
				require.EqualValues(t, toEncode.ID(), decoded.ID())
				require.EqualValues(t, toEncode.HashToSign(), decoded.HashToSign())
				require.EqualValues(t, toEncode.Size(), decoded.Size())
				require.EqualValues(t, toEncode.PayloadHash(), decoded.PayloadHash())
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

func TestEventSerialization_Flaky2(t *testing.T) {
	t.Parallel()
	for range 100 {
		t.Run("random", func(t *testing.T) {
			t.Parallel()
			event := FakeEvent(3, 2, 0, 0, false)
			buf, err := rlp.EncodeToBytes(&event)
			require.NoError(t, err)
			var decoded EventPayload
			err = rlp.DecodeBytes(buf, &decoded)
			require.NoError(t, err, "failed to decode encoding of %+v", event)
		})
	}
}

func TestEventSerialization_Flaky(t *testing.T) {

	// Failed data:
	// {SignedEvent:{Event:{baseEvent:{BaseEvent:{epoch:1234 seq:5762101 frame:26190 creator:1892640394 parents:[[0 0 4 210 0 0 1 244 83 219 144 86 250 59 149 6 145 72 153 169 9 48 136 75 176 41 48 74 182 165 28 34]] lamport:1000 id:[0 0 4 210 0 0 3 232 99 66 83 218 142 123 135 23 142 43 242 47 100 1 175 179 7 225 159 209 170 231 33 188]}} extEventData:{version:1 netForkID:14591 creationTime:1418823048812169170 medianTime:13446722647129416609 prevEpochHash:<nil> gasPowerLeft:{Gas:[13708822528903384745 16582119225744604457]} gasPowerUsed:8878188716898717410 extra:[45] anyTxs:true anyBlockVotes:true anyEpochVote:true anyMisbehaviourProofs:true hasProposal:false payloadHash:[146 25 217 255 222 158 125 169 223 156 40 127 240 134 135 235 88 61 242 50 224 40 219 2 138 65 53 11 170 113 173 86]} _baseHash:0xc0016000e0 _locatorHash:0xc0016000a0} sigData:{sig:[0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]}} payloadData:{txs:[0xc002ba2680 0xc002ba2000 0xc002ba2080 0xc002ba20c0 0xc002ba2100 0xc002ba2180 0xc002ba21c0 0xc002ba2200 0xc002ba2240 0xc002ba2280 0xc002ba22c0 0xc002ba2300] misbehaviourProofs:[{EventsDoublesign:0xc000036780 BlockVoteDoublesign:<nil> WrongBlockVote:<nil> EpochVoteDoublesign:<nil> WrongEpochVote:<nil>}] epochVote:{Epoch:529 Vote:[92 219 5 218 93 62 243 47 154 235 11 183 28 10 120 39 205 19 209 37 29 189 87 214 77 91 190 143 174 130 34 77]} blockVotes:{Start:804 Epoch:261 Votes:[[88 216 121 178 206 231 70 34 145 143 252 57 205 16 53 209 11 91 223 244 103 100 251 41 156 63 124 65 184 149 233 219]]} payload:{ProposalSyncState:{LastSeenProposalTurn:0 LastSeenProposalFrame:0} Proposal:<nil>}} _size:10491866}
	//
	//
	// Formatted:
	// {
	//   SignedEvent:{
	// 		Event:{
	// 			baseEvent:{
	// 				BaseEvent:{
	// 					epoch:1234
	// 					seq:5762101
	// 					frame:26190
	// 					creator:1892640394
	// 					parents:[[0 0 4 210 0 0 1 244 83 219 144 86 250 59 149 6 145 72 153 169 9 48 136 75 176 41 48 74 182 165 28 34]]
	// 					lamport:1000
	// 					id:[0 0 4 210 0 0 3 232 99 66 83 218 142 123 135 23 142 43 242 47 100 1 175 179 7 225 159 209 170 231 33 188]
	// 	 	    }
	// 			}
	// 			extEventData:{
	// 				version:1
	// 				netForkID:14591
	// 				creationTime:1418823048812169170
	// 				medianTime:13446722647129416609
	// 				prevEpochHash:<nil>
	// 				gasPowerLeft:{Gas:[13708822528903384745 16582119225744604457]}
	// 				gasPowerUsed:8878188716898717410
	// 				extra:[45]
	// 				anyTxs:true
	// 				anyBlockVotes:true
	// 				anyEpochVote:true
	// 				anyMisbehaviourProofs:true
	// 				hasProposal:false
	// 				payloadHash:[146 25 217 255 222 158 125 169 223 156 40 127 240 134 135 235 88 61 242 50 224 40 219 2 138 65 53 11 170 113 173 86]
	// 	    }
	// 	    _baseHash:0xc0016000e0
	// 	    _locatorHash:0xc0016000a0
	// 	}
	// 	sigData: {
	// 			sig:[0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]
	// 	}
	// }
	// payloadData:{
	// 		txs:[0xc002ba2680 0xc002ba2000 0xc002ba2080 0xc002ba20c0 0xc002ba2100 0xc002ba2180 0xc002ba21c0 0xc002ba2200 0xc002ba2240 0xc002ba2280 0xc002ba22c0 0xc002ba2300]
	// 		misbehaviourProofs:[{EventsDoublesign:0xc000036780 BlockVoteDoublesign:<nil> WrongBlockVote:<nil> EpochVoteDoublesign:<nil> WrongEpochVote:<nil>}]
	// 		epochVote:{Epoch:529 Vote:[92 219 5 218 93 62 243 47 154 235 11 183 28 10 120 39 205 19 209 37 29 189 87 214 77 91 190 143 174 130 34 77]}
	// 		blockVotes:{Start:804 Epoch:261 Votes:[[88 216 121 178 206 231 70 34 145 143 252 57 205 16 53 209 11 91 223 244 103 100 251 41 156 63 124 65 184 149 233 219]]}
	// 		payload:{ProposalSyncState:{LastSeenProposalTurn:0 LastSeenProposalFrame:0} Proposal:<nil>}}
	// _size:10491866
	//}

	builder := &MutableEventPayload{}
	builder.SetEpoch(1234)
	builder.SetSeq(5762101)
	builder.SetFrame(26190)
	builder.SetCreator(idx.ValidatorID(1892640394))
	builder.SetParents(hash.Events{{0, 0, 4, 210, 0, 0, 1, 244, 83, 219, 144, 86, 250, 59, 149, 6, 145, 72, 153, 169, 9, 48, 136, 75, 176, 41, 48, 74, 182, 165, 28, 34}})

	builder.SetLamport(1000)

	builder.SetVersion(1)
	builder.SetNetForkID(14591)
	builder.SetCreationTime(1418823048812169170)
	builder.SetMedianTime(13446722647129416609)

	builder.SetGasPowerLeft(GasPowerLeft{
		Gas: [2]uint64{13708822528903384745, 16582119225744604457},
	})
	builder.SetGasPowerUsed(8878188716898717410)
	builder.SetExtra([]byte{45})

	builder.SetTxs([]*types.Transaction{
		types.NewTx(&types.LegacyTx{}),
	})

	input := builder.Build()

	fmt.Printf("%+v\n", *input)
	buf, err := rlp.EncodeToBytes(&input)
	require.NoError(t, err)

	var decoded EventPayload
	err = rlp.DecodeBytes(buf, &decoded)
	require.NoError(t, err)
	t.Fail()
}

func TestEventUnmarshalCSER_Version2FailsIfHashOfEmptyPayloadIsIncluded(t *testing.T) {
	require := require.New(t)

	builder := MutableEventPayload{}
	builder.SetVersion(2)
	builder.SetTxs([]*types.Transaction{
		types.NewTx(&types.LegacyTx{Nonce: 12}),
	})
	event := builder.Build()

	// Deliberately set the hash to the value it should have if the payload was
	// empty. This should be detected by the decoder and identified as an error.
	event.payloadHash = EmptyPayloadHash(2)

	data, err := rlp.EncodeToBytes(&event)
	require.NoError(err)
	require.True(bytes.Contains(data, event.payloadHash[:]))

	var recovered EventPayload
	err = rlp.DecodeBytes(data, &recovered)
	require.ErrorIs(err, cser.ErrNonCanonicalEncoding)
}

func TestEventUnmarshalCSER_Version3AcceptsIfHashOfAnEmptyPayloadIsIncluded(t *testing.T) {
	require := require.New(t)

	builder := MutableEventPayload{}
	builder.SetVersion(3)
	builder.SetPayload(Payload{})
	event := builder.Build()

	require.Equal(event.payloadHash, (&Payload{}).Hash())
	require.Equal(event.payloadHash, EmptyPayloadHash(3))

	data, err := rlp.EncodeToBytes(&event)
	require.NoError(err)

	// The payload hash is always included in version3 events.
	require.True(bytes.Contains(data, event.payloadHash[:]))

	// During decoding, its presence is not considered an error.
	var recovered EventPayload
	err = rlp.DecodeBytes(data, &recovered)
	require.NoError(err)
}

func TestEventUnmarshalCSER_Version3DetectsUnsupportedPayload(t *testing.T) {
	require := require.New(t)

	tests := map[string]*EventPayload{
		"with transactions": func() *EventPayload {
			builder := MutableEventPayload{}
			builder.SetVersion(3)
			builder.SetTxs([]*types.Transaction{
				types.NewTx(&types.LegacyTx{Nonce: 12}),
			})
			return builder.Build()
		}(),
		"with epoch vote": func() *EventPayload {
			builder := MutableEventPayload{}
			builder.SetVersion(3)
			builder.SetEpochVote(LlrEpochVote{
				Epoch: 1,
			})
			return builder.Build()
		}(),
		"with block votes": func() *EventPayload {
			builder := MutableEventPayload{}
			builder.SetVersion(3)
			builder.SetBlockVotes(LlrBlockVotes{
				Start: 1,
				Votes: []hash.Hash{{}, {}},
			})
			return builder.Build()
		}(),
		"with misbehavior proofs": func() *EventPayload {
			builder := MutableEventPayload{}
			builder.SetVersion(3)
			builder.SetMisbehaviourProofs([]MisbehaviourProof{
				{
					EventsDoublesign: &EventsDoublesign{
						Pair: [2]SignedEventLocator{{}, {}},
					},
				},
			})
			return builder.Build()
		}(),
		"with proposal but missing has-proposal flag": func() *EventPayload {
			builder := MutableEventPayload{}
			builder.SetVersion(3)
			builder.SetPayload(Payload{
				Proposal: &Proposal{
					Number: 1,
				},
			})
			builder.hasProposal = false
			return builder.Build()
		}(),
		"without proposal but with has-proposal flag": func() *EventPayload {
			builder := MutableEventPayload{}
			builder.SetVersion(3)
			builder.hasProposal = true
			return builder.Build()
		}(),
	}

	for name, event := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := rlp.EncodeToBytes(event)
			require.ErrorIs(err, ErrSerMalformedEvent)
		})
	}
}

func TestEventPayloadMarshalCSER_DetectsInvalidTransactionEncoding(t *testing.T) {
	require := require.New(t)

	invalidTx := types.NewTx(&types.AccessListTx{
		ChainID: big.NewInt(-1),
	})
	_, want := invalidTx.MarshalBinary()
	require.Error(want)

	builder := MutableEventPayload{}
	builder.SetVersion(3)
	builder.SetPayload(Payload{
		Proposal: &Proposal{
			Transactions: []*types.Transaction{invalidTx},
		},
	})
	event := builder.Build()

	_, err := rlp.EncodeToBytes(&event)
	require.ErrorIs(err, want)
}

func TestEventPayloadUnmarshalCSER_DetectsInvalidPayloadEncoding(t *testing.T) {
	require := require.New(t)

	payload := Payload{ProposalSyncState: ProposalSyncState{
		LastSeenProposalTurn:  123,
		LastSeenProposalFrame: 456,
	}}
	payloadData, err := payload.Serialize()
	require.NoError(err)

	builder := MutableEventPayload{}
	builder.SetVersion(3)
	builder.SetPayload(payload)
	event := builder.Build()

	data, err := rlp.EncodeToBytes(&event)
	require.NoError(err)

	var restored EventPayload
	err = rlp.DecodeBytes(data, &restored)
	require.NoError(err)

	// Corrupt the payload data in the serialized event.
	data = bytes.Replace(data, payloadData, make([]byte, len(payloadData)), 1)
	err = rlp.DecodeBytes(data, &restored)
	require.ErrorContains(err, "invalid wire-format")
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
	for version := range MaxSerializationVersion + 1 {
		b.Run(fmt.Sprintf("version%d", version), func(b *testing.B) {
			e := emptyEvent(0)
			b.ResetTimer()

			for range b.N {
				buf, err := rlp.EncodeToBytes(&e)
				if err != nil {
					b.Fatal(err)
				}
				b.ReportMetric(float64(len(buf)), "size")
			}
		})
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
		switch i % 3 {
		case 0:
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
		case 1:
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
		default: // case 2:
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
					Pair: [2]SignedEventLocator{{}, {}},
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

	if version == 3 {
		random.SetTxs(nil)
		random.SetPayload(Payload{
			ProposalSyncState: ProposalSyncState{
				LastSeenProposalTurn:  Turn(rand.IntN(100)),
				LastSeenProposalFrame: idx.Frame(rand.IntN(100)),
			},
			Proposal: &Proposal{
				Number:       idx.Block(rand.IntN(10_000_000)),
				ParentHash:   common.Hash(randHash(r)),
				Time:         Timestamp(rand.Uint64()),
				RandaoReveal: randao.RandaoReveal(randBytes(r, 64)),
				Transactions: txs,
			},
		})
	}

	random.SetPayloadHash(CalcPayloadHash(random))

	parent := MutableEventPayload{}
	parent.SetVersion(1)
	parent.SetLamport(random.Lamport() - 500)
	parent.SetEpoch(random.Epoch())
	random.SetParents(hash.Events{parent.Build().ID()})

	return random.Build()
}
