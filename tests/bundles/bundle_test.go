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
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/0xsoniclabs/sonic/tests/contracts/revert"
	"github.com/0xsoniclabs/sonic/tests/gas_subsidies"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type txType interface {
	makeUnit(txMakeOptions, *AccountFactory) bundle.BundleUnit
}

type txIndex int

const (
	uncheckedTxIndex txIndex = -1
)

type txStatus uint64

const (
	successStatus txStatus = txStatus(types.ReceiptStatusSuccessful)
	failedStatus  txStatus = txStatus(types.ReceiptStatusFailed)
)

type Case struct {
	oneOf            bool
	tolerateFailed   bool
	tolerateInvalid  bool
	submittedTxTypes []txType
	blockTxIndices   []txIndex
	blockTxStatuses  []txStatus
	counter          int64
}

type NamedCase struct {
	name  string
	case_ Case
}

type SubCaseVariant struct {
	submittedTxTypes txType
	blockTxIndices   []txIndex
	blockTxStatuses  []txStatus
	counter          int64
}

type SubCase struct {
	success SubCaseVariant
	failed  SubCaseVariant
	invalid SubCaseVariant
}

// getSubcases returns a map from subcase names to subcases. Each subcase contains three variants: success, failed, and invalid, which specify the expected outcomes for each scenario.
// The subcases are intended to be used as part of a bundle.
func getSubcases() map[string]SubCase {
	return map[string]SubCase{
		"normal": {
			success: SubCaseVariant{
				successfulNormalTx{},
				[]txIndex{uncheckedTxIndex}, // relative 0
				[]txStatus{successStatus},
				1,
			},
			failed: SubCaseVariant{
				failedNormalTx{},
				[]txIndex{uncheckedTxIndex}, // relative 0
				[]txStatus{failedStatus},
				0,
			},
			invalid: SubCaseVariant{
				invalidNormalTx{},
				[]txIndex{},
				[]txStatus{},
				0,
			},
		},
		"sponsored": {
			success: SubCaseVariant{
				successfulSponsoredTx{},
				[]txIndex{uncheckedTxIndex, uncheckedTxIndex}, // relative 0, uncheckedTxIndex
				[]txStatus{successStatus, successStatus},
				1,
			},
			failed: SubCaseVariant{
				failedSponsoredTx{},
				[]txIndex{uncheckedTxIndex, uncheckedTxIndex}, // relative 0, uncheckedTxIndex
				[]txStatus{failedStatus, successStatus},
				0,
			},
			invalid: SubCaseVariant{
				invalidSponsoredTx{},
				[]txIndex{},
				[]txStatus{},
				0,
			},
		},
		"layered/OneOf=false/TolerateFailed=false/TolerateInvalid=false": {
			success: SubCaseVariant{
				subLayerTx{flags: 0, txTypes: []txType{successfulNormalTx{}, successfulNormalTx{}}},
				[]txIndex{uncheckedTxIndex, uncheckedTxIndex},
				[]txStatus{successStatus, successStatus},
				2,
			},
			failed: SubCaseVariant{
				subLayerTx{flags: 0, txTypes: []txType{successfulNormalTx{}, failedNormalTx{}}},
				[]txIndex{},
				[]txStatus{},
				0,
			},
			// skipped bundles are no longer possible, and all **/layered/**/invalid tests are skipped
		},
		"layered/OneOf=false/TolerateFailed=false/TolerateInvalid=true": {
			success: SubCaseVariant{
				subLayerTx{flags: bundle.EF_TolerateInvalid, txTypes: []txType{invalidNormalTx{}, successfulNormalTx{}}},
				[]txIndex{uncheckedTxIndex},
				[]txStatus{successStatus},
				1,
			},
			failed: SubCaseVariant{
				subLayerTx{flags: bundle.EF_TolerateInvalid, txTypes: []txType{successfulNormalTx{}, failedNormalTx{}}},
				[]txIndex{},
				[]txStatus{},
				0,
			},
			// skipped bundles are no longer possible, and all **/layered/**/invalid tests are skipped
		},
		"layered/OneOf=false/TolerateFailed=true/TolerateInvalid=false": {
			success: SubCaseVariant{
				subLayerTx{flags: bundle.EF_TolerateFailed, txTypes: []txType{failedNormalTx{}, successfulNormalTx{}}},
				[]txIndex{uncheckedTxIndex, uncheckedTxIndex},
				[]txStatus{failedStatus, successStatus},
				1,
			},
			failed: SubCaseVariant{
				subLayerTx{flags: bundle.EF_TolerateFailed, txTypes: []txType{successfulNormalTx{}, invalidNormalTx{}}},
				[]txIndex{},
				[]txStatus{},
				0,
			},
			// skipped bundles are no longer possible, and all **/layered/**/invalid tests are skipped
		},
		"layered/OneOf=false/TolerateFailed=true/TolerateInvalid=true": {
			success: SubCaseVariant{
				subLayerTx{flags: bundle.EF_TolerateFailed | bundle.EF_TolerateInvalid, txTypes: []txType{invalidNormalTx{}, failedNormalTx{}, successfulNormalTx{}}},
				[]txIndex{uncheckedTxIndex, uncheckedTxIndex},
				[]txStatus{failedStatus, successStatus},
				1,
			},
			// a bundle can not fail if OneOf is not set and both TolerateFailed and TolerateInvalid are set
			// skipped bundles are no longer possible, and all **/layered/**/invalid tests are skipped
		},
		"layered/OneOf=true/TolerateFailed=false/TolerateInvalid=false": {
			success: SubCaseVariant{
				subLayerTx{flags: bundle.EF_OneOf, txTypes: []txType{invalidNormalTx{}, failedNormalTx{}, successfulNormalTx{}}},
				[]txIndex{uncheckedTxIndex, uncheckedTxIndex},
				[]txStatus{failedStatus, successStatus},
				1,
			},
			failed: SubCaseVariant{
				subLayerTx{flags: bundle.EF_OneOf, txTypes: []txType{failedNormalTx{}, invalidNormalTx{}}},
				[]txIndex{},
				[]txStatus{},
				0,
			},
			// skipped bundles are no longer possible, and all **/layered/**/invalid tests are skipped
		},
		"layered/OneOf=true/TolerateFailed=false/TolerateInvalid=true": {
			success: SubCaseVariant{
				subLayerTx{flags: bundle.EF_OneOf | bundle.EF_TolerateInvalid, txTypes: []txType{failedNormalTx{}, invalidNormalTx{}}},
				[]txIndex{uncheckedTxIndex},
				[]txStatus{failedStatus},
				0,
			},
			failed: SubCaseVariant{
				subLayerTx{flags: bundle.EF_OneOf | bundle.EF_TolerateInvalid, txTypes: []txType{failedNormalTx{}, failedNormalTx{}}},
				[]txIndex{},
				[]txStatus{},
				0,
			},
			// skipped bundles are no longer possible, and all **/layered/**/invalid tests are skipped
		},
		"layered/OneOf=true/TolerateFailed=true/TolerateInvalid=false": {
			success: SubCaseVariant{
				subLayerTx{flags: bundle.EF_OneOf | bundle.EF_TolerateFailed, txTypes: []txType{invalidNormalTx{}, failedNormalTx{}}},
				[]txIndex{uncheckedTxIndex},
				[]txStatus{failedStatus},
				0,
			},
			failed: SubCaseVariant{
				subLayerTx{flags: bundle.EF_OneOf | bundle.EF_TolerateFailed, txTypes: []txType{invalidNormalTx{}, invalidNormalTx{}}},
				[]txIndex{},
				[]txStatus{},
				0,
			},
			// skipped bundles are no longer possible, and all **/layered/**/invalid tests are skipped
		},
		"layered/OneOf=true/TolerateFailed=true/TolerateInvalid=true": {
			success: SubCaseVariant{
				subLayerTx{flags: bundle.EF_OneOf | bundle.EF_TolerateFailed | bundle.EF_TolerateInvalid, txTypes: []txType{invalidNormalTx{}, successfulNormalTx{}}},
				[]txIndex{},
				[]txStatus{},
				0,
			},
			failed: SubCaseVariant{
				subLayerTx{flags: bundle.EF_OneOf | bundle.EF_TolerateFailed | bundle.EF_TolerateInvalid, txTypes: []txType{}},
				[]txIndex{},
				[]txStatus{},
				0,
			},
			// skipped bundles are no longer possible, and all **/layered/**/invalid tests are skipped
		},
		"bundled/OneOf=false/TolerateFailed=false/TolerateInvalid=false": {
			success: SubCaseVariant{
				subBundleTx{flags: 0, txTypes: []txType{successfulNormalTx{}, successfulNormalTx{}}},
				[]txIndex{uncheckedTxIndex, uncheckedTxIndex},
				[]txStatus{successStatus, successStatus},
				2,
			},
			failed: SubCaseVariant{
				subBundleTx{flags: 0, txTypes: []txType{successfulNormalTx{}, failedNormalTx{}}},
				[]txIndex{},
				[]txStatus{},
				0,
			},
			// skipped bundles are no longer possible, and all **/bundled/**/invalid tests are skipped
		},
		"bundled/OneOf=false/TolerateFailed=false/TolerateInvalid=true": {
			success: SubCaseVariant{
				subBundleTx{flags: bundle.EF_TolerateInvalid, txTypes: []txType{invalidNormalTx{}, successfulNormalTx{}}},
				[]txIndex{uncheckedTxIndex},
				[]txStatus{successStatus},
				1,
			},
			failed: SubCaseVariant{
				subBundleTx{flags: bundle.EF_TolerateInvalid, txTypes: []txType{successfulNormalTx{}, failedNormalTx{}}},
				[]txIndex{},
				[]txStatus{},
				0,
			},
			// skipped bundles are no longer possible, and all **/bundled/**/invalid tests are skipped
		},
		"bundled/OneOf=false/TolerateFailed=true/TolerateInvalid=false": {
			success: SubCaseVariant{
				subBundleTx{flags: bundle.EF_TolerateFailed, txTypes: []txType{failedNormalTx{}, successfulNormalTx{}}},
				[]txIndex{uncheckedTxIndex, uncheckedTxIndex},
				[]txStatus{failedStatus, successStatus},
				1,
			},
			failed: SubCaseVariant{
				subBundleTx{flags: bundle.EF_TolerateFailed, txTypes: []txType{successfulNormalTx{}, invalidNormalTx{}}},
				[]txIndex{},
				[]txStatus{},
				0,
			},
			// skipped bundles are no longer possible, and all **/bundled/**/invalid tests are skipped
		},
		"bundled/OneOf=false/TolerateFailed=true/TolerateInvalid=true": {
			success: SubCaseVariant{
				subBundleTx{flags: bundle.EF_TolerateFailed | bundle.EF_TolerateInvalid, txTypes: []txType{invalidNormalTx{}, failedNormalTx{}, successfulNormalTx{}}},
				[]txIndex{uncheckedTxIndex, uncheckedTxIndex},
				[]txStatus{failedStatus, successStatus},
				1,
			},
			// a bundle can not fail if OneOf is not set and both TolerateFailed and TolerateInvalid are set
			// skipped bundles are no longer possible, and all **/bundled/**/invalid tests are skipped
		},
		"bundled/OneOf=true/TolerateFailed=false/TolerateInvalid=false": {
			success: SubCaseVariant{
				subBundleTx{flags: bundle.EF_OneOf, txTypes: []txType{invalidNormalTx{}, failedNormalTx{}, successfulNormalTx{}}},
				[]txIndex{uncheckedTxIndex, uncheckedTxIndex},
				[]txStatus{failedStatus, successStatus},
				1,
			},
			failed: SubCaseVariant{
				subBundleTx{flags: bundle.EF_OneOf, txTypes: []txType{failedNormalTx{}, invalidNormalTx{}}},
				[]txIndex{},
				[]txStatus{},
				0,
			},
			// skipped bundles are no longer possible, and all **/bundled/**/invalid tests are skipped
		},
		"bundled/OneOf=true/TolerateFailed=false/TolerateInvalid=true": {
			success: SubCaseVariant{
				subBundleTx{flags: bundle.EF_OneOf | bundle.EF_TolerateInvalid, txTypes: []txType{failedNormalTx{}, invalidNormalTx{}}},
				[]txIndex{uncheckedTxIndex},
				[]txStatus{failedStatus},
				0,
			},
			failed: SubCaseVariant{
				subBundleTx{flags: bundle.EF_OneOf | bundle.EF_TolerateInvalid, txTypes: []txType{failedNormalTx{}, failedNormalTx{}}},
				[]txIndex{},
				[]txStatus{},
				0,
			},
			// skipped bundles are no longer possible, and all **/bundled/**/invalid tests are skipped
		},
		"bundled/OneOf=true/TolerateFailed=true/TolerateInvalid=false": {
			success: SubCaseVariant{
				subBundleTx{flags: bundle.EF_OneOf | bundle.EF_TolerateFailed, txTypes: []txType{invalidNormalTx{}, failedNormalTx{}}},
				[]txIndex{uncheckedTxIndex},
				[]txStatus{failedStatus},
				0,
			},
			failed: SubCaseVariant{
				subBundleTx{flags: bundle.EF_OneOf | bundle.EF_TolerateFailed, txTypes: []txType{invalidNormalTx{}, invalidNormalTx{}}},
				[]txIndex{},
				[]txStatus{},
				0,
			},
			// skipped bundles are no longer possible, and all **/bundled/**/invalid tests are skipped
		},
		"bundled/OneOf=true/TolerateFailed=true/TolerateInvalid=true": {
			success: SubCaseVariant{
				subBundleTx{flags: bundle.EF_OneOf | bundle.EF_TolerateFailed | bundle.EF_TolerateInvalid, txTypes: []txType{invalidNormalTx{}, successfulNormalTx{}}},
				[]txIndex{},
				[]txStatus{},
				0,
			},
			failed: SubCaseVariant{
				subBundleTx{flags: bundle.EF_OneOf | bundle.EF_TolerateFailed | bundle.EF_TolerateInvalid, txTypes: []txType{}},
				[]txIndex{},
				[]txStatus{},
				0,
			},
			// skipped bundles are no longer possible, and all **/bundled/**/invalid tests are skipped
		},
	}
}

