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

package drivercall

import (
	_ "embed"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver"
	"github.com/0xsoniclabs/sonic/opera/genesis/gpos"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
)

//go:embed NodeDriverAbi.json
var ContractABI string

var (
	sAbi, _ = abi.JSON(strings.NewReader(ContractABI))
)

type Delegation struct {
	Address            common.Address
	ValidatorID        idx.ValidatorID
	Stake              *big.Int
	LockedStake        *big.Int
	LockupFromEpoch    idx.Epoch
	LockupEndTime      idx.Epoch
	LockupDuration     uint64
	EarlyUnlockPenalty *big.Int
	Rewards            *big.Int
}

// Methods

func SealEpochValidators(_validators []idx.ValidatorID) []byte {
	newValidatorsIDs := make([]*big.Int, len(_validators))
	for i, v := range _validators {
		newValidatorsIDs[i] = utils.U64toBig(uint64(v))
	}
	data, _ := sAbi.Pack("sealEpochValidators", newValidatorsIDs)
	return data
}

type ValidatorEpochMetric struct {
	Missed          opera.BlocksMissed
	Uptime          inter.Timestamp
	OriginatedTxFee *big.Int
}

func SealEpoch(metrics []ValidatorEpochMetric) []byte {
	offlineTimes := make([]*big.Int, len(metrics))
	offlineBlocks := make([]*big.Int, len(metrics))
	uptimes := make([]*big.Int, len(metrics))
	originatedTxFees := make([]*big.Int, len(metrics))
	for i, m := range metrics {
		offlineTimes[i] = utils.U64toBig(uint64(m.Missed.Period.Unix()))
		offlineBlocks[i] = utils.U64toBig(uint64(m.Missed.BlocksNum))
		uptimes[i] = utils.U64toBig(uint64(m.Uptime.Unix()))
		originatedTxFees[i] = m.OriginatedTxFee
	}

	data, _ := sAbi.Pack("sealEpoch", offlineTimes, offlineBlocks, uptimes, originatedTxFees)
	return data
}

func SetGenesisValidator(v gpos.Validator) []byte {
	data, _ := sAbi.Pack("setGenesisValidator", v.Address, utils.U64toBig(uint64(v.ID)), v.PubKey.Bytes(), utils.U64toBig(uint64(v.CreationTime.Unix())))
	return data
}

func SetGenesisDelegation(d Delegation) []byte {
	data, _ := sAbi.Pack("setGenesisDelegation", d.Address, utils.U64toBig(uint64(d.ValidatorID)), d.Stake)
	return data
}

func DeactivateValidator(validatorID idx.ValidatorID, status uint64) []byte {
	data, _ := sAbi.Pack("deactivateValidator", utils.U64toBig(uint64(validatorID)), utils.U64toBig(status))
	return data
}

// ParseSealEpochArgs decodes a sealEpoch call and returns the validator epoch
// metrics (missed blocks, uptime, and originated transaction fee), one
// entry per validator in epoch order. Returns an error if data does not
// encode a sealEpoch call.
func ParseSealEpochArgs(tx *types.Transaction) ([]ValidatorEpochMetric, error) {
	if tx == nil || !internaltx.IsInternal(tx) {
		return nil, fmt.Errorf("transaction is nil or not internal")
	}
	if tx.To() == nil || *tx.To() != driver.ContractAddress {
		return nil, fmt.Errorf("transaction does not target the node driver contract")
	}
	data := tx.Data()

	if len(data) < 4 {
		return nil, fmt.Errorf("data too short to contain a function selector")
	}
	method, err := sAbi.MethodById(data[:4])
	if err != nil {
		return nil, fmt.Errorf("unknown method: %w", err)
	}
	if method.Name != "sealEpoch" {
		return nil, fmt.Errorf("expected sealEpoch, got %s", method.Name)
	}
	args, err := method.Inputs.Unpack(data[4:])
	if err != nil {
		return nil, fmt.Errorf("failed to unpack sealEpoch arguments: %w", err)
	}

	// Reconstruct the ValidatorEpochMetric slice from the unpacked arguments.
	return convertSealEpochArgs(args)
}

func convertSealEpochArgs(args []any) ([]ValidatorEpochMetric, error) {
	if len(args) != 4 {
		return nil, fmt.Errorf("expected 4 arguments for sealEpoch, got %d", len(args))
	}

	// Check the types.
	offlineTimes, ok := args[0].([]*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected type for offlineTimes argument")
	}
	offlineBlocks, ok := args[1].([]*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected type for offlineBlocks argument")
	}
	uptimes, ok := args[2].([]*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected type for uptimes argument")
	}
	fees, ok := args[3].([]*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected type for fees argument")
	}

	// Check the length.
	l := len(offlineTimes)
	if len(offlineBlocks) != l || len(uptimes) != l || len(fees) != l {
		return nil, fmt.Errorf("argument array lengths do not match")
	}

	// Check the range limits
	for _, cur := range offlineBlocks {
		if !cur.IsUint64() {
			return nil, fmt.Errorf("offlineBlocks contains value that doesn't fit in uint64")
		}
	}
	for _, cur := range offlineTimes {
		if !cur.IsInt64() {
			return nil, fmt.Errorf("offlineTimes contains value that doesn't fit in int64")
		}
		if val := cur.Int64(); val > math.MaxInt64/int64(time.Second) {
			return nil, fmt.Errorf("offlineTimes contains value that is too large to be a valid offline time in seconds")
		}
	}
	for _, cur := range uptimes {
		if !cur.IsInt64() {
			return nil, fmt.Errorf("uptimes contains value that doesn't fit in int64")
		}
		if val := cur.Int64(); val > math.MaxInt64/int64(time.Second) {
			return nil, fmt.Errorf("uptimes contains value that is too large to be a valid uptime in seconds")
		}
	}

	// Convert to metrics.
	metrics := make([]ValidatorEpochMetric, l)
	for i := range metrics {
		metrics[i] = ValidatorEpochMetric{
			Missed: opera.BlocksMissed{
				BlocksNum: idx.Block(offlineBlocks[i].Uint64()),
				Period:    inter.FromUnix(offlineTimes[i].Int64()),
			},
			Uptime:          inter.FromUnix(uptimes[i].Int64()),
			OriginatedTxFee: fees[i],
		}
	}
	return metrics, nil
}
