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

package fileshash

import (
	"bytes"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/utils/ioread"
)

type dropableFile struct {
	io.ReadWriteSeeker
	io.Closer
	path string
}

func (f dropableFile) Drop() error {
	return os.Remove(f.path)
}

func TestFileHash_ReadWrite(t *testing.T) {

	const (
		FILE_CONTENT = `Lorem ipsum dolor sit amet, consectetur adipiscing elit. 
			Nunc finibus ultricies interdum. Nulla porttitor arcu a tincidunt mollis. Aliquam erat volutpat. 
			Maecenas eget ligula mi. Maecenas in ligula non elit fringilla consequat. 
			Morbi non imperdiet odio. Integer viverra ligula a varius tempor. 
			Duis ac velit vel augue faucibus tincidunt ut ac nisl. Nulla sed magna est. 
			Etiam quis nunc in elit ultricies pulvinar sed at felis. 
			Suspendisse fringilla lectus vel est hendrerit pulvinar. 
			Vivamus nec lorem pharetra ligula pulvinar blandit in quis nunc. 
			Cras id eros fermentum mauris tristique faucibus. 
			Praesent vehicula lectus nec ipsum sollicitudin tempus. Nullam et massa velit.`
	)
	t.Run("Large file, pieceSize=10000", func(t *testing.T) {
		testFileHash_ReadWrite(t, bytes.Repeat([]byte(FILE_CONTENT), 20), hash.HexToHash("0xe3c3075749531525b472f4d6d70578e6d497d3e75b0727fea1ee10bdd1fcd490"), 10000)
	})
	t.Run("Large file, pieceSize=100", func(t *testing.T) {
		testFileHash_ReadWrite(t, bytes.Repeat([]byte(FILE_CONTENT), 20), hash.HexToHash("0xdc6d882fde82b2dd44884a97884d79be40e1d0f780a493dee0f7256d8261f7a5"), 100)
	})
	t.Run("Medium file, pieceSize=1", func(t *testing.T) {
		testFileHash_ReadWrite(t, bytes.Repeat([]byte(FILE_CONTENT), 1), hash.HexToHash("0x63a76929ee27decd5100d07a3cb626c05df1f1e927c5f27fa17c62459685ca6f"), 1)
	})
	t.Run("Medium file, pieceSize=2", func(t *testing.T) {
		testFileHash_ReadWrite(t, bytes.Repeat([]byte(FILE_CONTENT), 1), hash.HexToHash("0x2babd1049c449a60da62a1a0a3dc836e6a222dece07dcba1890a041d60ff29c7"), 2)
	})
	t.Run("Medium file, pieceSize=100", func(t *testing.T) {
		testFileHash_ReadWrite(t, bytes.Repeat([]byte(FILE_CONTENT), 1), hash.HexToHash("0x15c45aba675b7c49f5def32b4f24e827d478e5dfd712613fd6a5df69a2793b60"), 100)
	})
	t.Run("Tiny file, pieceSize=1", func(t *testing.T) {
		testFileHash_ReadWrite(t, []byte{0}, hash.HexToHash("0xbdda25ac486f2b8c0330a6fcace8f3d05d3e713b7920a39a1f60c0d8df024c0e"), 1)
	})
	t.Run("Tiny file, pieceSize=2", func(t *testing.T) {
		testFileHash_ReadWrite(t, []byte{0}, hash.HexToHash("0xf2b22424f7d1d01467d650f18ad49df8929fbefdeefe95e868d52eec6ea399e1"), 2)
	})
	t.Run("Empty file, pieceSize=1", func(t *testing.T) {
		testFileHash_ReadWrite(t, []byte{}, hash.HexToHash("0x9cbc73d18d70c94fe366e696035c4f2cffdbab7ea6d6c2c039ca185f9c9f2746"), 1)
	})
	t.Run("Empty file, pieceSize=2", func(t *testing.T) {
		testFileHash_ReadWrite(t, []byte{}, hash.HexToHash("0x163e7f66d58036ccb1d0b0058d8f46e7cd639816f570e5eb32853ea73634e4cd"), 2)
	})
}

