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

package bundle

import (
	"bytes"
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils/checked"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// gasRevisions lists the gas schedules an envelope can be priced with. Canto
// activates the Amsterdam schedule, everything before it shares the pre-
// Amsterdam one.
var gasRevisions = map[string]opera.Upgrades{
	"pre-canto": {},
	"canto":     {Canto: true},
}

const (
	// amsterdamBaseGas is the EIP-2780 intrinsic base cost of an envelope
	// without value: the transaction base cost plus the cold touch of the
	// BundleProcessor. An envelope is never a contract creation, so it is
	// charged ColdAccountAccessAmsterdam and not CreateAccessAmsterdam.
	amsterdamBaseGas = params.TxBaseCost2780 + params.ColdAccountAccessAmsterdam

	// amsterdamValueGas is what EIP-2780 adds for a non-zero envelope value.
	amsterdamValueGas = params.TransferLogCost2780 + params.TxValueCost2780

	// EIP-7981 charges the data of every access list entry on top of the base
	// per-entry cost.
	accessListAddressDataGas = common.AddressLength *
		params.TxCostFloorPerToken7976 * params.TxTokenPerNonZeroByte
	accessListStorageKeyDataGas = common.HashLength *
		params.TxCostFloorPerToken7976 * params.TxTokenPerNonZeroByte
)

func Test_CalculateEnvelopeGas_ChargesRevisionSpecificBaseCost(t *testing.T) {
	tests := map[string]struct {
		upgrades opera.Upgrades
		value    *uint256.Int
		want     uint64
	}{
		// Before Amsterdam the base cost is flat and ignores the value.
		"pre-canto without value": {
			upgrades: opera.Upgrades{},
			want:     params.TxGas,
		},
		"pre-canto with value": {
			upgrades: opera.Upgrades{},
			value:    uint256.NewInt(1),
			want:     params.TxGas,
		},
		// Since EIP-2780 the base cost is decomposed per resource, and paying
		// value adds the transfer log and the recipient's balance write.
		"canto without value": {
			upgrades: opera.Upgrades{Canto: true},
			want:     amsterdamBaseGas,
		},
		"canto with zero value": {
			upgrades: opera.Upgrades{Canto: true},
			value:    uint256.NewInt(0),
			want:     amsterdamBaseGas,
		},
		"canto with value": {
			upgrades: opera.Upgrades{Canto: true},
			value:    uint256.NewInt(1),
			want:     amsterdamBaseGas + amsterdamValueGas,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			have, err := CalculateEnvelopeGas(
				TransactionBundle{}, nil, nil, nil, tc.value, tc.upgrades,
			)
			require.NoError(t, err)
			require.Equal(t, tc.want, have)
		})
	}
}

func Test_CalculateEnvelopeGas_PricesEnvelopeContentPerRevision(t *testing.T) {
	payload := append(
		bytes.Repeat([]byte{0}, 500),
		bytes.Repeat([]byte{0xFF}, 700)...,
	)

	tests := map[string]struct {
		payload      []byte
		accessList   types.AccessList
		authList     []types.SetCodeAuthorization
		wantPreCanto uint64
		wantCanto    uint64
	}{
		"empty": {
			wantPreCanto: params.TxGas,
			wantCanto:    amsterdamBaseGas,
		},
		// For both revisions the floor data gas dominates a payload this size.
		"payload": {
			payload: payload,
			wantPreCanto: params.TxGas +
				(700*params.TxTokenPerNonZeroByte+500)*params.TxCostFloorPerToken,
			// EIP-7976 stops distinguishing zero from non-zero bytes.
			wantCanto: amsterdamBaseGas +
				1200*params.TxTokenPerNonZeroByte*params.TxCostFloorPerToken7976,
		},
		"access list addresses": {
			accessList:   make(types.AccessList, 1000),
			wantPreCanto: params.TxGas + 1000*params.TxAccessListAddressGas,
			wantCanto: amsterdamBaseGas + 1000*
				(params.TxAccessListAddressGasAmsterdam+accessListAddressDataGas),
		},
		"access list storage keys": {
			accessList: types.AccessList{{
				StorageKeys: make([]common.Hash, 1000),
			}},
			wantPreCanto: params.TxGas +
				params.TxAccessListAddressGas +
				1000*params.TxAccessListStorageKeyGas,
			wantCanto: amsterdamBaseGas +
				params.TxAccessListAddressGasAmsterdam + accessListAddressDataGas +
				1000*(params.TxAccessListStorageKeyGasAmsterdam+accessListStorageKeyDataGas),
		},
		"set code authorizations": {
			authList:     make([]types.SetCodeAuthorization, 1000),
			wantPreCanto: params.TxGas + 1000*params.CallNewAccountGas,
			wantCanto:    amsterdamBaseGas + 1000*params.RegularPerAuthBaseCost,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			for revision, upgrades := range gasRevisions {
				t.Run(revision, func(t *testing.T) {
					want := tc.wantPreCanto
					if upgrades.Canto {
						want = tc.wantCanto
					}
					have, err := CalculateEnvelopeGas(
						TransactionBundle{}, tc.payload, tc.accessList,
						tc.authList, nil, upgrades,
					)
					require.NoError(t, err)
					require.Equal(t, want, have)
				})
			}
		})
	}
}

