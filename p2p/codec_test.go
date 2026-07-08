// Copyright 2026 Sonic Operations Ltd
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

package p2p

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/p2p/pb"
)

func TestWriteMessage_RoundTrip_PreservesMessage(t *testing.T) {
	original := &pb.ScanStatusResponse{
		Role:          pb.NodeRole_NODE_ROLE_ARCHIVE,
		ClientVersion: "sonic/v2.2.0",
		BlockHeight:   1234567,
	}

	var buffer bytes.Buffer
	_, err := WriteMessage(&buffer, original, 1024)
	require.NoError(t, err, "WriteMessage failed")

	var decoded pb.ScanStatusResponse
	_, err = ReadMessage(&buffer, &decoded, 1024)
	require.NoError(t, err, "ReadMessage failed")

	require.Equal(t, original.ClientVersion, decoded.ClientVersion)
	require.Equal(t, original.BlockHeight, decoded.BlockHeight)
	require.Equal(t, original.Role, decoded.Role)
}

func TestWriteMessage_ExceedsCap_Rejected(t *testing.T) {
	message := &pb.ScanStatusResponse{ClientVersion: "a-client-version-string"}
	var buffer bytes.Buffer
	_, err := WriteMessage(&buffer, message, 4)
	require.ErrorIs(t, err, ErrMessageTooLarge)
	require.Zero(t, buffer.Len(), "expected nothing written on rejection")
}

func TestReadMessage_OversizedFrame_RejectedBeforeBody(t *testing.T) {
	// A frame declaring a huge body but carrying none: ReadMessage must reject
	// it from the length prefix alone, without blocking on the missing body.
	var header [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(header[:], 1<<20)
	reader := bytes.NewReader(header[:n])

	var decoded pb.ScanStatusResponse
	_, err := ReadMessage(reader, &decoded, 1024)
	require.ErrorIs(t, err, ErrMessageTooLarge)
}

func TestReadMessage_DifferentCapsPerType_Honored(t *testing.T) {
	message := &pb.ScanPeersResponse{PeerAddresses: []string{
		"/ip4/127.0.0.1/tcp/4002/p2p/12D3KooWExample",
	}}
	var buffer bytes.Buffer
	_, err := WriteMessage(&buffer, message, maxScanPeersLikeCap)
	require.NoError(t, err, "WriteMessage failed")
	encoded := buffer.Bytes()

	// A small cap rejects it; a large cap accepts it - same bytes, different
	// per-call limit.
	var decoded pb.ScanPeersResponse
	_, err = ReadMessage(bytes.NewReader(encoded), &decoded, 4)
	require.ErrorIs(t, err, ErrMessageTooLarge, "expected small cap to reject")
	_, err = ReadMessage(bytes.NewReader(encoded), &decoded, maxScanPeersLikeCap)
	require.NoError(t, err, "expected large cap to accept")
}

const maxScanPeersLikeCap = 1 << 20
