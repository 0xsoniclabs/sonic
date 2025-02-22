package genesis

import (
	"io"

	"github.com/Fantom-foundation/lachesis-base/hash"

	"github.com/0xsoniclabs/sonic/inter/ibr"
	"github.com/0xsoniclabs/sonic/inter/ier"
	"github.com/0xsoniclabs/sonic/scc/cert"
)

type (
	Hashes map[string]hash.Hash
	Header struct {
		GenesisID   hash.Hash
		NetworkID   uint64
		NetworkName string
	}
	Blocks interface {
		ForEach(fn func(ibr.LlrIdxFullBlockRecord) bool)
	}
	Epochs interface {
		ForEach(fn func(ier.LlrIdxFullEpochRecord) bool)
	}
	EvmItems interface {
		ForEach(fn func(key, value []byte) bool)
	}
	SccCommitteeCertificates interface {
		ForEach(fn func(cert.Certificate[cert.CommitteeStatement]) bool)
	}
	SccBlockCertificates interface {
		ForEach(fn func(cert.Certificate[cert.BlockStatement]) bool)
	}
	FwsLiveSection interface {
		GetReader() (io.Reader, error)
	}
	FwsArchiveSection interface {
		GetReader() (io.Reader, error)
	}
	SignatureSection interface {
		GetSignature() ([]byte, error)
	}
	SignedMetadata struct {
		Signature []byte
		Hashes    []byte
	}
	Genesis struct {
		Header

		Blocks                Blocks
		Epochs                Epochs
		RawEvmItems           EvmItems
		CommitteeCertificates SccCommitteeCertificates
		BlockCertificates     SccBlockCertificates
		FwsLiveSection
		FwsArchiveSection
		SignatureSection
	}
)

func (hh Hashes) Includes(hh2 Hashes) bool {
	for n, h := range hh {
		if hh2[n] != h {
			return false
		}
	}
	return true
}

func (hh Hashes) Equal(hh2 Hashes) bool {
	return hh.Includes(hh2) && hh2.Includes(hh)
}

func (h Header) Equal(h2 Header) bool {
	return h == h2
}
