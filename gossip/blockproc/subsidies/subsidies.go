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

// IsSponsorshipRequest checks if a transaction is requesting sponsorship from
// a pre-allocated sponsorship pool. A sponsorship request is defined as a
// transaction with a maximum gas price of zero.
func IsSponsorshipRequest(tx *types.Transaction) bool {
	return tx != nil &&
		!internaltx.IsInternal(tx) &&
		tx.To() != nil &&
		tx.GasPrice().Sign() == 0
}

// Identifier is a generic 32-byte Identifier type used for fund IDs and
// tracking IDs returned by the subsidies registry. The semantics of the
// Identifier are opaque to this code and only interpreted by the registry.
type Identifier [32]byte

// IsCovered checks if the given transaction is covered by available subsidies.
// If preconditions are met, it queries the subsidies registry contract. If
// there are sufficient funds, it returns a Sponsorship with IsSponsored true,
// otherwise IsSponsored returns false.
func IsCovered(
	upgrades opera.Upgrades,
	vm VirtualMachine,
	signer types.Signer,
	tx *types.Transaction,
	baseFee *big.Int,
) (Sponsorship, error) {
	if !upgrades.GasSubsidies {
		return Sponsorship{}, nil
	}
	if !IsSponsorshipRequest(tx) {
		return Sponsorship{}, nil
	}

	// Derive the sender of the transaction before interacting with the EVM.
	sender, err := types.Sender(signer, tx)
	if err != nil {
		return Sponsorship{}, fmt.Errorf("failed to derive sender: %w", err)
	}

	// Fetch the current configuration from the subsidies registry.
	gasConfig, err := getGasConfig(vm)
	if err != nil {
		return Sponsorship{}, fmt.Errorf("failed to get gas config: %w", err)
	}

	// Build the choose-fund query call to the subsidies registry contract.
	caller := common.Address{}
	target := registry.GetAddress()

	// Build the input data for the chooseFund call.
	maxOverhead := max(gasConfig.overheadFundBackedSponsoring, gasConfig.overheadNetworkTrackedSponsoring)
	maxGas := new(big.Int).Add(new(big.Int).SetUint64(tx.Gas()), new(big.Int).SetUint64(maxOverhead))
	maxFee := new(big.Int).Mul(baseFee, maxGas)
	input, err := createChooseFundInput(sender, tx, maxFee)
	if err != nil {
		return Sponsorship{}, fmt.Errorf("failed to create input for subsidies registry call: %w", err)
	}

	// Run the query on the EVM and the provided state.
	result, _, err := vm.Call(caller, target, input, gasConfig.gasLimitForChooseFund, uint256.NewInt(0))
	if err != nil {
		return Sponsorship{}, fmt.Errorf("EVM call failed: %w", err)
	}

	// An empty result indicates that there is no contract installed.
	if len(result) == 0 {
		return Sponsorship{}, fmt.Errorf("subsidies registry contract not found")
	}

	// Parse the result of the call.
	mode, id, err := parseChooseFundResult(result)
	if err != nil {
		return Sponsorship{}, fmt.Errorf("failed to parse result of subsidies registry call: %w", err)
	}
	return Sponsorship{
		mode:   mode,
		id:     id,
		config: gasConfig,
	}, nil
}

// Sponsorship holds the outcome of an IsCovered query and exposes the
// sponsorship details through behavioral methods.
type Sponsorship struct {
	mode   sponsorshipMode
	id     Identifier
	config gasConfig
}

// IsSponsored reports whether the transaction is covered by any sponsorship.
func (s Sponsorship) IsSponsored() bool {
	return s.mode != sponsorshipModeNotCovered
}

// Overhead returns the gas and size resources to reserve for post-execution
// transactions. Both fields are zero for modes that append no post-execution
// transaction (modes 0 and 2).
func (s Sponsorship) Overhead() Overhead {
	switch s.mode {
	case sponsorshipModeFundBacked:
		return Overhead{
			Gas:  s.config.overheadFundBackedSponsoring,
			Size: rlpEncodedRegistryCallTxSizeInBytes,
		}
	case sponsorshipModeNetworkWithTracking:
		return Overhead{
			Gas:  s.config.overheadNetworkTrackedSponsoring,
			Size: rlpEncodedRegistryCallTxSizeInBytes,
		}
	default:
		return Overhead{}
	}
}

