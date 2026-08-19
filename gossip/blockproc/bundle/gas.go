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
	"fmt"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils/checked"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// envelopeSender is the nominal sender used when computing the intrinsic and
// floor data gas of an envelope. Since EIP-2780 the sender influences those
// costs in a single way: a transaction sending value to itself pays neither for
// touching the recipient nor for the transfer. An envelope can never be such a
// self-transfer, since it is addressed to BundleProcessor, a system address no
// key can sign for. Any address other than BundleProcessor thus yields the
// correct result, independent of who actually signed the envelope.
var envelopeSender = common.Address{}

// CalculateEnvelopeGas calculates the gas limit for an envelope transaction
// carrying the given bundle. The envelope must declare enough gas to cover its
// own costs as a transaction - the intrinsic gas of its payload, access list
// and authorization list, and the floor data gas of EIP-7623 - as well as the
// combined gas limits of the transactions it delivers.
//
// The envelope's own costs are computed by go-ethereum, so that this function
// agrees with the intrinsic and floor data gas checks a validating node applies
// to the envelope, across all gas schedule revisions.
func CalculateEnvelopeGas(
	bundle TransactionBundle,
	payload []byte,
	accessList types.AccessList,
	authList []types.SetCodeAuthorization,
	value *uint256.Int,
	upgrades opera.Upgrades,
) (uint64, error) {
	rules := envelopeGasRules(upgrades)

	intrinsic, err := core.IntrinsicGas(
		payload, accessList, authList,
		envelopeSender, &BundleProcessor, value, rules,
	)
	if err != nil {
		return 0, fmt.Errorf("failed intrinsic gas calculation: %w", err)
	}

	floorDataGas, err := core.FloorDataGas(
		rules, envelopeSender, &BundleProcessor, value, payload, accessList,
	)
	if err != nil {
		return 0, fmt.Errorf("failed floor data gas calculation: %w", err)
	}

	txGas, err := calculateTxGasSum(bundle.GetTransactionsInReferencedOrder())
	if err != nil {
		return 0, fmt.Errorf("failed transaction gas sum calculation: %w", err)
	}

	return max(intrinsic, floorDataGas, txGas), nil
}

// envelopeGasRules reduces the given network upgrades to the Ethereum revision
// flags consulted by go-ethereum when pricing an envelope transaction.
// Transaction bundles require Brio, which succeeds both Istanbul and Shanghai,
// so those revisions are always active. Canto activates the Amsterdam gas
// schedule, which reprices the transaction base cost (EIP-2780), calldata
// (EIP-7976) and access list entries (EIP-7981).
func envelopeGasRules(upgrades opera.Upgrades) params.Rules {
	return params.Rules{
		IsHomestead: true,
		IsIstanbul:  true,
		IsShanghai:  true,
		IsAmsterdam: upgrades.Canto,
	}
}

// bundleOnlyMarkerGas returns the extra intrinsic gas a transaction requires
// once the bundle-only marker is appended to its access list. The marker adds
// one address and one storage key.
//
// The values below mirror the access list charges of go-ethereum's
// core.IntrinsicGas; Test_bundleOnlyMarkerGas_MatchesGethAccessListCharge keeps
// them in sync.
func bundleOnlyMarkerGas(upgrades opera.Upgrades) uint64 {
	if !upgrades.Canto {
		return params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas
	}
	// EIP-7981 charges the access list entry's data on top of the base cost.
	const dataCost = (common.AddressLength + common.HashLength) *
		params.TxCostFloorPerToken7976 * params.TxTokenPerNonZeroByte
	return params.TxAccessListAddressGasAmsterdam +
		params.TxAccessListStorageKeyGasAmsterdam +
		dataCost
}

// calculateTxGasSum sums up the gas limits of the given transactions. An
// error is returned if an overflow occurred.
func calculateTxGasSum(transactions []*types.Transaction) (uint64, error) {
	sum := checked.Uint64(0)
	for _, tx := range transactions {
		if tx != nil {
			sum = checked.Add(sum, tx.Gas())
		}
	}
	return sum.Unwrap()
}
