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
	"github.com/holiman/uint256"
)

// AggregatedSignature represents an aggregated BLS signature from a committee.
type AggregatedSignature[S Statement] struct {
	signers   BitSet[scc.MemberId]
	signature bls.Signature
}

// NewAggregatedSignature creates a new aggregated signature using the given signers
// and signature. All the parameters are shallow copied.
func NewAggregatedSignature[S Statement](
	signers BitSet[scc.MemberId],
	signature bls.Signature) AggregatedSignature[S] {
	return AggregatedSignature[S]{signers: signers, signature: signature}
}

// Signers returns the signers of the aggregated signature.
// The returned set of signers is a shallow copy of the internal set.
func (s *AggregatedSignature[S]) Signers() BitSet[scc.MemberId] {
	return s.signers
}

// Signature returns the aggregated BLS signature of the statement.
func (s *AggregatedSignature[S]) Signature() bls.Signature {
	return s.signature
}

// Add adds a signature from a member to the aggregated signature. The id
// identifies the member within the signing committee. The signature is the BLS
// signature of the statement produced by the respective member. The operation
// fails if a signature from the same member is already present. There is no
// check whether the signature is valid.
func (s *AggregatedSignature[S]) Add(id scc.MemberId, signature Signature[S]) error {
	if s.signers.Contains(id) {
		return fmt.Errorf("signature already added for signer %d", id)
	}
	s.signers.Add(id)
	s.signature = bls.AggregateSignatures(s.signature, signature.Signature)
	return nil
}

// Verify checks if the aggregated signature is valid for the given authority
// and message. The authority is the committee for which a 2/3 majority is
// required. The producers is the committee that signed the aggregated
// signature. These two committees may be the same.
func (s *AggregatedSignature[S]) Verify(
	authority scc.Committee, // < committee for which a 2/3 majority is required
	producers scc.Committee, // < committee that signed the aggregated signature
	statement S, // < the statement to be verified
) error {

	// The producer committee must be a valid committee. In particular, for each
	// public key in the producer committee, a valid proof of possession must be
	// present. Otherwise, a modified public key could be used to counterfeit
	// the aggregated signature.
	if err := producers.Validate(); err != nil {
		return fmt.Errorf("invalid producer committee: %w", err)
	}

	// The voting power of the authority must not exceed the maximum.
	totalPower, overflow := authority.GetTotalVotingPower()
	if overflow {
		return fmt.Errorf("total voting power exceeds maximum")
	}

	// Collect the signers and their voting power according to the authority.
	signers := []bls.PublicKey{}
	signersPower := uint256.NewInt(0)
	for _, i := range s.signers.Entries() {
		member, found := producers.GetMember(i)
		if !found {
			return fmt.Errorf("signer %d not found in producer committee", i)
		}
		signers = append(signers, member.PublicKey)
		if id, found := authority.GetMemberId(member.PublicKey); found {
			member, _ := authority.GetMember(id)
			signersPower.Add(signersPower, uint256.NewInt(member.VotingPower))
		}
	}

	// The total weight of the signers must be at least 2/3 of the total weight of the authority.
	have := new(uint256.Int).Mul(signersPower, uint256.NewInt(3))
	needed := new(uint256.Int).Mul(uint256.NewInt(totalPower), uint256.NewInt(2))
	if have.Cmp(needed) <= 0 {
		return fmt.Errorf("insufficient voting power: %d/%d", signersPower, totalPower)
	}

	// The aggregated signature must be valid.
	if !s.signature.VerifyAll(signers, statement.GetDataToSign()) {
		return fmt.Errorf("invalid aggregated signature")
	}
	return nil
}

// String returns a human-readable representation of the aggregated signature.
func (s *AggregatedSignature[S]) String() string {
	signature := s.signature.Serialize()
	return fmt.Sprintf(
		"AggregatedSignature(signers=%v, signature=0x%x..%x)",
		s.signers, signature[:2], signature[len(signature)-2:],
	)
}
