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

package sonic

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// executionProposal is the JSON form of a bundle to prepare: the blocks it
// may run in and its root group, whose leaves are unsigned transactions.
type executionProposal struct {
	BlockRange *blockRange `json:"blockRange,omitempty"`
	group
}

func (p *executionProposal) UnmarshalJSON(data []byte) error {
	var raw struct {
		BlockRange       *blockRange       `json:"blockRange,omitempty"`
		TolerateFailures bool              `json:"tolerateFailures,omitempty"`
		OneOf            bool              `json:"oneOf,omitempty"`
		Steps            []json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	steps, err := decodeSteps[proposedStep](raw.Steps, 1)
	if err != nil {
		return err
	}
	*p = executionProposal{BlockRange: raw.BlockRange, group: group{TolerateFailures: raw.TolerateFailures, OneOf: raw.OneOf, Steps: steps}}
	return nil
}

// proposedStep is an unsigned transaction of a proposal with its flags.
type proposedStep struct {
	TolerateFailed  bool `json:"tolerateFailed,omitempty"`
	TolerateInvalid bool `json:"tolerateInvalid,omitempty"`
	exec.CallArguments
}

// UnmarshalJSON decodes the flags and the arguments of a step apart, as the
// legacy handler does.
func (s *proposedStep) UnmarshalJSON(data []byte) error {
	var flags struct {
		TolerateFailed  bool `json:"tolerateFailed"`
		TolerateInvalid bool `json:"tolerateInvalid"`
	}
	if err := json.Unmarshal(data, &flags); err != nil {
		return err
	}
	s.TolerateFailed, s.TolerateInvalid = flags.TolerateFailed, flags.TolerateInvalid
	return json.Unmarshal(data, &s.CallArguments)
}

// transform answers the proposal with each step replaced by what change makes
// of it, in depth-first order. A group at a depth beyond the nesting limit,
// the root's depth being 0, or a step that is neither leaf nor group fails it.
func transform(proposal group, change func(proposedStep) (proposedStep, error), depth int) (group, error) {
	if depth+1 > bundle.MaxGroupNestingDepth {
		return proposal, fmt.Errorf("execution plan exceeds maximum nesting depth of %d", bundle.MaxGroupNestingDepth)
	}
	if proposal.Steps == nil {
		return proposal, nil
	}
	steps := make([]any, 0, len(proposal.Steps))
	for _, step := range proposal.Steps {
		switch step := step.(type) {
		case proposedStep:
			changed, err := change(step)
			if err != nil {
				return proposal, err
			}
			steps = append(steps, changed)
		case group:
			changed, err := transform(step, change, depth+1)
			if err != nil {
				return group{}, err
			}
			steps = append(steps, changed)
		default:
			return group{}, errNoStepOrGroup
		}
	}
	return group{OneOf: proposal.OneOf, TolerateFailures: proposal.TolerateFailures, Steps: steps}, nil
}

// flatten checks the transactions of a proposal and answers them in
// depth-first order.
func flatten(proposal group) ([]exec.CallArguments, error) {
	transactions := []exec.CallArguments{}
	_, err := transform(proposal, func(step proposedStep) (proposedStep, error) {
		switch {
		case step.Nonce == nil:
			return step, errors.New("proposed transaction is missing nonce")
		case step.From == nil:
			return step, errors.New("proposed transaction is missing from field")
		case step.GasPrice != nil && step.MaxFeePerGas != nil:
			return step, errors.New("proposed transaction cannot have both gasPrice and maxFeePerGas set")
		}
		transactions = append(transactions, step.CallArguments)
		return step, nil
	}, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to flatten transactions: %w", err)
	}
	return transactions, nil
}

func needsGas(transaction exec.CallArguments) bool {
	return transaction.Gas == nil || *transaction.Gas == 0
}

// checkSequentialEstimation refuses a proposal whose flags let a transaction
// not run or run in vain, after which the estimates of the transactions after
// it, which assume it ran, are of no use.
func checkSequentialEstimation(proposal group) error {
	if proposal.TolerateFailures {
		return errors.New("gas estimation not supported when tolerateFailures is set: failed transactions leave state uncertain for subsequent transactions")
	}
	if proposal.OneOf {
		return errors.New("gas estimation not supported when oneOf is set: only one branch executes, making state uncertain for sibling branches")
	}
	for _, step := range proposal.Steps {
		switch step := step.(type) {
		case proposedStep:
			if step.TolerateFailed {
				return errors.New("gas estimation not supported when tolerateFailed is set: failed transaction leaves state uncertain for subsequent transactions")
			}
			if step.TolerateInvalid {
				return errors.New("gas estimation not supported when tolerateInvalid is set: invalid transaction may be skipped, leaving state uncertain for subsequent transactions")
			}
		case group:
			if err := checkSequentialEstimation(step); err != nil {
				return err
			}
		}
	}
	return nil
}

// proposalToPlan is the execution plan of a proposal that flatten checked and
// whose transactions have their defaults: each step references its
// transaction by the hash the signer signs.
func proposalToPlan(signer types.Signer, blocks blockRange, proposal group) (bundle.ExecutionPlan, error) {
	root, err := proposedGroupToPlan(signer, proposal)
	if err != nil {
		return bundle.ExecutionPlan{}, err
	}
	return bundle.ExecutionPlan{Range: blocks.toBundle(), Root: root, Period: bundle.MakeUnrestrictedTimePeriod()}, nil
}

func proposedGroupToPlan(signer types.Signer, proposal group) (bundle.ExecutionStep, error) {
	if len(proposal.Steps) == 0 {
		return bundle.ExecutionStep{}, errors.New("proposed group must include at least one step")
	}
	steps := make([]bundle.ExecutionStep, len(proposal.Steps))
	for i, step := range proposal.Steps {
		var err error
		switch step := step.(type) {
		case proposedStep:
			steps[i], err = proposedStepToPlan(signer, step)
		case group:
			steps[i], err = proposedGroupToPlan(signer, step)
		default:
			err = errors.New("invalid execution proposal level: must have either executionStep or group")
		}
		if err != nil {
			return bundle.ExecutionStep{}, fmt.Errorf("invalid execution plan level: %w", err)
		}
	}
	return newGroupStep(proposal, steps), nil
}

func proposedStepToPlan(signer types.Signer, step proposedStep) (bundle.ExecutionStep, error) {
	transaction, err := bundleTransaction(step.CallArguments)
	if err != nil {
		return bundle.ExecutionStep{}, fmt.Errorf("invalid transaction in bundle: %w", err)
	}
	reference := bundle.TxReference{From: *step.From, Hash: signer.Hash(transaction)}
	return bundle.NewTxStep(reference).WithFlags(executionFlags(step.TolerateFailed, step.TolerateInvalid)), nil
}

// bundleTransaction is the transaction the arguments name. A legacy
// transaction cannot carry the bundle's marker in its access list, so it
// becomes an access list transaction.
func bundleTransaction(arguments exec.CallArguments) (*types.Transaction, error) {
	transaction, err := arguments.ToTransaction()
	if err != nil {
		return nil, err
	}
	if transaction.Type() != types.LegacyTxType {
		return transaction, nil
	}
	return types.NewTx(&types.AccessListTx{
		Nonce:    transaction.Nonce(),
		To:       transaction.To(),
		Value:    transaction.Value(),
		Gas:      transaction.Gas(),
		Data:     transaction.Data(),
		GasPrice: transaction.GasPrice(),
	}), nil
}

// addPlanMarker adds the bundle-only address with the hash of the plan to
// the access list of every transaction.
func addPlanMarker(transactions []exec.CallArguments, planHash common.Hash) {
	for i := range transactions {
		var accessList types.AccessList
		if transactions[i].AccessList != nil {
			accessList = *transactions[i].AccessList
		}
		accessList = append(accessList, types.AccessTuple{Address: bundle.BundleOnly, StorageKeys: []common.Hash{planHash}})
		transactions[i].AccessList = &accessList
	}
}
