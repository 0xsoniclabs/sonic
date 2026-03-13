package coretypes

import (
	"math/big"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
)

//go:generate mockgen -source=state_reader.go -destination=state_reader_mock.go -package=coretypes

// StateReader provides the state of blockchain and current gas limit to do
// some pre checks in tx pool and event subscribers.
type StateReader interface {
	CurrentBlock() *EvmBlock
	Block(hash common.Hash, number uint64) *EvmBlock
	CurrentStateDB() (state.StateDB, error)
	CurrentBaseFee() *big.Int
	CurrentMaxGasLimit() uint64
	SubscribeNewBlock(ch chan<- ChainHeadNotify) event.Subscription
	CurrentConfig() *params.ChainConfig
	CurrentRules() opera.Rules
	Header(hash common.Hash, number uint64) *EvmHeader
	HasBundleBeenProcessed(execPlanHash common.Hash) bool
}
