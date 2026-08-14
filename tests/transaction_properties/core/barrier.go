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
	"context"
	"fmt"
	"math/big"
	"slices"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ConfirmationBlocks is how many blocks must be produced after an injection before a transaction that
// has not appeared is declared absent. Two rather than one, because a forced event's transactions land
// in the next block formed after the event is processed, so a one-block budget would race with a
// block already being sealed and report a merely late transaction as absent.
const ConfirmationBlocks = 2

// markerRecipient receives the barrier's marker transactions; a plain address with no code, so a
// marker always succeeds.
var markerRecipient = common.Address{0xba, 0x11, 0x1e, 0x12}

// Outcomes holds the resolved fate of an injected batch, and the blocks produced to resolve it.
type Outcomes struct {
	Receipts     map[common.Hash]*types.Receipt
	MarkerBlocks []uint64
}

func (o Outcomes) OutcomeOf(hash common.Hash) Outcome {
	if _, ok := o.Receipts[hash]; ok {
		return OutcomeExecuted
	}
	return OutcomeDropped
}

// ConfirmOutcomes resolves the fate of every injected transaction, confirming absence in blocks
// rather than in seconds: it drives ConfirmationBlocks blocks and reports what still has no receipt,
// which ties the budget to the only thing that can change the answer instead of to a wall clock a
// loaded machine can exceed. The blocks must be driven rather than waited for, because Sonic paces
// empty blocks and an idle network would otherwise take two full skip periods.
func ConfirmOutcomes(
	ctx context.Context,
	session tests.IntegrationTestNetSession,
	client *tests.PooledEhtClient,
	hashes []common.Hash,
) (Outcomes, error) {

	result := Outcomes{Receipts: make(map[common.Hash]*types.Receipt, len(hashes))}
	pending := slices.Clone(hashes)

	collect := func() error {
		remaining := pending[:0]
		for _, hash := range pending {
			receipt, err := client.TransactionReceipt(ctx, hash)
			switch err {
			case nil:
				result.Receipts[hash] = receipt
			case ethereum.NotFound:
				remaining = append(remaining, hash)
			default:
				return fmt.Errorf("failed to query receipt of %v: %w", hash, err)
			}
		}
		pending = remaining
		return nil
	}

	if err := collect(); err != nil {
		return Outcomes{}, err
	}

	for len(result.MarkerBlocks) < ConfirmationBlocks {
		blockNumber, err := advanceOneBlock(ctx, session, client)
		if err != nil {
			return Outcomes{}, err
		}
		result.MarkerBlocks = append(result.MarkerBlocks, blockNumber)

		if len(pending) == 0 {
			continue
		}
		if err := collect(); err != nil {
			return Outcomes{}, err
		}
	}

	return result, nil
}

// advanceOneBlock forces a single block and returns its number, by running one ordinary transaction
// from the session sponsor, which is never an account under test.
func advanceOneBlock(
	ctx context.Context,
	session tests.IntegrationTestNetSession,
	client *tests.PooledEhtClient,
) (uint64, error) {
	before, err := client.BlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to read the current block number: %w", err)
	}

	receipt, err := session.EndowAccount(markerRecipient, big.NewInt(1))
	if err != nil {
		return 0, fmt.Errorf("failed to force a block: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return 0, fmt.Errorf("marker transaction failed with status %d", receipt.Status)
	}
	if blockNumber := receipt.BlockNumber.Uint64(); blockNumber > before {
		return blockNumber, nil
	}
	return 0, fmt.Errorf(
		"marker transaction landed in block %v, which does not advance past %d",
		receipt.BlockNumber, before,
	)
}
