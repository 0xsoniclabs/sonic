package state

import (
	"github.com/0xsoniclabs/carmen/go/common/witness"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

//go:generate mockgen -source adapter.go -destination adapter_mock.go -package state

type StateDB interface {
	vm.StateDB

	Error() error
	GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log
	SetTxContext(thash common.Hash, ti int)
	TxIndex() int
	GetProof(addr common.Address, keys []common.Hash) (witness.Proof, error)
	SetBalance(addr common.Address, amount *uint256.Int)
	SetStorage(addr common.Address, storage map[common.Hash]common.Hash)
	Copy() StateDB
	GetStateHash() common.Hash

	BeginBlock(number uint64)
	EndBlock(number uint64)
	EndTransaction()
	Release()
}
