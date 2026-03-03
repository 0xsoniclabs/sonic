package bundle

import "github.com/ethereum/go-ethereum/common"

// ExecutionInfo contains information about a processed bundle that can be used
// for tracking and querying purposes. It includes the bundle's hash, the block
// number, and the position within the block.
type ExecutionInfo struct {
	Hash     common.Hash
	BlockNum uint64
	Position uint32
}
