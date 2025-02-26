package ethapi

import (
	"testing"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBlockCertificateJson_ToBlockCertificate_ConvertsToHealthyCertificate(t *testing.T) {
	require := require.New(t)
	bitSet := cert.BitSet[scc.MemberId]{}
	bitSet.Add(1)
	newPrivateKey := bls.NewPrivateKey()
	sig := newPrivateKey.GetProofOfPossession()

	b := blockCertificateJson{
		ChainId:   123,
		Number:    456,
		Hash:      common.Hash{0x1},
		StateRoot: common.Hash{0x2},
		Signers:   bitSet,
		Signature: sig,
	}

	got := b.ToBlockCertificate()
	aggregatedSignature := cert.NewAggregatedSignature[cert.BlockStatement](
		b.Signers,
		b.Signature,
	)
	want := cert.NewCertificateWithSignature(cert.NewBlockStatement(
		b.ChainId,
		idx.Block(b.Number),
		b.Hash,
		b.StateRoot),
		aggregatedSignature)

	require.Equal(want, got)
}
