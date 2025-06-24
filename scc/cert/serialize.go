// Copyright 2025 Sonic Operations Ltd
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

// Serialize serializes the certificate into a byte slice. The internal encoding
// uses the protobuf format, enabling future backward compatibility.
func (c Certificate[S]) Serialize() ([]byte, error) {
	if subject, ok := any(&c.subject).(serializableSubject); ok {
		return subject.serialize(toProtoSignature(&c.signature))
	}
	return nil, fmt.Errorf("unsupported subject type: %T", c.subject)
}

// Deserialize restores the certificate from a byte slice. The accepted input
// data must follow the protobuf format used by the Serialize method.
func (c *Certificate[S]) Deserialize(data []byte) error {
	subject, ok := any(&c.subject).(serializableSubject)
	if !ok {
		return fmt.Errorf("unsupported subject type: %T", c.subject)
	}
	sig, err := subject.deserialize(data)
	if err != nil {
		return err
	}
	c.signature, err = fromProtoSignature[S](sig)
	return err
}

// --- internal ---

type serializableSubject interface {
	serialize(*pb.AggregatedSignature) ([]byte, error)
	deserialize([]byte) (*pb.AggregatedSignature, error)
}

func (s CommitteeStatement) serialize(signature *pb.AggregatedSignature) ([]byte, error) {
	var members []*pb.Member
	for _, member := range s.Committee.Members() {
		key := member.PublicKey.Serialize()
		proof := member.ProofOfPossession.Serialize()
		members = append(members, &pb.Member{
			PublicKey:         key[:],
			ProofOfPossession: proof[:],
			VotingPower:       member.VotingPower,
		})
	}

	return proto.Marshal(&pb.CommitteeCertificate{
		ChainId:   s.ChainId,
		Period:    uint64(s.Period),
		Members:   members,
		Signature: signature,
	})
}

func (s *CommitteeStatement) deserialize(data []byte) (*pb.AggregatedSignature, error) {
	var pb pb.CommitteeCertificate
	if err := proto.Unmarshal(data, &pb); err != nil {
		return nil, err
	}

	var members []scc.Member
	if len(pb.Members) > 0 {
		members = make([]scc.Member, 0, len(pb.Members))
	}
	for _, cur := range pb.Members {
		if len := len(cur.PublicKey); len != 48 {
			return nil, fmt.Errorf("invalid public key length: %d", len)
		}
		key, err := bls.DeserializePublicKey([48]byte(cur.PublicKey))
		if err != nil {
			return nil, fmt.Errorf("failed to decode public key, %w", err)
		}

		if len := len(cur.ProofOfPossession); len != 96 {
			return nil, fmt.Errorf("invalid proof of possession length: %d", len)
		}
		proof, err := bls.DeserializeSignature([96]byte(cur.ProofOfPossession))
		if err != nil {
			return nil, fmt.Errorf("failed to decode proof of possession, %w", err)
		}

		members = append(members, scc.Member{
			PublicKey:         key,
			ProofOfPossession: proof,
			VotingPower:       cur.VotingPower,
		})
	}

	*s = CommitteeStatement{
		statement: statement{
			ChainId: pb.ChainId,
		},
		Period:    scc.Period(pb.Period),
		Committee: scc.NewCommittee(members...),
	}

	return pb.Signature, nil
}

func (s BlockStatement) serialize(signature *pb.AggregatedSignature) ([]byte, error) {
	return proto.Marshal(&pb.BlockCertificate{
		ChainId:   s.ChainId,
		Number:    uint64(s.Number),
		Hash:      s.Hash[:],
		StateRoot: s.StateRoot[:],
		Signature: signature,
	})
}

func (s *BlockStatement) deserialize(data []byte) (*pb.AggregatedSignature, error) {
	var pb pb.BlockCertificate
	if err := proto.Unmarshal(data, &pb); err != nil {
		return nil, err
	}

	if len := len(pb.Hash); len != 32 {
		return nil, fmt.Errorf("invalid hash length: %d", len)
	}
	if len := len(pb.StateRoot); len != 32 {
		return nil, fmt.Errorf("invalid state root length: %d", len)
	}

	*s = BlockStatement{
		statement: statement{
			ChainId: pb.ChainId,
		},
		Number:    idx.Block(pb.Number),
		Hash:      common.Hash(pb.Hash),
		StateRoot: common.Hash(pb.StateRoot),
	}

	return pb.Signature, nil
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
