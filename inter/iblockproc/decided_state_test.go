// Copyright 2026 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package iblockproc

import (
	"bytes"
	"crypto/sha256"
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/drivertype"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func TestEpochState_Hash_BeforeBrio_HashDoesNotCoverV2Fields(t *testing.T) {
	require := require.New(t)
	state := makeExampleEpochState()
	state.Rules.Upgrades.Brio = false

	hasher := sha256.New()
	require.NoError(rlp.Encode(hasher, state.asV1()))
	wantedHash := hash.BytesToHash(hasher.Sum(nil))

	hashBeforeBrio := state.Hash()
	require.Equal(wantedHash, hashBeforeBrio)

	// Changing V2 fields does not change the hash.
	state.EpochEndBlockHash[5]++
	require.Equal(wantedHash, state.Hash())

	state.EpochEndExecutionPlanChainHash[7]++
	require.Equal(wantedHash, state.Hash())

	state.EpochSealingTxHashes = append(state.EpochSealingTxHashes, hash.Hash{1, 2, 3})
	require.Equal(wantedHash, state.Hash())
}

func TestEpochState_Hash_AfterBrio_CoversV2Fields(t *testing.T) {
	require := require.New(t)
	state := makeExampleEpochState()
	state.Rules.Upgrades.Brio = true

	hasher := sha256.New()
	require.NoError(rlp.Encode(hasher, state.asV1()))
	v1Hash := hash.BytesToHash(hasher.Sum(nil))

	hashAfterBrio := state.Hash()
	require.NotEqual(v1Hash, hashAfterBrio)

	// Changes to V2 fields do change the hash.
	state.EpochEndBlockHash[5]++
	require.NotEqual(hashAfterBrio, state.Hash())
	state.EpochEndBlockHash[5]--
	require.Equal(hashAfterBrio, state.Hash())

	state.EpochEndExecutionPlanChainHash[7]++
	require.NotEqual(hashAfterBrio, state.Hash())
	state.EpochEndExecutionPlanChainHash[7]--
	require.Equal(hashAfterBrio, state.Hash())

	state.EpochSealingTxHashes = append(state.EpochSealingTxHashes, hash.Hash{1, 2, 3})
	require.NotEqual(hashAfterBrio, state.Hash())

	beforeTxHashUpdate := state.Hash()
	state.EpochSealingTxHashes[0][12]++
	require.NotEqual(beforeTxHashUpdate, state.Hash())
	state.EpochSealingTxHashes[0][12]--
	require.Equal(beforeTxHashUpdate, state.Hash())
}

func TestEpochState_Copy_CreatesDeepCopy(t *testing.T) {
	require := require.New(t)
	orig := makeExampleEpochState()
	copy := orig.Copy()

	require.Equal(orig, copy)

	// Check that copy was indeed a deep copy.
	require.NotSame(
		&orig.ValidatorStates[0],
		&copy.ValidatorStates[0],
	)
	require.NotSame(
		orig.ValidatorProfiles[1].Weight,
		copy.ValidatorProfiles[1].Weight,
	)
	require.NotSame(
		orig.Rules.Economy.MinGasPrice,
		copy.Rules.Economy.MinGasPrice,
	)
	require.NotSame(
		&orig.EpochSealingTxHashes[1],
		&copy.EpochSealingTxHashes[1],
	)
}

func TestEpochState_asV1_ExtractsV1Fields(t *testing.T) {
	original := makeExampleEpochStateV1()
	state := makeEpochStateFromV1(original)
	restored := state.asV1()
	require.Equal(t, original, restored)
}

func TestEpochState_CanBeSerializedAndDeserialized(t *testing.T) {
	state := makeExampleEpochState()
	serialized, err := rlp.EncodeToBytes(state)
	require.NoError(t, err)

	var deserialized EpochState
	err = rlp.DecodeBytes(serialized, &deserialized)
	require.NoError(t, err)
	require.Equal(t, state, deserialized)
}

func TestEpochState_Decode_CanBeDecodedFromV1Serialization(t *testing.T) {
	// This test verifies that V2 can be read from the encoding of V1, using
	// default values for the added fields. This ensures backward compatibility,
	// and enables reading state information from a DB or other file containing
	// serialized V1 data.
	var v1 EpochStateV1 = makeExampleEpochStateV1()
	data, err := rlp.EncodeToBytes(v1)
	require.NoError(t, err)

	var state EpochState
	require.NoError(t, rlp.DecodeBytes(data, &state))

	require.Equal(t, v1, state.asV1())
	require.Zero(t, state.EpochEndBlockHash)
	require.Zero(t, state.EpochEndExecutionPlanChainHash)
	require.Zero(t, state.EpochSealingTxHashes)
	require.Equal(t, makeEpochStateFromV1(v1), state)
}

func TestEpochState_Decode_DecodingFromV1Serialization_ResetV2Fields(t *testing.T) {
	var v1 EpochStateV1 = makeExampleEpochStateV1()
	data, err := rlp.EncodeToBytes(v1)
	require.NoError(t, err)

	// write some state into target state before restoring
	state := makeExampleEpochState()
	require.NotZero(t, state.EpochEndBlockHash)
	require.NotZero(t, state.EpochEndExecutionPlanChainHash)
	require.NotZero(t, state.EpochSealingTxHashes)

	// Decoding should cause a reset
	require.NoError(t, rlp.DecodeBytes(data, &state))

	require.Equal(t, v1, state.asV1())
	require.Zero(t, state.EpochEndBlockHash)
	require.Zero(t, state.EpochEndExecutionPlanChainHash)
	require.Zero(t, state.EpochSealingTxHashes)
	require.Equal(t, makeEpochStateFromV1(v1), state)
}