// Test_RunAllOf_Works tests that if OneOf is not set, all transactions in the bundle are executed, unless they are not tolerated according to the flags.
// If all transactions are tolerated, the bundle should succeed with the effect of all successful transactions applied. If some transactions are not tolerated, the bundle should not have any effect.
// The submitted transactions are a successful transaction, depending on the subcase an successful, failed, or invalid transaction, and another successful transaction.
// The second transaction might be a normal transaction, a sponsored transaction, or a sub-bundle, depending on the subcase.
func Test_RunAllOf_Works(t *testing.T) {
	t.Parallel()

	cases := []NamedCase{}
	for name, subcase := range getSubcases() {
		cases = append(cases, []NamedCase{
			{
				name + "/success",
				Case{false, false, false,
					Merge[txType](successfulNormalTx{}, subcase.success.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](txIndex(0), subcase.success.blockTxIndices, txIndex(2)),
					Merge[txStatus](successStatus, subcase.success.blockTxStatuses, successStatus),
					1 + subcase.success.counter + 1,
				},
			},
			{
				name + "/failed",
				Case{false, false, false,
					Merge[txType](successfulNormalTx{}, subcase.failed.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](),
					Merge[txStatus](),
					0,
				},
			},
			{
				name + "/invalid",
				Case{false, false, false,
					Merge[txType](successfulNormalTx{}, subcase.invalid.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](),
					Merge[txStatus](),
					0,
				},
			},
			// TolerateInvalid
			{
				name + "/success",
				Case{false, false, true,
					Merge[txType](successfulNormalTx{}, subcase.success.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](txIndex(0), subcase.success.blockTxIndices, txIndex(2)),
					Merge[txStatus](successStatus, subcase.success.blockTxStatuses, successStatus),
					1 + subcase.success.counter + 1,
				},
			},
			{
				name + "/failed",
				Case{false, false, true,
					Merge[txType](successfulNormalTx{}, subcase.failed.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](),
					Merge[txStatus](),
					0,
				},
			},
			{
				name + "/invalid",
				Case{false, false, true,
					Merge[txType](successfulNormalTx{}, subcase.invalid.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](txIndex(0), txIndex(2)),
					Merge[txStatus](successStatus, successStatus),
					1 + 1,
				},
			},
			// TolerateFailed
			{
				name + "/success",
				Case{false, true, false,
					Merge[txType](successfulNormalTx{}, subcase.success.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](txIndex(0), subcase.success.blockTxIndices, txIndex(2)),
					Merge[txStatus](successStatus, subcase.success.blockTxStatuses, successStatus),
					1 + subcase.success.counter + 1,
				},
			},
			{
				name + "/failed",
				Case{false, true, false,
					Merge[txType](successfulNormalTx{}, subcase.failed.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](txIndex(0), subcase.failed.blockTxIndices, txIndex(2)),
					Merge[txStatus](successStatus, subcase.failed.blockTxStatuses, successStatus),
					1 + subcase.failed.counter + 1,
				},
			},
			{
				name + "/invalid",
				Case{false, true, false,
					Merge[txType](successfulNormalTx{}, subcase.invalid.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](),
					Merge[txStatus](),
					0,
				},
			},
			// TolerateFailed & TolerateInvalid
			{
				name + "/success",
				Case{false, true, true,
					Merge[txType](successfulNormalTx{}, subcase.success.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](txIndex(0), subcase.success.blockTxIndices, txIndex(2)),
					Merge[txStatus](successStatus, subcase.success.blockTxStatuses, successStatus),
					1 + subcase.success.counter + 1,
				},
			},
			{
				name + "/failed",
				Case{false, true, true,
					Merge[txType](successfulNormalTx{}, subcase.failed.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](txIndex(0), subcase.failed.blockTxIndices, txIndex(2)),
					Merge[txStatus](successStatus, subcase.failed.blockTxStatuses, successStatus),
					1 + subcase.failed.counter + 1,
				},
			},
			{
				name + "/invalid",
				Case{false, true, true,
					Merge[txType](successfulNormalTx{}, subcase.invalid.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](txIndex(0), txIndex(2)),
					Merge[txStatus](successStatus, successStatus),
					1 + 1,
				},
			},
		}...)
	}
	net := startTestnet(t)
	factory := &AccountFactory{session: net}
	sessions := net.SpawnSessions(t, len(cases))
	for i, c := range cases {
		if c.name == "bundled/OneOf=false/TolerateFailed=true/TolerateInvalid=true/failed" ||
			c.name == "layered/OneOf=false/TolerateFailed=true/TolerateInvalid=true/failed" ||
			(strings.HasPrefix(c.name, "layered") && strings.HasSuffix(c.name, "invalid")) ||
			(strings.HasPrefix(c.name, "bundled") && strings.HasSuffix(c.name, "invalid")) {
			continue
		}
		checkCase(t, sessions[i], factory, c)
	}
}

