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
	"testing"

	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/stretchr/testify/require"
)

func TestSignature_Sign_ProducesValidSignature(t *testing.T) {
	key := bls.NewPrivateKey()
	statement := testStatement(1)

	sig := Sign(statement, key)
	require.Equal(t, key.Sign([]byte{1}), sig.Signature)

	if !sig.Verify(key.PublicKey(), statement) {
		t.Error("signature is not valid")
	}
}

func TestSignature_Verify_RejectsInvalidSignature(t *testing.T) {
	key := bls.NewPrivateKey()
	stmt := testStatement(1)
	tests := map[string]struct {
		key  bls.PublicKey
		stmt testStatement
	}{
		"wrong key": {
			key:  bls.NewPrivateKey().PublicKey(),
			stmt: stmt,
		},
		"wrong statement": {
			key:  key.PublicKey(),
			stmt: stmt + 1,
		},
		"both wrong": {
			key:  bls.NewPrivateKey().PublicKey(),
			stmt: stmt + 1,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			sig := Sign(testStatement(1), key)
			if sig.Verify(test.key, test.stmt) {
				t.Error("invalid signature is reported to be valid")
			}
		})
	}
}

type testStatement byte

func (s testStatement) GetDataToSign() []byte {
	return []byte{byte(s)}
}
