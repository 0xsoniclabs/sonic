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
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/0xsoniclabs/sonic/scc/bls"
)

// Committee is a group of Members that can sign statements to produce
// certificates.
//
// On the Sonic Certification Chain (SCC), time is divided into periods. For
// each period, a committee is authorized to sign the statement for that period.
// Among those statements is the confirmation of the succeeding period's
// committee.
//
// Committees are composed of an ordered list of Members, each weighted with a
// non-zero voting power. Members are identified by their public keys, for which
// they are required to provide a proof of possession.
type Committee struct {
	members []Member
}

// MaxCommitteeSize is the maximum number of members that can be in a committee.
// The number needs to be limited to prevent Committee certificates from becoming
// too large.
const MaxCommitteeSize = 512

// MemberId is used to identify a member in a committee by its position in the
// ordered list of members.
type MemberId uint16

// NewCommittee creates a new committee with the provided members. The list of
// members is not implicitly validated, thus it is possible to create invalid
// committees. To ensure that the committee is well-formed, call the Validate
// method.
func NewCommittee(members ...Member) Committee {
	return Committee{members: slices.Clone(members)}
}

// Members returns a copy of the members in the committee.
func (c Committee) Members() []Member {
	return slices.Clone(c.members)
}

// GetMember returns the member with the given id. If the id is out of bounds,
// the second return value is false.
func (c Committee) GetMember(id MemberId) (Member, bool) {
	if int(id) >= len(c.members) {
		return Member{}, false
	}
	return c.members[id], true
}

// GetMemberId returns the id of the member with the given public key. If the
// public key is not found, the second return value is false.
func (c Committee) GetMemberId(publicKey bls.PublicKey) (MemberId, bool) {
	for id, m := range c.members {
		if m.PublicKey == publicKey {
			return MemberId(id), true
		}
	}
	return 0, false
}

// GetTotalVotingPower returns the sum of the voting power of all members in the
// committee. Valid committees have a total voting power that does not exceed
// 2^64 - 1, but invalid committees may have a total voting power that exceeds
// this limit. In the latter case, the second return value is true.
func (c Committee) GetTotalVotingPower() (total uint64, overflow bool) {
	for _, m := range c.members {
		next := total + m.VotingPower
		if next < total {
			return 0, true
		}
		total = next
	}
	return total, false
}

// Validate checks that the committee is well-formed. For a committee to be well
// formed, the following properties need to be satisfied:
// - The committee must have at least one member.
// - The committee size must not exceed the maximum committee size.
// - All members must be valid.
// - No two members can have the same public key.
// - The sum of the voting power must not exceed 2^64 - 1.
// If any of these properties are violated, an error is returned.
func (c Committee) Validate() error {
	if len(c.members) == 0 {
		return fmt.Errorf("committee must have at least one member")
	}

	if len(c.members) > MaxCommitteeSize {
		return fmt.Errorf("committee size exceeds the maximum of %d", MaxCommitteeSize)
	}

	for _, m := range c.members {
		if err := m.Validate(); err != nil {
			return fmt.Errorf("invalid member %v, %w", m, err)
		}
	}

	for i, a := range c.members {
		for j, b := range c.members {
			if i != j && a.PublicKey == b.PublicKey {
				return fmt.Errorf("duplicate members at indexes %d and %d", i, j)
			}
		}
	}

	sum := uint64(0)
	for _, m := range c.members {
		next := sum + m.VotingPower
		if next < sum {
			return fmt.Errorf("voting power overflow")
		}
		sum = next
	}

	return nil
}

// String produces a human-readable summary of the Committee information mainly
// for debugging purposes. The output is not sufficient to reconstruct the
// committee.
func (c Committee) String() string {
	result := strings.Builder{}
	result.WriteString("Committee{")
	for i, m := range c.Members() {
		key := m.PublicKey.Serialize()
		result.WriteString(
			fmt.Sprintf(
				"%d: 0x%x..%x -> %d, ",
				i, key[:2], key[len(key)-2:],
				m.VotingPower,
			),
		)
	}
	result.WriteString(fmt.Sprintf("Valid: %t}", c.Validate() == nil))
	return result.String()
}

// Serialize serializes the committee into a byte slice. The serialization format
// is a concatenation of the serialized members. Note that also invalid committees
// can be serialized.
func (c Committee) Serialize() []byte {
	if len(c.members) == 0 {
		return nil
	}
	res := make([]byte, 0, len(c.members)*len(EncodedMember{}))
	for _, m := range c.Members() {
		cur := m.Serialize()
		res = append(res, cur[:]...)
	}
	return res
}

// DeserializeCommittee deserializes a committee from a byte slice. An error is
// returned if the provided data does not contain a valid encoding of a
// committee. Note, this function does not validate members nor the committee.
// Thus, it is possible to deserialize invalid committees.
func DeserializeCommittee(data []byte) (Committee, error) {
	if len(data) == 0 {
		return Committee{}, nil
	}
	if len(data)%len(EncodedMember{}) != 0 {
		return Committee{}, fmt.Errorf("invalid committee data length")
	}

	members := make([]Member, 0, len(data)/len(EncodedMember{}))
	for len(data) > 0 {
		var m Member
		m, err := DeserializeMember(*(*EncodedMember)(data))
		if err != nil {
			return Committee{}, fmt.Errorf("invalid member, %w", err)
		}
		members = append(members, m)
		data = data[len(EncodedMember{}):]
	}

	return Committee{members}, nil
}

// MarshalJSON implements the json.Marshaler interface. The committee is
// serialized into a JSON array of members.
func (c Committee) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.members)
}

// UnmarshalJSON implements the json.Unmarshaler interface. The committee is
// deserialized from a JSON array of members.
func (c *Committee) UnmarshalJSON(data []byte) error {
	var members []Member
	if err := json.Unmarshal(data, &members); err != nil {
		return err
	}
	*c = NewCommittee(members...)
	return nil
}
