package cert

import (
	"fmt"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert/pb"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/protobuf/proto"
)

// CommitteeCertificate is a certificate for a committee. It is a specialization
// of the generic Certificate type offering additional methods for serialization.
type CommitteeCertificate Certificate[CommitteeStatement]

// Serialize serializes the certificate into a byte slice. The internal encoding
// uses the protobuf format, enabling future backward compatibility.
func (c *CommitteeCertificate) Serialize() ([]byte, error) {
	return marshalCommitteeCertificate(c)
}

// Deserialize restores the certificate from a byte slice. The accepted input
// data must follow the protobuf format used by the Serialize method.
func (c *CommitteeCertificate) Deserialize(data []byte) error {
	restored, err := unmarshalCommitteeCertificate(data)
	if err != nil {
		return err
	}
	*c = restored
	return nil
}

// BlockCertificate is a certificate for a block. It is a specialization of the
// generic Certificate type offering additional methods for serialization.
type BlockCertificate Certificate[BlockStatement]

// Serialize serializes the certificate into a byte slice. The internal encoding
// uses the protobuf format, enabling future backward compatibility.
func (c *BlockCertificate) Serialize() ([]byte, error) {
	return marshalBlockCertificate(c)
}

// Deserialize restores the certificate from a byte slice. The accepted input
// data must follow the protobuf format used by the Serialize method.
func (c *BlockCertificate) Deserialize(data []byte) error {
	restored, err := unmarshalBlockCertificate(data)
	if err != nil {
		return err
	}
	*c = restored
	return nil
}

// --- internal ---

func marshalCommitteeCertificate(cert *CommitteeCertificate) ([]byte, error) {
	var members []*pb.Member
	for _, member := range cert.subject.Committee.Members() {
		key := member.PublicKey.Serialize()
		proof := member.ProofOfPossession.Serialize()
		members = append(members, &pb.Member{
			PublicKey:         key[:],
			ProofOfPossession: proof[:],
			VotingPower:       member.VotingPower,
		})
	}

	return proto.Marshal(&pb.CommitteeCertificate{
		ChainId:   cert.subject.ChainId,
		Period:    uint64(cert.subject.Period),
		Members:   members,
		Signature: toProtoSignature(&cert.signature),
	})
}

func unmarshalCommitteeCertificate(data []byte) (CommitteeCertificate, error) {
	var none CommitteeCertificate
	var pb pb.CommitteeCertificate
	if err := proto.Unmarshal(data, &pb); err != nil {
		return none, err
	}
	signature, err := fromProtoSignature[CommitteeStatement](pb.Signature)
	if err != nil {
		return none, fmt.Errorf("failed to decode signature, %w", err)
	}

	var members []scc.Member
	if len(pb.Members) > 0 {
		members = make([]scc.Member, 0, len(pb.Members))
	}
	for _, cur := range pb.Members {
		if len := len(cur.PublicKey); len != 48 {
			return none, fmt.Errorf("invalid public key length: %d", len)
		}
		key, err := bls.DeserializePublicKey([48]byte(cur.PublicKey))
		if err != nil {
			return none, fmt.Errorf("failed to decode public key, %w", err)
		}

		if len := len(cur.ProofOfPossession); len != 96 {
			return none, fmt.Errorf("invalid proof of possession length: %d", len)
		}
		proof, err := bls.DeserializeSignature([96]byte(cur.ProofOfPossession))
		if err != nil {
			return none, fmt.Errorf("failed to decode proof of possession, %w", err)
		}

		members = append(members, scc.Member{
			PublicKey:         key,
			ProofOfPossession: proof,
			VotingPower:       cur.VotingPower,
		})
	}

	return CommitteeCertificate{
		subject: CommitteeStatement{
			statement: statement{
				ChainId: pb.ChainId,
			},
			Period:    scc.Period(pb.Period),
			Committee: scc.NewCommittee(members...),
		},
		signature: signature,
	}, nil
}

func marshalBlockCertificate(cert *BlockCertificate) ([]byte, error) {
	return proto.Marshal(&pb.BlockCertificate{
		ChainId:   cert.subject.ChainId,
		Number:    uint64(cert.subject.Number),
		Hash:      cert.subject.Hash[:],
		StateRoot: cert.subject.StateRoot[:],
		Signature: toProtoSignature(&cert.signature),
	})
}

func unmarshalBlockCertificate(data []byte) (BlockCertificate, error) {
	var none BlockCertificate
	var pb pb.BlockCertificate
	if err := proto.Unmarshal(data, &pb); err != nil {
		return none, err
	}

	if len := len(pb.Hash); len != 32 {
		return none, fmt.Errorf("invalid hash length: %d", len)
	}
	if len := len(pb.StateRoot); len != 32 {
		return none, fmt.Errorf("invalid state root length: %d", len)
	}

	signature, err := fromProtoSignature[BlockStatement](pb.Signature)
	if err != nil {
		return none, fmt.Errorf("failed to decode signature, %w", err)
	}

	return BlockCertificate{
		subject: BlockStatement{
			statement: statement{
				ChainId: pb.ChainId,
			},
			Number:    idx.Block(pb.Number),
			Hash:      common.Hash(pb.Hash),
			StateRoot: common.Hash(pb.StateRoot),
		},
		signature: signature,
	}, nil
}

func toProtoSignature[S Statement](
	signature *AggregatedSignature[S],
) *pb.AggregatedSignature {
	sig := signature.signature.Serialize()
	return &pb.AggregatedSignature{
		SignerMask: signature.signers.mask,
		Signature:  sig[:],
	}
}

func fromProtoSignature[S Statement](
	pb *pb.AggregatedSignature,
) (AggregatedSignature[S], error) {
	var none AggregatedSignature[S]
	if pb == nil {
		return none, fmt.Errorf("no signature")
	}
	if len := len(pb.Signature); len != 96 {
		return none, fmt.Errorf("invalid signature length: %d", len)
	}

	signature, err := bls.DeserializeSignature([96]byte(pb.Signature))
	if err != nil {
		return none, fmt.Errorf("failed to decode signature, %w", err)
	}

	return AggregatedSignature[S]{
		signers:   BitSet[scc.MemberId]{mask: pb.SignerMask},
		signature: signature,
	}, nil
}
