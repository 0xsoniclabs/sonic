package ethapi

import (
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
)

type blockCertificateJson struct {
	ChainId   uint64                    `json:"chainId"`
	Number    uint64                    `json:"period"`
	Hash      common.Hash               `json:"height"`
	StateRoot common.Hash               `json:"hash"`
	Signers   cert.BitSet[scc.MemberId] `json:"signers"`
	Signature bls.Signature             `json:"signature"`
}

func (b blockCertificateJson) ToBlockCertificate() cert.BlockCertificate {
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
