package execution_apis

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	rpctest "github.com/0xsoniclabs/sonic/api/rpc_test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

// LoadGenesis loads a genesis.json file and returns the parsed genesis object.
func LoadGenesis(path string) (*core.Genesis, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading genesis file: %w", err)
	}

	var genesis core.Genesis
	if err := json.Unmarshal(data, &genesis); err != nil {
		return nil, fmt.Errorf("parsing genesis JSON: %w", err)
	}

	return &genesis, nil
}

// GenesisAccounts converts a go-ethereum Genesis alloc into a map of
// address to rpctest.AccountState, suitable for the fake backend builder.
func GenesisAccounts(genesis *core.Genesis) map[common.Address]rpctest.AccountState {
	accounts := make(map[common.Address]rpctest.AccountState, len(genesis.Alloc))

	for addr, account := range genesis.Alloc {
		state := rpctest.AccountState{
			Nonce:   account.Nonce,
			Balance: account.Balance,
			Code:    account.Code,
		}

		if len(account.Storage) > 0 {
			state.Store = make(map[common.Hash]common.Hash, len(account.Storage))
			for k, v := range account.Storage {
				state.Store[k] = v
			}
		}

		accounts[addr] = state
	}

	return accounts
}

// GenesisBlock returns a rpctest.Block representing the genesis block (block 0)
// derived from the genesis specification.
func GenesisBlock(genesis *core.Genesis) rpctest.Block {
	var baseFee *big.Int
	if genesis.BaseFee != nil {
		baseFee = genesis.BaseFee
	}

	return rpctest.Block{
		Number:  0,
		Hash:    common.Hash{}, // will be derived from first chain block's parent
		BaseFee: baseFee,
	}
}
