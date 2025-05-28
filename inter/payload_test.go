package inter

import (
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/pb"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestPayload_Hash_IsShaOfFieldConcatenation(t *testing.T) {

	// The procedure of computing the hash of a payload is critical for
	// the consensus protocol. It is important to ensure that the hash is
	// computed correctly and consistently. Thus, this test provides a
	// second implementation of the hash function. If you have to change this
	// test, make sure you understand the implications on consensus.

	for i := range 5 {

		payload := &Payload{
			ProposalSyncState: ProposalSyncState{
				LastSeenProposalTurn:  Turn(1 + i),
				LastSeenProposalFrame: idx.Frame(2 + i),
			},
			Proposal: &Proposal{
				Number: idx.Block(3 + i),
			},
		}

		data := []byte{currentPayloadVersion}
		data = binary.BigEndian.AppendUint32(data, uint32(payload.LastSeenProposalTurn))
		data = binary.BigEndian.AppendUint32(data, uint32(payload.LastSeenProposalFrame))
		proposalHash := payload.Proposal.Hash()
		data = append(data, proposalHash[:]...)
		require.Equal(t, hash.Hash(sha256.Sum256(data)), payload.Hash())
	}
}

func TestPayload_Hash_MissingPayloadIsOmittedInHashInput(t *testing.T) {
	payload := &Payload{
		ProposalSyncState: ProposalSyncState{
			LastSeenProposalTurn:  1,
			LastSeenProposalFrame: 2,
		},
		Proposal: nil,
	}

	data := []byte{currentPayloadVersion}
	data = binary.BigEndian.AppendUint32(data, uint32(payload.LastSeenProposalTurn))
	data = binary.BigEndian.AppendUint32(data, uint32(payload.LastSeenProposalFrame))
	require.Equal(t, hash.Hash(sha256.Sum256(data)), payload.Hash())
}

func TestPayload_Hash_ModifyingContent_ChangesHash(t *testing.T) {
	tests := map[string]func(*Payload){
		"change last seen proposal turn": func(p *Payload) {
			p.LastSeenProposalTurn = p.LastSeenProposalTurn + 1
		},
		"change last seen proposal frame": func(p *Payload) {
			p.LastSeenProposalFrame = p.LastSeenProposalFrame + 1
		},
		"change proposal": func(p *Payload) {
			p.Proposal = &Proposal{
				Number: p.Proposal.Number + 1,
			}
		},
		"remove proposal": func(p *Payload) {
			p.Proposal = nil
		},
	}

	for name, modify := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			payload := &Payload{
				ProposalSyncState: ProposalSyncState{
					LastSeenProposalTurn:  1,
					LastSeenProposalFrame: 2,
				},
				Proposal: &Proposal{
					Number: 4,
				},
			}

			hashBefore := payload.Hash()
			modify(payload)
			hashAfter := payload.Hash()

			require.NotEqual(hashBefore, hashAfter)
		})
	}
}

func TestPayload_CanBeSerializedAndRestored(t *testing.T) {
	for _, proposal := range []*Proposal{nil, {}} {
		require := require.New(t)
		original := &Payload{
			ProposalSyncState: ProposalSyncState{
				LastSeenProposalTurn:  1,
				LastSeenProposalFrame: 2,
			},
			Proposal: proposal,
		}

		data, err := original.Serialize()
		require.NoError(err)
		require.NotEmpty(data)
		restored := &Payload{}
		err = restored.Deserialize(data)
		require.NoError(err)

		// Check individual fields. Note: a full Deep-Equal comparison is not
		// possible because transactions have insignificant meta-information that
		// is not serialized and restored.
		require.Equal(original.LastSeenProposalTurn, restored.LastSeenProposalTurn)
		require.Equal(original.LastSeenProposalFrame, restored.LastSeenProposalFrame)

		if original.Proposal == nil {
			require.Nil(restored.Proposal)
		} else {
			require.NotNil(restored.Proposal)
			require.Equal(original.Proposal.Number, restored.Proposal.Number)
		}

		require.Equal(original.Hash(), restored.Hash())
	}
}

func TestPayload_Serialize_InvalidTransaction_FailsSerialization(t *testing.T) {
	require := require.New(t)

	// Negative chain IDs are invalid and fail the serialization.
	invalidTx := types.NewTx(&types.AccessListTx{
		ChainID: big.NewInt(-1),
	})
	_, want := invalidTx.MarshalBinary()
	require.Error(want)

	payload := &Payload{
		Proposal: &Proposal{
			Transactions: []*types.Transaction{invalidTx},
		},
	}

	_, got := payload.Serialize()
	require.Error(got)
	require.Equal(want, got)
}

func TestPayload_Deserialize_InvalidEncoding_FailsDecoding(t *testing.T) {
	require := require.New(t)
	var payload Payload
	want := payload.Deserialize([]byte{0, 1})
	require.Error(want)
	require.ErrorContains(want, "invalid wire-format")
}

func TestPayload_Deserialize_UnsupportedVersionNumber_FailsDecoding(t *testing.T) {
	require := require.New(t)

	pb := pb.Payload{Version: currentPayloadVersion + 1}
	data, err := proto.Marshal(&pb)
	require.NoError(err)

	var payload Payload
	want := payload.Deserialize(data)
	require.Error(want)
	require.ErrorContains(want, "unsupported payload version")
}

func TestPayload_Deserialize_InvalidTransaction_FailsDecoding(t *testing.T) {
	require := require.New(t)

	invalidTxEncoding := []byte{0, 1}
	var tx types.Transaction
	want := tx.UnmarshalBinary(invalidTxEncoding)
	require.Error(want)

	pb := &pb.Payload{
		Version: currentPayloadVersion,
		Proposal: &pb.Proposal{
			Transactions: []*pb.Transaction{
				{Encoded: invalidTxEncoding},
			},
		},
	}
	data, err := proto.Marshal(pb)
	require.NoError(err)

	payload := &Payload{}
	got := payload.Deserialize(data)
	require.Error(got)
	require.Equal(want, got)
}
