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

package genesisstore

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/sonic/inter/ibr"
	"github.com/0xsoniclabs/sonic/inter/ier"
	"github.com/0xsoniclabs/sonic/opera/genesis"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/0xsoniclabs/sonic/utils/iodb"
	"github.com/0xsoniclabs/sonic/utils/objstream"
)

type (
	Blocks struct {
		fMap FilesMap
	}
	Epochs struct {
		fMap FilesMap
	}
	RawEvmItems struct {
		fMap FilesMap
	}
	RawCommitteeCertificates struct {
		fMap FilesMap
	}
	RawBlockCertificates struct {
		fMap FilesMap
	}
	RawFwsLiveSection struct {
		fMap FilesMap
	}
	RawFwsArchiveSection struct {
		fMap FilesMap
	}
	SignatureSection struct {
		fMap FilesMap
	}
)

func (s *Store) Genesis() genesis.Genesis {
	return genesis.Genesis{
		Header:                s.head,
		Blocks:                s.Blocks(),
		Epochs:                s.Epochs(),
		RawEvmItems:           s.RawEvmItems(),
		CommitteeCertificates: s.CommitteeCertificates(),
		BlockCertificates:     s.BlockCertificates(),
		FwsLiveSection:        s.FwsLiveSection(),
		FwsArchiveSection:     s.FwsArchiveSection(),
		SignatureSection:      s.SignatureSection(),
	}
}

func getSectionName(base string, i int) string {
	if i == 0 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, i)
}

func (s Store) Header() genesis.Header {
	return s.head
}

func (s *Store) Blocks() genesis.Blocks {
	return Blocks{s.fMap}
}

func (s Blocks) ForEach(fn func(ibr.LlrIdxFullBlockRecord) bool) {
	for i := 1000; i >= 0; i-- {
		f, err := s.fMap(BlocksSection(i))
		if err != nil {
			continue
		}
		stream := rlp.NewStream(f, 0)
		for {
			br := ibr.LlrIdxFullBlockRecord{}
			err = stream.Decode(&br)
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Crit("Failed to decode Blocks genesis section", "err", err)
			}
			if !fn(br) {
				break
			}
		}
	}
}

func (s *Store) Epochs() genesis.Epochs {
	return Epochs{s.fMap}
}

func (s Epochs) ForEach(fn func(ier.LlrIdxFullEpochRecord) bool) {
	for i := 1000; i >= 0; i-- {
		f, err := s.fMap(EpochsSection(i))
		if err != nil {
			continue
		}
		stream := rlp.NewStream(f, 0)
		for {
			er := ier.LlrIdxFullEpochRecord{}
			err = stream.Decode(&er)
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Crit("Failed to decode Epochs genesis section", "err", err)
			}
			if !fn(er) {
				break
			}
		}
	}
}

func (s *Store) RawEvmItems() genesis.EvmItems {
	return RawEvmItems{s.fMap}
}

func (s RawEvmItems) ForEach(fn func(key, value []byte) bool) {
	for i := 1000; i >= 0; i-- {
		f, err := s.fMap(EvmSection(i))
		if err != nil {
			continue
		}
		it := iodb.NewIterator(f)
		for it.Next() {
			if !fn(it.Key(), it.Value()) {
				break
			}
		}
		if it.Error() != nil {
			log.Crit("Failed to decode RawEvmItems genesis section", "err", it.Error())
		}
		it.Release()
	}
}

func (s *Store) CommitteeCertificates() genesis.SccCommitteeCertificates {
	return RawCommitteeCertificates{s.fMap}
}

func (s RawCommitteeCertificates) ForEach(fn func(cert.Certificate[cert.CommitteeStatement]) bool) {
	for i := range 1000 {
		f, err := s.fMap(SccCommitteeSection(i))
		if err != nil {
			continue
		}

		reader := objstream.NewReader[*cert.Certificate[cert.CommitteeStatement]](f)
		var cur cert.Certificate[cert.CommitteeStatement]
		for {
			err = reader.Read(&cur)
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Crit("Failed to decode committee certificate genesis section", "err", err)
			}
			if !fn(cur) {
				break
			}
		}
	}
}

func (s *Store) BlockCertificates() genesis.SccBlockCertificates {
	return RawBlockCertificates{s.fMap}
}

func (s RawBlockCertificates) ForEach(fn func(cert.Certificate[cert.BlockStatement]) bool) {
	for i := range 1000 {
		f, err := s.fMap(SccBlockSection(i))
		if err != nil {
			continue
		}

		reader := objstream.NewReader[*cert.Certificate[cert.BlockStatement]](f)
		var cur cert.Certificate[cert.BlockStatement]
		for {
			err = reader.Read(&cur)
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Crit("Failed to decode block certificate genesis section", "err", err)
			}
			if !fn(cur) {
				break
			}
		}
	}
}

func (s *Store) FwsLiveSection() genesis.FwsLiveSection {
	return RawFwsLiveSection{s.fMap}
}

func (s RawFwsLiveSection) GetReader() (io.Reader, error) {
	return s.fMap(FwsLiveSection(0))
}

func (s *Store) FwsArchiveSection() genesis.FwsLiveSection {
	return RawFwsArchiveSection{s.fMap}
}

func (s RawFwsArchiveSection) GetReader() (io.Reader, error) {
	return s.fMap(FwsArchiveSection(0))
}

func (s *Store) SignatureSection() genesis.SignatureSection {
	return SignatureSection{s.fMap}
}

func (s SignatureSection) GetSignature() ([]byte, error) {
	f, err := s.fMap("signature")
	if err != nil {
		return nil, err
	}
	return io.ReadAll(f)
}