// Test_RunOneOf_Works tests that if OneOf is set, transactions in the bundle are executed until a transaction is tolerated according to the flags.
// If a transaction is tolerated, the bundle should succeed with the effect of all successful transactions up to and including the tolerated transaction applied. If no transaction is tolerated, the bundle should not have any effect.
// The submitted transactions are a successful, failed, or invalid transaction, depending on the subcase, and another two successful transactions.
// The first transaction might be a normal transaction, a sponsored transaction, or a sub-bundle, depending on the subcase.
func Test_RunOneOf_Works(t *testing.T) {
	t.Parallel()

	cases := []NamedCase{}
	for name, subcase := range getSubcases() {
		cases = append(cases, []NamedCase{
			{
				name + "/success",
				Case{true, false, false,
					Merge[txType](subcase.success.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](subcase.success.blockTxIndices),
					Merge[txStatus](subcase.success.blockTxStatuses),
					subcase.success.counter,
				},
			},
			{
				name + "/failed",
				Case{true, false, false,
					Merge[txType](subcase.failed.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](subcase.failed.blockTxIndices, txIndex(1)),
					Merge[txStatus](subcase.failed.blockTxStatuses, successStatus),
					subcase.failed.counter + 1,
				},
			},
			{
				name + "/invalid",
				Case{true, false, false,
					Merge[txType](subcase.invalid.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](txIndex(1)),
					Merge[txStatus](successStatus),
					1,
				},
			},
			// TolerateInvalid
			{
				name + "/success",
				Case{true, false, true,
					Merge[txType](subcase.success.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](subcase.success.blockTxIndices),
					Merge[txStatus](subcase.success.blockTxStatuses),
					subcase.success.counter,
				},
			},
			{
				name + "/failed",
				Case{true, false, true,
					Merge[txType](subcase.failed.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](subcase.failed.blockTxIndices, txIndex(1)),
					Merge[txStatus](subcase.failed.blockTxStatuses, successStatus),
					subcase.failed.counter + 1,
				},
			},
			{
				name + "/invalid",
				Case{true, false, true,
					Merge[txType](subcase.invalid.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](),
					Merge[txStatus](),
					0,
				},
			},
			// TolerateFailed
			{
				name + "/success",
				Case{true, true, false,
					Merge[txType](subcase.success.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](subcase.success.blockTxIndices),
					Merge[txStatus](subcase.success.blockTxStatuses),
					subcase.success.counter,
				},
			},
			{
				name + "/failed",
				Case{true, true, false,
					Merge[txType](subcase.failed.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](subcase.failed.blockTxIndices),
					Merge[txStatus](subcase.failed.blockTxStatuses),
					subcase.failed.counter,
				},
			},
			{
				name + "/invalid",
				Case{true, true, false,
					Merge[txType](subcase.invalid.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](txIndex(1)),
					Merge[txStatus](successStatus),
					1,
				},
			},
			// TolerateFailed & TolerateInvalid
			{
				name + "/success",
				Case{true, true, true,
					Merge[txType](subcase.success.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](subcase.success.blockTxIndices),
					Merge[txStatus](subcase.success.blockTxStatuses),
					subcase.success.counter,
				},
			},
			{
				name + "/failed",
				Case{true, true, true,
					Merge[txType](subcase.failed.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](subcase.failed.blockTxIndices),
					Merge[txStatus](subcase.failed.blockTxStatuses),
					subcase.failed.counter,
				},
			},
			{
				name + "/invalid",
				Case{true, true, true,
					Merge[txType](subcase.invalid.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](),
					Merge[txStatus](),
					0,
				},
			},
		}...)
	}
	net := startTestnet(t)
	factory := &AccountFactory{session: net}
	sessions := net.SpawnSessions(t, len(cases))
	for i, c := range cases {
		if c.name == "bundled/OneOf=false/TolerateFailed=true/TolerateInvalid=true/failed" ||
			c.name == "layered/OneOf=false/TolerateFailed=true/TolerateInvalid=true/failed" ||
			(strings.HasPrefix(c.name, "layered") && strings.HasSuffix(c.name, "invalid")) ||
			(strings.HasPrefix(c.name, "bundled") && strings.HasSuffix(c.name, "invalid")) {
			continue
		}
		checkCase(t, sessions[i], factory, c)
	}
}

