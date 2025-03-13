package drivercall

import (
	_ "embed"
	"math/big"
	"strings"

	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/genesis/gpos"
	"github.com/0xsoniclabs/sonic/utils"
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