func TestEpochState_DecodeRLP_FailsOnInvalidInput(t *testing.T) {
	in := rlp.NewStream(bytes.NewBuffer([]byte("not a valid rlp encoding")), 0)
	state := EpochState{}
	require.Error(t, state.DecodeRLP(in))
}

func TestEpochState_DecodeRLP_FailsOnIncorrectInputLength(t *testing.T) {

	state := makeExampleEpochState()
	encoded, err := rlp.EncodeToBytes(state)
	require.NoError(t, err)

	// get individual fields as a list
	var list []any
	require.NoError(t, rlp.DecodeBytes(encoded, &list))

	numV2Fields := len(list)
	numV1Fields := numV2Fields - 3

	list = append(list, list...)

	for i := range list {
		encode, err := rlp.EncodeToBytes(list[:i])
		require.NoError(t, err)

		var restored EpochState
		switch i {
		case numV1Fields:
			// All V1 fields are here, the V1 part should have been decoded.
			require.NoError(t, rlp.DecodeBytes(encode, &restored))
			require.Equal(t, makeEpochStateFromV1(state.asV1()), restored)
		case numV2Fields:
			// All V2 fields are here, the state should have been fully decoded.
			require.NoError(t, rlp.DecodeBytes(encode, &restored))
			require.Equal(t, state, restored)
		default:
			// Too little or too many fields should cause failures.
			err := rlp.DecodeBytes(encode, &restored)
			require.Error(t, err)
			if i < numV2Fields {
				require.ErrorContains(t, err, "end of list")
			} else {
				require.ErrorContains(t, err, "ListEnd not positioned at EOL")
			}
		}
	}
}

func TestEpochState_DecodeRLP_FailsOnIncorrectListElementEncoding(t *testing.T) {

	state := makeExampleEpochState()
	encoded, err := rlp.EncodeToBytes(state)
	require.NoError(t, err)

	// get individual fields as a list
	var list []any
	require.NoError(t, rlp.DecodeBytes(encoded, &list))

	for i := range list {
		// The full list can be parsed
		copy := slices.Clone(list)
		encoded, err := rlp.EncodeToBytes(copy)
		require.NoError(t, err)

		var restored EpochState
		require.NoError(t, rlp.DecodeBytes(encoded, &restored))
		require.Equal(t, restored, state)

		// Swapping element i between a list and scalar breaks the decoding.
		if _, isList := copy[i].([]any); isList {
			copy[i] = uint(12)
		} else {
			copy[i] = []any{uint(12), uint(14)}
		}

		// The modified list should no longer be decodable.
		encoded, err = rlp.EncodeToBytes(copy)
		require.NoError(t, err)

		require.Error(t, rlp.DecodeBytes(encoded, &restored))
	}
}

// --- example data ---

func makeEpochStateFromV1(v1 EpochStateV1) EpochState {
	return EpochState{
		Epoch:             v1.Epoch,
		EpochStart:        v1.EpochStart,
		PrevEpochStart:    v1.PrevEpochStart,
		EpochStateRoot:    v1.EpochStateRoot,
		Validators:        v1.Validators,
		ValidatorStates:   v1.ValidatorStates,
		ValidatorProfiles: v1.ValidatorProfiles,
		Rules:             v1.Rules,
	}
}

func makeExampleEpochState() EpochState {
	v1 := makeExampleEpochStateV1()
	res := makeEpochStateFromV1(v1)
	res.Rules.Upgrades.Brio = true
	res.EpochEndBlockHash = hash.Hash{0xBE, 0xEF}
	res.EpochEndExecutionPlanChainHash = hash.Hash{0xFE, 0xED}
	res.EpochSealingTxHashes = []hash.Hash{{0xBA, 0xAD}, {0xF0, 0x0D}}
	return res
}

func makeExampleEpochStateV1() EpochStateV1 {
	validatorBuilder := pos.NewBuilder()
	validatorBuilder.Set(1, 200)
	validatorBuilder.Set(2, 100)
	validators := validatorBuilder.Build()

	return EpochStateV1{
		Epoch:          123,
		EpochStart:     456,
		PrevEpochStart: 789,
		EpochStateRoot: hash.Hash{0xAC, 0xDC},
		Validators:     validators,
		ValidatorStates: []ValidatorEpochState{
			{
				GasRefund: 1000,
				PrevEpochEvent: EventInfo{
					ID: hash.Event{0xDE, 0xAD},
					GasPowerLeft: inter.GasPowerLeft{
						Gas: [2]uint64{10, 20},
					},
					Time: 123456789,
				},
			},
		},
		ValidatorProfiles: map[idx.ValidatorID]drivertype.Validator{
			1: {Weight: big.NewInt(50), PubKey: validatorpk.PubKey{Type: 2, Raw: []byte{1, 2, 3}}},
			2: {Weight: big.NewInt(100), PubKey: validatorpk.PubKey{Type: 3, Raw: []byte{4, 5}}},
		},
		Rules: opera.MainNetRules(),
	}
}