func Test_CalculateEnvelopeGas_UsesMaximumOfIntrinsicFloorAndTransactionGas(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	// A payload of only non-zero bytes is more expensive in floor costs than in
	// intrinsic costs, while access list addresses only add to intrinsic costs.
	tests := map[string]struct {
		payload    []byte
		accessList types.AccessList
		txGas      []uint64
		dominant   string
	}{
		"intrinsic gas": {
			accessList: make(types.AccessList, 100),
			dominant:   "intrinsic",
		},
		"floor data gas": {
			payload:  bytes.Repeat([]byte{0xFF}, 100),
			dominant: "floor",
		},
		"transaction gas": {
			txGas:    []uint64{1_000_000},
			dominant: "transactions",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			for revision, upgrades := range gasRevisions {
				t.Run(revision, func(t *testing.T) {
					require := require.New(t)

					txBundle := TransactionBundle{}
					if len(tc.txGas) > 0 {
						steps := make([]BuilderStep, 0, len(tc.txGas))
						for _, gas := range tc.txGas {
							steps = append(steps, Step(key, types.AccessListTx{Gas: gas}))
						}
						txBundle = NewBuilder().
							WithUpgrades(upgrades).
							AllOf(steps...).
							BuildBundle()
					}

					rules := envelopeGasRules(upgrades)
					intrinsic, err := core.IntrinsicGas(
						tc.payload, tc.accessList, nil,
						envelopeSender, &BundleProcessor, nil, rules,
					)
					require.NoError(err)
					floor, err := core.FloorDataGas(
						rules, envelopeSender, &BundleProcessor, nil,
						tc.payload, tc.accessList,
					)
					require.NoError(err)
					txGasSum, err := calculateTxGasSum(
						txBundle.GetTransactionsInReferencedOrder(),
					)
					require.NoError(err)

					switch tc.dominant {
					case "intrinsic":
						require.Greater(intrinsic, floor)
						require.Greater(intrinsic, txGasSum)
					case "floor":
						require.Greater(floor, intrinsic)
						require.Greater(floor, txGasSum)
					case "transactions":
						require.Greater(txGasSum, intrinsic)
						require.Greater(txGasSum, floor)
					default:
						require.FailNow("unsupported test case spec")
					}

					have, err := CalculateEnvelopeGas(
						txBundle, tc.payload, tc.accessList, nil, nil, upgrades,
					)
					require.NoError(err)
					require.Equal(max(intrinsic, floor, txGasSum), have)
				})
			}
		})
	}
}

func Test_CalculateEnvelopeGas_CoversTheGasLimitsOfTheBundledTransactions(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	for revision, upgrades := range gasRevisions {
		t.Run(revision, func(t *testing.T) {
			txBundle := NewBuilder().
				WithUpgrades(upgrades).
				AllOf(
					Step(key, types.AccessListTx{Gas: 100_000}),
					Step(key, types.AccessListTx{Gas: 250_000}),
				).
				BuildBundle()

			// The builder raises each transaction's gas limit to account for
			// the bundle-only marker it adds to their access lists.
			want := uint64(100_000+250_000) + 2*bundleOnlyMarkerGas(upgrades)

			have, err := CalculateEnvelopeGas(
				txBundle, nil, nil, nil, nil, upgrades,
			)
			require.NoError(t, err)
			require.Equal(t, want, have)
		})
	}
}

func Test_CalculateEnvelopeGas_SatisfiesTheChecksAppliedToASignedEnvelope(t *testing.T) {
	// The gas limit computed for an envelope must be enough to pass the
	// intrinsic and floor data gas checks a validating node applies to it. Those
	// checks use the envelope's real sender, whereas the computation here uses a
	// nominal one, so this pins that the substitution is sound.
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := types.LatestSignerForChainID(big.NewInt(1))

	for revision, upgrades := range gasRevisions {
		t.Run(revision, func(t *testing.T) {
			require := require.New(t)

			envelope := NewBuilder().
				WithSigner(signer).
				WithUpgrades(upgrades).
				AllOf(Step(key, types.AccessListTx{Gas: 21_000})).
				Build()

			sender, err := types.Sender(signer, envelope)
			require.NoError(err)
			require.NotEqual(BundleProcessor, sender)

			value, overflow := uint256.FromBig(envelope.Value())
			require.False(overflow)

			rules := envelopeGasRules(upgrades)
			intrinsic, err := core.IntrinsicGas(
				envelope.Data(), envelope.AccessList(),
				envelope.SetCodeAuthorizations(),
				sender, envelope.To(), value, rules,
			)
			require.NoError(err)
			require.GreaterOrEqual(envelope.Gas(), intrinsic)

			floor, err := core.FloorDataGas(
				rules, sender, envelope.To(), value,
				envelope.Data(), envelope.AccessList(),
			)
			require.NoError(err)
			require.GreaterOrEqual(envelope.Gas(), floor)
		})
	}
}

