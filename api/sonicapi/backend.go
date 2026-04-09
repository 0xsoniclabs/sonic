package sonicapi

import (
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
)

//go:generate mockgen -source=backend.go -destination=backend_mock.go -package=sonicapi

// backend is an interface that abstracts the necessary backend functions for the
// sonic API.
type backend interface {
	GetBundleExecutionInfo(executionPlanHash common.Hash) *bundle.ExecutionInfo
}
