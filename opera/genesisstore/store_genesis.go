package genesisstore

import (
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"io"

	"github.com/0xsoniclabs/sonic/inter/ibr"
	"github.com/0xsoniclabs/sonic/inter/ier"
	"github.com/0xsoniclabs/sonic/opera/genesis"
	"github.com/0xsoniclabs/sonic/utils/iodb"
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
		Header:      s.head,
		Blocks:      s.Blocks(),
		Epochs:      s.Epochs(),
		RawEvmItems: s.RawEvmItems(),
		FwsLiveSection:  s.FwsLiveSection(),
		FwsArchiveSection: s.FwsArchiveSection(),
		SignatureSection: s.SignatureSection(),
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
