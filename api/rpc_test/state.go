package rpctest

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type TestAccount struct {
	Nonce   uint64
	Balance *big.Int
	Code    []byte
	Store   map[common.Hash]common.Hash
}

type testState struct {
	state map[common.Address]TestAccount
}

func newTestState() testState {
	return testState{state: make(map[common.Address]TestAccount)}
}
