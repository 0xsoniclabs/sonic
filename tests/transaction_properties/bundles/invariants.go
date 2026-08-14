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

package bundles

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/0xsoniclabs/sonic/api/sonicapi"
	testbundles "github.com/0xsoniclabs/sonic/tests/bundles"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
)

// checkBundle verifies one bundle. The first property holds whatever anything predicted, and is the
// one worth having most: a bundle is all or nothing. What the chain says the bundle did has to agree
// with that, and with the model.
func (d *Domain) checkBundle(ctx context.Context, envelope *Envelope, obs core.Observation) error {
	if envelope.Built == nil {
		return nil // the batch it belonged to never reached a node
	}

	receipts, err := d.receiptsOf(ctx, envelope)
	if err != nil {
		return err
	}

	// 1. Atomicity.
	included := 0
	for _, receipt := range receipts {
		if receipt != nil {
			included++
		}
	}
	if included != 0 && included != len(receipts) {
		return fmt.Errorf(
			"%s put %d of its %d transactions in a block: a bundle is all or nothing",
			envelope, included, len(receipts))
	}
	fate := FateSkipped
	if included > 0 {
		fate = FateExecuted
	}
	d.observed[fate]++

	// 2. What the chain says it did. A bundle that ran and failed records its plan too, with a count
	// of zero -- which is how a failed bundle is told apart from one that was never opened, and why the
	// count is compared against what is in the block rather than against what the bundle carries.
	info, err := testbundles.GetBundleInfo(ctx, d.client.Client(), envelope.Built.PlanHash)
	switch {
	case errors.Is(err, ethereum.NotFound):
		if fate == FateExecuted {
			return fmt.Errorf("%s executed, but no bundle info is recorded for its plan", envelope)
		}
	case err != nil:
		return fmt.Errorf("failed to read the bundle info of %s: %w", envelope, err)
	default:
		if err := d.checkInfo(envelope, info, receipts, included, obs); err != nil {
			return err
		}
	}

	// 3. What the model expected of it.
	expectation := d.Expect(envelope, baseFeeOf(obs))
	d.reasons[expectation.Reason]++
	if !expectation.permits(fate) {
		return fmt.Errorf("%s was %v but the model expects %v", envelope, fate, expectation)
	}

	// 4. The envelope itself never enters a block once bundles mean anything.
	if d.cfg.Upgrades.Brio {
		for _, tx := range obs.Txs {
			if tx.Spec == core.TxSpec(envelope) && tx.Outcome == core.OutcomeExecuted {
				return fmt.Errorf("%s has a receipt of its own, which a carrier must never get",
					envelope)
			}
		}
	}
	return nil
}

// checkInfo verifies the record of a bundle against the receipts of its transactions: the count it
// reports, the block it names, and that its transactions occupy exactly the positions it claims, in
// the order the plan referenced them.
func (d *Domain) checkInfo(
	envelope *Envelope,
	info *sonicapi.RPCBundleInfo,
	receipts []*types.Receipt,
	included int,
	obs core.Observation,
) error {

	if got := int(info.Count); got != included {
		return fmt.Errorf(
			"%s reports %d of its transactions in the block, but %d have a receipt",
			envelope, got, included)
	}
	if included == 0 {
		return nil // the bundle was opened and gave up; the atomicity check has already had its say
	}

	for i, receipt := range receipts {
		position := uint64(info.Position) + uint64(i)
		switch {
		case receipt.BlockNumber.Uint64() != uint64(info.Block):
			return fmt.Errorf(
				"transaction %s of %s is in block %v, but its bundle info names block %d",
				reference(i), envelope, receipt.BlockNumber, uint64(info.Block))
		case uint64(receipt.TransactionIndex) != position:
			return fmt.Errorf(
				"transaction %s of %s is at index %d of its block, but its bundle info puts it at %d",
				reference(i), envelope, receipt.TransactionIndex, position)
		}

		block := blockOf(obs, receipt.BlockNumber)
		if block == nil {
			continue // outside the blocks this iteration read; the indices above already agree
		}
		txs := block.Transactions()
		if position >= uint64(len(txs)) || txs[position].Hash() != envelope.Built.Inner[i].Hash() {
			return fmt.Errorf(
				"transaction %s of %s is not at index %d of block %d, where its bundle info puts it",
				reference(i), envelope, position, block.NumberU64())
		}
	}
	return nil
}

// receiptsOf looks up what became of every transaction of a bundle, nil where there is no receipt.
func (d *Domain) receiptsOf(ctx context.Context, envelope *Envelope) ([]*types.Receipt, error) {
	receipts := make([]*types.Receipt, len(envelope.Built.Inner))
	for i, tx := range envelope.Built.Inner {
		receipt, err := d.client.TransactionReceipt(ctx, tx.Hash())
		switch {
		case errors.Is(err, ethereum.NotFound):
		case err != nil:
			return nil, fmt.Errorf("failed to query the receipt of %v: %w", tx.Hash(), err)
		default:
			receipts[i] = receipt
		}
	}
	return receipts, nil
}

// blockOf finds one of the blocks this iteration read, or nil when the block is not among them.
func blockOf(obs core.Observation, number *big.Int) *types.Block {
	for _, block := range obs.Blocks {
		if block.Number().Cmp(number) == 0 {
			return block
		}
	}
	return nil
}

// baseFeeOf is the base fee the model prices the contents at, taken from the first block this
// iteration produced, which is the one they would have landed in.
func baseFeeOf(obs core.Observation) *big.Int {
	if len(obs.Blocks) == 0 {
		return new(big.Int)
	}
	return obs.Blocks[0].BaseFee()
}
