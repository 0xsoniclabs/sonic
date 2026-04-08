package bundle

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/core/types"
)

// NewBuilder2 creates a new bundle builder to create a custom bundle.
func NewBuilder2() *builder2 {
	return &builder2{}
}

type builder2 struct {
}

func (b *builder2) AllOf(steps ...BundleStep2) *builder2 {
	return b
}

func (b *builder2) OneOf(steps ...BundleStep2) *builder2 {
	return b
}

func (b *builder2) Build() *types.Transaction {
	return nil
}

func Run(flags ExecutionFlags, steps ...BundleStep2) BundleStep2 {
	return BundleStep2{
		flags: flags,
		steps: steps,
	}
}

func AllOf2(steps ...BundleStep2) BundleStep2 {
	return Run(EF_AllOf, steps...)
}

func OneOf2(steps ...BundleStep2) BundleStep2 {
	return Run(EF_OneOf, steps...)
}

func Step2(key *ecdsa.PrivateKey, tx types.TxData) BundleStep2 {
	return BundleStep2{
		key: key,
		tx:  tx,
	}
}

type BundleStep2 struct {
	// -- fields for a single transaction step --
	key *ecdsa.PrivateKey
	tx  types.TxData

	// -- fields for a group step --
	flags ExecutionFlags
	steps []BundleStep2
}
