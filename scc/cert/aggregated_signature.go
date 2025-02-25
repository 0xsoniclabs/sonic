package cert

import (
	"fmt"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/holiman/uint256"
)

// AggregatedSignature represents an aggregated BLS signature from a committee.
type AggregatedSignature[S Statement] struct {
	Signers   BitSet[scc.MemberId]
	Signature bls.Signature
}

// Add adds a signature from a member to the aggregated signature. The id
// identifies the member within the signing committee. The signature is the BLS
// signature of the statement produced by the respective member. The operation
// fails if a signature from the same member is already present. There is no
// check whether the signature is valid.
func (s *AggregatedSignature[S]) Add(id scc.MemberId, signature Signature[S]) error {
	if s.Signers.Contains(id) {
		return fmt.Errorf("signature already added for signer %d", id)
	}
	s.Signers.Add(id)
	s.Signature = bls.AggregateSignatures(s.Signature, signature.Signature)
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
	for _, i := range s.Signers.Entries() {
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
	if !s.Signature.VerifyAll(signers, statement.GetDataToSign()) {
		return fmt.Errorf("invalid aggregated signature")
	}
	return nil
}

// String returns a human-readable representation of the aggregated signature.
func (s *AggregatedSignature[S]) String() string {
	signature := s.Signature.Serialize()
	return fmt.Sprintf(
		"AggregatedSignature(signers=%v, signature=0x%x..%x)",
		s.Signers, signature[:2], signature[len(signature)-2:],
	)
}

// func (s *AggregatedSignature[S]) MarshalJSON() ([]byte, error) {
// 	signers, err := s.signers.MarshalJSON()
// 	if err != nil {
// 		return nil, err
// 	}
// 	signature, err := s.signature.MarshalJSON()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return []byte(fmt.Sprintf(`{"signers":"%s","signature":"%s"}`, signers, signature)), nil
// }

// func (s *AggregatedSignature[S]) UnmarshalJSON(data []byte) error {
// 	var signers BitSet[scc.MemberId]
// 	if err := signers.UnmarshalJSON(data); err != nil {
// 		return err
// 	}
// 	s.signers = signers
// 	var signature bls.Signature
// 	if err := signature.UnmarshalJSON(data); err != nil {
// 		return err
// 	}
// 	s.signature = signature
// 	return nil
// }
