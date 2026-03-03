// Copyright 2024 The go-ethereum Authors
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

package ethapi

import (
	"errors"

	"github.com/ethereum/go-ethereum/core"
)

const (
	errCodeReverted                = -32000
	errCodeVMError                 = -32015
	errCodeInvalidParams           = -32602
	errCodeInternalError           = -32603
	errCodeNonceTooHigh            = -38011
	errCodeNonceTooLow             = -38010
	errCodeIntrinsicGas            = -38013
	errCodeInsufficientFunds       = -38014
	errCodeBlockGasLimitReached    = -38015
	errCodeBlockNumberInvalid      = -38020
	errCodeBlockTimestampInvalid   = -38021
	errCodeSenderIsNotEOA          = -38024
	errCodeMaxInitCodeSizeExceeded = -38025
	errCodeClientLimitExceeded     = -38026
)

// simInvalidTxError is an error type for invalid transactions
// in simulation, with an associated error code for JSON-RPC responses.
type simInvalidTxError struct {
	Message string
	Code    int
}

func (e *simInvalidTxError) Error() string  { return e.Message }
func (e *simInvalidTxError) ErrorCode() int { return e.Code }

func simInvalidParamsError() *simInvalidTxError {
	return &simInvalidTxError{Message: "empty input", Code: errCodeInvalidParams}
}
func simClientLimitExceededError() *simInvalidTxError {
	return &simInvalidTxError{Message: "too many blocks", Code: errCodeClientLimitExceeded}
}
func simInvalidBlockNumberError(message string) *simInvalidTxError {
	return &simInvalidTxError{Message: message, Code: errCodeBlockNumberInvalid}
}
func simInvalidBlockTimestampError(message string) *simInvalidTxError {
	return &simInvalidTxError{Message: message, Code: errCodeBlockTimestampInvalid}
}
func simBlockGasLimitReachedError(message string) *simInvalidTxError {
	return &simInvalidTxError{Message: message, Code: errCodeBlockGasLimitReached}
}

// simTxValidationError maps core transaction validation errors
// to simInvalidTxError with appropriate message and error codes.
func simTxValidationError(err error) *simInvalidTxError {
	switch {
	case errors.Is(err, core.ErrNonceTooHigh):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeNonceTooHigh}
	case errors.Is(err, core.ErrNonceTooLow):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeNonceTooLow}
	case errors.Is(err, core.ErrSenderNoEOA):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeSenderIsNotEOA}
	case errors.Is(err, core.ErrFeeCapTooLow):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrTipAboveFeeCap):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrTipVeryHigh):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrFeeCapVeryHigh):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrInsufficientFunds):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeInsufficientFunds}
	case errors.Is(err, core.ErrIntrinsicGas):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeIntrinsicGas}
	case errors.Is(err, core.ErrMaxInitCodeSizeExceeded):
		return &simInvalidTxError{Message: err.Error(), Code: errCodeMaxInitCodeSizeExceeded}
	default:
		return &simInvalidTxError{
			Message: err.Error(),
			Code:    errCodeInternalError,
		}
	}
}
