package inter

import (
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/randao"
	"github.com/0xsoniclabs/sonic/inter/pb"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestProposal_Hash_IsShaOfFieldConcatenation(t *testing.T) {

	// The procedure of computing the hash of a proposal is critical for
	// the consensus protocol. It is important to ensure that the hash is
	// computed correctly and consistently. Thus, this test provides a
	// second implementation of the hash function. If you have to change this
	// test, make sure you understand the implications on consensus.

	for i := range 5 {

		proposal := &Proposal{
			Number:       idx.Block(1 + i),
			ParentHash:   [32]byte{0: 1, 1: byte(i), 31: 2},
			RandaoReveal: randao.RandaoReveal([]byte{4: 4, 5: byte(i), 63: 5}),
			Transactions: []*types.Transaction{
				types.NewTx(&types.LegacyTx{Nonce: 1}),
				types.NewTx(&types.LegacyTx{Nonce: 2}),
				types.NewTx(&types.LegacyTx{Nonce: uint64(i)}),
			},
		}

		hash := func(proposal *Proposal) hash.Hash {
			data := []byte{}
			data = binary.BigEndian.AppendUint64(data, uint64(proposal.Number))
			data = append(data, proposal.ParentHash[:]...)
			data = append(data, proposal.RandaoReveal[:]...)
			for _, tx := range proposal.Transactions {
				txHash := tx.Hash()
				data = append(data, txHash[:]...)
			}
			return sha256.Sum256(data)
		}

		require.Equal(t, proposal.Hash(), hash(proposal))
	}
}

func TestProposal_Hash_ModifyingContent_ChangesHash(t *testing.T) {
	tests := map[string]func(*Proposal){
		"change number": func(p *Proposal) {
			p.Number = p.Number + 1
		},
		"change parent hash": func(p *Proposal) {
			p.ParentHash[0] = p.ParentHash[0] + 1
		},
		"change randao reveal": func(p *Proposal) {
			p.RandaoReveal[0] = p.RandaoReveal[0] + 1
		},
		"change transaction": func(p *Proposal) {
			p.Transactions[0] = types.NewTx(&types.LegacyTx{Nonce: 3})
		},
		"add transaction": func(p *Proposal) {
			p.Transactions = append(p.Transactions, types.NewTx(
				&types.LegacyTx{Nonce: 4},
			))
		},
		"remove transaction": func(p *Proposal) {
			p.Transactions = p.Transactions[:len(p.Transactions)-1]
		},
	}

	for name, modify := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			proposal := &Proposal{
				Number:       1,
				ParentHash:   [32]byte{1},
				RandaoReveal: randao.RandaoReveal([]byte{3, 63: 0}),
				Transactions: []*types.Transaction{
					types.NewTx(&types.LegacyTx{Nonce: 1}),
					types.NewTx(&types.LegacyTx{Nonce: 2}),
				},
			}

			hashBefore := proposal.Hash()
			modify(proposal)
			hashAfter := proposal.Hash()

			require.NotEqual(hashBefore, hashAfter)
		})
	}
}

func TestProposal_CanBeSerializedAndRestored(t *testing.T) {
	require := require.New(t)
	original := &Proposal{
		Number:       1,
		ParentHash:   [32]byte{2},
		RandaoReveal: randao.RandaoReveal([]byte{4, 64: 0}),
		Transactions: []*types.Transaction{
			types.NewTx(&types.LegacyTx{Nonce: 1}),
			types.NewTx(&types.LegacyTx{Nonce: 2}),
		},
	}

	data, err := original.Serialize()
	require.NoError(err)
	require.NotEmpty(data)
	restored := &Proposal{}
	err = restored.Deserialize(data)
	require.NoError(err)

	// Check individual fields. Note: a full Deep-Equal comparison is not
	// possible because transactions have insignificant meta-information that
	// is not serialized and restored.
	require.Equal(original.Number, restored.Number)
	require.Equal(original.ParentHash, restored.ParentHash)
	require.Equal(original.RandaoReveal, restored.RandaoReveal)

	require.Equal(len(original.Transactions), len(restored.Transactions))
	for i := range original.Transactions {
		require.Equal(original.Transactions[i].Hash(), restored.Transactions[i].Hash())
	}

	require.Equal(original.Hash(), restored.Hash())
}

func TestProposal_Serialize_FailsOnInvalidTransaction(t *testing.T) {
	require := require.New(t)

	// Negative chain IDs are invalid and fail the serialization.
	invalidTx := types.NewTx(&types.AccessListTx{
		ChainID: big.NewInt(-1),
	})
	_, want := invalidTx.MarshalBinary()
	require.Error(want)

	// The serialization of a proposal containing an invalid transaction
	// should also fail.
	proposal := &Proposal{Transactions: []*types.Transaction{invalidTx}}
	_, got := proposal.Serialize()
	require.Error(got)
	require.Equal(want, got)
}

func TestProposal_Deserialize_FailsOnInvalidInputData(t *testing.T) {
	require := require.New(t)
	invalidData := []byte{0, 1, 2, 3, 4, 5}
	restored := &Proposal{}
	err := restored.Deserialize(invalidData)
	require.Error(err)
	require.ErrorContains(err, "invalid wire-format")
}

func TestProposal_fromProto_FailsOnInvalidTransaction(t *testing.T) {
	require := require.New(t)

	invalidTxEncoding := []byte{0, 1, 2, 3, 4, 5}
	var tx types.Transaction
	want := tx.UnmarshalBinary(invalidTxEncoding)
	require.Error(want)

	pbProposal := &pb.Proposal{
		Transactions: []*pb.Transaction{
			{Encoded: invalidTxEncoding},
		},
	}

	proposal := &Proposal{}
	got := proposal.fromProto(pbProposal)
	require.Error(got)
	require.Equal(want, got)
}

func TestProposal_fromProto_FailsOnNilTransaction(t *testing.T) {
	require := require.New(t)

	pbProposal := &pb.Proposal{
		Transactions: []*pb.Transaction{nil},
	}

	proposal := &Proposal{}
	got := proposal.fromProto(pbProposal)
	require.ErrorContains(got, "nil transaction in proposal")
}

func TestProposal_fromProto_FailsOnEmptyTransactionEncoding(t *testing.T) {
	require := require.New(t)

	pbProposal := &pb.Proposal{
		Transactions: []*pb.Transaction{
			{Encoded: nil},
		},
	}

	proposal := &Proposal{}
	got := proposal.fromProto(pbProposal)
	require.ErrorContains(got, "typed transaction too short")
}