func testFileHash_ReadWrite(t *testing.T, content []byte, expRoot hash.Hash, pieceSize uint64) {
	require := require.New(t)
	tmpDirPath := t.TempDir()
	file, err := os.CreateTemp(tmpDirPath, "testnet.g")
	filePath := file.Name()
	require.NoError(err)
	writer := WrapWriter(file, pieceSize, func(i int) TmpWriter {
		tmpFh, err := os.OpenFile(path.Join(tmpDirPath, fmt.Sprintf("genesis%d.dat", i)), os.O_CREATE|os.O_RDWR, os.ModePerm)
		require.NoError(err)
		return dropableFile{
			ReadWriteSeeker: tmpFh,
			Closer:          tmpFh,
			path:            tmpFh.Name(),
		}
	})

	// write out the (secure) self-hashed file properly
	_, err = writer.Write(content)
	require.NoError(err)
	root, err := writer.Flush()
	require.NoError(err)
	require.Equal(expRoot.Hex(), root.Hex())
	err = file.Close()
	require.NoError(err)

	maxMemUsage := memUsageOf(pieceSize, getPiecesNum(uint64(len(content)), pieceSize))

	// normal case: correct root hash and content after reading file partially
	if len(content) > 0 {
		file, err := os.OpenFile(filePath, os.O_RDONLY, 0600)
		require.NoError(err)
		reader := WrapReader(file, maxMemUsage, root)
		readB := make([]byte, rand.Int64N(int64(len(content))))
		err = ioread.ReadAll(reader, readB)
		require.NoError(err)
		require.Equal(content[:len(readB)], readB)
		err = reader.Close()
		require.NoError(err)
	}

	// normal case: correct root hash and content after reading the whole file
	{
		file, err := os.OpenFile(filePath, os.O_RDONLY, 0600)
		require.NoError(err)
		reader := WrapReader(file, maxMemUsage, root)
		readB := make([]byte, len(content))
		err = ioread.ReadAll(reader, readB)
		require.NoError(err)
		require.Equal(content, readB)
		// try to read one more byte
		require.Error(ioread.ReadAll(reader, make([]byte, 1)), io.EOF)
		err = reader.Close()
		require.NoError(err)
	}

	// correct root hash and reading too much content
	{
		file, err := os.OpenFile(filePath, os.O_RDONLY, 0600)
		require.NoError(err)
		reader := WrapReader(file, maxMemUsage, root)
		readB := make([]byte, len(content)+1)
		require.Error(ioread.ReadAll(reader, readB), io.EOF)
		err = reader.Close()
		require.NoError(err)
	}

	// passing the wrong root hash to reader
	{
		file, err := os.OpenFile(filePath, os.O_RDONLY, 0600)
		require.NoError(err)
		maliciousReader := WrapReader(file, maxMemUsage, hash.HexToHash("0x00"))
		data := make([]byte, 1)
		err = ioread.ReadAll(maliciousReader, data)
		require.Contains(err.Error(), ErrInit.Error())
		err = maliciousReader.Close()
		require.NoError(err)
	}

	// modify piece data to make the mismatch piece hash
	headerOffset := 4 + 8 + getPiecesNum(uint64(len(content)), pieceSize)*32
	if len(content) > 0 {
		// mutate content byte
		file, err = os.OpenFile(filePath, os.O_RDWR, 0600)
		require.NoError(err)
		s := []byte{0}
		contentPos := rand.Int64N(int64(len(content)))
		pos := int64(headerOffset) + contentPos
		_, err = file.ReadAt(s, pos)
		require.NoError(err)
		s[0]++
		_, err = file.WriteAt(s, pos)
		require.NoError(err)
		require.NoError(file.Close())

		// try to read
		file, err = os.OpenFile(filePath, os.O_RDONLY, 0600)
		require.NoError(err)
		maliciousReader := WrapReader(file, maxMemUsage, root)
		data := make([]byte, contentPos+1)
		err = ioread.ReadAll(maliciousReader, data)
		require.Contains(err.Error(), ErrHashMismatch.Error())
		require.NoError(maliciousReader.Close())

		// restore
		s[0]--
		file, err = os.OpenFile(filePath, os.O_RDWR, 0600)
		require.NoError(err)
		_, err = file.WriteAt(s, pos)
		require.NoError(err)
		require.NoError(file.Close())
	}

	// modify a piece hash in file to make the wrong one
	{
		// mutate content byte
		file, err = os.OpenFile(filePath, os.O_RDWR, 0600)
		require.NoError(err)
		pos := rand.Int64N(int64(headerOffset))
		s := []byte{0}
		_, err = file.ReadAt(s, pos)
		require.NoError(err)
		s[0]++
		_, err = file.WriteAt(s, pos)
		require.NoError(err)
		require.NoError(file.Close())

		// try to read
		file, err = os.OpenFile(filePath, os.O_RDONLY, 0600)
		require.NoError(err)
		maliciousReader := WrapReader(file, maxMemUsage*2, root)
		data := make([]byte, 1)
		err = ioread.ReadAll(maliciousReader, data)
		require.Contains(err.Error(), ErrInit.Error())
		require.NoError(maliciousReader.Close())

		// restore
		s[0]--
		file, err = os.OpenFile(filePath, os.O_RDWR, 0600)
		require.NoError(err)
		_, err = file.WriteAt(s, pos)
		require.NoError(err)
		require.NoError(file.Close())
	}

	// hashed file requires too much memory
	{
		file, err = os.OpenFile(filePath, os.O_WRONLY, 0600)
		require.NoError(err)
		oomReader := WrapReader(file, maxMemUsage-1, root)
		data := make([]byte, 1)
		err = ioread.ReadAll(oomReader, data)
		require.Errorf(err, "hashed file requires too much memory")
	}
}
