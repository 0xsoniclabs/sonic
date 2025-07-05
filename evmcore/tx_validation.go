// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package evmcore

import (
	"fmt"
	"math/big"

	"github.com/0xsoniclabs/sonic/gossip/gasprice/gaspricelimits"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// poolOptions is a set of options to adjust the validation of transactions
// according to the current state of the transaction pool.
type poolOptions struct {
	currentState TxPoolStateDB // Current state in the blockchain head
	minTip       *big.Int      // Minimum gas tip to enforce for acceptance into the pool
	locals       *accountSet   // Set of local transaction to exempt from eviction rules
	isLocal      bool          // Whether the transaction came from a local source
}

// NetworkRules is a set of network rules to validate transactions
// according to the current state of the blockchain.
type NetworkRules struct {
	istanbul bool // Fork indicator whether we are in the istanbul revision.
	shanghai bool // Fork indicator whether we are in the shanghai revision.

	eip2718 bool // Fork indicator whether we are using EIP-2718 type transactions.
	eip1559 bool // Fork indicator whether we are using EIP-1559 type transactions.
	eip4844 bool // Fork indicator whether we are using EIP-4844 type transactions.
	eip7623 bool // Fork indicator whether we are using EIP-7623 floor gas validation.
	eip7702 bool // Fork indicator whether we are using EIP-7702 set code transactions.

	signer types.Signer // Signer to use for transaction validation
}

// blockState is a set of options to adjust the validation of transactions
// according to the current state of the blockchain, specifically the max gas
// and base fee of the current block.
type blockState struct {
	maxGas  uint64   // Current gas limit for transaction caps
	baseFee *big.Int // Current base fee for transaction
}

// validateTx checks whether a transaction is valid according to the current
// options and adheres to some heuristic limits of the local node (price and size).
func validateTx(
	tx *types.Transaction,
	opt poolOptions,
	currentBlockState blockState,
	netRules NetworkRules) error {

	// The transaction is checked against network rules to detect unsupported
	// transaction types first, ensuring protocol compliance before performing
	// further validation.
	if err := ValidateTxForNetwork(tx, netRules); err != nil {
		return err
	}

	if err := ValidateTxStatic(tx); err != nil {
		return err
	}

	if err := ValidateTxForBlock(tx, currentBlockState); err != nil {
		return err
	}

	if err := validateTxForPool(tx, opt, netRules.signer); err != nil {
		return err
	}

	if err := ValidateTxForState(tx, opt.currentState, netRules.signer); err != nil {
		return err
	}

	return nil
}

// ValidateTxForNetwork runs a set of verifications against the given
// network rules. It checks that:
// - the transaction's type is supported in the current network
// - if the network supports shanghai, it checks the init code size
// - the transaction's gas is enough to cover for the intrinsic gas
// - the transaction's gas is enough to cover for the floor data gas
// - the transaction has been signed with a valid signer for the current chain
//
// It returns an error if any of the checks fail.
func ValidateTxForNetwork(tx *types.Transaction, opt NetworkRules) error {
	// With the introduction of EIP-2718 transaction envelope, the transaction
	// can be of different types. Before the Berlin fork
	// (https://blog.ethereum.org/2021/03/08/ethereum-berlin-upgrade-announcement),
	// the only accepted type of transaction is legacy.
	if !opt.eip2718 && tx.Type() != types.LegacyTxType {
		return ErrTxTypeNotSupported
	}
	// NOTE: between eip-2718 and eip-1559, access list transactions are allowed as well.

	// Reject dynamic fee transactions until EIP-1559 activates.
	if !opt.eip1559 && tx.Type() == types.DynamicFeeTxType {
		return ErrTxTypeNotSupported
	}
	// Reject blob transactions until EIP-4844 activates or if is already EIP-4844 and they are not empty
	if tx.Type() == types.BlobTxType {
		if !opt.eip4844 {
			return ErrTxTypeNotSupported
		}
		// Sonic only supports Blob transactions without blob data.
		if len(tx.BlobHashes()) > 0 ||
			(tx.BlobTxSidecar() != nil && len(tx.BlobTxSidecar().BlobHashes()) > 0) {
			return ErrNonEmptyBlobTx
		}
	}

	// validate EIP-7702 transactions, part of prague revision
	if tx.Type() == types.SetCodeTxType && !opt.eip7702 {
		return ErrTxTypeNotSupported
	}

	// This check does not validate gas, but depends on active revision.
	// Check whether the init code size has been exceeded, introduced in EIP-3860
	if opt.shanghai && tx.To() == nil && len(tx.Data()) > params.MaxInitCodeSize {
		return fmt.Errorf("%w: code size %v, limit %v", ErrMaxInitCodeSizeExceeded, len(tx.Data()), params.MaxInitCodeSize)
	}

	// Ensure the transaction has more gas than the basic tx fee.
	intrGas, err := core.IntrinsicGas(
		tx.Data(),
		tx.AccessList(),
		tx.SetCodeAuthorizations(),
		tx.To() == nil, // is contract creation
		true,           // is homestead
		opt.istanbul,   // is eip-2028 (transactional data gas cost reduction)
		opt.shanghai,   // is eip-3860 (limit and meter init-code )
	)
	if err != nil {
		return err
	}
	if tx.Gas() < intrGas {
		return ErrIntrinsicGas
	}

	// EIP-7623 part of Prague revision: Floor data gas
	// see: https://eips.ethereum.org/EIPS/eip-7623
	if opt.eip7623 {
		floorDataGas, err := core.FloorDataGas(tx.Data())
		if err != nil {
			return err
		}
		if tx.Gas() < floorDataGas {
			return fmt.Errorf("%w: have %d, want %d", ErrFloorDataGas, tx.Gas(), floorDataGas)
		}
	}

	if _, err := types.Sender(opt.signer, tx); err != nil {
		return ErrInvalidSender
	}

	return nil
}

// ValidateTxStatic runs a set of verification independent from any context with
// the aim to identify malformed transactions. It checks the transaction's:
// - value (must be positive)
// - gas fee cap and tip cap must be within the 256 bit range
// - gas fee cap must be greater than or equal to the tip cap
// - set code transactions must not have an empty authorization list
//
// It returns an error if any of the checks fail.
func ValidateTxStatic(tx *types.Transaction) error {

	// Transactions can't be negative. This may never happen using RLP decoded
	// transactions but may occur if you create a transaction using the RPC.
	if tx.Value().Sign() < 0 {
		return ErrNegativeValue
	}

	// Sanity check for extremely large numbers
	if tx.GasFeeCap().BitLen() > 256 {
		return ErrFeeCapVeryHigh
	}
	if tx.GasTipCap().BitLen() > 256 {
		return ErrTipVeryHigh
	}

	// Ensure gasFeeCap is greater than or equal to gasTipCap.
	if tx.GasFeeCapIntCmp(tx.GasTipCap()) < 0 {
		return ErrTipAboveFeeCap
	}

	// Check non-empty authorization list
	if tx.Type() == types.SetCodeTxType && len(tx.SetCodeAuthorizations()) == 0 {
		return ErrEmptyAuthorizations
	}

	return nil
}

// ValidateTxForBlock checks if a transaction is valid based on the max gas limit
// and base fee of the current block.
// An error is returned if the transaction's gas exceeds the block's gas limit
// or if the transaction's gas fee cap is below the minimum required base fee.
func ValidateTxForBlock(tx *types.Transaction, blockState blockState) error {

	// Ensure Sonic-specific hard bounds
	if baseFee := blockState.baseFee; baseFee != nil {
		limit := gaspricelimits.GetMinimumFeeCapForTransactionPool(baseFee)
		if tx.GasFeeCapIntCmp(limit) < 0 {
			log.Trace("Rejecting underpriced tx: minimumBaseFee", "minimumBaseFee", baseFee, "limit", limit, "tx.GasFeeCap", tx.GasFeeCap())
			return ErrUnderpriced
		}
	}

	// Ensure the transaction doesn't exceed the current block limit gas.
	if blockState.maxGas < tx.Gas() {
		return ErrGasLimit
	}

	return nil
}

// ValidateTxForState checks if a transaction is valid based on the sender's current state.
// Specifically, it ensures the sender has sufficient balance to cover the transaction cost,
// and that the transaction's nonce is not lower than the sender's nonce in the state.
// Returns an error if any of these conditions are not met.
func ValidateTxForState(tx *types.Transaction, state TxPoolStateDB,
	signer types.Signer) error {

	// Make sure the transaction is signed properly.
	from, err := types.Sender(signer, tx)
	if err != nil {
		return ErrInvalidSender
	}

	// Ensure the transaction adheres to nonce ordering
	if state.GetNonce(from) > tx.Nonce() {
		return ErrNonceTooLow
	}

	// Transactor should have enough funds to cover the costs
	// cost == Value + GasPrice * Gas
	if utils.Uint256ToBigInt(state.GetBalance(from)).Cmp(tx.Cost()) < 0 {
		return ErrInsufficientFunds
	}
	return nil
}

// validateTxForPool checks whether a transaction is valid according to the
// current minimum tip for the pool. It returns an error if the transaction's
// tip is lower than the minimum tip.
func validateTxForPool(tx *types.Transaction, opt poolOptions,
	signer types.Signer) error {

	// Reject transactions over defined size to prevent DoS attacks
	if uint64(tx.Size()) > txMaxSize {
		return ErrOversizedData
	}

	// Make sure the transaction is signed properly.
	from, err := types.Sender(signer, tx)
	if err != nil {
		return ErrInvalidSender
	}

	// A transaction is local if received from the local RPC or if the sender belongs to the local accounts set.
	// For local transactions, minimum gas tips are not enforced.
	local := opt.isLocal || opt.locals.contains(from)
	if local {
		return nil
	}

	// Drop non-local transactions under our own minimal accepted gas price or tip.
	if tx.GasTipCapIntCmp(opt.minTip) < 0 {
		log.Trace("Rejecting underpriced tx: pool.minTip", "pool.minTip",
			opt.minTip, "tx.GasTipCap", tx.GasTipCap())
		return ErrUnderpriced
	}
	return nil
}
