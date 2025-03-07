package inter

import (
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

type BlockSignature struct {
	Number    idx.Block
	Signature cert.Signature[cert.BlockStatement]
}

type CommitteeSignature struct {
	Period    scc.Period
	Signature cert.Signature[cert.CommitteeStatement]
}