func Test_CalculateEnvelopeGas_DetectsTxGasSumOverflow(t *testing.T) {
	first := TxReference{From: common.Address{1}}
	second := TxReference{From: common.Address{2}}
	txBundle := TransactionBundle{
		Transactions: map[TxReference]*types.Transaction{
			first:  types.NewTx(&types.LegacyTx{Gas: math.MaxUint64 - 1000}),
			second: types.NewTx(&types.LegacyTx{Gas: 2000}),
		},
		Plan: ExecutionPlan{
			Root: NewAllOfStep(NewTxStep(first), NewTxStep(second)),
		},
	}

	for revision, upgrades := range gasRevisions {
		t.Run(revision, func(t *testing.T) {
			_, err := CalculateEnvelopeGas(
				txBundle, nil, nil, nil, nil, upgrades,
			)
			require.ErrorContains(t, err, "transaction gas sum")
			require.ErrorIs(t, err, checked.ErrOverflow)
		})
	}
}

func Test_envelopeGasRules_ActivatesAmsterdamWithCanto(t *testing.T) {
	for revision, upgrades := range gasRevisions {
		t.Run(revision, func(t *testing.T) {
			rules := envelopeGasRules(upgrades)
			require.Equal(t, upgrades.Canto, rules.IsAmsterdam)

			// Bundles require Brio, which succeeds both revisions.
			require.True(t, rules.IsIstanbul)
			require.True(t, rules.IsShanghai)
		})
	}
}

func Test_bundleOnlyMarkerGas_MatchesGethAccessListCharge(t *testing.T) {
	// The marker adds one address and one storage key to a transaction's access
	// list. Its cost is tracked as constants, so compare them against what
	// go-ethereum charges for such an entry.
	marker := types.AccessList{{
		Address:     BundleOnly,
		StorageKeys: []common.Hash{{}},
	}}

	for revision, upgrades := range gasRevisions {
		t.Run(revision, func(t *testing.T) {
			require := require.New(t)
			rules := envelopeGasRules(upgrades)

			withMarker, err := core.IntrinsicGas(
				nil, marker, nil,
				envelopeSender, &BundleProcessor, nil, rules,
			)
			require.NoError(err)
			withoutMarker, err := core.IntrinsicGas(
				nil, nil, nil,
				envelopeSender, &BundleProcessor, nil, rules,
			)
			require.NoError(err)

			require.Equal(withMarker-withoutMarker, bundleOnlyMarkerGas(upgrades))
		})
	}
}

func Test_calculateTxGasSum_sumsGasLimits(t *testing.T) {
	tests := map[string][]uint64{
		"no transactions":       {},
		"one transaction":       {21000},
		"multiple transactions": {21000, 50000, 100000},
	}

	for name, gasLimits := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			var transactions []*types.Transaction
			var expectedSum uint64
			for _, gasLimit := range gasLimits {
				tx := types.NewTx(&types.LegacyTx{
					Gas: gasLimit,
				})
				transactions = append(transactions, tx)
				expectedSum += gasLimit
			}

			sum, err := calculateTxGasSum(transactions)
			require.NoError(err)
			require.Equal(expectedSum, sum)
		})
	}
}

func Test_calculateTxGasSum_detectsOverflows(t *testing.T) {
	tests := map[string][]uint64{
		"overflow from two transactions": {
			math.MaxUint64 - 1000, 2000,
		},
		"overflow from multiple transactions": {
			math.MaxUint64 - 1000, 500, 300, 200, 1,
		},
	}

	for name, gasLimits := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			var transactions []*types.Transaction
			for _, gasLimit := range gasLimits {
				tx := types.NewTx(&types.LegacyTx{
					Gas: gasLimit,
				})
				transactions = append(transactions, tx)
			}

			_, err := calculateTxGasSum(transactions)
			require.ErrorIs(err, checked.ErrOverflow)
		})
	}
}

func Test_calculateTxGasSum_ignoresNilTransactions(t *testing.T) {
	tests := map[string][]*types.Transaction{
		"all nil transactions": {
			nil, nil, nil,
		},
		"some nil transactions": {
			types.NewTx(&types.LegacyTx{Gas: 21000}), nil,
			types.NewTx(&types.LegacyTx{Gas: 50000}), nil,
		},
	}

	for name, transactions := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			expectedSum := uint64(0)
			for _, tx := range transactions {
				if tx != nil {
					expectedSum += tx.Gas()
				}
			}

			sum, err := calculateTxGasSum(transactions)
			require.NoError(err)
			require.Equal(expectedSum, sum)
		})
	}
}