func Merge[T any](items ...any) []T {
	var result []T
	if len(items) == 0 {
		return result
	}

	for _, item := range items {
		if item == nil {
			continue
		}
		switch v := item.(type) {
		case T:
			result = append(result, v)
		case []T:
			result = append(result, v...)
		default:
			panic(fmt.Sprintf("unexpected type %T in Merge", v))
		}
	}

	return result
}

func printLayer(layer bundle.BundleLayer, indent string) {
	for _, unit := range layer.Units {
		if tx := unit.AsTransaction(); tx != nil {
			entry := tx.Tx.AccessList()[0]

			bundleOnly := ""
			if entry.Address == bundle.BundleOnly {
				bundleOnly = " (bundle-only)"
			}

			envelope := ""
			if bundle.IsEnvelope(tx.Tx) {
				envelope = " (envelope)"
			}

			fmt.Printf("%s%s%s tx, part of %v\n", indent, bundleOnly, envelope, entry.StorageKeys[0].Hex()[:8])

			if bundle.IsEnvelope(tx.Tx) {
				txBundle, _ := bundle.OpenEnvelope(tx.Tx)
				printLayer(txBundle.Layer, indent+"    ")
			}
		} else if subLayer := unit.AsBundleLayer(); subLayer != nil {
			printLayer(*subLayer, indent+"    ")
		} else {
			panic(fmt.Sprintf("unknown unit type \n"))
		}
	}
}

