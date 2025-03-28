package inter

import (
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/0xsoniclabs/sonic/utils/cser"
)

type CommitteeSignature struct {
	Period    scc.Period
	Signature cert.Signature[cert.CommitteeStatement]
}

func (e *CommitteeSignature) MarshalCSER(w *cser.Writer) error {
	w.U64(uint64(e.Period))
	sig := e.Signature.Signature.Serialize()
	w.FixedBytes(sig[:])
	return nil
}

func (e *CommitteeSignature) UnmarshalCSER(r *cser.Reader) error {
	period := r.U64()
	sig := [96]byte{}
	r.FixedBytes(sig[:])
	signature, err := bls.DeserializeSignature(sig)
	if err != nil {
		return err
	}
	e.Period = scc.Period(period)
	e.Signature = cert.Signature[cert.CommitteeStatement]{
		Signature: signature,
	}
	return nil
}

type BlockSignature struct {
	Number    idx.Block
	Signature cert.Signature[cert.BlockStatement]
}

func (e *BlockSignature) MarshalCSER(w *cser.Writer) error {
	w.U64(uint64(e.Number))
	sig := e.Signature.Signature.Serialize()
	w.FixedBytes(sig[:])
	return nil
}

func (e *BlockSignature) UnmarshalCSER(r *cser.Reader) error {
	number := r.U64()
	sig := [96]byte{}
	r.FixedBytes(sig[:])
	signature, err := bls.DeserializeSignature(sig)
	if err != nil {
		return err
	}
	e.Number = idx.Block(number)
	e.Signature = cert.Signature[cert.BlockStatement]{
		Signature: signature,
	}
	return nil
}