// GetPostTransactions returns the post-execution transactions to append to the
// block after a sponsored transaction has been executed. For fund-backed
// sponsorships (mode 1) this is a deductFees call; for network-sponsored with
// tracking (mode 3) this is a track call; for all other modes the slice is nil.
func (s Sponsorship) GetPostTransactions(
	nonceSource NonceSource,
	gasUsed uint64,
	gasPrice *big.Int,
) ([]*types.Transaction, error) {
	overheadGasCosts := s.Overhead().Gas
	switch s.mode {
	case sponsorshipModeFundBacked:
		tx, err := buildRegistryCallTransaction(
			nonceSource, s.id, overheadGasCosts, gasUsed, gasPrice,
			registry.DeductFeesFunctionSelector, s.config.gasLimitForDeductFees,
		)
		if err != nil {
			return nil, err
		}
		return []*types.Transaction{tx}, nil
	case sponsorshipModeNetworkWithTracking:
		tx, err := buildRegistryCallTransaction(
			nonceSource, s.id, overheadGasCosts, gasUsed, gasPrice,
			registry.TrackFunctionSelector, s.config.gasLimitForTrack,
		)
		if err != nil {
			return nil, err
		}
		return []*types.Transaction{tx}, nil
	default:
		return nil, nil
	}
}

