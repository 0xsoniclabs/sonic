package rpctest

import (
	"math/big"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

//go:generate mockgen -source=backend.go -destination=backend_mock.go -package=rpctest

type TestAccount struct {
	Nonce   uint64
	Balance *big.Int
	Code    []byte
	Store   map[common.Hash]common.Hash
}

type backendBuilder struct {
	be backend
}

func NewBackendBuilder() backendBuilder {
	return backendBuilder{
		be: backend{
			chainId:      1,
			rules:        opera.FakeNetRules(opera.GetBrioUpgrades()),
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

func (b backendBuilder) WithUpgrade(upgrades opera.Upgrades) backendBuilder {
	b.be.rules = opera.FakeNetRules(upgrades)
	return b
}

func (b backendBuilder) Build() *backend {
	return &b.be
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

type backend struct {
	ethapi.Backend

	chainId      uint64
	rules        opera.Rules
	state        *testState
	pool         txPool
	blockHistory []TestBlock
}

func (b *backend) GetSigner() types.Signer {
	return types.LatestSignerForChainID(b.ChainID())
}
