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

package subsidies

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// GasConfig is what the registry charges beyond the transaction's own gas, read from the chain once
// per network rather than assumed, since a registry is free to answer differently.
type GasConfig struct {
	// FundBackedOverhead is the gas a fund-backed sponsorship is charged for its deductFees
	// follow-up, on top of the transaction's own gas used.
	FundBackedOverhead uint64

	// TrackedOverhead is the same for a network sponsorship with tracking.
	TrackedOverhead uint64
}

// MaxOverhead is the overhead IsCovered reserves when it asks the registry whether a fund covers a
// transaction: the larger of the two, whatever mode the answer turns out to be.
func (c GasConfig) MaxOverhead() uint64 {
	return max(c.FundBackedOverhead, c.TrackedOverhead)
}

// Registry is the subsidies registry as this domain uses it: reading the gas configuration and the
// balance of a fund, and paying into one.
type Registry struct {
	registry *registry.Registry
	session  tests.IntegrationTestNetSession
	client   *tests.PooledEhtClient
	chainId  *big.Int

	// treasurer pays for every top-up. It is one account of the pool, claimed for the life of the
	// network and never released, so it is never an account under observation, and stocked on claim,
	// since the pool may hand over one that earlier iterations have drained. The session sponsor
	// cannot do this job: it also drives every block of the barrier, and a run of a few hundred
	// iterations paying several funds apiece drains it.
	treasurer core.PooledAccount
}

// NewRegistry binds the registry deployed at the well-known address and claims the account that pays
// into the funds.
func NewRegistry(
	session tests.IntegrationTestNetSession,
	client *tests.PooledEhtClient,
	network *core.Network,
) (*Registry, error) {

	bound, err := registry.NewRegistry(registry.GetAddress(), client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind the subsidies registry: %w", err)
	}

	treasurer, err := network.ClaimPayer()
	if err != nil {
		return nil, fmt.Errorf("failed to claim the account paying into the funds: %w", err)
	}

	return &Registry{
		registry:  bound,
		session:   session,
		client:    client,
		chainId:   network.Cfg.ChainId,
		treasurer: treasurer,
	}, nil
}

// GasConfig reads the overheads the registry charges.
func (r *Registry) GasConfig() (GasConfig, error) {
	config, err := r.registry.GetGasConfig(nil)
	if err != nil {
		return GasConfig{}, fmt.Errorf("failed to read the registry's gas config: %w", err)
	}
	if !config.OverheadChargeForFundBackedSponsorships.IsUint64() ||
		!config.OverheadChargeForNetworkSponsorshipsWithTracking.IsUint64() {
		return GasConfig{}, fmt.Errorf("the registry reports an overhead beyond a gas limit")
	}
	return GasConfig{
		FundBackedOverhead: config.OverheadChargeForFundBackedSponsorships.Uint64(),
		TrackedOverhead:    config.OverheadChargeForNetworkSponsorshipsWithTracking.Uint64(),
	}, nil
}

// AccountFund is the fund covering everything one account sends, which is the only kind this domain
// ever pays into: chooseFund tries the approval, call, account, contract, bootstrap and global funds
// in that order, and leaving every other one empty is what lets the model decide coverage from a
// single balance. EmptyFundsElsewhere checks that assumption holds.
func (r *Registry) AccountFund(account common.Address) ([32]byte, error) {
	ok, id, err := r.registry.AccountSponsorshipFundId(nil, account)
	if err != nil {
		return id, fmt.Errorf("failed to derive the fund of %v: %w", account, err)
	}
	if !ok {
		return id, fmt.Errorf("the registry has no account fund for %v", account)
	}
	return id, nil
}

// Funds reads what a fund holds, at a block the archive can answer for.
//
// The pinning is not caution, it is the whole point. A call resolves state against the archive, which
// lags the head while blocks are being produced -- six blocks, in a run that drives them as fast as
// this one does. A call against a block the archive has not ingested yet answers **zero** rather than
// failing, and a fund of zero is exactly what an unfunded sender looks like: the model would then
// expect every request to be refused while the chain sponsors them all. Pinned to a block, the same
// call says so instead, and this waits for the archive to catch up rather than believing it.
//
// tests.WaitForProofOf is the same guard by another name, which is why the hand-written funding
// helper in tests/gas_subsidies calls it around every read.
func (r *Registry) Funds(ctx context.Context, id [32]byte, block uint64) (*big.Int, error) {
	head := block
	if head == 0 {
		var err error
		if head, err = r.client.BlockNumber(ctx); err != nil {
			return nil, fmt.Errorf("failed to read the block number: %w", err)
		}
	}
	at := &bind.CallOpts{BlockNumber: new(big.Int).SetUint64(head), Context: ctx}

	deadline := time.Now().Add(archivePatience)
	for {
		fund, err := r.registry.Sponsorships(at, id)
		if err == nil {
			return fund.Funds, nil
		}
		if !strings.Contains(err.Error(), "not present in the archive") {
			return nil, fmt.Errorf("failed to read fund %x at block %d: %w", id, head, err)
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("%w: block %d within %v, so what fund %x holds cannot be read",
				ErrArchiveBehind, head, archivePatience, id)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(archivePollInterval):
		}
	}
}

