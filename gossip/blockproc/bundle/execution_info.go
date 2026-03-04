package bundle

import "github.com/ethereum/go-ethereum/common"

// ExecutionInfo contains information about a processed bundle. It connects an
// execution plan's hash with the block number and position in which it got
// executed.
type ExecutionInfo struct {
	ExecutionPlanHash common.Hash
	BlockNum          uint64
	Position          uint32
}
