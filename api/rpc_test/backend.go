package rpctest

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

//go:generate mockgen -source=backend.go -destination=backend_mock.go -package=rpctest

type backend struct {
	chainId      uint64
	state        testState
	pool         txPool
	blockHistory []TestBlock
}

type backendBuilder struct {
	be backend
}

func NewBackendBuilder() backendBuilder {
	return backendBuilder{
		be: backend{
			chainId:      1,
			state:        newTestState(),
			blockHistory: DefaultBlockHistory(),
		},
	}
}

func (b backendBuilder) WithChainId(chainId uint64) backendBuilder {
	b.be.chainId = chainId
	return b
}

func (b backendBuilder) WithBlockHistory(blocks []TestBlock) backendBuilder {
	b.be.blockHistory = blocks
	return b
}

func (b backendBuilder) WithPool(pool txPool) backendBuilder {
	b.be.pool = pool
	return b
}

func (b backendBuilder) WithAccount(addr common.Address, account TestAccount) backendBuilder {
	b.be.state.setAccount(addr, account)
	return b
}

func (b backendBuilder) Build() backend {
	return b.be
}

func DefaultBlockHistory() []TestBlock {
	return []TestBlock{
		{Number: 1},
	}
}

type TestBlock struct {
	Number uint64
}

type txPool interface {
	AddLocal(*types.Transaction) error
}