func checkCase(t *testing.T, session tests.IntegrationTestNetSession, accounts *AccountFactory, namedCase NamedCase) {
	c := namedCase.case_
	name := fmt.Sprintf("OneOf=%v/TolerateFailed=%v/TolerateInvalid=%v/%s", c.oneOf, c.tolerateFailed, c.tolerateInvalid, namedCase.name)
	t.Run(name, func(t *testing.T) {
		t.Parallel()
		flags := bundle.ExecutionFlag(0)
		flags.SetTolerateInvalid(c.tolerateInvalid)
		flags.SetTolerateFailed(c.tolerateFailed)
		flags.SetOneOf(c.oneOf)

		client, err := session.GetClient()
		require.NoError(t, err, "failed to get client; %v", err)
		defer client.Close()

		contractInfo := deployContracts(t, session)

		envelopeTx, plan, layer := buildBundle(t, session, contractInfo, c.submittedTxTypes, flags, false, accounts)
		require.NotNil(t, envelopeTx)

		fmt.Println()
		printLayer(layer, "")
		fmt.Println()

		err = client.SendTransaction(t.Context(), envelopeTx)
		if err != nil {
			// Check whether the bundle was rejected by the pre-check.
			require.ErrorContains(t, err, "permanently blocked")
			// This is only allowed for transactions that should fail.
			require.Zero(t, c.counter)
			require.Empty(t, c.blockTxIndices)
			return
		}

		// Wait for the bundle to be processed.
		info, err := waitForBundleExecution(t.Context(), client.Client(), plan.Hash())
		require.NoError(t, err)
		require.NotNil(t, info.Block)

		fmt.Printf("Bundle got processed in block %d, position %d (%d transactions)\n", *info.Block, *info.Position, *info.Count)

		// Check transactions hashes and statuses
		transactionHashes := getTransactionsInBlock(t, session, big.NewInt(int64(*info.Block)))

		// Consider only transactions corresponding to this bundle.
		require.LessOrEqual(t, int(*info.Position), len(transactionHashes))
		from := *info.Position
		until := from + *info.Count
		transactionHashes = transactionHashes[from:until]

		require.Len(t, transactionHashes, len(c.blockTxIndices))
		for i, txIndex := range c.blockTxIndices {
			if txIndex == uncheckedTxIndex {
				checkStatus(t, session, c.blockTxStatuses[i], transactionHashes[i])
			} else {
				checkStatus(t, session, c.blockTxStatuses[i], transactionHashes[i])
				// checkHashesEqAndStatus(t, session, layer[txIndex].Hash(), c.blockTxStatuses[i], transactionHashes[i])
			}
		}

		// Check the final state is correct
		require.Equal(t, c.counter, getCounterValue(t, client, contractInfo))
	})
}

