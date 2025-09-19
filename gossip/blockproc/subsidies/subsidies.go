// Copyright 2025 Sonic Operations Ltd
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

package subsidies

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// SponsorshipOverheadGasCost is the additional gas cost incurred when a
// transaction is sponsored. It covers the overhead of calling the subsidies
// registry contract to check for available funds and to deduct the fees after
// the sponsored transaction has been executed.
const SponsorshipOverheadGasCost = 50_000 // < TODO: reevaluate this value

// IsSponsorshipRequest checks if a transaction is requesting sponsorship from
// a pre-allocated sponsorship pool. A sponsorship request is defined as a
// transaction with zero value and zero gas price (legacy) or zero gas fee cap
// and zero gas tip cap (EIP-1559).
func IsSponsorshipRequest(tx *types.Transaction) bool {
	return tx.To() != nil &&
		tx.Value().Sign() == 0 &&
		tx.GasPrice().Sign() == 0 &&
		tx.GasFeeCap().Sign() == 0 &&
		tx.GasTipCap().Sign() == 0
}

// IsCovered checks if the given transaction is covered by the subsidies
// registry contract. It returns true if sponsorship funds are available to
// cover the given transaction, false otherwise.
func IsCovered(
	blockContext vm.BlockContext,
	signer types.Signer,
	chainConfig *params.ChainConfig,
	rules opera.Rules,
	state state.StateDB,
	tx *types.Transaction,
) (bool, error) {
	if !rules.Upgrades.GasSubsidies {
		return false, nil
	}
	if !IsSponsorshipRequest(tx) {
		return false, nil
	}

	// Create a EVM processor instance to run the IsCovered query.
	vmConfig := opera.GetVmConfig(rules)
	vm := vm.NewEVM(blockContext, state, chainConfig, vmConfig)
	return IsCoveredBy(vm, signer, tx, blockContext.BaseFee)
}

func IsCoveredBy( // < TODO: find a better name
	vm *vm.EVM,
	signer types.Signer,
	tx *types.Transaction,
	baseFee *big.Int,
) (bool, error) {
	// Build the example query call to the subsidies registry contract.
	caller := common.Address{}
	target := registry.GetAddress()

	// Build the input data for the IsCovered call.
	from, to, selector, err := getTransactionDetails(signer, tx)
	if err != nil {
		return false, fmt.Errorf("failed to get transaction details: %v", err)
	}
	maxGas := tx.Gas() + SponsorshipOverheadGasCost
	maxFee := new(big.Int).Mul(baseFee, new(big.Int).SetUint64(maxGas))
	input := packIsCoveredInput(from, to, selector, maxFee)

	// Run the query on the EVM and the provided state.
	const initialGas = 7_500 // TODO: figure out a sensible value
	result, _, err := vm.Call(caller, target, input, initialGas, uint256.NewInt(0))
	if err != nil {
		return false, fmt.Errorf("EVM call failed: %v", err)
	}

	return len(result) == 32 && result[31] == 1, nil
}

// GetFeeChargeTransaction builds a transaction that deducts the given fee
// amount from the sponsorship pool of the given subsidies registry contract.
// The returned transaction is unsigned and has zero value and gas price. It is
// intended to be introduced by the block processor after the sponsored
// transaction has been executed.
func GetFeeChargeTransaction(
	nonceSource NonceSource,
	signer types.Signer,
	tx *types.Transaction,
	gasUsed uint64,
	gasPrice *big.Int,
) (*types.Transaction, error) {
	const gasLimit = 100_000 // TODO: re-evaluate this value
	sender := common.Address{}
	nonce := nonceSource.GetNonce(sender)
	from, to, selector, err := getTransactionDetails(signer, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction details: %v", err)
	}
	// TODO: add overflow checks
	fee := new(big.Int).Mul(
		new(big.Int).SetUint64(gasUsed+SponsorshipOverheadGasCost),
		gasPrice,
	)
	input := packDeductFeesInput(from, to, selector, fee)
	return types.NewTransaction(
		nonce, registry.GetAddress(), common.Big0, gasLimit, common.Big0, input,
	), nil
}

type NonceSource interface {
	GetNonce(addr common.Address) uint64
}

// --- utility functions ---

func getTransactionDetails(
	signer types.Signer,
	tx *types.Transaction,
) (
	from common.Address,
	to common.Address,
	functionSelector [4]byte,
	err error,
) {
	from, err = signer.Sender(tx)
	if err != nil {
		return common.Address{}, common.Address{}, [4]byte{}, fmt.Errorf("failed to derive sender: %v", err)
	}
	if tx.To() == nil {
		return common.Address{}, common.Address{}, [4]byte{}, fmt.Errorf("transaction has no recipient")
	}
	to = *tx.To()
	if data := tx.Data(); len(data) >= 4 {
		copy(functionSelector[:], data[:4])
	}
	return from, to, functionSelector, nil
}

func packIsCoveredInput(from, to common.Address, functionSelector [4]byte, fee *big.Int) []byte {
	input := make([]byte, 0, 4+32*4) // selector + 4x 32-byte arguments
	input = binary.BigEndian.AppendUint32(input, registry.IsCoveredFunctionSelector)
	return appendPackedParameters(input, from, to, functionSelector, fee)
}

func packDeductFeesInput(from, to common.Address, functionSelector [4]byte, fee *big.Int) []byte {
	input := make([]byte, 0, 4+32*4) // selector + 4x 32-byte arguments
	input = binary.BigEndian.AppendUint32(input, registry.DeductFeesFunctionSelector)
	return appendPackedParameters(input, from, to, functionSelector, fee)
}

func appendPackedParameters(
	input []byte,
	from, to common.Address,
	functionSelector [4]byte,
	fee *big.Int,
) []byte {
	addressPadding := [12]byte{}
	input = append(input, addressPadding[:]...)
	input = append(input, from[:]...)
	input = append(input, addressPadding[:]...)
	input = append(input, to[:]...)
	input = append(input, functionSelector[:]...)
	padding := [28]byte{}
	input = append(input, padding[:]...)
	input = append(input, fee.FillBytes(make([]byte, 32))...)
	return input
}
