package ethapi

import (
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
)

// BlockCertificateJson is a JSON representation of a block certificate
// as returned by the Sonic API. This type provides a conversion between the
// internal certificate representation and the JSON representation provided to
// the API clients. The external API is expected to be stable over time and
// should only be updated in a backward compatible way.
type BlockCertificateJson struct {
	ChainId   uint64                    `json:"chainId"`
	Number    uint64                    `json:"number"`
	Hash      common.Hash               `json:"hash"`
	StateRoot common.Hash               `json:"stateRoot"`
	Signers   cert.BitSet[scc.MemberId] `json:"signers"`
	Signature bls.Signature             `json:"signature"`
}

func (b BlockCertificateJson) ToCertificate() cert.BlockCertificate {
	aggregatedSignature := cert.NewAggregatedSignature[cert.BlockStatement](
		b.Signers, b.Signature)

	newCert := cert.NewCertificateWithSignature(
		cert.NewBlockStatement(
			b.ChainId,
			idx.Block(b.Number),
			b.Hash,
			b.StateRoot),
		aggregatedSignature)

	return newCert
}

func toJsonBlockCertificate(b cert.BlockCertificate) BlockCertificateJson {
	sub := b.Subject()
	agg := b.Signature()
	return BlockCertificateJson{
		ChainId:   sub.ChainId,
		Number:    uint64(sub.Number),
		Hash:      sub.Hash,
		StateRoot: sub.StateRoot,
		Signers:   agg.Signers(),
		Signature: agg.Signature(),
	}
}
