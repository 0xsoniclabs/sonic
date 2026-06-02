package rpctest

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

// extractGenesisAccounts converts a go-ethereum Genesis alloc into a map of
// address to AccountState, suitable for the fake backend builder.
func extractGenesisAccounts(genesis *core.Genesis) map[common.Address]AccountState {
	accounts := make(map[common.Address]AccountState, len(genesis.Alloc))

	for addr, account := range genesis.Alloc {
		state := AccountState{
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
