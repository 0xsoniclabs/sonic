package genesisstore

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/utils/byteutils"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/status-im/keycard-go/hexutils"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/0xsoniclabs/sonic/opera/genesis"
	"github.com/0xsoniclabs/sonic/opera/genesisstore/filelog"
	"github.com/0xsoniclabs/sonic/opera/genesisstore/fileshash"
	"github.com/0xsoniclabs/sonic/opera/genesisstore/readersmap"
	"github.com/0xsoniclabs/sonic/utils/ioread"
)

var (
	FileHeader  = hexutils.HexToBytes("641b00ac")
	FileVersion = hexutils.HexToBytes("00020001")
)

const (
	FilesHashMaxMemUsage = 256 * opt.MiB
	FilesHashPieceSize   = 64 * opt.MiB
)

type dummyByteReader struct {
	io.Reader
}

func (r dummyByteReader) ReadByte() (byte, error) {
	b := make([]byte, 1)
	err := ioread.ReadAll(r.Reader, b)
	return b[0], err
}

func checkFileHeader(reader io.Reader) error {
	headerAndVersion := make([]byte, len(FileHeader)+len(FileVersion))
	err := ioread.ReadAll(reader, headerAndVersion)
	if err != nil {
		return err
	}
	if !bytes.Equal(headerAndVersion[:len(FileHeader)], FileHeader) {
		return errors.New("expected a genesis file, mismatched file header")
	}
	if !bytes.Equal(headerAndVersion[len(FileHeader):], FileVersion) {
		got := hexutils.BytesToHex(headerAndVersion[len(FileHeader):])
		expected := hexutils.BytesToHex(FileVersion)
		return fmt.Errorf("wrong version of genesis file, got=%s, expected=%s", got, expected)
	}
	return nil
}

type ReadAtSeekerCloser interface {
	io.ReaderAt
	io.Seeker
	io.Closer
}

type Unit struct {
	UnitName string
	Header   genesis.Header
}

func OpenGenesisStore(rawReader ReadAtSeekerCloser) (*Store, genesis.Hashes, error) {
	header := genesis.Header{}
	hashes := genesis.Hashes{}
	units := make([]readersmap.Unit, 0, 3)
	offset := int64(0)
	for i := 0; ; i++ {
		// header cannot be long, cap it with 100000 bytes
		headerReader := io.NewSectionReader(rawReader, offset, offset+100000)
		err := checkFileHeader(headerReader)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, hashes, err
		}
		unit := Unit{}
		err = rlp.Decode(dummyByteReader{headerReader}, &unit)
		if err != nil {
			return nil, hashes, err
		}
		if i == 0 {
			header = unit.Header
		} else {
			if !header.Equal(unit.Header) {
				return nil, hashes, errors.New("subsequent genesis header doesn't match the first header")
			}
		}

		var h consensus.Hash
		err = ioread.ReadAll(headerReader, h[:])
		if err != nil {
			return nil, hashes, err
		}
		hashes[unit.UnitName] = h

		var numB [8]byte
		err = ioread.ReadAll(headerReader, numB[:])
		if err != nil {
			return nil, hashes, err
		}
		dataCompressedSize := byteutils.BigEndianToUint64(numB[:])

		err = ioread.ReadAll(headerReader, numB[:])
		if err != nil {
			return nil, hashes, err
		}
		uncompressedSize := byteutils.BigEndianToUint64(numB[:])

		headerSize, err := headerReader.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, hashes, err
		}

		off := offset // standalone variable for each Unit instance
		units = append(units, readersmap.Unit{
			Name: unit.UnitName,
			ReaderProvider: func() (io.Reader, error) {
				unitReader := io.NewSectionReader(rawReader, off+headerSize, off+headerSize+int64(dataCompressedSize))
				gzipReader, err := gzip.NewReader(unitReader)
				if err != nil {
					return nil, err
				}
				return filelog.Wrap(gzipReader, getLoggerName(unit.UnitName), uncompressedSize, time.Minute), nil
			},
		})
		offset += headerSize + int64(dataCompressedSize)
	}

	unitsMap, err := readersmap.Wrap(units)
	if err != nil {
		return nil, hashes, err
	}

	hashedMap := fileshash.Wrap(unitsMap.Open, FilesHashMaxMemUsage, hashes)

	return NewStore(hashedMap, header, rawReader.Close), hashes, nil
}

// getLoggerName provides a human-readable name of a unit for logging purposes
func getLoggerName(name string) string {
	scanfName := strings.ReplaceAll(name, "-", "")
	if scanfName[len(scanfName)-1] < '0' || scanfName[len(scanfName)-1] > '9' {
		scanfName += "0"
	}
	var part int
	if _, err := fmt.Sscanf(scanfName, "brs%d", &part); err == nil {
		name = fmt.Sprintf("blocks unit %d", part)
	}
	if _, err := fmt.Sscanf(scanfName, "ers%d", &part); err == nil {
		name = fmt.Sprintf("epochs unit %d", part)
	}
	if _, err := fmt.Sscanf(scanfName, "evm%d", &part); err == nil {
		name = fmt.Sprintf("EVM unit %d", part)
	}
	return name
}
