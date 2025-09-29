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
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

//go:generate mockgen -source=subsidies.go -destination=subsidies_mock.go -package=subsidies

// SponsorshipOverheadGasCost is the additional gas cost incurred when a
// transaction is sponsored. It covers the overhead of calling the subsidies
// registry contract to check for available funds and to deduct the fees after
// the sponsored transaction has been executed.
const SponsorshipOverheadGasCost = 0 +
	registry.GasLimitForChooseFundCall +
	registry.GasLimitForDeductFeesCall

// IsSponsorshipRequest checks if a transaction is requesting sponsorship from
// a pre-allocated sponsorship pool. A sponsorship request is defined as a
// transaction with a maximum gas price of zero.
func IsSponsorshipRequest(tx *types.Transaction) bool {
	return tx != nil &&
		!internaltx.IsInternal(tx) &&
		tx.To() != nil &&
		tx.GasPrice().Sign() == 0
}

// FundId is an identifier for a fund in the subsidies registry contract.
type FundId [32]byte

func (id FundId) String() string {
	return fmt.Sprintf("0x%x", id[:])
}

// IsCovered checks if the given transaction is covered by available subsidies.
// If preconditions are met, it queries the subsidies registry contract. If
// there are sufficient funds, it returns true, otherwise false.
func IsCovered(
	upgrades opera.Upgrades,
	vm VirtualMachine,
	reader SenderReader,
	tx *types.Transaction,
	baseFee *big.Int,
) (bool, FundId, error) {
	if !upgrades.GasSubsidies {
		return false, FundId{}, nil
	}
	if !IsSponsorshipRequest(tx) {
		return false, FundId{}, nil
	}

	// Build the example query call to the subsidies registry contract.
	caller := common.Address{}
	target := registry.GetAddress()

	// Build the input data for the IsCovered call.
	maxGas := tx.Gas() + SponsorshipOverheadGasCost
	maxFee := new(big.Int).Mul(baseFee, new(big.Int).SetUint64(maxGas))
	input, err := createChooseFundInput(reader, tx, maxFee)
	if err != nil {
		return false, FundId{}, fmt.Errorf("failed to create input for subsidies registry call: %w", err)
	}

	// Run the query on the EVM and the provided state.
	const initialGas = registry.GasLimitForChooseFundCall
	result, _, err := vm.Call(caller, target, input, initialGas, uint256.NewInt(0))
	if err != nil {
		return false, FundId{}, fmt.Errorf("EVM call failed: %w", err)
	}

	// An empty result indicates that there is no contract installed.
	if len(result) == 0 {
		return false, FundId{}, fmt.Errorf("subsidies registry contract not found")
	}

	// Parse the result of the call.
	covered, fundID, err := parseChooseFundResult(result)
	if err != nil {
		return false, FundId{}, fmt.Errorf("failed to parse result of subsidies registry call: %w", err)
	}
	return covered, fundID, nil
}

// VirtualMachine is a minimal interface for an EVM instance that can be used
// to query the subsidies registry contract.
type VirtualMachine interface {
	Call(
		from common.Address,
		to common.Address,
		input []byte,
		gas uint64,
		value *uint256.Int,
	) (
		result []byte,
		gasLeft uint64,
		err error,
	)
}

// GetFeeChargeTransaction builds a transaction that deducts the given fee
// amount from the sponsorship pool of the given subsidies registry contract.
// The returned transaction is unsigned and has zero value and gas price. It is
// intended to be introduced by the block processor after the sponsored
// transaction has been executed.
func GetFeeChargeTransaction(
	nonceSource NonceSource,
	fundId FundId,
	gasUsed uint64,
	gasPrice *big.Int,
) (*types.Transaction, error) {
	const gasLimit = registry.GasLimitForDeductFeesCall
	sender := common.Address{}
	nonce := nonceSource.GetNonce(sender)

	// Calculate the fee to be deducted: (gasUsed + overhead) * gasPrice
	fee, overflow := uint256.FromBig(new(big.Int).Mul(
		new(big.Int).Add(
			new(big.Int).SetUint64(gasUsed),
			new(big.Int).SetUint64(SponsorshipOverheadGasCost),
		),
		gasPrice,
	))
	if overflow {
		return nil, fmt.Errorf("fee calculation overflow")
	}

	input := createDeductFeesInput(fundId, *fee)
	return types.NewTransaction(
		nonce, registry.GetAddress(), common.Big0, gasLimit, common.Big0, input,
	), nil
}