func startTestnet(t *testing.T) tests.IntegrationTestNetSession {
	updates := opera.GetBrioUpgrades()
	updates.GasSubsidies = true
	updates.TransactionBundles = true
	net := sharedNetwork.GetIntegrationTestNetSession(t, updates)
	return net
}

// --- Contract deployment and helper functions ---

type ContractInfo struct {
	counterAddress  common.Address
	counterGasLimit uint64
	counterInput    []byte

	revertAddress  common.Address
	revertGasLimit uint64
	revertInput    []byte
}

// deployContracts deploys the counter and revert contracts, and returns their addresses, the input data for calling them,
// and the estimated gas limits for calling them with the input data. The gas limit estimation includes an additional entry
// in the access list to account for the bundle-only marker.
// The counter contract is used to check whether the effects of transactions in a bundle are applied as expected,
// and the revert contract is used to create transactions that fail by reverting.
func deployContracts(t *testing.T, net tests.IntegrationTestNetSession) ContractInfo {
	counterAddress, counterInput := counterAddressAndInput(t, net)
	revertAddress, revertInput := revertAddressAndInput(t, net)

	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client; %v", err)
	defer client.Close()

	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err, "failed to suggest gas price; %v", err)

	counterGasLimit, err := client.EstimateGas(t.Context(), ethereum.CallMsg{
		From:     net.GetSessionSponsor().Address(),
		To:       &counterAddress,
		Data:     counterInput,
		GasPrice: gasPrice,
		AccessList: types.AccessList{
			// add one entry to the estimation, to allocate gas for the bundle-only marker
			{Address: bundle.BundleOnly, StorageKeys: []common.Hash{{}}},
		},
	})
	require.NoError(t, err, "failed to estimate gas")

	return ContractInfo{
		counterAddress:  counterAddress,
		counterGasLimit: counterGasLimit,
		counterInput:    counterInput,

		revertAddress:  revertAddress,
		revertGasLimit: counterGasLimit,
		revertInput:    revertInput,
	}
}

func counterAddressAndInput(t *testing.T, net tests.IntegrationTestNetSession) (common.Address, []byte) {
	_, counterAbi, counterAddress := prepareContract(t, net, counter.CounterMetaData.GetAbi, counter.DeployCounter)
	counterInput := generateCallData(t, counterAbi, "incrementCounter")
	return counterAddress, counterInput
}

func revertAddressAndInput(t *testing.T, net tests.IntegrationTestNetSession) (common.Address, []byte) {
	_, revertABI, revertAddress := prepareContract(t, net, revert.RevertMetaData.GetAbi, revert.DeployRevert)
	revertInput := generateCallData(t, revertABI, "doCrash")
	return revertAddress, revertInput
}

func getCounterValue(t *testing.T, client *tests.PooledEhtClient, contractInfo ContractInfo) int64 {
	counterInstance, err := counter.NewCounter(contractInfo.counterAddress, client)
	require.NoError(t, err, "failed to create counter instance; %v", err)
	count, err := counterInstance.GetCount(nil)
	require.NoError(t, err, "failed to get counter value; %v", err)
	return count.Int64()
}

// --- Tx creation ---

type txMakeOptions struct {
	t   *testing.T
	net tests.IntegrationTestNetSession

	contractInfo ContractInfo
	gasPrice     *big.Int
}

type successfulNormalTx struct{}

