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
	"errors"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	// ErrNoGenesis is returned when there is no Genesis Block.
	ErrNoGenesis = errors.New("genesis not found in chain")
)

// List of evm-call-message pre-checking errors. All state transition messages will
// be pre-checked before execution. If any invalidation detected, the corresponding
// error should be returned which is defined here.
//
// - If the pre-checking happens in the miner, then the transaction won't be packed.
// - If the pre-checking happens in the block processing procedure, then a "BAD BLOCK"
// error should be emitted.
var (
	// ErrNonceTooLow is returned if the nonce of a transaction is lower than the
	// one present in the local chain.
	ErrNonceTooLow = errors.New("nonce too low")

	// ErrNonceTooHigh is returned if the nonce of a transaction is higher than the
	// next one expected based on the local chain.
	ErrNonceTooHigh = errors.New("nonce too high")

	// ErrGasLimitReached is returned by the gas pool if the amount of gas required
	// by a transaction is higher than what's left in the block.
	ErrGasLimitReached = errors.New("gas limit reached")

	// ErrInsufficientFundsForTransfer is returned if the transaction sender doesn't
	// have enough funds for transfer(topmost call only).
	ErrInsufficientFundsForTransfer = errors.New("insufficient funds for transfer")

	// ErrInsufficientFunds is returned if the total cost of executing a transaction
	// is higher than the balance of the user's account.
	ErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")

	// ErrIntrinsicGas is returned if the transaction is specified to use less gas
	// than required to start the invocation.
	ErrIntrinsicGas = core.ErrIntrinsicGas

	// ErrFloorDataGas is returned if the transaction is specified to use less gas
	// than required for the data floor cost.
	ErrFloorDataGas = errors.New("insufficient gas for floor data gas cost")

	// ErrTxTypeNotSupported is returned if a transaction is not supported in the
	// current network configuration.
	ErrTxTypeNotSupported = types.ErrTxTypeNotSupported

	// ErrValueMissing is returned if the transaction value is missing.
	ErrValueMissing = errors.New("value missing")

	// ErrValueNegative is returned if the transaction value is negative.
	ErrValueNegative = errors.New("negative value")

	// ErrValueTooHigh is returned if the transaction value is higher than 2^256-1.
	ErrValueTooHigh = errors.New("value higher than 2^256-1")

	// ErrFeeCapMissing is returned if the transaction fee cap is missing in
	// a transaction.
	ErrFeeCapMissing = errors.New("max fee per gas missing")

	// ErrFeeCapNegative is returned if the transaction fee cap is negative.
	ErrFeeCapNegative = errors.New("max fee per gas negative")

	// ErrFeeCapTooHigh is a sanity error to avoid extremely big numbers specified
	// in the fee cap field.
	ErrFeeCapTooHigh = errors.New("max fee per gas higher than 2^256-1")

	// ErrTipCapMissing is returned if the transaction tip cap is missing in
	// a transaction.
	ErrTipCapMissing = errors.New("max priority fee per gas missing")

	// ErrTipCapNegative is a sanity error to ensure no one is able to specify a
	// transaction with a negative tip.
	ErrTipCapNegative = errors.New("max priority fee per gas negative")

	// ErrTipCapTooHigh is a sanity error to avoid extremely big numbers specified
	// in the tip cap field.
	ErrTipCapTooHigh = errors.New("max priority fee per gas higher than 2^256-1")

	// ErrTipAboveFeeCap is a sanity error to ensure no one is able to specify a
	// transaction with a tip higher than the total fee cap.
	ErrTipAboveFeeCap = errors.New("max priority fee per gas higher than max fee per gas")

	// ErrBlobGasFeeCapMissing is returned if the transaction blob fee cap is
	// missing in a blob transaction.
	ErrBlobGasFeeCapMissing = errors.New("max fee per blob gas missing")

	// ErrBlobGasFeeCapNegative is returned if the transaction blob fee cap is negative.
	ErrBlobGasFeeCapNegative = errors.New("max fee per blob gas negative")

	// ErrBlobGasFeeCapTooHigh is a sanity error to avoid extremely big numbers specified
	// in the blob gas fee cap field.
	ErrBlobGasFeeCapTooHigh = errors.New("max fee per blob gas higher than 2^256-1")

	// ErrGasPriceMissing is returned if the transaction gas price is missing.
	ErrGasPriceMissing = errors.New("gas price missing")

	// ErrGasPriceNegative is returned if the transaction gas price is negative.
	ErrGasPriceNegative = errors.New("gas price negative")

	// ErrGasPriceTooHigh is a sanity error to avoid extremely big numbers specified
	// in the gas price field.
	ErrGasPriceTooHigh = errors.New("gas price higher than 2^256-1")

	// ErrGasPriceAboveFeeCap is a sanity error to ensure no one is able to
	// specify a transaction with a gas price higher than the max fee per gas.
	ErrGasPriceAboveFeeCap = errors.New("gas price higher than max fee per gas")

	// ErrCostNegative is returned if the total cost of a transaction is negative.
	ErrCostNegative = errors.New("total cost negative")

	// ErrCostMissing is returned if the total cost of a transaction is missing.
	ErrCostMissing = errors.New("total cost missing")

	// ErrCostTooHigh is returned if the total cost of a transaction is higher than 2^256-1.
	ErrCostTooHigh = errors.New("total cost higher than 2^256-1")

	// ErrFeeCapTooLow is returned if the transaction fee cap is less than the
	// the base fee of the block.
	ErrFeeCapTooLow = errors.New("max fee per gas less than block base fee")

	// ErrSenderNoEOA is returned if the sender of a transaction is a contract.
	ErrSenderNoEOA = errors.New("sender not an eoa")

	// ErrEmptyAuthorizations is returned if a SetCode transaction has no authorizations.
	ErrEmptyAuthorizations = errors.New("empty authorizations")

	// ErrNonEmptyAuthorizations is returned if a non-SetCode transaction has authorizations.
	ErrNonEmptyAuthorizations = errors.New("non-empty authorizations")

	// ErrMaxInitCodeSizeExceeded is returned if creation transaction provides the init code bigger
	// than init code size limit.
	ErrMaxInitCodeSizeExceeded = errors.New("max initcode size exceeded")

	// ErrInflightTxLimitReached is returned when the maximum number of in-flight
	// transactions is reached for specific accounts.
	ErrInflightTxLimitReached = errors.New("in-flight transaction limit reached for delegated accounts")

	// ErrAuthorityReserved is returned if a transaction has an authorization
	// signed by an address which already has in-flight transactions known to the
	// pool.
	ErrAuthorityReserved = errors.New("authority already reserved")

	// ErrNonEmptyBlobTx is returned if a blob transaction has non-empty blob data.
	ErrNonEmptyBlobTx = errors.New("non-empty blob transaction are not supported")

	// -- EIP-7825 error --
	ErrGasLimitTooHigh = errors.New("transaction gas limit too high")
)