// Overhead holds the block resources to reserve for any
// post-execution transactions appended after a sponsored transaction.
type Overhead struct {
	// Gas is the extra gas to reserve in the block gas pool.
	Gas uint64
	// Size is the upper-bound byte size of the post-execution transactions.
	Size uint64
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

// buildRegistryCallTransaction is the shared implementation used to construct
// post-execution registry call transactions (deductFees and track). It encodes
// the ABI call (selector, id, fee) and wraps it in an internal transaction
// targeting the subsidies registry.
func buildRegistryCallTransaction(
	nonceSource NonceSource,
	payload Identifier,
	overheadGasCosts uint64,
	gasUsed uint64,
	gasPrice *big.Int,
	selector uint32,
	gasLimit uint64,
) (*types.Transaction, error) {
	sender := common.Address{}
	nonce := nonceSource.GetNonce(sender)

	// Calculate the fee to be charged: (gasUsed + overhead) * gasPrice
	fee, overflow := uint256.FromBig(new(big.Int).Mul(
		new(big.Int).Add(
			new(big.Int).SetUint64(gasUsed),
			new(big.Int).SetUint64(overheadGasCosts),
		),
		gasPrice,
	))
	if overflow {
		return nil, fmt.Errorf("fee calculation overflow")
	}

	input := createRegistryCallInput(selector, payload, *fee)
	return types.NewTransaction(
		nonce, registry.GetAddress(), common.Big0,
		gasLimit, common.Big0, input,
	), nil
}

// NonceSource provides nonces for addresses. It is used to determine the
// correct nonce for the fee deduction transaction.
type NonceSource interface {
	GetNonce(addr common.Address) uint64
}

// IsFeeChargeTransaction returns true if the transaction is a fee-charge
// transaction created as a follow-up to fund-backed sponsorships. It checks
// that the transaction is internal, targets the subsidies registry, and carries
// calldata with the correct length and deductFees function selector.
func IsFeeChargeTransaction(tx *types.Transaction) bool {
	return isRegistryCallTransaction(tx, registry.DeductFeesFunctionSelector)
}

// IsTrackTransaction returns true if the transaction is a track transaction
// created as a follow-up to network-sponsored transactions with tracking. It
// checks that the transaction is internal, targets the subsidies registry, and
// carries calldata with the correct length and track function selector.
func IsTrackTransaction(tx *types.Transaction) bool {
	return isRegistryCallTransaction(tx, registry.TrackFunctionSelector)
}

// isRegistryCallTransaction returns true if tx is an internal transaction
// targeting the subsidies registry with the given function selector.
func isRegistryCallTransaction(tx *types.Transaction, selector uint32) bool {
	if tx == nil || !internaltx.IsInternal(tx) {
		return false
	}
	if tx.To() == nil || *tx.To() != registry.GetAddress() {
		return false
	}
	input := tx.Data()
	if len(input) != 4+2*32 {
		return false
	}
	return binary.BigEndian.Uint32(input[:4]) == selector
}

// ParseFeeChargeAmount extracts the fee amount from the input data of a fee
// charge transaction inserted after a fund-backed sponsorship.
func ParseFeeChargeAmount(tx *types.Transaction) (*uint256.Int, error) {
	if !IsFeeChargeTransaction(tx) {
		return nil, fmt.Errorf("transaction is not a fee charge transaction")
	}
	input := tx.Data()
	return new(uint256.Int).SetBytes32(input[36:68]), nil
}

// ParseTrackAmount extracts the fee amount from the input data of a track
// transaction inserted after a network-sponsored transaction with tracking.
func ParseTrackAmount(tx *types.Transaction) (*uint256.Int, error) {
	if !IsTrackTransaction(tx) {
		return nil, fmt.Errorf("transaction is not a track transaction")
	}
	input := tx.Data()
	return new(uint256.Int).SetBytes32(input[36:68]), nil
}

// --- utility functions ---

// rlpEncodedRegistryCallTxSizeInBytes is an upper bound for the size of an
// RLP-encoded registry call transaction (deductFees or track) introduced for
// gas subsidies. Both have identical ABI shape (two 32-byte params).
const rlpEncodedRegistryCallTxSizeInBytes = 128

// sponsorshipMode is the mode returned by the subsidies registry to indicate
// how a sponsored transaction should be handled.
type sponsorshipMode uint8

const (
	// sponsorshipModeNotCovered means the transaction is not sponsored.
	sponsorshipModeNotCovered sponsorshipMode = 0
	// sponsorshipModeFundBacked means the transaction is sponsored by a fund.
	// A deductFees call is appended after execution.
	sponsorshipModeFundBacked sponsorshipMode = 1
	// sponsorshipModeNetwork means the transaction is sponsored by the network.
	// No post-execution call is appended; the cost is absorbed by block producers.
	sponsorshipModeNetwork sponsorshipMode = 2
	// sponsorshipModeNetworkWithTracking means the transaction is sponsored by
	// the network with on-chain tracking. A track call is appended after execution.
	sponsorshipModeNetworkWithTracking sponsorshipMode = 3
)

func allModes() []sponsorshipMode {
	// Make sure to keep this up-to-date with the constant definitions above.
	return []sponsorshipMode{
		sponsorshipModeNotCovered,
		sponsorshipModeFundBacked,
		sponsorshipModeNetwork,
		sponsorshipModeNetworkWithTracking,
	}
}

// getGasConfig queries the subsidies registry contract for the current gas
// configuration. It returns the gas limits to be used when calling the
// `chooseFund`, `deductFees`, and `track` functions, as well as the per-mode
// overhead to charge for sponsored transactions.
func getGasConfig(
	vm VirtualMachine,
) (gasConfig, error) {
	// Call the getGasConfig function on the subsidies registry contract, which
	// takes no arguments and returns three or five uint64 values.
	caller := common.Address{}
	target := registry.GetAddress()

	// Build the input data for the getGasConfig call.
	input := make([]byte, 4) // function selector only
	binary.BigEndian.PutUint32(input, registry.GetGasConfigFunctionSelector)

	// Run the query on the EVM and the provided state.
	const initialGas = registry.GasLimitForGetGasConfig
	result, _, err := vm.Call(caller, target, input, initialGas, uint256.NewInt(0))
	if err != nil {
		return gasConfig{}, fmt.Errorf("EVM call failed: %w", err)
	}

	// An empty result indicates that there is no contract installed.
	if len(result) == 0 {
		return gasConfig{}, fmt.Errorf("subsidies registry contract not found")
	}

	if len(result) != 3*32 && len(result) != 5*32 {
		return gasConfig{}, fmt.Errorf("invalid result length from getGasConfig call: %d", len(result))
	}

	// check for uint64 overflows in the first two words (common to both formats)
	type bytes24 [24]byte
	zero := bytes24{}
	if bytes24(result[0:32-8]) != zero ||
		bytes24(result[32:64-8]) != zero {
		return gasConfig{}, fmt.Errorf("invalid result from getGasConfig call, values do not fit into uint64")
	}

	chooseFundGasLimit := binary.BigEndian.Uint64(result[32-8 : 32])
	deductFeesGasLimit := binary.BigEndian.Uint64(result[64-8 : 64])

	if len(result) == 3*32 {
		// Legacy 3-field format: chooseFundGasLimit, deductFeesGasLimit, overheadCharge.
		// Apply the shared overhead to both sponsored modes conservatively.
		if bytes24(result[64:96-8]) != zero {
			return gasConfig{}, fmt.Errorf("invalid result from getGasConfig call, values do not fit into uint64")
		}
		overheadCharge := binary.BigEndian.Uint64(result[96-8 : 96])
		return gasConfig{
			gasLimitForChooseFund:            chooseFundGasLimit,
			gasLimitForDeductFees:            deductFeesGasLimit,
			gasLimitForTrack:                 0,
			overheadFundBackedSponsoring:     overheadCharge,
			overheadNetworkTrackedSponsoring: overheadCharge,
		}, nil
	}

	// 5-field format: chooseFundGasLimit, deductFeesGasLimit, trackGasLimit,
	// fundBackedOverheadCharge, networkTrackedOverheadCharge.
	if bytes24(result[64:96-8]) != zero ||
		bytes24(result[96:128-8]) != zero ||
		bytes24(result[128:160-8]) != zero {
		return gasConfig{}, fmt.Errorf("invalid result from getGasConfig call, values do not fit into uint64")
	}

	trackGasCost := binary.BigEndian.Uint64(result[96-8 : 96])
	fundBackedOverhead := binary.BigEndian.Uint64(result[128-8 : 128])
	networkTrackedOverhead := binary.BigEndian.Uint64(result[160-8 : 160])

	return gasConfig{
		gasLimitForChooseFund:            chooseFundGasLimit,
		gasLimitForDeductFees:            deductFeesGasLimit,
		gasLimitForTrack:                 trackGasCost,
		overheadFundBackedSponsoring:     fundBackedOverhead,
		overheadNetworkTrackedSponsoring: networkTrackedOverhead,
	}, nil
}

type gasConfig struct {
	gasLimitForChooseFund            uint64
	gasLimitForDeductFees            uint64
	gasLimitForTrack                 uint64
	overheadFundBackedSponsoring     uint64
	overheadNetworkTrackedSponsoring uint64
}

// createChooseFundInput creates the input data for the chooseFund call to the
// subsidies registry contract.
func createChooseFundInput(
	sender common.Address,
	tx *types.Transaction,
	fee *big.Int,
) ([]byte, error) {
	if tx == nil || fee == nil {
		return nil, fmt.Errorf("invalid transaction or fee")
	}
	if fee.BitLen() > 256 {
		return nil, fmt.Errorf("fee does not fit into 32 bytes")
	}

	to := common.Address{}
	if tx.To() != nil {
		to = *tx.To()
	}

	// Add the function selector for `chooseFund`.
	input := []byte{}
	input = binary.BigEndian.AppendUint32(input, registry.ChooseFundFunctionSelector)

	// The from and to addresses are padded to 32 bytes.
	addressPadding := [12]byte{}
	input = append(input, addressPadding[:]...)
	input = append(input, sender[:]...)
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

// parseChooseFundResult parses the result of the chooseFund call.
// Legacy registries return 32 bytes (bare fundId); extended registries return
// 64 bytes (mode uint256, payload bytes32).
func parseChooseFundResult(data []byte) (mode sponsorshipMode, id Identifier, err error) {
	switch len(data) {
	case 32:
		// Legacy registry: bare fundId. Zero means not covered (mode 0),
		// non-zero means fund-backed (mode 1).
		if Identifier(data[0:32]) == (Identifier{}) {
			return sponsorshipModeNotCovered, Identifier{}, nil
		}
		return sponsorshipModeFundBacked, Identifier(data[0:32]), nil

	case 64:
		// Extended registry: (mode uint256, payload bytes32).
		// Validate that the mode fits in a uint8 and is a known value.
		type bytes31 [31]byte
		if bytes31(data[0:31]) != (bytes31{}) {
			return 0, Identifier{}, fmt.Errorf("mode value out of range")
		}
		rawMode := data[31]
		if rawMode > 3 {
			return 0, Identifier{}, fmt.Errorf("unknown sponsorship mode: %d", rawMode)
		}
		m := sponsorshipMode(rawMode)
		id := Identifier(data[32:64])
		if m == sponsorshipModeNotCovered || m == sponsorshipModeNetwork {
			id = Identifier{} // ignore the payload for modes that don't use it
		}
		return m, id, nil

	default:
		return 0, Identifier{}, fmt.Errorf("invalid result length from chooseFund call: %d", len(data))
	}
}

// createRegistryCallInput encodes an ABI call with the given selector, a
// bytes32 id, and a uint256 fee — the shared shape of both deductFees and track.
func createRegistryCallInput(selector uint32, id Identifier, fee uint256.Int) []byte {
	input := make([]byte, 4+2*32) // selector + 2 × 32-byte params
	binary.BigEndian.PutUint32(input, selector)
	copy(input[4:36], id[:])
	fee.WriteToArray32((*[32]byte)(input[36:68]))
	return input
}
