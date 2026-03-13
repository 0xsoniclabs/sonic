package coretypes

import (
	"github.com/ethereum/go-ethereum/common"
)

//go:generate mockgen -source=interface.go -destination=interface_mock.go -package=coretypes

// DummyChain supports retrieving headers and consensus parameters from the
// current blockchain to be used during transaction processing.
type DummyChain interface {
	// Header returns the header of the block with the given number.
	// If the block is not found, nil is returned.
	// If the hash provided is not zero and does not match, nil is returned.
	Header(hash common.Hash, number uint64) *EvmHeader
}
