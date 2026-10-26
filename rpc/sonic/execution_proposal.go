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
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// RPCExecutionProposal is the JSON-serializable representation of the execution proposal
// that is returned by the API. It is designed to be easily serializable to JSON
// and human-readable for integration purposes.
//
// Each leaf step contains the same fields as the eth_call transaction arguments,
// with the addition of the tolerateFailed and tolerateInvalid flags that
// indicate whether a transaction is allowed to fail or be invalid without
// causing the entire proposal to be rejected.
//
// Steps can be nested into groups with optional oneOf and tolerateFailures
// semantics. The top-level object is the root group of the plan; its oneOf and
// tolerateFailures flags apply to that group. An example of the JSON
// representation:
//
//	{
//	  "blockRange": {
//	    "first": "0xbc614e",
//	    "length": "0x64"
//	  },
//	  "steps": [
//	    {
//	      "from": "0xabc1230000000000000000000000000000000001",
//	      "to": "0xdef4560000000000000000000000000000000002",
//	      "gas": "0x5208",
//	      "value": "0xde0b6b3a7640000",
//	      "chainId": "0x1"
//	    },
//	    {
//	      "oneOf": true,
//	      "steps": [
//	        {
//	          "tolerateFailed": true,
//	          "from": "0xabc1230000000000000000000000000000000001",
//	          "to": "0x1111111111111111111111111111111111111111",
//	          "data": "0xa9059cbb",
//	          "chainId": "0x1"
//	        },
//	        {
//	          "tolerateInvalid": true,
//	          "from": "0xabc1230000000000000000000000000000000001",
//	          "to": "0x2222222222222222222222222222222222222222",
//	          "data": "0xa9059cbb",
//	          "chainId": "0x1"
//	        }
//	      ]
//	    }
//	  ]
//	}
type RPCExecutionProposal struct {
	BlockRange *RPCRange `json:"blockRange,omitempty"`
	RPCExecutionPlanGroup
}

type RPCExecutionStepProposal struct {
	TolerateFailed  bool `json:"tolerateFailed,omitempty"`
	TolerateInvalid bool `json:"tolerateInvalid,omitempty"`
	exec.CallArguments
}