func (t successfulNormalTx) makeUnit(opts txMakeOptions, a *AccountFactory) bundle.BundleUnit {
	s, err := a.Create()
	require.NoError(opts.t, err)
	sender := s.Into()
	return &bundle.BundleTransaction{
		Tx: types.NewTx(&types.AccessListTx{
			To:       &opts.contractInfo.counterAddress,
			Gas:      opts.contractInfo.counterGasLimit,
			Data:     opts.contractInfo.counterInput,
			GasPrice: opts.gasPrice,
		}),
		Sender: &sender,
	}
}

type failedNormalTx struct{}

func (t failedNormalTx) makeUnit(opts txMakeOptions, a *AccountFactory) bundle.BundleUnit {
	s, err := a.Create()
	require.NoError(opts.t, err)
	sender := s.Into()
	return &bundle.BundleTransaction{
		Tx: types.NewTx(&types.AccessListTx{
			To:       &opts.contractInfo.revertAddress,
			Gas:      opts.contractInfo.revertGasLimit,
			Data:     opts.contractInfo.revertInput,
			GasPrice: opts.gasPrice,
		}),
		Sender: &sender,
	}
}

type invalidNormalTx struct{}

func (t invalidNormalTx) makeUnit(opts txMakeOptions, a *AccountFactory) bundle.BundleUnit {
	s, err := a.Create()
	require.NoError(opts.t, err)
	sender := s.Into()
	return &bundle.BundleTransaction{
		Tx: types.NewTx(&types.AccessListTx{
			To:       &opts.contractInfo.counterAddress,
			Gas:      1, // invalid
			Data:     opts.contractInfo.counterInput,
			GasPrice: opts.gasPrice,
		}),
		Sender: &sender,
	}
}

type successfulSponsoredTx struct{}

func (t successfulSponsoredTx) makeUnit(opts txMakeOptions, a *AccountFactory) bundle.BundleUnit {
	s, err := a.Create()
	require.NoError(opts.t, err)
	sender := s.Into()
	donation := big.NewInt(1e16)
	gas_subsidies.Fund(opts.t, opts.net, sender.Address(), donation)
	return &bundle.BundleTransaction{
		Tx: types.NewTx(&types.AccessListTx{
			To:       &opts.contractInfo.counterAddress,
			Gas:      opts.contractInfo.counterGasLimit,
			Data:     opts.contractInfo.counterInput,
			GasPrice: big.NewInt(0),
		}),
		Sender: &sender,
	}
}

type failedSponsoredTx struct{}

func (t failedSponsoredTx) makeUnit(opts txMakeOptions, a *AccountFactory) bundle.BundleUnit {
	sender := tests.MakeAccountWithBalance(opts.t, opts.net, big.NewInt(1e18))
	donation := big.NewInt(1e16)
	gas_subsidies.Fund(opts.t, opts.net, sender.Address(), donation)
	s := sender.Into()
	return &bundle.BundleTransaction{
		Tx: types.NewTx(&types.AccessListTx{
			To:       &opts.contractInfo.revertAddress,
			Gas:      opts.contractInfo.revertGasLimit,
			Data:     opts.contractInfo.revertInput,
			GasPrice: big.NewInt(0),
		}),
		Sender: &s,
	}
}

type invalidSponsoredTx struct{}

func (t invalidSponsoredTx) makeUnit(opts txMakeOptions, a *AccountFactory) bundle.BundleUnit {
	s, err := a.Create()
	require.NoError(opts.t, err)
	sender := s.Into()
	return &bundle.BundleTransaction{
		Tx: types.NewTx(&types.AccessListTx{
			To:       &opts.contractInfo.counterAddress,
			Gas:      opts.contractInfo.counterGasLimit,
			Data:     opts.contractInfo.counterInput,
			GasPrice: big.NewInt(0),
		}),
		Sender: &sender,
	}
}

type subLayerTx struct {
	txTypes []txType
	flags   bundle.ExecutionFlag
}

func (t subLayerTx) makeUnit(opts txMakeOptions, a *AccountFactory) bundle.BundleUnit {
	units := make([]bundle.BundleUnit, len(t.txTypes))
	for i, txType := range t.txTypes {
		units[i] = txType.makeUnit(opts, a)
	}

	return &bundle.BundleLayer{Units: units, Flags: t.flags}
}

type subBundleTx struct {
	txTypes []txType
	flags   bundle.ExecutionFlag
}

func (t subBundleTx) makeUnit(opts txMakeOptions, a *AccountFactory) bundle.BundleUnit {
	s, err := a.Create()
	require.NoError(opts.t, err)
	sender := s.Into()
	envelopeTx, _, _ := buildBundle(opts.t, opts.net, opts.contractInfo, t.txTypes, t.flags, true, a)
	require.NotNil(opts.t, envelopeTx)

	// remove signature
	envelopeTx = types.NewTx(&types.AccessListTx{
		Nonce:      envelopeTx.Nonce(),
		GasPrice:   envelopeTx.GasPrice(),
		Gas:        envelopeTx.Gas(),
		To:         envelopeTx.To(),
		Value:      envelopeTx.Value(),
		Data:       envelopeTx.Data(),
		AccessList: envelopeTx.AccessList(),
	})

	return &bundle.BundleTransaction{Tx: envelopeTx, Sender: &sender}
}

// --- transaction bundling and signing ---