// ErrArchiveBehind says the archive never reached the block a read needed. It is worth telling apart,
// because a fork that reproduces known defect 2 leaves state the blocks do not account for, and an
// archive is exactly a re-derivation of state from blocks -- so on such a fork the archive stops and no
// fund can be read at all.
var ErrArchiveBehind = errors.New("the archive did not reach the block a fund had to be read at")

// archivePatience is how long a read waits for the archive to reach the head, and how often it asks.
// A caught-up archive answers in well under a millisecond; the wait is for the backlog a busy
// iteration leaves behind.
const (
	archivePatience     = 20 * time.Second
	archivePollInterval = 5 * time.Millisecond
)

// FundsOf reads what the fund of one account holds.
func (r *Registry) FundsOf(ctx context.Context, account common.Address, block uint64) (*big.Int, error) {
	id, err := r.AccountFund(account)
	if err != nil {
		return nil, err
	}
	return r.Funds(ctx, id, block)
}

// Donation is wei to pay into the fund of one account.
type Donation struct {
	Account common.Address
	Amount  *big.Int
}

// TopUp pays into the funds of several accounts at once, sending every payment before waiting for any
// of them, so a batch that funds three senders still costs about one block rather than three. It
// returns the newest block one of the payments landed in, which is what a later read of those funds
// pins itself to: a block that old the archive has, while the head it is still catching up with.
func (r *Registry) TopUp(ctx context.Context, donations []Donation) (uint64, error) {
	if len(donations) == 0 {
		return 0, nil
	}

	nonce, err := r.client.PendingNonceAt(ctx, r.treasurer.Address())
	if err != nil {
		return 0, fmt.Errorf("failed to read the nonce of the funding account: %w", err)
	}

	txs := make([]*types.Transaction, 0, len(donations))
	for _, donation := range donations {
		id, err := r.AccountFund(donation.Account)
		if err != nil {
			return 0, err
		}

		opts, err := bind.NewKeyedTransactorWithChainID(r.treasurer.PrivateKey, r.chainId)
		if err != nil {
			return 0, fmt.Errorf("failed to build a transactor for the funding account: %w", err)
		}
		opts.Value = donation.Amount
		opts.Nonce = new(big.Int).SetUint64(nonce)
		opts.NoSend = true

		tx, err := r.registry.Sponsor(opts, id)
		if err != nil {
			return 0, fmt.Errorf("failed to build the payment into the fund of %v: %w",
				donation.Account, err)
		}
		txs = append(txs, tx)
		nonce++
	}

	receipts, err := r.session.RunAll(txs)
	if err != nil {
		return 0, fmt.Errorf("failed to pay into %d fund(s): %w", len(txs), err)
	}
	landed := uint64(0)
	for i, receipt := range receipts {
		if receipt.Status != types.ReceiptStatusSuccessful {
			return 0, fmt.Errorf("paying into the fund of %v failed with status %d",
				donations[i].Account, receipt.Status)
		}
		landed = max(landed, receipt.BlockNumber.Uint64())
	}
	return landed, nil
}

// EmptyFundsElsewhere verifies that the funds the model treats as empty really are, so that a
// coverage decision follows from the account fund alone. The global fund covers everything and the
// bootstrap fund covers an account's first three transactions -- every pooled account starts at
// nonce zero -- so either one holding anything would silently sponsor transactions the model
// expects nobody to pay for.
func (r *Registry) EmptyFundsElsewhere(ctx context.Context) error {
	global, id, err := r.registry.GlobalSponsorshipFundId(nil)
	if err != nil {
		return fmt.Errorf("failed to derive the global fund: %w", err)
	}
	if global {
		if funds, err := r.Funds(ctx, id, 0); err != nil {
			return err
		} else if funds.Sign() != 0 {
			return fmt.Errorf("the global sponsorship fund holds %v wei", funds)
		}
	}

	for nonce := range uint64(3) {
		bootstrap, id, err := r.registry.BootstrapSponsorshipFund(nil, new(big.Int).SetUint64(nonce))
		if err != nil {
			return fmt.Errorf("failed to derive the bootstrap fund for nonce %d: %w", nonce, err)
		}
		if !bootstrap {
			continue
		}
		if funds, err := r.Funds(ctx, id, 0); err != nil {
			return err
		} else if funds.Sign() != 0 {
			return fmt.Errorf("the bootstrap sponsorship fund holds %v wei", funds)
		}
	}
	return nil
}

// label names a draw after the transaction it belongs to, matching how the generators in core label
// theirs.
func label(name string, index int) string {
	return name + "_" + strconv.Itoa(index)
}
