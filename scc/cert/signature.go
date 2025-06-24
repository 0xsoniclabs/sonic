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
	"github.com/0xsoniclabs/sonic/scc/bls"
)

// Signature is a BLS signature of a statement.
type Signature[S Statement] struct {
	Signature bls.Signature
}

// Sign creates a signature for the given statement using the given key.
func Sign[S Statement](subject S, key bls.PrivateKey) Signature[S] {
	data := subject.GetDataToSign()
	return Signature[S]{Signature: key.Sign(data)}
}

// Verify checks if the signature is valid for the given key and statement.
func (s Signature[S]) Verify(key bls.PublicKey, statement S) bool {
	data := statement.GetDataToSign()
	return s.Signature.Verify(key, data)
}
