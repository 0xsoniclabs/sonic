// Copyright 2026 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"

	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// AccountBalance is the balance every pooled account is given in the genesis block. It is a constant
// rather than a generated value so that the model can decide the Affordability boundary exactly
// without observing the chain; the transaction's cost is the generated quantity instead, which is
// strictly more controllable.
var AccountBalance = new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil) // 1e24 wei

// MaxGenesisAccounts caps the pool. This is a workaround, not a considered limit: starting a test net
// with roughly 1536 or more genesis accounts panics inside Carmen with "unable to store account node
// with dirty hash", which reproduces in seconds with nothing but StartIntegrationTestNet and that
// many plain accounts, while 1024 import fine. The cap costs nothing here, since every account is
// returned after use.
const MaxGenesisAccounts = 512

// AccountPool hands out accounts funded in the genesis block, keeping those still at their genesis
// nonce apart from those a transaction has executed from and handing the untouched ones out first.
// Funding at run time would cost a block per iteration and dominate the test; reusing accounts is
// safe because every nonce and balance is read from the chain before use, and preferring untouched
// ones only makes a shrink replay more likely to see what the original run saw.
type AccountPool struct {
	Accounts []*tests.Account

	mu        sync.Mutex
	untouched []int
	used      []int
	handedOut int
	dirtied   int
}

// NewAccountPool derives size accounts from a fixed seed, so that the genesis written at startup and
// the keys used by the test agree without anything being persisted between them.
func NewAccountPool(size int) (*AccountPool, error) {
	pool := &AccountPool{
		Accounts:  make([]*tests.Account, size),
		untouched: make([]int, size),
	}
	for i := range pool.Accounts {
		key, err := DeriveKey(i)
		if err != nil {
			return nil, fmt.Errorf("failed to derive key %d: %w", i, err)
		}
		pool.Accounts[i] = &tests.Account{PrivateKey: key}
		pool.untouched[i] = i
	}
	return pool, nil
}

func DeriveKey(index int) (*ecdsa.PrivateKey, error) {
	return crypto.ToECDSA(crypto.Keccak256(fmt.Appendf(nil, "txprop-account-%d", index)))
}

// GenesisAccounts returns the genesis entries funding the whole pool, to be passed as
// IntegrationTestNetOptions.Accounts.
func (p *AccountPool) GenesisAccounts() []makefakegenesis.Account {
	balance := uint256.MustFromBig(AccountBalance)
	out := make([]makefakegenesis.Account, len(p.Accounts))
	for i, account := range p.Accounts {
		out[i] = makefakegenesis.Account{
			Name:    fmt.Sprintf("txprop-%d", i),
			Address: account.Address(),
			Balance: balance,
		}
	}
	return out
}

// PooledAccount is an account on loan from the pool.
type PooledAccount struct {
	*tests.Account
	Index int
}

// Claim hands out n accounts that no other iteration is currently using. It cannot run dry, because
// everything handed out is returned; it fails only on a static configuration mistake.
func (p *AccountPool) Claim(n int) ([]PooledAccount, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if free := len(p.untouched) + len(p.used); free < n {
		return nil, fmt.Errorf(
			"%d accounts requested but only %d of %d are free", n, free, len(p.Accounts),
		)
	}

	out := make([]PooledAccount, n)
	for i := range out {
		var index int
		if len(p.untouched) > 0 {
			index, p.untouched = p.untouched[0], p.untouched[1:]
		} else {
			index, p.used = p.used[0], p.used[1:]
		}
		out[i] = PooledAccount{Account: p.Accounts[index], Index: index}
	}
	p.handedOut += n
	return out, nil
}

// Release returns an account to the pool, which every claimed account must do whether or not its
// state changed. The dirty flag says whether a transaction of this account executed, and so which
// list it rejoins.
func (p *AccountPool) Release(account PooledAccount, dirty bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if dirty {
		p.used = append(p.used, account.Index)
		p.dirtied++
		return
	}
	p.untouched = append(p.untouched, account.Index)
}

// Stats reports pool usage, so a run can show whether the sizing is sound and whether transactions
// reach execution at all.
func (p *AccountPool) Stats() (handedOut, dirtied, untouched int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.handedOut, p.dirtied, len(p.untouched)
}
