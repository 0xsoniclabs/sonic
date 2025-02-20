package cert

import (
	"fmt"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert/pb"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/protobuf/proto"
)

func SerializeCommitteeCertificate(
	certificate Certificate[CommitteeStatement],
) ([]byte, error) {
	return marshalCommitteeCertificate(
		&certificate.subject,
		toProtoSignature(&certificate.signature),
	)
}

func DeserializeCommitteeCertificate(data []byte) (
	Certificate[CommitteeStatement],
	error,
) {
	return unmarshalCommitteeCertificate(data)
}

func SerializeBlockCertificate(
	certificate Certificate[BlockStatement],
) ([]byte, error) {
	return marshalBlockCertificate(
		&certificate.subject,
		toProtoSignature(&certificate.signature),
	)
}

func DeserializeBlockCertificate(data []byte) (
	Certificate[BlockStatement],
	error,
) {
	return unmarshalBlockCertificate(data)
}

// --- internal ---

func marshalCommitteeCertificate(
	statement *CommitteeStatement,
	signature *pb.AggregatedSignature,
) ([]byte, error) {
	members := []*pb.Member{}
	for _, member := range statement.Committee.Members() {
		key := member.PublicKey.Serialize()
		proof := member.ProofOfPossession.Serialize()
		members = append(members, &pb.Member{
			PublicKey:         key[:],
			ProofOfPossession: proof[:],
			VotingPower:       member.VotingPower,
		})
	}

	return proto.Marshal(&pb.CommitteeCertificate{
		ChainId:   statement.ChainId,
		Period:    statement.Period,
		Members:   members,
		Signature: signature,
	})
}

func unmarshalCommitteeCertificate(data []byte) (Certificate[CommitteeStatement], error) {
	var none Certificate[CommitteeStatement]
	var pb pb.CommitteeCertificate
	if err := proto.Unmarshal(data, &pb); err != nil {
		return none, err
	}
	signature, err := fromProtoSignature[CommitteeStatement](pb.Signature)
	if err != nil {
		return none, fmt.Errorf("failed to decode signature, %w", err)
	}

	members := make([]scc.Member, 0, len(pb.Members))
	for _, pbMember := range pb.Members {
		if len(pbMember.PublicKey) != 48 {
			return none, fmt.Errorf("invalid public key length: %d", len(pbMember.PublicKey))
		}
		key, err := bls.DeserializePublicKey([48]byte(pbMember.PublicKey))
		if err != nil {
			return none, fmt.Errorf("failed to decode public key, %w", err)
		}

		if len(pbMember.ProofOfPossession) != 96 {
			return none, fmt.Errorf("invalid proof of possession length: %d", len(pbMember.ProofOfPossession))
		}
		proof, err := bls.DeserializeSignature([96]byte(pbMember.ProofOfPossession))
		if err != nil {
			return none, fmt.Errorf("failed to decode proof of possession, %w", err)
		}

		members = append(members, scc.Member{
			PublicKey:         key,
			ProofOfPossession: proof,
			VotingPower:       pbMember.VotingPower,
		})
	}

	return Certificate[CommitteeStatement]{
		subject: CommitteeStatement{
			statement: statement{
				ChainId: pb.ChainId,
			},
			Period:    pb.Period,
			Committee: scc.NewCommittee(members...),
		},
		signature: signature,
	}, nil
}

func marshalBlockCertificate(
	statement *BlockStatement,
	signature *pb.AggregatedSignature,
) ([]byte, error) {
	return proto.Marshal(&pb.BlockCertificate{
		ChainId:   statement.ChainId,
		Number:    statement.Number,
		Hash:      statement.Hash[:],
		StateRoot: statement.StateRoot[:],
		Signature: signature,
	})
}

func unmarshalBlockCertificate(data []byte) (Certificate[BlockStatement], error) {
	var none Certificate[BlockStatement]
	var pb pb.BlockCertificate
	if err := proto.Unmarshal(data, &pb); err != nil {
		return none, err
	}

	if len(pb.Hash) != 32 {
		return none, fmt.Errorf("invalid hash length: %d", len(pb.Hash))
	}
	if len(pb.StateRoot) != 32 {
		return none, fmt.Errorf("invalid state root length: %d", len(pb.StateRoot))
	}

	signature, err := fromProtoSignature[BlockStatement](pb.Signature)
	if err != nil {
		return none, fmt.Errorf("failed to decode signature, %w", err)
	}

	return Certificate[BlockStatement]{
		subject: BlockStatement{
			statement: statement{
				ChainId: pb.ChainId,
			},
			Number:    pb.Number,
			Hash:      common.Hash(pb.Hash),
			StateRoot: common.Hash(pb.StateRoot),
		},
		signature: signature,
	}, nil
}

func toProtoSignature[S Statement](
	signature *AggregatedSignature[S],
) *pb.AggregatedSignature {
	sig := signature.Signature.Serialize()
	return &pb.AggregatedSignature{
		SignerMask: signature.Signers.mask,
		Signature:  sig[:],
	}
}

func fromProtoSignature[S Statement](
	pb *pb.AggregatedSignature,
) (AggregatedSignature[S], error) {
	if len(pb.Signature) != 96 {
		return AggregatedSignature[S]{}, fmt.Errorf("invalid signature length: %d", len(pb.Signature))
	}

	signature, err := bls.DeserializeSignature([96]byte(pb.Signature))
	if err != nil {
		return AggregatedSignature[S]{}, fmt.Errorf("failed to decode signature, %w", err)
	}

	return AggregatedSignature[S]{
		Signers:   BitSet[scc.MemberId]{mask: pb.SignerMask},
		Signature: signature,
	}, nil
}
