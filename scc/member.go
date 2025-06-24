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

package scc

import (
	"encoding/binary"
	"fmt"

	"github.com/0xsoniclabs/sonic/scc/bls"
)

// Member is a member of a committee. Members are identified by their public key.
// To defend against rogue key attacks, members must provide a proof of possession
// for their public key. The voting power of a member determines their relative
// influence in committees.
type Member struct {
	PublicKey         bls.PublicKey
	ProofOfPossession bls.Signature
	VotingPower       uint64
}

func (m Member) Validate() error {
	if !m.PublicKey.Validate() {
		return fmt.Errorf("invalid public key")
	}
	if !m.PublicKey.CheckProofOfPossession(m.ProofOfPossession) {
		return fmt.Errorf("invalid proof of possession")
	}
	if m.VotingPower == 0 {
		return fmt.Errorf("invalid zero voting power")
	}
	return nil
}

// String produces a human-readable summary of the member information mainly for
// debugging purposes. The output is not sufficient to reconstruct the member.
func (m Member) String() string {
	key := m.PublicKey.Serialize()
	return fmt.Sprintf(
		"Member{PublicKey: 0x%x..%x, Valid: %t, VotingPower: %d}",
		key[:2],
		key[len(key)-2:],
		m.Validate() == nil,
		m.VotingPower,
	)
}

// EncodedMember is a fixed-length byte array that represents a serialized member.
type EncodedMember [48 + 96 + 8]byte

// Serialize serializes the member into a fixed-length byte array.
func (m Member) Serialize() EncodedMember {
	res := EncodedMember{}
	*(*[48]byte)(res[0:]) = m.PublicKey.Serialize()
	*(*[96]byte)(res[48:]) = m.ProofOfPossession.Serialize()
	binary.BigEndian.PutUint64(res[48+96:], m.VotingPower)
	return res
}

// DeserializeMember deserializes a member from a fixed-length byte array. An
// error is returned if the provided data does not contain a valid encoding of
// a public key or proof of possession.
func DeserializeMember(data EncodedMember) (Member, error) {
	var m Member
	var err error
	m.PublicKey, err = bls.DeserializePublicKey(*(*[48]byte)(data[0:]))
	if err != nil {
		return m, fmt.Errorf("invalid public key, %w", err)
	}
	m.ProofOfPossession, err = bls.DeserializeSignature(*(*[96]byte)(data[48:]))
	if err != nil {
		return m, fmt.Errorf("invalid proof of possession, %w", err)
	}
	m.VotingPower = binary.BigEndian.Uint64(data[48+96:])
	return m, nil
}