// UnmarshalJSON implements json.Unmarshaler for RPCExecutionProposal.
// Steps []any requires custom handling: each element is either an
// RPCExecutionStepProposal (leaf, no "steps" key) or an RPCExecutionPlanGroup
// (has "steps" key), resolved by unmarshalBundleGroup.
func (proposal *RPCExecutionProposal) UnmarshalJSON(data []byte) error {
	var raw struct {
		BlockRange       *RPCRange         `json:"blockRange,omitempty"`
		TolerateFailures bool              `json:"tolerateFailures,omitempty"`
		OneOf            bool              `json:"oneOf,omitempty"`
		Steps            []json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	proposal.BlockRange = raw.BlockRange
	proposal.OneOf = raw.OneOf
	proposal.TolerateFailures = raw.TolerateFailures
	proposal.Steps = make([]any, len(raw.Steps))
	for index, rawStep := range raw.Steps {
		step, empty, err := unmarshalBundleGroup[RPCExecutionStepProposal](rawStep, 1)
		if err != nil {
			return err
		}
		if !empty {
			proposal.Steps[index] = step
		}
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler for RPCExecutionStepProposal.
// An embedded type with a custom UnmarshalJSON would have it promoted
// to RPCExecutionStepProposal, which would silently drop TolerateFailed and
// TolerateInvalid. This method handles both.
func (step *RPCExecutionStepProposal) UnmarshalJSON(data []byte) error {
	var flags struct {
		TolerateFailed  bool `json:"tolerateFailed"`
		TolerateInvalid bool `json:"tolerateInvalid"`
	}
	if err := json.Unmarshal(data, &flags); err != nil {
		return err
	}
	step.TolerateFailed = flags.TolerateFailed
	step.TolerateInvalid = flags.TolerateInvalid
	return json.Unmarshal(data, &step.CallArguments)
}

// unmarshalBundleGroup discriminates between a leaf (i.e. RPCExecutionStepProposal,
// no "steps" key) and a group (RPCExecutionPlanGroup, has "steps" key).
// This function returns the unmarshaled element, a boolean indicating whenever the
// element can be optimized away and an error if the JSON is invalid. The optimization
// boolean is used to indicate to the caller that the resulting group is empty.
//
// depth is the nesting depth of data, with 0 for the root group, i.e. the
// top-level object (matching the convention of validateStep,
// gossip/blockproc/bundle/validate.go). It is checked against
// bundle.MaxGroupNestingDepth before any parsing of data is attempted, so
// maliciously deep input is rejected cheaply instead of costing quadratic
// decode time (every recursion level re-parses the still fully nested
// remainder of the input).
func unmarshalBundleGroup[LeafType any](data []byte, depth int) (any, bool, error) {
	if depth > bundle.MaxGroupNestingDepth {
		return nil, false, fmt.Errorf(
			"execution plan exceeds maximum nesting depth of %d", bundle.MaxGroupNestingDepth)
	}
	var probe struct {
		Steps *json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, false, err
	}
	if probe.Steps != nil {
		var rawGroup struct {
			TolerateFailures bool              `json:"tolerateFailures"`
			OneOf            bool              `json:"oneOf"`
			Steps            []json.RawMessage `json:"steps"`
		}
		if err := json.Unmarshal(data, &rawGroup); err != nil {
			return nil, false, err
		}
		children := make([]any, len(rawGroup.Steps))
		for index, raw := range rawGroup.Steps {
			child, empty, err := unmarshalBundleGroup[LeafType](raw, depth+1)
			if err != nil {
				return nil, false, err
			}
			if !empty {
				children[index] = child
			}
		}
		return RPCExecutionPlanGroup{
			OneOf:            rawGroup.OneOf,
			TolerateFailures: rawGroup.TolerateFailures,
			Steps:            children,
		}, len(children) == 0, nil
	}
	var step LeafType
	if err := json.Unmarshal(data, &step); err != nil {
		return nil, false, err
	}
	return step, false, nil
}

func convertProposalToPlan(signer types.Signer, proposal RPCExecutionProposal) (bundle.ExecutionPlan, error) {

	root, err := convertProposalToPlanInternal(signer, proposal.RPCExecutionPlanGroup)
	if err != nil {
		return bundle.ExecutionPlan{}, err
	}

	if proposal.BlockRange == nil {
		return bundle.ExecutionPlan{}, fmt.Errorf("execution proposal must include block range")
	}

	return bundle.ExecutionPlan{
		Range:  proposal.BlockRange.toBundleBlockRange(),
		Root:   root,
		Period: bundle.MakeUnrestrictedTimePeriod(),
	}, nil
}

func convertProposalToPlanInternal(signer types.Signer, proposalStep any) (bundle.ExecutionStep, error) {
	empty := bundle.ExecutionStep{}

	switch step := proposalStep.(type) {
	case RPCExecutionStepProposal:
		if step.From == nil {
			return empty, fmt.Errorf("transaction in bundle must include from")
		}

		transaction, err := toTransactionForBundles(step.CallArguments)
		if err != nil {
			return empty, fmt.Errorf("invalid transaction in bundle: %w", err)
		}
		hash := signer.Hash(transaction)

		return bundle.NewTxStep(bundle.TxReference{
			From: *step.From,
			Hash: hash,
		}).WithFlags(func() bundle.ExecutionFlags {
			flags := bundle.EF_Default
			if step.TolerateFailed {
				flags |= bundle.EF_TolerateFailed
			}
			if step.TolerateInvalid {
				flags |= bundle.EF_TolerateInvalid
			}
			return flags
		}()), nil

	case RPCExecutionPlanGroup:

		if len(step.Steps) == 0 {
			return empty, fmt.Errorf("proposed group must include at least one step")
		}

		steps := make([]bundle.ExecutionStep, len(step.Steps))
		for index, stepLevel := range step.Steps {
			childStep, err := convertProposalToPlanInternal(signer, stepLevel)
			if err != nil {
				return empty, fmt.Errorf("invalid execution plan level: %w", err)
			}
			steps[index] = childStep
		}

		group := bundle.NewGroupStep(step.OneOf, steps...)
		if step.TolerateFailures {
			group = group.WithFlags(bundle.EF_TolerateFailed)
		}
		return group, nil
	}

	return empty, fmt.Errorf("invalid execution proposal level: must have either executionStep or group")
}

// toTransactionForBundles converts exec.CallArguments to types.Transaction
// for computing execution plan transaction reference hashes. Legacy transactions
// are promoted to AccessList transactions to prevent poisoning the plan hash.
func toTransactionForBundles(step exec.CallArguments) (*types.Transaction, error) {
	transaction, err := step.ToTransaction()
	if err != nil {
		return nil, err
	}

	// legacy transactions cannot be included in a bundle.
	// if the transaction arguments correspond to a legacy transaction,
	// promote it to an access list transaction to host the bundle marker.
	if transaction.Type() == types.LegacyTxType {
		transaction = types.NewTx(&types.AccessListTx{
			Nonce:    transaction.Nonce(),
			To:       transaction.To(),
			Value:    transaction.Value(),
			Gas:      transaction.Gas(),
			Data:     transaction.Data(),
			GasPrice: transaction.GasPrice(),
		})
	}
	return transaction, nil
}

func transform(
	proposal RPCExecutionProposal,
	function func(step RPCExecutionStepProposal) (RPCExecutionStepProposal, error),
	depth int,
) (RPCExecutionProposal, error) {
	// The steps of this group are one level deeper than the group itself.
	if depth+1 > bundle.MaxGroupNestingDepth {
		return proposal, fmt.Errorf(
			"execution plan exceeds maximum nesting depth of %d", bundle.MaxGroupNestingDepth)
	}

	if proposal.Steps == nil {
		return proposal, nil
	}

	resultSteps := make([]any, 0, len(proposal.Steps))
	for _, step := range proposal.Steps {
		switch step := step.(type) {
		case RPCExecutionStepProposal:
			step, err := function(step)
			if err != nil {
				return proposal, err
			}
			resultSteps = append(resultSteps, step)

		case RPCExecutionPlanGroup:
			result, err := transform(RPCExecutionProposal{
				BlockRange:            proposal.BlockRange,
				RPCExecutionPlanGroup: step,
			}, function, depth+1)
			if err != nil {
				return RPCExecutionProposal{}, err
			}
			resultSteps = append(resultSteps, result.RPCExecutionPlanGroup)

		default:
			return RPCExecutionProposal{}, fmt.Errorf("invalid execution plan level: must have either executionStep or group")
		}
	}
	return RPCExecutionProposal{
		BlockRange: proposal.BlockRange,
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			OneOf:            proposal.OneOf,
			TolerateFailures: proposal.TolerateFailures,
			Steps:            resultSteps,
		},
	}, nil
}
