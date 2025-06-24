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

package objstream

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestReader_Read_FromEmptyStreamProducesEof(t *testing.T) {
	reader := NewReader[*testMsg](bytes.NewReader(nil))
	require.ErrorIs(t, reader.Read(nil), io.EOF)
}

func TestReader_Read_CanReadEncodedMessage(t *testing.T) {
	data := []byte{0x2, 0x1, 0x3}
	reader := NewReader[*testMsg](bytes.NewReader(data))

	var restored testMsg
	require.NoError(t, reader.Read(&restored))
	require.Equal(t, restored, testMsg([]byte{0x1, 0x3}))

	require.ErrorIs(t, reader.Read(&restored), io.EOF)
}

func TestReader_Read_ReportsDeserializationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	trg := NewMockDeserializer(ctrl)

	injectedError := fmt.Errorf("injected")
	trg.EXPECT().Deserialize(gomock.Any()).Return(injectedError)

	reader := NewReader[*MockDeserializer](bytes.NewReader([]byte{0x2, 0x1, 0x3}))
	require.ErrorIs(t, reader.Read(trg), injectedError)
}

func TestWriter_Write_EncodesMessageWithLengthPrefix(t *testing.T) {
	buffer := bytes.Buffer{}
	writer := bufio.NewWriter(&buffer)
	out := NewWriter[testMsg](writer)

	require.NoError(t, out.Write(testMsg("hello")))
	require.NoError(t, writer.Flush())

	require.Equal(t, []byte{0x5, 'h', 'e', 'l', 'l', 'o'}, buffer.Bytes())
}

func TestWriter_Write_ReportsSerializationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	trg := NewMockSerializer(ctrl)

	injectedError := fmt.Errorf("injected")
	trg.EXPECT().Serialize().Return(nil, injectedError)

	buffer := bytes.Buffer{}
	writer := bufio.NewWriter(&buffer)
	out := NewWriter[*MockSerializer](writer)

	require.ErrorIs(t, out.Write(trg), injectedError)
}

func TestIntegration_CanEncodeAndDecodeListOfMessages(t *testing.T) {
	require := require.New(t)

	input := []string{"hello", "world", "this", "is", "a", "test"}

	buffer := bytes.Buffer{}
	writer := bufio.NewWriter(&buffer)
	out := NewWriter[testMsg](writer)
	for _, str := range input {
		require.NoError(out.Write(testMsg(str)))
	}
	require.NoError(writer.Flush())

	in := NewReader[*testMsg](bufio.NewReader(&buffer))
	var output []string
	for {
		var str testMsg
		err := in.Read(&str)
		if err == io.EOF {
			break
		}
		require.NoError(err)
		output = append(output, string(str))
	}

	require.Equal(input, output)
}

type testMsg string

func (m testMsg) Serialize() ([]byte, error) {
	return []byte(m), nil
}

func (m *testMsg) Deserialize(data []byte) error {
	*m = testMsg(data)
	return nil
}