// NonceSource provides nonces for addresses. It is used to determine the
// correct nonce for the fee deduction transaction.
type NonceSource interface {
	GetNonce(addr common.Address) uint64
}

// SenderReader is an interface for types that can extract the sender
// address from a transaction. Typically, this is an implementation of
// types.Signer.
type SenderReader interface {
	Sender(*types.Transaction) (common.Address, error)
}

// --- utility functions ---

// createChooseFundInput creates the input data for the chooseFund call to the
// subsidies registry contract.
func createChooseFundInput(
	reader SenderReader,
	tx *types.Transaction,
	fee *big.Int,
) ([]byte, error) {
	if reader == nil || tx == nil || fee == nil {
		return nil, fmt.Errorf("invalid transaction, reader, or fee")
	}
	if fee.BitLen() > 256 {
		return nil, fmt.Errorf("fee does not fit into 32 bytes")
	}
	from, err := reader.Sender(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to derive sender: %w", err)
	}

	to := common.Address{}
	if tx.To() != nil {
		to = *tx.To()
	}

	// Add the function selector for `isCovered`.
	input := []byte{}
	input = binary.BigEndian.AppendUint32(input, registry.ChooseFundFunctionSelector)

	// The from and to addresses are padded to 32 bytes.
	addressPadding := [12]byte{}
	input = append(input, addressPadding[:]...)
	input = append(input, from[:]...)
	input = append(input, addressPadding[:]...)
	input = append(input, to[:]...)

	// The value is padded to 32 bytes.
	input = append(input, tx.Value().FillBytes(make([]byte, 32))...)

	// The nonce is padded to 32 bytes.
	uint64Padding := [24]byte{}
	input = append(input, uint64Padding[:]...)
	input = binary.BigEndian.AppendUint64(input, tx.Nonce())

	// The calldata is a dynamic parameter, encoded as its offset in the input
	// data. Dynamic sized parameters are at the end of the input data.
	input = append(input, uint64Padding[:]...)
	input = binary.BigEndian.AppendUint64(input, 32*6) // 6 32-byte parameters

	// The fee is padded to 32 bytes.
	input = append(input, fee.FillBytes(make([]byte, 32))...)

	// -- dynamic sized parameters --

	// The input data is prefixed by its length as a 32-byte value,
	// followed by the actual data, padded to a multiple of 32 bytes.
	input = append(input, uint64Padding[:]...)
	input = binary.BigEndian.AppendUint64(input, uint64(len(tx.Data())))
	input = append(input, tx.Data()...)
	if len(tx.Data())%32 != 0 {
		dataPadding := make([]byte, 32-len(tx.Data())%32)
		input = append(input, dataPadding...)
	}

	return input, nil
}

// parseChooseFundResult parses the result of the IsCovered call to the
// subsidies registry contract.
func parseChooseFundResult(data []byte) (covered bool, fundID FundId, err error) {
	// The result is a 32-byte long FundId.
	if len(data) != 32 {
		return false, FundId{}, fmt.Errorf("invalid result length from chooseFund call: %d", len(data))
	}
	fundId := FundId(data[0:32])
	return fundId != (FundId{}), fundId, nil
}

// createDeductFeesInput creates the input data for the DeductFees call to the
// subsidies registry contract.
func createDeductFeesInput(fundId FundId, fee uint256.Int) []byte {
	// Signature: deductFees(bytes32 fundId, uint256 fee)
	input := make([]byte, 4+2*32) // function selector + 2 parameters

	binary.BigEndian.PutUint32(input, registry.DeductFeesFunctionSelector)
	copy(input[4:36], fundId[:])
	fee.WriteToArray32((*[32]byte)(input[36:68]))
	return input
}
