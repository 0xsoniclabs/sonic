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

package evmcore

import (
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type BundleState int

const (
	BundleStateRunnable BundleState = iota
	BundleStateTemporaryBlocked
	BundleStatePermanentlyBlocked
)

// GetBundleState determines the state of a transaction bundle by try-running it
// the given state. If it can be successfully executed, it is considered
// runnable. If it fails with a temporary error (e.g., nonce too high), it is
// considered temporarily blocked. If it fails with a permanent error (e.g.,
// nonce too low or invalid signature), it is considered permanently blocked.
func GetBundleState(
	bundle bundle.TransactionBundle,
	state state.StateDB,
	signer types.Signer,
) BundleState {

	// TODO: use the actual evmcore processor for the determining the bundle state,
	// including the support for nested bundles and sponsorships.

	// For now, we simply check the nonces and assume no nested transactions or
	// sponsorships to provide a fake implementation.

	if bundle.Flags&0x3 == 0 { // AllOf mode -- all need to be valid in sequence

		nonces := map[common.Address]uint64{}
		for _, tx := range bundle.Bundle {
			from, nonce, err := getNonceAndSender(tx, signer)
			if err != nil {
				return BundleStatePermanentlyBlocked
			}
			if expectedNonce, ok := nonces[from]; ok {
				if nonce != expectedNonce {
					return BundleStatePermanentlyBlocked
				}
				nonces[from] = expectedNonce + 1
			} else {
				accountNonce := state.GetNonce(from)
				if nonce < accountNonce {
					return BundleStatePermanentlyBlocked
				}
				if nonce > accountNonce {
					return BundleStateTemporaryBlocked
				}
				nonces[from] = nonce + 1
			}
		}
		return BundleStateRunnable

	}

	// AnyOf mode -- one needs to be valid
	for _, tx := range bundle.Bundle {
		from, nonce, err := getNonceAndSender(tx, signer)

		// If the nonce is correct, the transaction is runnable now.
		if err == nil && nonce == state.GetNonce(from) {
			return BundleStateRunnable
		}
		// If the nonce is too high, it may be runnable in the future.
		if err != nil || nonce > state.GetNonce(from) {
			return BundleStateTemporaryBlocked
		}
	}
	return BundleStatePermanentlyBlocked
}

func getNonceAndSender(tx *types.Transaction, signer types.Signer) (common.Address, uint64, error) {
	sender, err := signer.Sender(tx)
	return sender, tx.Nonce(), err
}