func makeUnsignedBundleUnits(
	t *testing.T,
	net tests.IntegrationTestNetSession,
	txTypes []txType,
	contractInfo ContractInfo,
	factory *AccountFactory,
) []bundle.BundleUnit {
	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client; %v", err)
	defer client.Close()

	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err, "failed to suggest gas price; %v", err)

	bundleUnits := make([]bundle.BundleUnit, len(txTypes))
	for i, tType := range txTypes {
		bundleUnits[i] = tType.makeUnit(txMakeOptions{t, net, contractInfo, gasPrice}, factory)
	}

	return bundleUnits
}

func signBundleUnits(
	t *testing.T,
	net tests.IntegrationTestNetSession,
	bundleUnits []bundle.BundleUnit,
	plan bundle.ExecutionPlan,
) {
	bundleMarkerWithPlanHash := types.AccessTuple{Address: bundle.BundleOnly, StorageKeys: []common.Hash{plan.Hash()}}
	for i, bundleUnit := range bundleUnits {
		if tx := bundleUnit.AsTransaction(); tx != nil {
			bundleOnlyTx := &types.AccessListTx{
				Nonce:      tx.Tx.Nonce(),
				GasPrice:   tx.Tx.GasPrice(),
				Gas:        tx.Tx.Gas(),
				To:         tx.Tx.To(),
				Value:      tx.Tx.Value(),
				Data:       tx.Tx.Data(),
				AccessList: append(tx.Tx.AccessList(), bundleMarkerWithPlanHash),
			}
			bundleUnits[i] = &bundle.BundleTransaction{Tx: tests.SignTransaction(t, net.GetChainId(), bundleOnlyTx, tests.From(tx.Sender))}
		} else {
			signBundleUnits(t, net, bundleUnit.AsBundleLayer().Units, plan)
		}
	}
}

func buildPlan(
	t *testing.T,
	net tests.IntegrationTestNetSession,
	flags bundle.ExecutionFlag,
	bundleUnits []bundle.BundleUnit,
) bundle.ExecutionPlan {
	signer := types.NewCancunSigner(net.GetChainId())

	steps := make([]bundle.ExecutionUnit, len(bundleUnits))
	for i, bundleUnit := range bundleUnits {
		unit, err := bundleUnit.UnsignedToExecutionUnit(signer)
		require.NoError(t, err, "failed to convert bundle unit to execution unit; %v", err)
		steps[i] = unit
	}

	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client; %v", err)
	defer client.Close()

	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err, "failed to get block number; %v", err)

	plan := bundle.ExecutionPlan{
		Layer: bundle.ExecutionLayer{
			Flags: flags,
			Units: steps,
		},
		Earliest: blockNumber,
		Latest:   blockNumber + 100,
	}

	return plan
}

func buildBundle(
	t *testing.T,
	net tests.IntegrationTestNetSession,
	contractInfo ContractInfo,
	txTypes []txType,
	flags bundle.ExecutionFlag,
	nested bool,
	accountFactory *AccountFactory,
) (*types.Transaction, bundle.ExecutionPlan, bundle.BundleLayer) {
	bundleUnits := makeUnsignedBundleUnits(t, net, txTypes, contractInfo, accountFactory)

	plan := buildPlan(t, net, flags, bundleUnits)

	signBundleUnits(t, net, bundleUnits, plan)

	bundleLayer := bundle.BundleLayer{
		Units: bundleUnits,
		Flags: flags,
	}

	envelopeTx := makeEnvelopeTransaction(t, net, bundleLayer, plan, nested)

	return envelopeTx, plan, bundleLayer
}

func checkHashesEqAndStatus(
	t *testing.T,
	net tests.IntegrationTestNetSession,
	expectedHash common.Hash,
	expectedStatus txStatus,
	txHash common.Hash,
) {
	t.Helper()
	require.Equal(t, expectedHash, txHash)
	checkStatus(t, net, expectedStatus, txHash)
}

func checkStatus(
	t *testing.T,
	net tests.IntegrationTestNetSession,
	status txStatus,
	txHash common.Hash,
) {
	t.Helper()
	receipt, err := net.GetReceipt(txHash)
	require.NoError(t, err, "failed to get transaction receipt; %v", err)
	require.Equal(t, status, txStatus(receipt.Status))
}

type AccountFactory struct {
	session  tests.IntegrationTestNetSession
	accounts []*tests.Account
	mutex    sync.Mutex
}

func (f *AccountFactory) Create() (*tests.Account, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if len(f.accounts) == 0 {
		const batchSize = 100
		accounts := tests.NewAccounts(batchSize)
		addresses := make([]common.Address, len(accounts))
		for i, cur := range accounts {
			addresses[i] = cur.Address()
		}

		receipts, err := f.session.EndowAccounts(addresses, big.NewInt(1e16))
		if err != nil {
			return nil, err
		}

		for _, receipt := range receipts {
			if receipt.Status != types.ReceiptStatusSuccessful {
				return nil, fmt.Errorf("failed to endow account")
			}
		}

		f.accounts = accounts
	}
	res := f.accounts[0]
	f.accounts = f.accounts[1:]
	return res, nil
}

func (f *AccountFactory) CreateMultiple(num int) ([]*tests.Account, error) {
	res := make([]*tests.Account, num)
	for i := range res {
		next, err := f.Create()
		if err != nil {
			return nil, err
		}
		res[i] = next
	}
	return res, nil
}
