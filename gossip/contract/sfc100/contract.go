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

// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package sfc100

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// ContractMetaData contains all meta data concerning the Contract contract.
var ContractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"}],\"name\":\"AddressEmptyCode\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AlreadyRedirected\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"ERC1967InvalidImplementation\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ERC1967NonPayable\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"FailedCall\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InsufficientSelfStake\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidInitialization\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MalformedPubkey\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotAuthorized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotDeactivatedStatus\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotDriverAuth\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotEnoughEpochsPassed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotEnoughTimePassed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotInitializing\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NothingToStash\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"OwnableInvalidOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"OwnableUnauthorizedAccount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PubkeyUsedByOtherValidator\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Redirected\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"RefundRatioTooHigh\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"RequestExists\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"RequestNotExists\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"SameAddress\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"SameRedirectionAuthorizer\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"StakeIsFullySlashed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"StakeSubscriberFailed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TransferFailed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TransfersNotAllowed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"UUPSUnauthorizedCallContext\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"slot\",\"type\":\"bytes32\"}],\"name\":\"UUPSUnsupportedProxiableUUID\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ValidatorDelegationLimitExceeded\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ValidatorExists\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ValidatorNotActive\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ValidatorNotExists\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ValidatorNotSlashed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ZeroAddress\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ZeroAmount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ZeroRewards\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"AnnouncedRedirection\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"BurntFTM\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"status\",\"type\":\"uint256\"}],\"name\":\"ChangedValidatorStatus\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"toValidatorID\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"rewards\",\"type\":\"uint256\"}],\"name\":\"ClaimedRewards\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"auth\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"createdEpoch\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"createdTime\",\"type\":\"uint256\"}],\"name\":\"CreatedValidator\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"deactivatedEpoch\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"deactivatedTime\",\"type\":\"uint256\"}],\"name\":\"DeactivatedValidator\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"toValidatorID\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Delegated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"version\",\"type\":\"uint64\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"RefundedSlashedLegacyDelegation\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"toValidatorID\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"rewards\",\"type\":\"uint256\"}],\"name\":\"RestakedRewards\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"toValidatorID\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"wrID\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Undelegated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"refundRatio\",\"type\":\"uint256\"}],\"name\":\"UpdatedSlashingRefundRatio\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"Upgraded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"toValidatorID\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"wrID\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Withdrawn\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"UPGRADE_INTERFACE_VERSION\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"syncPubkey\",\"type\":\"bool\"}],\"name\":\"_syncValidator\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"announceRedirection\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"burnFTM\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"toValidatorID\",\"type\":\"uint256\"}],\"name\":\"claimRewards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"constsAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"pubkey\",\"type\":\"bytes\"}],\"name\":\"createValidator\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentEpoch\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentSealedEpoch\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"status\",\"type\":\"uint256\"}],\"name\":\"deactivateValidator\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"toValidatorID\",\"type\":\"uint256\"}],\"name\":\"delegate\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"epoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"}],\"name\":\"getEpochAccumulatedOriginatedTxsFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"epoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"}],\"name\":\"getEpochAccumulatedRewardPerToken\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"epoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"}],\"name\":\"getEpochAccumulatedUptime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"epoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"}],\"name\":\"getEpochAverageUptime\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"epoch\",\"type\":\"uint256\"}],\"name\":\"getEpochEndBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"epoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"}],\"name\":\"getEpochOfflineBlocks\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"epoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"}],\"name\":\"getEpochOfflineTime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"epoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"}],\"name\":\"getEpochReceivedStake\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"epoch\",\"type\":\"uint256\"}],\"name\":\"getEpochSnapshot\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"endTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"endBlock\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"epochFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseRewardPerSecond\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalStake\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"epoch\",\"type\":\"uint256\"}],\"name\":\"getEpochValidatorIDs\",\"outputs\":[{\"internalType\":\"uint256[]\",\"name\":\"\",\"type\":\"uint256[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"}],\"name\":\"getRedirection\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"receiver\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"}],\"name\":\"getRedirectionRequest\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"receiver\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"}],\"name\":\"getSelfStake\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"}],\"name\":\"getStake\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"}],\"name\":\"getValidator\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"status\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"receivedStake\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"auth\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"createdEpoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"createdTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deactivatedTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deactivatedEpoch\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"auth\",\"type\":\"address\"}],\"name\":\"getValidatorID\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"}],\"name\":\"getValidatorPubkey\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"pubkey\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wrID\",\"type\":\"uint256\"}],\"name\":\"getWithdrawalRequest\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"epoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"time\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"sealedEpoch\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"nodeDriver\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_c\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"initiateRedirection\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"}],\"name\":\"isSlashed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"lastValidatorID\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"toValidatorID\",\"type\":\"uint256\"}],\"name\":\"pendingRewards\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proxiableUUID\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"pubkeyAddress\",\"type\":\"address\"}],\"name\":\"pubkeyAddressToValidatorID\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"redirect\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"redirectionAuthorizer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"toValidatorID\",\"type\":\"uint256\"}],\"name\":\"restakeRewards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"}],\"name\":\"rewardsStash\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"offlineTime\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"offlineBlocks\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"uptimes\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"originatedTxsFee\",\"type\":\"uint256[]\"}],\"name\":\"sealEpoch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"nextValidatorIDs\",\"type\":\"uint256[]\"}],\"name\":\"sealEpochValidators\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"toValidatorID\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"}],\"name\":\"setGenesisDelegation\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"auth\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"pubkey\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"createdTime\",\"type\":\"uint256\"}],\"name\":\"setGenesisValidator\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"v\",\"type\":\"address\"}],\"name\":\"setRedirectionAuthorizer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"}],\"name\":\"slashingRefundRatio\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"refundRatio\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"stakeSubscriberAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"toValidatorID\",\"type\":\"uint256\"}],\"name\":\"stashRewards\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"}],\"name\":\"stashedRewardsUntilEpoch\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"epoch\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalActiveStake\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalStake\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"treasuryAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"toValidatorID\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wrID\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"undelegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"v\",\"type\":\"address\"}],\"name\":\"updateConstsAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"refundRatio\",\"type\":\"uint256\"}],\"name\":\"updateSlashingRefundRatio\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"v\",\"type\":\"address\"}],\"name\":\"updateStakeSubscriberAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"v\",\"type\":\"address\"}],\"name\":\"updateTreasuryAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newImplementation\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"upgradeToAndCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"bytes3\",\"name\":\"\",\"type\":\"bytes3\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"toValidatorID\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wrID\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x60a0604052306080523480156012575f5ffd5b50608051614bd76100395f395f81816125a7015281816125d001526129a90152614bd75ff3fe6080604052600436106103b6575f3560e01c80638cddb015116101e9578063c7be95de11610108578063df00c9221161009d578063e9a505a71161006d578063e9a505a714610dc6578063ebdf104c14610de5578063f2fde38b14610e04578063fb36025f14610e23575f5ffd5b8063df00c92214610d14578063e08d7e6614610d4e578063e261641a14610d6d578063e880a15914610da7575f5ffd5b8063d46fa518116100d8578063d46fa51814610c70578063d725e91f14610c8d578063db5ca3e514610cac578063dc31e1af14610cda575f5ffd5b8063c7be95de14610be7578063cc17278414610bfc578063cc8343aa14610c1b578063cfd4766314610c3a575f5ffd5b8063aa5d82921161017e578063b88a37e21161014e578063b88a37e214610b31578063c3de580e14610b5d578063c5f956af14610b9d578063c65ee0e114610bbc575f5ffd5b8063aa5d8292146109e4578063ad3cb1cc14610a3f578063b0ef386c14610a6f578063b5d8962714610a8e575f5ffd5b8063a198d229116101b9578063a198d22914610942578063a5a470ad1461097c578063a86a056f1461098f578063a8ab09ba146109c5575f5ffd5b80638cddb015146108b55780638da5cb5b146108d457806390a6c475146109105780639fa6dd351461092f575f5ffd5b806352d1902d116102d5578063736de9ae1161026a578063841e45611161023a578063841e456114610836578063854873e114610855578063860c2750146108815780638b0e9f3f146108a0575f5ffd5b8063736de9ae146107ba57806376671808146107ee57806376fed43a146108025780637cacb1d614610821575f5ffd5b80636099ecb2116102a55780636099ecb21461070c57806361e53fcc1461072b5780636f49866314610765578063715018a6146107a6575f5ffd5b806352d1902d1461069c57806354fd4d50146106b05780635601fe01146106ce57806358f95b80146106ed575f5ffd5b806339b80c001161034b57806346f1ca351161031b57806346f1ca351461062c5780634f1ef2861461064b5780634f7c4efb1461065e5780634f864df41461067d575f5ffd5b806339b80c001461053c5780633fbfd4df146105ba578063441a3e70146105d9578063468f35ee146105f8575f5ffd5b806318160ddd1161038657806318160ddd1461048d5780631e702f83146104a25780631f270152146104c157806328f7314814610527575f5ffd5b80630135b1db146103d857806308c3687414610416578063093b41d0146104375780630962ef791461046e575f5ffd5b366103d45760405163ab064ad360e01b815260040160405180910390fd5b5f5ffd5b3480156103e3575f5ffd5b506104036103f2366004614299565b60036020525f908152604090205481565b6040519081526020015b60405180910390f35b348015610421575f5ffd5b506104356104303660046142b2565b610e4e565b005b348015610442575f5ffd5b50601154610456906001600160a01b031681565b6040516001600160a01b03909116815260200161040d565b348015610479575f5ffd5b506104356104883660046142b2565b610eb0565b348015610498575f5ffd5b50610403600c5481565b3480156104ad575f5ffd5b506104356104bc3660046142c9565b610f80565b3480156104cc575f5ffd5b5061050c6104db3660046142e9565b600a60209081525f938452604080852082529284528284209052825290208054600182015460029092015490919083565b6040805193845260208401929092529082015260600161040d565b348015610532575f5ffd5b5061040360075481565b348015610547575f5ffd5b5061058d6105563660046142b2565b600d60208190525f918252604090912060088101546009820154600a830154600b840154600c850154949095015492949193909286565b604080519687526020870195909552938501929092526060840152608083015260a082015260c00161040d565b3480156105c5575f5ffd5b506104356105d4366004614319565b611009565b3480156105e4575f5ffd5b506104356105f33660046142c9565b611172565b348015610603575f5ffd5b50610456610612366004614299565b60146020525f90815260409020546001600160a01b031681565b348015610637575f5ffd5b50610435610646366004614299565b61118a565b610435610659366004614380565b6111c2565b348015610669575f5ffd5b506104356106783660046142c9565b6111dd565b348015610688575f5ffd5b50610435610697366004614441565b61128c565b3480156106a7575f5ffd5b506104036113d9565b3480156106bb575f5ffd5b50604051600160fa1b815260200161040d565b3480156106d9575f5ffd5b506104036106e83660046142b2565b6113f4565b3480156106f8575f5ffd5b506104036107073660046142c9565b611426565b348015610717575f5ffd5b5061040361072636600461446a565b611446565b348015610736575f5ffd5b506104036107453660046142c9565b5f918252600d602090815260408084209284526001909201905290205490565b348015610770575f5ffd5b5061040361077f36600461446a565b6001600160a01b03919091165f908152600860209081526040808320938352929052205490565b3480156107b1575f5ffd5b5061043561148b565b3480156107c5575f5ffd5b506104566107d4366004614299565b60156020525f90815260409020546001600160a01b031681565b3480156107f9575f5ffd5b5061040361149e565b34801561080d575f5ffd5b5061043561081c3660046144d6565b6114b3565b34801561082c575f5ffd5b5061040360015481565b348015610841575f5ffd5b50610435610850366004614299565b611505565b348015610860575f5ffd5b5061087461086f3660046142b2565b61152f565b60405161040d9190614560565b34801561088c575f5ffd5b5061043561089b366004614299565b6115c6565b3480156108ab575f5ffd5b5061040360065481565b3480156108c0575f5ffd5b506104356108cf36600461446a565b6115f0565b3480156108df575f5ffd5b507f9016d09d72d40fdae2fd8ceac6b6234c7706214fd39c1cd1e609a0528c199300546001600160a01b0316610456565b34801561091b575f5ffd5b5061043561092a3660046142b2565b611617565b61043561093d3660046142b2565b61162b565b34801561094d575f5ffd5b5061040361095c3660046142c9565b5f918252600d602090815260408084209284526006909201905290205490565b61043561098a366004614572565b611636565b34801561099a575f5ffd5b506104036109a936600461446a565b600960209081525f928352604080842090915290825290205481565b3480156109d0575f5ffd5b506104356109df3660046142e9565b61177d565b3480156109ef575f5ffd5b50610a276109fe3660046142c9565b5f918252600d60209081526040808420928452600390920190529020546001600160401b031690565b6040516001600160401b03909116815260200161040d565b348015610a4a575f5ffd5b50610874604051806040016040528060058152602001640352e302e360dc1b81525081565b348015610a7a575f5ffd5b50610435610a89366004614299565b6117bc565b348015610a99575f5ffd5b50610aee610aa83660046142b2565b600260208190525f918252604090912080546001820154928201546003830154600484015460058501546006909501549395946001600160a01b03909316939192909187565b6040805197885260208801969096526001600160a01b03909416948601949094526060850191909152608084015260a083019190915260c082015260e00161040d565b348015610b3c575f5ffd5b50610b50610b4b3660046142b2565b611814565b60405161040d91906145b0565b348015610b68575f5ffd5b50610b8d610b773660046142b2565b5f90815260026020526040902054608016151590565b604051901515815260200161040d565b348015610ba8575f5ffd5b50600f54610456906001600160a01b031681565b348015610bc7575f5ffd5b50610403610bd63660046142b2565b600e6020525f908152604090205481565b348015610bf2575f5ffd5b5061040360055481565b348015610c07575f5ffd5b50610435610c163660046145f2565b611876565b348015610c26575f5ffd5b50610435610c35366004614623565b61193c565b348015610c45575f5ffd5b50610403610c5436600461446a565b600b60209081525f928352604080842090915290825290205481565b348015610c7b575f5ffd5b506010546001600160a01b0316610456565b348015610c98575f5ffd5b50610435610ca7366004614299565b611a66565b348015610cb7575f5ffd5b50610403610cc63660046142b2565b5f908152600d602052604090206009015490565b348015610ce5575f5ffd5b50610403610cf43660046142c9565b5f918252600d602090815260408084209284526004909201905290205490565b348015610d1f575f5ffd5b50610403610d2e3660046142c9565b5f918252600d602090815260408084209284526002909201905290205490565b348015610d59575f5ffd5b50610435610d68366004614695565b611b0c565b348015610d78575f5ffd5b50610403610d873660046142c9565b5f918252600d602090815260408084209284526005909201905290205490565b348015610db2575f5ffd5b50610435610dc1366004614299565b611bcd565b348015610dd1575f5ffd5b50601354610456906001600160a01b031681565b348015610df0575f5ffd5b50610435610dff3660046146c7565b611bf7565b348015610e0f575f5ffd5b50610435610e1e366004614299565b611e9d565b348015610e2e575f5ffd5b50610403610e3d366004614299565b60126020525f908152604090205481565b335f610e5a8284611edc565b9050610e67828483611f5f565b82826001600160a01b03167f663e0f63f4fc6b01be195c4b56111fd6f14b947d6264497119b08daf77e26da583604051610ea391815260200190565b60405180910390a3505050565b335f610ebc8284611edc565b90505f610ec883611fec565b6001600160a01b0316826040515f6040518083038185875af1925050503d805f8114610f0f576040519150601f19603f3d011682016040523d82523d5f602084013e610f14565b606091505b5050905080610f36576040516312171d8360e31b815260040160405180910390fd5b83836001600160a01b03167f70de20a533702af05c8faf1637846c4586a021bbc71b6928b089b6d555e4fbc284604051610f7291815260200190565b60405180910390a350505050565b5f546001600160a01b03163314610faa5760405163c78d372960e01b815260040160405180910390fd5b80610fc85760405163396bd83560e21b815260040160405180910390fd5b610fd28282612014565b610fdc825f61193c565b5f828152600260208190526040822001546001600160a01b0316906110049082908190612116565b505050565b7ff0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a008054600160401b810460ff1615906001600160401b03165f8115801561104d5750825b90505f826001600160401b031660011480156110685750303b155b905081158015611076575080155b156110945760405163f92ee8a960e01b815260040160405180910390fd5b845467ffffffffffffffff1916600117855583156110be57845460ff60401b1916600160401b1785555b6110c7866121f2565b6110cf612203565b60018a90555f80546001600160a01b03808b166001600160a01b03199283161790925560108054928a1692909116919091179055600c89905561110f4290565b5f8b8152600d6020526040902060080155831561116657845460ff60401b19168555604051600181527fc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d29060200160405180910390a15b50505050505050505050565b61118633838361118133611fec565b61220b565b5050565b6040516001600160a01b0382169033907f857125196131cfcd709c738c6d1fd2701ce70f2a03785aeadae6f4b47fe73c1d905f90a350565b6111ca61259c565b6111d382612640565b6111868282612648565b6111e5612704565b5f82815260026020526040902054608016611213576040516321b6a8f960e11b815260040160405180910390fd5b670de0b6b3a764000081111561123c576040516357c70d6360e01b815260040160405180910390fd5b5f828152600e6020526040908190208290555182907f047575f43f09a7a093d94ec483064acfc61b7e25c0de28017da442abf99cb917906112809084815260200190565b60405180910390a25050565b33611297818561275f565b50815f036112b857604051631f2a200560e01b815260040160405180910390fd5b6001600160a01b0381165f908152600a602090815260408083208784528252808320868452909152902060020154156113045760405163756f5c2d60e11b815260040160405180910390fd5b61131481858460015f60016127cd565b6001600160a01b0381165f908152600a602090815260408083208784528252808320868452909152902060020182905561134c61149e565b6001600160a01b0382165f908152600a60209081526040808320888452825280832087845290915281209182554260019092019190915561138e90859061193c565b8284826001600160a01b03167fd3bb4e423fbea695d16b982f9f682dc5f35152e5411646a8a5a79a6b02ba8d57856040516113cb91815260200190565b60405180910390a450505050565b5f6113e261299e565b505f516020614b825f395f51905f5290565b5f818152600260208181526040808420909201546001600160a01b03168352600b815281832093835292909252205490565b5f828152600d602090815260408083208484529091529020545b92915050565b5f5f61145284846129e7565b6001600160a01b0385165f9081526008602090815260408083208784529091529020549091506114839082906147a5565b949350505050565b611493612704565b61149c5f612a51565b565b5f60015460016114ae91906147a5565b905090565b5f546001600160a01b031633146114dd5760405163c78d372960e01b815260040160405180910390fd5b6114ee858585855f5f875f5f612ac1565b6005548411156114fe5760058490555b5050505050565b61150d612704565b600f80546001600160a01b0319166001600160a01b0392909216919091179055565b60046020525f908152604090208054611547906147b8565b80601f0160208091040260200160405190810160405280929190818152602001828054611573906147b8565b80156115be5780601f10611595576101008083540402835291602001916115be565b820191905f5260205f20905b8154815290600101906020018083116115a157829003601f168201915b505050505081565b6115ce612704565b601080546001600160a01b0319166001600160a01b0392909216919091179055565b6115fa828261275f565b6111865760405163208e0a4160e11b815260040160405180910390fd5b61161f612704565b61162881612c6c565b50565b611628338234611f5f565b60105f9054906101000a90046001600160a01b03166001600160a01b031663c5f530af6040518163ffffffff1660e01b8152600401602060405180830381865afa158015611686573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906116aa91906147f0565b3410156116ca5760405163047447a360e11b815260040160405180910390fd5b604281141580611703575081815f8181106116e7576116e7614807565b9050013560f81c60f81b6001600160f81b03191660c060f81b14155b15611721576040516338497f4960e11b815260040160405180910390fd5b60125f61172e8484612cd3565b6001600160a01b0316815260208101919091526040015f2054156117655760405163028aeb6760e21b815260040160405180910390fd5b611770338383612cff565b6111863360055434611f5f565b5f546001600160a01b031633146117a75760405163c78d372960e01b815260040160405180910390fd5b6117b38383835f612d2d565b61100481612e7b565b6117c4612704565b6013546001600160a01b038083169116036117f257604051639b92bed360e01b815260040160405180910390fd5b601380546001600160a01b0319166001600160a01b0392909216919091179055565b5f818152600d602090815260409182902060070180548351818402810184019094528084526060939283018282801561186a57602002820191905f5260205f20905b815481526020019060010190808311611856575b50505050509050919050565b6013546001600160a01b031633146118a15760405163ea8e4eb560e01b815260040160405180910390fd5b6001600160a01b038281165f908152601560205260409020548183169116036118dd5760405163eb81e1a360e01b815260040160405180910390fd5b806001600160a01b0316826001600160a01b03160361190f5760405163367558c360e01b815260040160405180910390fd5b6001600160a01b039182165f90815260146020526040902080546001600160a01b03191691909216179055565b5f8281526002602052604090206004015461196a57604051635926e0c360e01b815260040160405180910390fd5b5f828152600260205260409020600181015490541561198657505f5b5f5460405163520337df60e11b815260048101859052602481018390526001600160a01b039091169063a4066fbe906044015f604051808303815f87803b1580156119cf575f5ffd5b505af11580156119e1573d5f5f3e3d5ffd5b505050508180156119f157508015155b15611004575f805484825260046020819052604092839020925163242a6e3f60e01b81526001600160a01b039092169263242a6e3f92611a34928892910161481b565b5f604051808303815f87803b158015611a4b575f5ffd5b505af1158015611a5d573d5f5f3e3d5ffd5b50505050505050565b336001600160a01b038216611a8e5760405163d92e233d60e01b815260040160405180910390fd5b6001600160a01b038181165f90815260146020526040902054811690831614611aca57604051630fe3b3c160e31b815260040160405180910390fd5b6001600160a01b039081165f9081526015602090815260408083208054949095166001600160a01b031994851617909455601490529190912080549091169055565b5f546001600160a01b03163314611b365760405163c78d372960e01b815260040160405180910390fd5b5f600d5f611b4261149e565b81526020019081526020015f2090505f5f90505b82811015611bb8575f848483818110611b7157611b71614807565b602090810292909201355f81815260028452604080822060010154948890529020839055600c860154909350611ba9915082906147a5565b600c8501555050600101611b56565b50611bc7600782018484614221565b50505050565b611bd5612704565b601180546001600160a01b0319166001600160a01b0392909216919091179055565b5f546001600160a01b03163314611c215760405163c78d372960e01b815260040160405180910390fd5b5f600d5f611c2d61149e565b81526020019081526020015f2090505f81600701805480602002602001604051908101604052809291908181526020018280548015611c8957602002820191905f5260205f20905b815481526020019060010190808311611c75575b50505050509050611d0e82828c8c808060200260200160405190810160405280939291908181526020018383602002808284375f81840152601f19601f820116905080830192505050505050508b8b808060200260200160405190810160405280939291908181526020018383602002808284375f92019190915250612eed92505050565b600180545f908152600d602052604090206008810154909190421115611d40576008820154611d3d90426148a9565b90505b611dc0818584868c8c808060200260200160405190810160405280939291908181526020018383602002808284375f81840152601f19601f820116905080830192505050505050508b8b808060200260200160405190810160405280939291908181526020018383602002808284375f9201919091525061310792505050565b611dff818584868c8c808060200260200160405190810160405280939291908181526020018383602002808284375f9201919091525061382592505050565b5050611e0961149e565b6001554260088301554360098301556010546040805163d9a7c1f960e01b815290516001600160a01b039092169163d9a7c1f9916004808201926020929091908290030181865afa158015611e60573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190611e8491906147f0565b600b83015550600c54600d909101555050505050505050565b611ea5612704565b6001600160a01b038116611ed357604051631e4fbdf760e01b81525f60048201526024015b60405180910390fd5b61162881612a51565b5f611ee7838361275f565b506001600160a01b0383165f90815260086020908152604080832085845290915281205490819003611f2c5760405163899aaa9d60e01b815260040160405180910390fd5b6001600160a01b0384165f908152600860209081526040808320868452909152812055611f5881612e7b565b9392505050565b5f82815260026020526040902060040154611f8d57604051635926e0c360e01b815260040160405180910390fd5b5f8281526002602052604090205415611fb9576040516353670afb60e11b815260040160405180910390fd5b611fc68383836001612d2d565b611fcf82613aa3565b6110045760405163c2eb4ead60e01b815260040160405180910390fd5b6001600160a01b038082165f9081526015602052604081205490911680611440575090919050565b5f8281526002602052604090205415801561202e57508015155b15612055575f8281526002602052604090206001015460075461205191906148a9565b6007555b5f82815260026020526040902054811115611186575f8281526002602052604081208281556006015490036120e45761208c61149e565b5f8381526002602090815260409182902060068101849055426005909101819055825193845290830152805184927fac4801c32a6067ff757446524ee4e7a373797278ac3c883eac5c693b4ad72e4792908290030190a25b817fcd35267e7654194727477d6c78b541a553483cff7f92a055d17868d3da6e953e8260405161128091815260200190565b6011546001600160a01b031615611004576011546040516001600160a01b03858116602483015284811660448301525f921690627a12009060640160408051601f198184030181529181526020820180516001600160e01b0316631fbcb08360e11b1790525161218691906148bc565b5f604051808303815f8787f1925050503d805f81146121c0576040519150601f19603f3d011682016040523d82523d5f602084013e6121c5565b606091505b50509050801580156121d45750815b15611bc7576040516347b4be6960e11b815260040160405180910390fd5b6121fa613b5a565b61162881613ba3565b61149c613b5a565b6001600160a01b0384165f908152600a60209081526040808320868452825280832085845282528083208151606081018352815480825260018301549482019490945260029091015491810191909152910361227a57604051630fe3b3c160e31b815260040160405180910390fd5b60208082015182515f8781526002909352604090922060050154909190158015906122b457505f8681526002602052604090206005015482115b156122d45750505f84815260026020526040902060058101546006909101545b60105f9054906101000a90046001600160a01b03166001600160a01b031663b82b84276040518163ffffffff1660e01b8152600401602060405180830381865afa158015612324573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061234891906147f0565b61235290836147a5565b42101561237257604051635ada9a9960e01b815260040160405180910390fd5b60105f9054906101000a90046001600160a01b03166001600160a01b031663650acd666040518163ffffffff1660e01b8152600401602060405180830381865afa1580156123c2573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906123e691906147f0565b6123f090826147a5565b6123f861149e565b1015612417576040516323ea994d60e01b815260040160405180910390fd5b6001600160a01b0387165f908152600a60209081526040808320898452825280832088845282528083206002908101548a855290835281842054600e9093529083205490926080909216151591906124729084908490613bab565b6001600160a01b038b165f908152600a602090815260408083208d845282528083208c84529091528120818155600181018290556002015590508083116124cc576040516318f967fb60e01b815260040160405180910390fd5b5f6001600160a01b0388166124e183866148a9565b6040515f81818185875af1925050503d805f811461251a576040519150601f19603f3d011682016040523d82523d5f602084013e61251f565b606091505b5050905080612541576040516312171d8360e31b815260040160405180910390fd5b61254a82612c6c565b888a8c6001600160a01b03167f75e161b3e824b114fc1a33274bd7091918dd4e639cede50b78b15a4eea956a218760405161258791815260200190565b60405180910390a45050505050505050505050565b306001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016148061262257507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03166126165f516020614b825f395f51905f52546001600160a01b031690565b6001600160a01b031614155b1561149c5760405163703e46dd60e11b815260040160405180910390fd5b611628612704565b816001600160a01b03166352d1902d6040518163ffffffff1660e01b8152600401602060405180830381865afa9250505080156126a2575060408051601f3d908101601f1916820190925261269f918101906147f0565b60015b6126ca57604051634c9c8ce360e01b81526001600160a01b0383166004820152602401611eca565b5f516020614b825f395f51905f5281146126fa57604051632a87526960e21b815260048101829052602401611eca565b6110048383613c11565b336127367f9016d09d72d40fdae2fd8ceac6b6234c7706214fd39c1cd1e609a0528c199300546001600160a01b031690565b6001600160a01b03161461149c5760405163118cdaa760e01b8152336004820152602401611eca565b5f5f61276b84846129e7565b905061277683613c66565b6001600160a01b0385165f8181526009602090815260408083208884528252808320949094559181526008825282812086825290915290812080548392906127bf9084906147a5565b909155505015159392505050565b6001600160a01b0386165f908152600b60209081526040808320888452909152812080548692906127ff9084906148a9565b90915550505f858152600260205260409020600101546128209085906148a9565b5f8681526002602052604090206001015560065461283f9085906148a9565b6006555f85815260026020526040902054612866578360075461286291906148a9565b6007555b5f612870866113f4565b9050801580159061288c57505f86815260026020526040902054155b1561296c5760105f9054906101000a90046001600160a01b03166001600160a01b031663c5f530af6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156128e1573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061290591906147f0565b81101561293657821561292b5760405163047447a360e11b815260040160405180910390fd5b612936866001612014565b818015612949575061294786613aa3565b155b156129675760405163c2eb4ead60e01b815260040160405180910390fd5b612977565b612977866001612014565b5f8681526002602081905260409091200154611a5d9088906001600160a01b031686612116565b306001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000161461149c5760405163703e46dd60e11b815260040160405180910390fd5b6001600160a01b0382165f90815260096020908152604080832084845290915281205481612a1484613c66565b6001600160a01b0386165f908152600b60209081526040808320888452909152812054919250612a4682878686613cbb565b979650505050505050565b7f9016d09d72d40fdae2fd8ceac6b6234c7706214fd39c1cd1e609a0528c19930080546001600160a01b031981166001600160a01b03848116918217845560405192169182907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0905f90a3505050565b6001600160a01b0389165f9081526003602052604090205415612af757604051633f4dc7d360e11b815260040160405180910390fd5b6001600160a01b0389165f8181526003602081815260408084208d90558c845260028083528185208b81559384018a905560048085018a90556005850188905560068501899055930180546001600160a01b0319169095179094555220612b5f878983614916565b508760125f612b6e8a8a612cd3565b6001600160a01b03166001600160a01b031681526020019081526020015f2081905550886001600160a01b0316887f49bca1ed2666922f9f1690c26a569e1299c2a715fe57647d77e81adfabbf25bf8686604051612bd6929190918252602082015260400190565b60405180910390a38115612c2057604080518381526020810183905289917fac4801c32a6067ff757446524ee4e7a373797278ac3c883eac5c693b4ad72e47910160405180910390a25b8415612c6157877fcd35267e7654194727477d6c78b541a553483cff7f92a055d17868d3da6e953e86604051612c5891815260200190565b60405180910390a25b505050505050505050565b8015611628576040515f9082156108fc0290839083818181858288f19350505050158015612c9c573d5f5f3e3d5ffd5b506040518181527f8918bd6046d08b314e457977f29562c5d76a7030d79b1edba66e8a5da0b77ae89060200160405180910390a150565b5f612ce182600281866149cf565b604051612cef9291906149f6565b6040519081900390209392505050565b5f60055f8154612d0e90614a05565b91829055509050611bc7848285855f612d2561149e565b425f5f612ac1565b815f03612d4d57604051631f2a200560e01b815260040160405180910390fd5b612d57848461275f565b506001600160a01b0384165f908152600b60209081526040808320868452909152902054612d869083906147a5565b6001600160a01b0385165f908152600b60209081526040808320878452825280832093909355600290522060010154612dbf83826147a5565b5f85815260026020526040902060010155600654612dde9084906147a5565b6006555f84815260026020526040902054612e055782600754612e0191906147a5565b6007555b612e1084821561193c565b83856001600160a01b03167f9a8f44850296624dadfd9c246d17e47171d35727a181bd090aa14bbbe00238bb85604051612e4c91815260200190565b60405180910390a35f84815260026020819052604090912001546114fe9086906001600160a01b031684612116565b5f546040516366e7ea0f60e01b8152306004820152602481018390526001600160a01b03909116906366e7ea0f906044015f604051808303815f87803b158015612ec3575f5ffd5b505af1158015612ed5573d5f5f3e3d5ffd5b5050505080600c54612ee791906147a5565b600c5550565b5f5b83518110156114fe5760105f9054906101000a90046001600160a01b03166001600160a01b0316635a68f01a6040518163ffffffff1660e01b8152600401602060405180830381865afa158015612f48573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190612f6c91906147f0565b828281518110612f7e57612f7e614807565b602002602001015111801561301e575060105f9054906101000a90046001600160a01b03166001600160a01b031662cc7f836040518163ffffffff1660e01b8152600401602060405180830381865afa158015612fdd573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061300191906147f0565b83828151811061301357613013614807565b602002602001015110155b1561306a5761304784828151811061303857613038614807565b60200260200101516008612014565b61306a84828151811061305c5761305c614807565b60200260200101515f61193c565b82818151811061307c5761307c614807565b6020026020010151856005015f86848151811061309b5761309b614807565b602002602001015181526020019081526020015f20819055508181815181106130c6576130c6614807565b6020026020010151856006015f8684815181106130e5576130e5614807565b60209081029190910181015182528101919091526040015f2055600101612eef565b5f6040518060a0016040528085516001600160401b0381111561312c5761312c61436c565b604051908082528060200260200182016040528015613155578160200160208202803683370190505b5081526020015f815260200185516001600160401b0381111561317a5761317a61436c565b6040519080825280602002602001820160405280156131a3578160200160208202803683370190505b5081526020015f81526020015f81525090505f5f90505b84518110156132e3575f866004015f8784815181106131db576131db614807565b602002602001015181526020019081526020015f205490505f5f90508185848151811061320a5761320a614807565b60200260200101511115613240578185848151811061322b5761322b614807565b602002602001015161323d91906148a9565b90505b8986848151811061325357613253614807565b6020026020010151826132669190614a1d565b6132709190614a48565b8460400151848151811061328657613286614807565b602002602001018181525050836040015183815181106132a8576132a8614807565b602002602001015184606001516132bf91906147a5565b606085015260808401516132d49082906147a5565b608085015250506001016131ba565b505f5b84518110156133d0578784828151811061330257613302614807565b60200260200101518986848151811061331d5761331d614807565b60200260200101518a5f015f8a878151811061333b5761333b614807565b602002602001015181526020019081526020015f205461335b9190614a1d565b6133659190614a48565b61336f9190614a1d565b6133799190614a48565b825180518390811061338d5761338d614807565b602090810291909101015281518051829081106133ac576133ac614807565b602002602001015182602001516133c391906147a5565b60208301526001016132e6565b505f5b84518110156136d4575f61347b8960105f9054906101000a90046001600160a01b03166001600160a01b031663d9a7c1f96040518163ffffffff1660e01b8152600401602060405180830381865afa158015613431573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061345591906147f0565b855180518690811061346957613469614807565b60200260200101518660200151613d24565b90506134ad83608001518460400151848151811061349b5761349b614807565b60200260200101518560600151613d5f565b6134b790826147a5565b90505f8683815181106134cc576134cc614807565b6020908102919091018101515f8181526002808452604080832090910154601054825163a778651560e01b815292519496506001600160a01b03918216959394613564948994929093169263a778651592600480820193918290030181865afa15801561353b573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061355f91906147f0565b613eb0565b6001600160a01b0383165f908152600b6020908152604080832087845290915290205490915080156135c8576001600160a01b0383165f908152600860209081526040808320878452909152812080548492906135c29084906147a5565b90915550505b5f6135d383876148a9565b5f86815260026020526040812060010154919250811561360d5781613600670de0b6b3a764000085614a1d565b61360a9190614a48565b90505b5f87815260018f0160205260409020546136289082906147a5565b8f6001015f8981526020019081526020015f20819055508a898151811061365157613651614807565b60200260200101518f6004015f8981526020019081526020015f20819055508b898151811061368257613682614807565b60200260200101518e6002015f8981526020019081526020015f20546136a891906147a5565b8f6002015f8981526020019081526020015f2081905550505050505050505080806001019150506133d3565b506080810151600a8701819055600c54111561370a5785600a0154600c5f8282546136ff91906148a9565b9091555061370f9050565b5f600c555b600f546001600160a01b031615611a5d575f670de0b6b3a764000060105f9054906101000a90046001600160a01b03166001600160a01b03166394c3e9146040518163ffffffff1660e01b8152600401602060405180830381865afa15801561377a573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061379e91906147f0565b83608001516137ad9190614a1d565b6137b79190614a48565b90506137c281612e7b565b600f546040515f916001600160a01b031690620f424090849084818181858888f193505050503d805f8114613812576040519150601f19603f3d011682016040523d82523d5f602084013e613817565b606091505b505050505050505050505050565b5f5b8251811015613a9b575f83828151811061384357613843614807565b602002602001015190505f8761385e670de0b6b3a764000090565b85858151811061387057613870614807565b60200260200101516138829190614a1d565b61388c9190614a48565b9050670de0b6b3a76400008111156138a95750670de0b6b3a76400005b5f8281526003870160209081526040808320815160608101835290546001600160401b038116825263ffffffff600160401b8204811694830194909452600160601b900490921690820152906138ff8383613ece565b5f85815260038b0160209081526040918290208351815485840151868601516001600160401b039093166bffffffffffffffffffffffff1990921691909117600160401b63ffffffff928316021763ffffffff60601b1916600160601b91909216021790556010548251631c25433760e01b815292519394506001600160a01b031692631c2543379260048082019392918290030181865afa1580156139a7573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906139cb9190614a5b565b6001600160401b0316815f01516001600160401b0316108015613a71575060105f9054906101000a90046001600160a01b03166001600160a01b0316633fa225486040518163ffffffff1660e01b8152600401602060405180830381865afa158015613a39573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190613a5d9190614a81565b63ffffffff16816040015163ffffffff1610155b15613a8b57613a81846010612014565b613a8b845f61193c565b5050600190920191506138279050565b505050505050565b5f670de0b6b3a764000060105f9054906101000a90046001600160a01b03166001600160a01b0316632265f2846040518163ffffffff1660e01b8152600401602060405180830381865afa158015613afd573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190613b2191906147f0565b613b2a846113f4565b613b349190614a1d565b613b3e9190614a48565b5f92835260026020526040909220600101549190911115919050565b7ff0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a0054600160401b900460ff1661149c57604051631afcd79f60e31b815260040160405180910390fd5b611ea5613b5a565b5f821580613bc15750670de0b6b3a76400008210155b15613bcd57505f611f58565b670de0b6b3a7640000613be083826148a9565b613bea9086614a1d565b613bf49190614a48565b613bff9060016147a5565b905083811115611f5857509192915050565b613c1a826140b1565b6040516001600160a01b038316907fbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b905f90a2805115613c5e576110048282614114565b61118661417d565b5f8181526002602052604081206006015415613cb3575f828152600260205260409020600601546001541015613c9e57505060015490565b505f9081526002602052604090206006015490565b505060015490565b5f818310613cca57505f611483565b5f838152600d6020818152604080842088855260019081018352818520548786529383528185208986520190915290912054670de0b6b3a764000087613d1084846148a9565b613d1a9190614a1d565b612a469190614a48565b5f825f03613d3357505f611483565b5f613d3e8587614a1d565b905082613d4b8583614a1d565b613d559190614a48565b9695505050505050565b5f825f03613d6e57505f611f58565b5f82613d7a8587614a1d565b613d849190614a48565b9050670de0b6b3a764000060105f9054906101000a90046001600160a01b03166001600160a01b03166394c3e9146040518163ffffffff1660e01b8152600401602060405180830381865afa158015613ddf573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190613e0391906147f0565b60105f9054906101000a90046001600160a01b03166001600160a01b031663c74dd6216040518163ffffffff1660e01b8152600401602060405180830381865afa158015613e53573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190613e7791906147f0565b613e8990670de0b6b3a76400006148a9565b613e9391906148a9565b613e9d9083614a1d565b613ea79190614a48565b95945050505050565b5f670de0b6b3a7640000613ec48385614a1d565b611f589190614a48565b604080516060810182525f8082526020820181905291810191909152604080516060810182525f8082526020820181905291810191909152826040015163ffffffff165f03613f31576001600160401b0384168152600160408201529050611440565b5f83604001516001613f439190614aa4565b63ffffffff1690505f846020015163ffffffff16866001600160401b0316865f01516001600160401b0316600185613f7b9190614ac0565b613f859190614adf565b613f8f9190614b08565b613f999190614b08565b9050613fa58282614b27565b6001600160401b03168352613fba8282614b54565b63ffffffff166020840152670de0b6b3a764000083516001600160401b03161115613feb57670de0b6b3a764000083525b60105f9054906101000a90046001600160a01b03166001600160a01b0316633fa225486040518163ffffffff1660e01b8152600401602060405180830381865afa15801561403b573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061405f9190614a81565b63ffffffff16856040015163ffffffff161015614096576040850151614086906001614aa4565b63ffffffff1660408401526140a7565b60408086015163ffffffff16908401525b5090949350505050565b806001600160a01b03163b5f036140e657604051634c9c8ce360e01b81526001600160a01b0382166004820152602401611eca565b5f516020614b825f395f51905f5280546001600160a01b0319166001600160a01b0392909216919091179055565b60605f5f846001600160a01b03168460405161413091906148bc565b5f60405180830381855af49150503d805f8114614168576040519150601f19603f3d011682016040523d82523d5f602084013e61416d565b606091505b5091509150613ea785838361419c565b341561149c5760405163b398979f60e01b815260040160405180910390fd5b6060826141b1576141ac826141f8565b611f58565b81511580156141c857506001600160a01b0384163b155b156141f157604051639996b31560e01b81526001600160a01b0385166004820152602401611eca565b5080611f58565b8051156142085780518082602001fd5b60405163d6bda27560e01b815260040160405180910390fd5b828054828255905f5260205f2090810192821561425a579160200282015b8281111561425a57823582559160200191906001019061423f565b5061426692915061426a565b5090565b5b80821115614266575f815560010161426b565b80356001600160a01b0381168114614294575f5ffd5b919050565b5f602082840312156142a9575f5ffd5b611f588261427e565b5f602082840312156142c2575f5ffd5b5035919050565b5f5f604083850312156142da575f5ffd5b50508035926020909101359150565b5f5f5f606084860312156142fb575f5ffd5b6143048461427e565b95602085013595506040909401359392505050565b5f5f5f5f5f60a0868803121561432d575f5ffd5b85359450602086013593506143446040870161427e565b92506143526060870161427e565b91506143606080870161427e565b90509295509295909350565b634e487b7160e01b5f52604160045260245ffd5b5f5f60408385031215614391575f5ffd5b61439a8361427e565b915060208301356001600160401b038111156143b4575f5ffd5b8301601f810185136143c4575f5ffd5b80356001600160401b038111156143dd576143dd61436c565b604051601f8201601f19908116603f011681016001600160401b038111828210171561440b5761440b61436c565b604052818152828201602001871015614422575f5ffd5b816020840160208301375f602083830101528093505050509250929050565b5f5f5f60608486031215614453575f5ffd5b505081359360208301359350604090920135919050565b5f5f6040838503121561447b575f5ffd5b6144848361427e565b946020939093013593505050565b5f5f83601f8401126144a2575f5ffd5b5081356001600160401b038111156144b8575f5ffd5b6020830191508360208285010111156144cf575f5ffd5b9250929050565b5f5f5f5f5f608086880312156144ea575f5ffd5b6144f38661427e565b94506020860135935060408601356001600160401b03811115614514575f5ffd5b61452088828901614492565b96999598509660600135949350505050565b5f81518084528060208401602086015e5f602082860101526020601f19601f83011685010191505092915050565b602081525f611f586020830184614532565b5f5f60208385031215614583575f5ffd5b82356001600160401b03811115614598575f5ffd5b6145a485828601614492565b90969095509350505050565b602080825282518282018190525f918401906040840190835b818110156145e75783518352602093840193909201916001016145c9565b509095945050505050565b5f5f60408385031215614603575f5ffd5b61460c8361427e565b915061461a6020840161427e565b90509250929050565b5f5f60408385031215614634575f5ffd5b823591506020830135801515811461464a575f5ffd5b809150509250929050565b5f5f83601f840112614665575f5ffd5b5081356001600160401b0381111561467b575f5ffd5b6020830191508360208260051b85010111156144cf575f5ffd5b5f5f602083850312156146a6575f5ffd5b82356001600160401b038111156146bb575f5ffd5b6145a485828601614655565b5f5f5f5f5f5f5f5f6080898b0312156146de575f5ffd5b88356001600160401b038111156146f3575f5ffd5b6146ff8b828c01614655565b90995097505060208901356001600160401b0381111561471d575f5ffd5b6147298b828c01614655565b90975095505060408901356001600160401b03811115614747575f5ffd5b6147538b828c01614655565b90955093505060608901356001600160401b03811115614771575f5ffd5b61477d8b828c01614655565b999c989b5096995094979396929594505050565b634e487b7160e01b5f52601160045260245ffd5b8082018082111561144057611440614791565b600181811c908216806147cc57607f821691505b6020821081036147ea57634e487b7160e01b5f52602260045260245ffd5b50919050565b5f60208284031215614800575f5ffd5b5051919050565b634e487b7160e01b5f52603260045260245ffd5b828152604060208201525f5f8354614832816147b8565b806040860152600182165f8114614850576001811461486c5761489d565b60ff1983166060870152606082151560051b870101935061489d565b865f5260205f205f5b8381101561489457815488820160600152600190910190602001614875565b87016060019450505b50919695505050505050565b8181038181111561144057611440614791565b5f82518060208501845e5f920191825250919050565b601f82111561100457805f5260205f20601f840160051c810160208510156148f75750805b601f840160051c820191505b818110156114fe575f8155600101614903565b6001600160401b0383111561492d5761492d61436c565b6149418361493b83546147b8565b836148d2565b5f601f841160018114614972575f851561495b5750838201355b5f19600387901b1c1916600186901b1783556114fe565b5f83815260208120601f198716915b828110156149a15786850135825560209485019460019092019101614981565b50868210156149bd575f1960f88860031b161c19848701351681555b505060018560011b0183555050505050565b5f5f858511156149dd575f5ffd5b838611156149e9575f5ffd5b5050820193919092039150565b818382375f9101908152919050565b5f60018201614a1657614a16614791565b5060010190565b808202811582820484141761144057611440614791565b634e487b7160e01b5f52601260045260245ffd5b5f82614a5657614a56614a34565b500490565b5f60208284031215614a6b575f5ffd5b81516001600160401b0381168114611f58575f5ffd5b5f60208284031215614a91575f5ffd5b815163ffffffff81168114611f58575f5ffd5b63ffffffff818116838216019081111561144057611440614791565b6001600160801b03828116828216039081111561144057611440614791565b6001600160801b038181168382160290811690818114614b0157614b01614791565b5092915050565b6001600160801b03818116838216019081111561144057611440614791565b5f6001600160801b03831680614b3f57614b3f614a34565b806001600160801b0384160491505092915050565b5f6001600160801b03831680614b6c57614b6c614a34565b806001600160801b038416069150509291505056fe360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbca2646970667358221220b85ce566241cb74ec11834f70798de9be5c9b0494cacd0dd034911314e5aff8364736f6c634300081b0033",
}

// ContractABI is the input ABI used to generate the binding from.
// Deprecated: Use ContractMetaData.ABI instead.
var ContractABI = ContractMetaData.ABI

// ContractBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ContractMetaData.Bin instead.
var ContractBin = ContractMetaData.Bin

// DeployContract deploys a new Ethereum contract, binding an instance of Contract to it.
func DeployContract(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Contract, error) {
	parsed, err := ContractMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ContractBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Contract{ContractCaller: ContractCaller{contract: contract}, ContractTransactor: ContractTransactor{contract: contract}, ContractFilterer: ContractFilterer{contract: contract}}, nil
}

// Contract is an auto generated Go binding around an Ethereum contract.
type Contract struct {
	ContractCaller     // Read-only binding to the contract
	ContractTransactor // Write-only binding to the contract
	ContractFilterer   // Log filterer for contract events
}

// ContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type ContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ContractSession struct {
	Contract     *Contract         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ContractCallerSession struct {
	Contract *ContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// ContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ContractTransactorSession struct {
	Contract     *ContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type ContractRaw struct {
	Contract *Contract // Generic contract binding to access the raw methods on
}

// ContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ContractCallerRaw struct {
	Contract *ContractCaller // Generic read-only contract binding to access the raw methods on
}

// ContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ContractTransactorRaw struct {
	Contract *ContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewContract creates a new instance of Contract, bound to a specific deployed contract.
func NewContract(address common.Address, backend bind.ContractBackend) (*Contract, error) {
	contract, err := bindContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Contract{ContractCaller: ContractCaller{contract: contract}, ContractTransactor: ContractTransactor{contract: contract}, ContractFilterer: ContractFilterer{contract: contract}}, nil
}

// NewContractCaller creates a new read-only instance of Contract, bound to a specific deployed contract.
func NewContractCaller(address common.Address, caller bind.ContractCaller) (*ContractCaller, error) {
	contract, err := bindContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ContractCaller{contract: contract}, nil
}

// NewContractTransactor creates a new write-only instance of Contract, bound to a specific deployed contract.
func NewContractTransactor(address common.Address, transactor bind.ContractTransactor) (*ContractTransactor, error) {
	contract, err := bindContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ContractTransactor{contract: contract}, nil
}

// NewContractFilterer creates a new log filterer instance of Contract, bound to a specific deployed contract.
func NewContractFilterer(address common.Address, filterer bind.ContractFilterer) (*ContractFilterer, error) {
	contract, err := bindContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ContractFilterer{contract: contract}, nil
}

// bindContract binds a generic wrapper to an already deployed contract.
func bindContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ContractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Contract *ContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Contract.Contract.ContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Contract *ContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contract.Contract.ContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Contract *ContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Contract.Contract.ContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Contract *ContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Contract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Contract *ContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Contract *ContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Contract.Contract.contract.Transact(opts, method, params...)
}

// UPGRADEINTERFACEVERSION is a free data retrieval call binding the contract method 0xad3cb1cc.
//
// Solidity: function UPGRADE_INTERFACE_VERSION() view returns(string)
func (_Contract *ContractCaller) UPGRADEINTERFACEVERSION(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "UPGRADE_INTERFACE_VERSION")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// UPGRADEINTERFACEVERSION is a free data retrieval call binding the contract method 0xad3cb1cc.
//
// Solidity: function UPGRADE_INTERFACE_VERSION() view returns(string)
func (_Contract *ContractSession) UPGRADEINTERFACEVERSION() (string, error) {
	return _Contract.Contract.UPGRADEINTERFACEVERSION(&_Contract.CallOpts)
}

// UPGRADEINTERFACEVERSION is a free data retrieval call binding the contract method 0xad3cb1cc.
//
// Solidity: function UPGRADE_INTERFACE_VERSION() view returns(string)
func (_Contract *ContractCallerSession) UPGRADEINTERFACEVERSION() (string, error) {
	return _Contract.Contract.UPGRADEINTERFACEVERSION(&_Contract.CallOpts)
}

// ConstsAddress is a free data retrieval call binding the contract method 0xd46fa518.
//
// Solidity: function constsAddress() view returns(address)
func (_Contract *ContractCaller) ConstsAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "constsAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ConstsAddress is a free data retrieval call binding the contract method 0xd46fa518.
//
// Solidity: function constsAddress() view returns(address)
func (_Contract *ContractSession) ConstsAddress() (common.Address, error) {
	return _Contract.Contract.ConstsAddress(&_Contract.CallOpts)
}

// ConstsAddress is a free data retrieval call binding the contract method 0xd46fa518.
//
// Solidity: function constsAddress() view returns(address)
func (_Contract *ContractCallerSession) ConstsAddress() (common.Address, error) {
	return _Contract.Contract.ConstsAddress(&_Contract.CallOpts)
}

// CurrentEpoch is a free data retrieval call binding the contract method 0x76671808.
//
// Solidity: function currentEpoch() view returns(uint256)
func (_Contract *ContractCaller) CurrentEpoch(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "currentEpoch")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CurrentEpoch is a free data retrieval call binding the contract method 0x76671808.
//
// Solidity: function currentEpoch() view returns(uint256)
func (_Contract *ContractSession) CurrentEpoch() (*big.Int, error) {
	return _Contract.Contract.CurrentEpoch(&_Contract.CallOpts)
}

// CurrentEpoch is a free data retrieval call binding the contract method 0x76671808.
//
// Solidity: function currentEpoch() view returns(uint256)
func (_Contract *ContractCallerSession) CurrentEpoch() (*big.Int, error) {
	return _Contract.Contract.CurrentEpoch(&_Contract.CallOpts)
}

// CurrentSealedEpoch is a free data retrieval call binding the contract method 0x7cacb1d6.
//
// Solidity: function currentSealedEpoch() view returns(uint256)
func (_Contract *ContractCaller) CurrentSealedEpoch(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "currentSealedEpoch")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CurrentSealedEpoch is a free data retrieval call binding the contract method 0x7cacb1d6.
//
// Solidity: function currentSealedEpoch() view returns(uint256)
func (_Contract *ContractSession) CurrentSealedEpoch() (*big.Int, error) {
	return _Contract.Contract.CurrentSealedEpoch(&_Contract.CallOpts)
}

// CurrentSealedEpoch is a free data retrieval call binding the contract method 0x7cacb1d6.
//
// Solidity: function currentSealedEpoch() view returns(uint256)
func (_Contract *ContractCallerSession) CurrentSealedEpoch() (*big.Int, error) {
	return _Contract.Contract.CurrentSealedEpoch(&_Contract.CallOpts)
}

// GetEpochAccumulatedOriginatedTxsFee is a free data retrieval call binding the contract method 0xdc31e1af.
//
// Solidity: function getEpochAccumulatedOriginatedTxsFee(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractCaller) GetEpochAccumulatedOriginatedTxsFee(opts *bind.CallOpts, epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getEpochAccumulatedOriginatedTxsFee", epoch, validatorID)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetEpochAccumulatedOriginatedTxsFee is a free data retrieval call binding the contract method 0xdc31e1af.
//
// Solidity: function getEpochAccumulatedOriginatedTxsFee(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractSession) GetEpochAccumulatedOriginatedTxsFee(epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetEpochAccumulatedOriginatedTxsFee(&_Contract.CallOpts, epoch, validatorID)
}

// GetEpochAccumulatedOriginatedTxsFee is a free data retrieval call binding the contract method 0xdc31e1af.
//
// Solidity: function getEpochAccumulatedOriginatedTxsFee(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractCallerSession) GetEpochAccumulatedOriginatedTxsFee(epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetEpochAccumulatedOriginatedTxsFee(&_Contract.CallOpts, epoch, validatorID)
}

// GetEpochAccumulatedRewardPerToken is a free data retrieval call binding the contract method 0x61e53fcc.
//
// Solidity: function getEpochAccumulatedRewardPerToken(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractCaller) GetEpochAccumulatedRewardPerToken(opts *bind.CallOpts, epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getEpochAccumulatedRewardPerToken", epoch, validatorID)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetEpochAccumulatedRewardPerToken is a free data retrieval call binding the contract method 0x61e53fcc.
//
// Solidity: function getEpochAccumulatedRewardPerToken(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractSession) GetEpochAccumulatedRewardPerToken(epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetEpochAccumulatedRewardPerToken(&_Contract.CallOpts, epoch, validatorID)
}

// GetEpochAccumulatedRewardPerToken is a free data retrieval call binding the contract method 0x61e53fcc.
//
// Solidity: function getEpochAccumulatedRewardPerToken(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractCallerSession) GetEpochAccumulatedRewardPerToken(epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetEpochAccumulatedRewardPerToken(&_Contract.CallOpts, epoch, validatorID)
}

// GetEpochAccumulatedUptime is a free data retrieval call binding the contract method 0xdf00c922.
//
// Solidity: function getEpochAccumulatedUptime(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractCaller) GetEpochAccumulatedUptime(opts *bind.CallOpts, epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getEpochAccumulatedUptime", epoch, validatorID)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetEpochAccumulatedUptime is a free data retrieval call binding the contract method 0xdf00c922.
//
// Solidity: function getEpochAccumulatedUptime(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractSession) GetEpochAccumulatedUptime(epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetEpochAccumulatedUptime(&_Contract.CallOpts, epoch, validatorID)
}

// GetEpochAccumulatedUptime is a free data retrieval call binding the contract method 0xdf00c922.
//
// Solidity: function getEpochAccumulatedUptime(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractCallerSession) GetEpochAccumulatedUptime(epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetEpochAccumulatedUptime(&_Contract.CallOpts, epoch, validatorID)
}

// GetEpochAverageUptime is a free data retrieval call binding the contract method 0xaa5d8292.
//
// Solidity: function getEpochAverageUptime(uint256 epoch, uint256 validatorID) view returns(uint64)
func (_Contract *ContractCaller) GetEpochAverageUptime(opts *bind.CallOpts, epoch *big.Int, validatorID *big.Int) (uint64, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getEpochAverageUptime", epoch, validatorID)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetEpochAverageUptime is a free data retrieval call binding the contract method 0xaa5d8292.
//
// Solidity: function getEpochAverageUptime(uint256 epoch, uint256 validatorID) view returns(uint64)
func (_Contract *ContractSession) GetEpochAverageUptime(epoch *big.Int, validatorID *big.Int) (uint64, error) {
	return _Contract.Contract.GetEpochAverageUptime(&_Contract.CallOpts, epoch, validatorID)
}

// GetEpochAverageUptime is a free data retrieval call binding the contract method 0xaa5d8292.
//
// Solidity: function getEpochAverageUptime(uint256 epoch, uint256 validatorID) view returns(uint64)
func (_Contract *ContractCallerSession) GetEpochAverageUptime(epoch *big.Int, validatorID *big.Int) (uint64, error) {
	return _Contract.Contract.GetEpochAverageUptime(&_Contract.CallOpts, epoch, validatorID)
}

// GetEpochEndBlock is a free data retrieval call binding the contract method 0xdb5ca3e5.
//
// Solidity: function getEpochEndBlock(uint256 epoch) view returns(uint256)
func (_Contract *ContractCaller) GetEpochEndBlock(opts *bind.CallOpts, epoch *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getEpochEndBlock", epoch)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetEpochEndBlock is a free data retrieval call binding the contract method 0xdb5ca3e5.
//
// Solidity: function getEpochEndBlock(uint256 epoch) view returns(uint256)
func (_Contract *ContractSession) GetEpochEndBlock(epoch *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetEpochEndBlock(&_Contract.CallOpts, epoch)
}

// GetEpochEndBlock is a free data retrieval call binding the contract method 0xdb5ca3e5.
//
// Solidity: function getEpochEndBlock(uint256 epoch) view returns(uint256)
func (_Contract *ContractCallerSession) GetEpochEndBlock(epoch *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetEpochEndBlock(&_Contract.CallOpts, epoch)
}

// GetEpochOfflineBlocks is a free data retrieval call binding the contract method 0xa198d229.
//
// Solidity: function getEpochOfflineBlocks(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractCaller) GetEpochOfflineBlocks(opts *bind.CallOpts, epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getEpochOfflineBlocks", epoch, validatorID)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetEpochOfflineBlocks is a free data retrieval call binding the contract method 0xa198d229.
//
// Solidity: function getEpochOfflineBlocks(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractSession) GetEpochOfflineBlocks(epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetEpochOfflineBlocks(&_Contract.CallOpts, epoch, validatorID)
}

// GetEpochOfflineBlocks is a free data retrieval call binding the contract method 0xa198d229.
//
// Solidity: function getEpochOfflineBlocks(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractCallerSession) GetEpochOfflineBlocks(epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetEpochOfflineBlocks(&_Contract.CallOpts, epoch, validatorID)
}

// GetEpochOfflineTime is a free data retrieval call binding the contract method 0xe261641a.
//
// Solidity: function getEpochOfflineTime(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractCaller) GetEpochOfflineTime(opts *bind.CallOpts, epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getEpochOfflineTime", epoch, validatorID)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetEpochOfflineTime is a free data retrieval call binding the contract method 0xe261641a.
//
// Solidity: function getEpochOfflineTime(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractSession) GetEpochOfflineTime(epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetEpochOfflineTime(&_Contract.CallOpts, epoch, validatorID)
}

// GetEpochOfflineTime is a free data retrieval call binding the contract method 0xe261641a.
//
// Solidity: function getEpochOfflineTime(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractCallerSession) GetEpochOfflineTime(epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetEpochOfflineTime(&_Contract.CallOpts, epoch, validatorID)
}

// GetEpochReceivedStake is a free data retrieval call binding the contract method 0x58f95b80.
//
// Solidity: function getEpochReceivedStake(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractCaller) GetEpochReceivedStake(opts *bind.CallOpts, epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getEpochReceivedStake", epoch, validatorID)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetEpochReceivedStake is a free data retrieval call binding the contract method 0x58f95b80.
//
// Solidity: function getEpochReceivedStake(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractSession) GetEpochReceivedStake(epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetEpochReceivedStake(&_Contract.CallOpts, epoch, validatorID)
}

// GetEpochReceivedStake is a free data retrieval call binding the contract method 0x58f95b80.
//
// Solidity: function getEpochReceivedStake(uint256 epoch, uint256 validatorID) view returns(uint256)
func (_Contract *ContractCallerSession) GetEpochReceivedStake(epoch *big.Int, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetEpochReceivedStake(&_Contract.CallOpts, epoch, validatorID)
}

// GetEpochSnapshot is a free data retrieval call binding the contract method 0x39b80c00.
//
// Solidity: function getEpochSnapshot(uint256 epoch) view returns(uint256 endTime, uint256 endBlock, uint256 epochFee, uint256 baseRewardPerSecond, uint256 totalStake, uint256 totalSupply)
func (_Contract *ContractCaller) GetEpochSnapshot(opts *bind.CallOpts, epoch *big.Int) (struct {
	EndTime             *big.Int
	EndBlock            *big.Int
	EpochFee            *big.Int
	BaseRewardPerSecond *big.Int
	TotalStake          *big.Int
	TotalSupply         *big.Int
}, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getEpochSnapshot", epoch)

	outstruct := new(struct {
		EndTime             *big.Int
		EndBlock            *big.Int
		EpochFee            *big.Int
		BaseRewardPerSecond *big.Int
		TotalStake          *big.Int
		TotalSupply         *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.EndTime = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.EndBlock = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.EpochFee = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.BaseRewardPerSecond = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.TotalStake = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.TotalSupply = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetEpochSnapshot is a free data retrieval call binding the contract method 0x39b80c00.
//
// Solidity: function getEpochSnapshot(uint256 epoch) view returns(uint256 endTime, uint256 endBlock, uint256 epochFee, uint256 baseRewardPerSecond, uint256 totalStake, uint256 totalSupply)
func (_Contract *ContractSession) GetEpochSnapshot(epoch *big.Int) (struct {
	EndTime             *big.Int
	EndBlock            *big.Int
	EpochFee            *big.Int
	BaseRewardPerSecond *big.Int
	TotalStake          *big.Int
	TotalSupply         *big.Int
}, error) {
	return _Contract.Contract.GetEpochSnapshot(&_Contract.CallOpts, epoch)
}

// GetEpochSnapshot is a free data retrieval call binding the contract method 0x39b80c00.
//
// Solidity: function getEpochSnapshot(uint256 epoch) view returns(uint256 endTime, uint256 endBlock, uint256 epochFee, uint256 baseRewardPerSecond, uint256 totalStake, uint256 totalSupply)
func (_Contract *ContractCallerSession) GetEpochSnapshot(epoch *big.Int) (struct {
	EndTime             *big.Int
	EndBlock            *big.Int
	EpochFee            *big.Int
	BaseRewardPerSecond *big.Int
	TotalStake          *big.Int
	TotalSupply         *big.Int
}, error) {
	return _Contract.Contract.GetEpochSnapshot(&_Contract.CallOpts, epoch)
}

// GetEpochValidatorIDs is a free data retrieval call binding the contract method 0xb88a37e2.
//
// Solidity: function getEpochValidatorIDs(uint256 epoch) view returns(uint256[])
func (_Contract *ContractCaller) GetEpochValidatorIDs(opts *bind.CallOpts, epoch *big.Int) ([]*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getEpochValidatorIDs", epoch)

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// GetEpochValidatorIDs is a free data retrieval call binding the contract method 0xb88a37e2.
//
// Solidity: function getEpochValidatorIDs(uint256 epoch) view returns(uint256[])
func (_Contract *ContractSession) GetEpochValidatorIDs(epoch *big.Int) ([]*big.Int, error) {
	return _Contract.Contract.GetEpochValidatorIDs(&_Contract.CallOpts, epoch)
}

// GetEpochValidatorIDs is a free data retrieval call binding the contract method 0xb88a37e2.
//
// Solidity: function getEpochValidatorIDs(uint256 epoch) view returns(uint256[])
func (_Contract *ContractCallerSession) GetEpochValidatorIDs(epoch *big.Int) ([]*big.Int, error) {
	return _Contract.Contract.GetEpochValidatorIDs(&_Contract.CallOpts, epoch)
}

// GetRedirection is a free data retrieval call binding the contract method 0x736de9ae.
//
// Solidity: function getRedirection(address delegator) view returns(address receiver)
func (_Contract *ContractCaller) GetRedirection(opts *bind.CallOpts, delegator common.Address) (common.Address, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getRedirection", delegator)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetRedirection is a free data retrieval call binding the contract method 0x736de9ae.
//
// Solidity: function getRedirection(address delegator) view returns(address receiver)
func (_Contract *ContractSession) GetRedirection(delegator common.Address) (common.Address, error) {
	return _Contract.Contract.GetRedirection(&_Contract.CallOpts, delegator)
}

// GetRedirection is a free data retrieval call binding the contract method 0x736de9ae.
//
// Solidity: function getRedirection(address delegator) view returns(address receiver)
func (_Contract *ContractCallerSession) GetRedirection(delegator common.Address) (common.Address, error) {
	return _Contract.Contract.GetRedirection(&_Contract.CallOpts, delegator)
}

// GetRedirectionRequest is a free data retrieval call binding the contract method 0x468f35ee.
//
// Solidity: function getRedirectionRequest(address delegator) view returns(address receiver)
func (_Contract *ContractCaller) GetRedirectionRequest(opts *bind.CallOpts, delegator common.Address) (common.Address, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getRedirectionRequest", delegator)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetRedirectionRequest is a free data retrieval call binding the contract method 0x468f35ee.
//
// Solidity: function getRedirectionRequest(address delegator) view returns(address receiver)
func (_Contract *ContractSession) GetRedirectionRequest(delegator common.Address) (common.Address, error) {
	return _Contract.Contract.GetRedirectionRequest(&_Contract.CallOpts, delegator)
}

// GetRedirectionRequest is a free data retrieval call binding the contract method 0x468f35ee.
//
// Solidity: function getRedirectionRequest(address delegator) view returns(address receiver)
func (_Contract *ContractCallerSession) GetRedirectionRequest(delegator common.Address) (common.Address, error) {
	return _Contract.Contract.GetRedirectionRequest(&_Contract.CallOpts, delegator)
}

// GetSelfStake is a free data retrieval call binding the contract method 0x5601fe01.
//
// Solidity: function getSelfStake(uint256 validatorID) view returns(uint256)
func (_Contract *ContractCaller) GetSelfStake(opts *bind.CallOpts, validatorID *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getSelfStake", validatorID)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetSelfStake is a free data retrieval call binding the contract method 0x5601fe01.
//
// Solidity: function getSelfStake(uint256 validatorID) view returns(uint256)
func (_Contract *ContractSession) GetSelfStake(validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetSelfStake(&_Contract.CallOpts, validatorID)
}

// GetSelfStake is a free data retrieval call binding the contract method 0x5601fe01.
//
// Solidity: function getSelfStake(uint256 validatorID) view returns(uint256)
func (_Contract *ContractCallerSession) GetSelfStake(validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetSelfStake(&_Contract.CallOpts, validatorID)
}

// GetStake is a free data retrieval call binding the contract method 0xcfd47663.
//
// Solidity: function getStake(address delegator, uint256 validatorID) view returns(uint256 stake)
func (_Contract *ContractCaller) GetStake(opts *bind.CallOpts, delegator common.Address, validatorID *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getStake", delegator, validatorID)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetStake is a free data retrieval call binding the contract method 0xcfd47663.
//
// Solidity: function getStake(address delegator, uint256 validatorID) view returns(uint256 stake)
func (_Contract *ContractSession) GetStake(delegator common.Address, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetStake(&_Contract.CallOpts, delegator, validatorID)
}

// GetStake is a free data retrieval call binding the contract method 0xcfd47663.
//
// Solidity: function getStake(address delegator, uint256 validatorID) view returns(uint256 stake)
func (_Contract *ContractCallerSession) GetStake(delegator common.Address, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.GetStake(&_Contract.CallOpts, delegator, validatorID)
}

// GetValidator is a free data retrieval call binding the contract method 0xb5d89627.
//
// Solidity: function getValidator(uint256 validatorID) view returns(uint256 status, uint256 receivedStake, address auth, uint256 createdEpoch, uint256 createdTime, uint256 deactivatedTime, uint256 deactivatedEpoch)
func (_Contract *ContractCaller) GetValidator(opts *bind.CallOpts, validatorID *big.Int) (struct {
	Status           *big.Int
	ReceivedStake    *big.Int
	Auth             common.Address
	CreatedEpoch     *big.Int
	CreatedTime      *big.Int
	DeactivatedTime  *big.Int
	DeactivatedEpoch *big.Int
}, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getValidator", validatorID)

	outstruct := new(struct {
		Status           *big.Int
		ReceivedStake    *big.Int
		Auth             common.Address
		CreatedEpoch     *big.Int
		CreatedTime      *big.Int
		DeactivatedTime  *big.Int
		DeactivatedEpoch *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Status = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.ReceivedStake = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Auth = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	outstruct.CreatedEpoch = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.CreatedTime = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.DeactivatedTime = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)
	outstruct.DeactivatedEpoch = *abi.ConvertType(out[6], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetValidator is a free data retrieval call binding the contract method 0xb5d89627.
//
// Solidity: function getValidator(uint256 validatorID) view returns(uint256 status, uint256 receivedStake, address auth, uint256 createdEpoch, uint256 createdTime, uint256 deactivatedTime, uint256 deactivatedEpoch)
func (_Contract *ContractSession) GetValidator(validatorID *big.Int) (struct {
	Status           *big.Int
	ReceivedStake    *big.Int
	Auth             common.Address
	CreatedEpoch     *big.Int
	CreatedTime      *big.Int
	DeactivatedTime  *big.Int
	DeactivatedEpoch *big.Int
}, error) {
	return _Contract.Contract.GetValidator(&_Contract.CallOpts, validatorID)
}

// GetValidator is a free data retrieval call binding the contract method 0xb5d89627.
//
// Solidity: function getValidator(uint256 validatorID) view returns(uint256 status, uint256 receivedStake, address auth, uint256 createdEpoch, uint256 createdTime, uint256 deactivatedTime, uint256 deactivatedEpoch)
func (_Contract *ContractCallerSession) GetValidator(validatorID *big.Int) (struct {
	Status           *big.Int
	ReceivedStake    *big.Int
	Auth             common.Address
	CreatedEpoch     *big.Int
	CreatedTime      *big.Int
	DeactivatedTime  *big.Int
	DeactivatedEpoch *big.Int
}, error) {
	return _Contract.Contract.GetValidator(&_Contract.CallOpts, validatorID)
}

// GetValidatorID is a free data retrieval call binding the contract method 0x0135b1db.
//
// Solidity: function getValidatorID(address auth) view returns(uint256 validatorID)
func (_Contract *ContractCaller) GetValidatorID(opts *bind.CallOpts, auth common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getValidatorID", auth)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetValidatorID is a free data retrieval call binding the contract method 0x0135b1db.
//
// Solidity: function getValidatorID(address auth) view returns(uint256 validatorID)
func (_Contract *ContractSession) GetValidatorID(auth common.Address) (*big.Int, error) {
	return _Contract.Contract.GetValidatorID(&_Contract.CallOpts, auth)
}

// GetValidatorID is a free data retrieval call binding the contract method 0x0135b1db.
//
// Solidity: function getValidatorID(address auth) view returns(uint256 validatorID)
func (_Contract *ContractCallerSession) GetValidatorID(auth common.Address) (*big.Int, error) {
	return _Contract.Contract.GetValidatorID(&_Contract.CallOpts, auth)
}

// GetValidatorPubkey is a free data retrieval call binding the contract method 0x854873e1.
//
// Solidity: function getValidatorPubkey(uint256 validatorID) view returns(bytes pubkey)
func (_Contract *ContractCaller) GetValidatorPubkey(opts *bind.CallOpts, validatorID *big.Int) ([]byte, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getValidatorPubkey", validatorID)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// GetValidatorPubkey is a free data retrieval call binding the contract method 0x854873e1.
//
// Solidity: function getValidatorPubkey(uint256 validatorID) view returns(bytes pubkey)
func (_Contract *ContractSession) GetValidatorPubkey(validatorID *big.Int) ([]byte, error) {
	return _Contract.Contract.GetValidatorPubkey(&_Contract.CallOpts, validatorID)
}

// GetValidatorPubkey is a free data retrieval call binding the contract method 0x854873e1.
//
// Solidity: function getValidatorPubkey(uint256 validatorID) view returns(bytes pubkey)
func (_Contract *ContractCallerSession) GetValidatorPubkey(validatorID *big.Int) ([]byte, error) {
	return _Contract.Contract.GetValidatorPubkey(&_Contract.CallOpts, validatorID)
}

// GetWithdrawalRequest is a free data retrieval call binding the contract method 0x1f270152.
//
// Solidity: function getWithdrawalRequest(address delegator, uint256 validatorID, uint256 wrID) view returns(uint256 epoch, uint256 time, uint256 amount)
func (_Contract *ContractCaller) GetWithdrawalRequest(opts *bind.CallOpts, delegator common.Address, validatorID *big.Int, wrID *big.Int) (struct {
	Epoch  *big.Int
	Time   *big.Int
	Amount *big.Int
}, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getWithdrawalRequest", delegator, validatorID, wrID)

	outstruct := new(struct {
		Epoch  *big.Int
		Time   *big.Int
		Amount *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Epoch = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Time = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Amount = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetWithdrawalRequest is a free data retrieval call binding the contract method 0x1f270152.
//
// Solidity: function getWithdrawalRequest(address delegator, uint256 validatorID, uint256 wrID) view returns(uint256 epoch, uint256 time, uint256 amount)
func (_Contract *ContractSession) GetWithdrawalRequest(delegator common.Address, validatorID *big.Int, wrID *big.Int) (struct {
	Epoch  *big.Int
	Time   *big.Int
	Amount *big.Int
}, error) {
	return _Contract.Contract.GetWithdrawalRequest(&_Contract.CallOpts, delegator, validatorID, wrID)
}

// GetWithdrawalRequest is a free data retrieval call binding the contract method 0x1f270152.
//
// Solidity: function getWithdrawalRequest(address delegator, uint256 validatorID, uint256 wrID) view returns(uint256 epoch, uint256 time, uint256 amount)
func (_Contract *ContractCallerSession) GetWithdrawalRequest(delegator common.Address, validatorID *big.Int, wrID *big.Int) (struct {
	Epoch  *big.Int
	Time   *big.Int
	Amount *big.Int
}, error) {
	return _Contract.Contract.GetWithdrawalRequest(&_Contract.CallOpts, delegator, validatorID, wrID)
}

// IsSlashed is a free data retrieval call binding the contract method 0xc3de580e.
//
// Solidity: function isSlashed(uint256 validatorID) view returns(bool)
func (_Contract *ContractCaller) IsSlashed(opts *bind.CallOpts, validatorID *big.Int) (bool, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "isSlashed", validatorID)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsSlashed is a free data retrieval call binding the contract method 0xc3de580e.
//
// Solidity: function isSlashed(uint256 validatorID) view returns(bool)
func (_Contract *ContractSession) IsSlashed(validatorID *big.Int) (bool, error) {
	return _Contract.Contract.IsSlashed(&_Contract.CallOpts, validatorID)
}

// IsSlashed is a free data retrieval call binding the contract method 0xc3de580e.
//
// Solidity: function isSlashed(uint256 validatorID) view returns(bool)
func (_Contract *ContractCallerSession) IsSlashed(validatorID *big.Int) (bool, error) {
	return _Contract.Contract.IsSlashed(&_Contract.CallOpts, validatorID)
}

// LastValidatorID is a free data retrieval call binding the contract method 0xc7be95de.
//
// Solidity: function lastValidatorID() view returns(uint256)
func (_Contract *ContractCaller) LastValidatorID(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "lastValidatorID")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LastValidatorID is a free data retrieval call binding the contract method 0xc7be95de.
//
// Solidity: function lastValidatorID() view returns(uint256)
func (_Contract *ContractSession) LastValidatorID() (*big.Int, error) {
	return _Contract.Contract.LastValidatorID(&_Contract.CallOpts)
}

// LastValidatorID is a free data retrieval call binding the contract method 0xc7be95de.
//
// Solidity: function lastValidatorID() view returns(uint256)
func (_Contract *ContractCallerSession) LastValidatorID() (*big.Int, error) {
	return _Contract.Contract.LastValidatorID(&_Contract.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Contract *ContractCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Contract *ContractSession) Owner() (common.Address, error) {
	return _Contract.Contract.Owner(&_Contract.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Contract *ContractCallerSession) Owner() (common.Address, error) {
	return _Contract.Contract.Owner(&_Contract.CallOpts)
}

// PendingRewards is a free data retrieval call binding the contract method 0x6099ecb2.
//
// Solidity: function pendingRewards(address delegator, uint256 toValidatorID) view returns(uint256)
func (_Contract *ContractCaller) PendingRewards(opts *bind.CallOpts, delegator common.Address, toValidatorID *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "pendingRewards", delegator, toValidatorID)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PendingRewards is a free data retrieval call binding the contract method 0x6099ecb2.
//
// Solidity: function pendingRewards(address delegator, uint256 toValidatorID) view returns(uint256)
func (_Contract *ContractSession) PendingRewards(delegator common.Address, toValidatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.PendingRewards(&_Contract.CallOpts, delegator, toValidatorID)
}

// PendingRewards is a free data retrieval call binding the contract method 0x6099ecb2.
//
// Solidity: function pendingRewards(address delegator, uint256 toValidatorID) view returns(uint256)
func (_Contract *ContractCallerSession) PendingRewards(delegator common.Address, toValidatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.PendingRewards(&_Contract.CallOpts, delegator, toValidatorID)
}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_Contract *ContractCaller) ProxiableUUID(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "proxiableUUID")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_Contract *ContractSession) ProxiableUUID() ([32]byte, error) {
	return _Contract.Contract.ProxiableUUID(&_Contract.CallOpts)
}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_Contract *ContractCallerSession) ProxiableUUID() ([32]byte, error) {
	return _Contract.Contract.ProxiableUUID(&_Contract.CallOpts)
}

// PubkeyAddressToValidatorID is a free data retrieval call binding the contract method 0xfb36025f.
//
// Solidity: function pubkeyAddressToValidatorID(address pubkeyAddress) view returns(uint256 validatorID)
func (_Contract *ContractCaller) PubkeyAddressToValidatorID(opts *bind.CallOpts, pubkeyAddress common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "pubkeyAddressToValidatorID", pubkeyAddress)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PubkeyAddressToValidatorID is a free data retrieval call binding the contract method 0xfb36025f.
//
// Solidity: function pubkeyAddressToValidatorID(address pubkeyAddress) view returns(uint256 validatorID)
func (_Contract *ContractSession) PubkeyAddressToValidatorID(pubkeyAddress common.Address) (*big.Int, error) {
	return _Contract.Contract.PubkeyAddressToValidatorID(&_Contract.CallOpts, pubkeyAddress)
}

// PubkeyAddressToValidatorID is a free data retrieval call binding the contract method 0xfb36025f.
//
// Solidity: function pubkeyAddressToValidatorID(address pubkeyAddress) view returns(uint256 validatorID)
func (_Contract *ContractCallerSession) PubkeyAddressToValidatorID(pubkeyAddress common.Address) (*big.Int, error) {
	return _Contract.Contract.PubkeyAddressToValidatorID(&_Contract.CallOpts, pubkeyAddress)
}

// RedirectionAuthorizer is a free data retrieval call binding the contract method 0xe9a505a7.
//
// Solidity: function redirectionAuthorizer() view returns(address)
func (_Contract *ContractCaller) RedirectionAuthorizer(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "redirectionAuthorizer")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// RedirectionAuthorizer is a free data retrieval call binding the contract method 0xe9a505a7.
//
// Solidity: function redirectionAuthorizer() view returns(address)
func (_Contract *ContractSession) RedirectionAuthorizer() (common.Address, error) {
	return _Contract.Contract.RedirectionAuthorizer(&_Contract.CallOpts)
}

// RedirectionAuthorizer is a free data retrieval call binding the contract method 0xe9a505a7.
//
// Solidity: function redirectionAuthorizer() view returns(address)
func (_Contract *ContractCallerSession) RedirectionAuthorizer() (common.Address, error) {
	return _Contract.Contract.RedirectionAuthorizer(&_Contract.CallOpts)
}

// RewardsStash is a free data retrieval call binding the contract method 0x6f498663.
//
// Solidity: function rewardsStash(address delegator, uint256 validatorID) view returns(uint256)
func (_Contract *ContractCaller) RewardsStash(opts *bind.CallOpts, delegator common.Address, validatorID *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "rewardsStash", delegator, validatorID)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// RewardsStash is a free data retrieval call binding the contract method 0x6f498663.
//
// Solidity: function rewardsStash(address delegator, uint256 validatorID) view returns(uint256)
func (_Contract *ContractSession) RewardsStash(delegator common.Address, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.RewardsStash(&_Contract.CallOpts, delegator, validatorID)
}

// RewardsStash is a free data retrieval call binding the contract method 0x6f498663.
//
// Solidity: function rewardsStash(address delegator, uint256 validatorID) view returns(uint256)
func (_Contract *ContractCallerSession) RewardsStash(delegator common.Address, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.RewardsStash(&_Contract.CallOpts, delegator, validatorID)
}

// SlashingRefundRatio is a free data retrieval call binding the contract method 0xc65ee0e1.
//
// Solidity: function slashingRefundRatio(uint256 validatorID) view returns(uint256 refundRatio)
func (_Contract *ContractCaller) SlashingRefundRatio(opts *bind.CallOpts, validatorID *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "slashingRefundRatio", validatorID)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SlashingRefundRatio is a free data retrieval call binding the contract method 0xc65ee0e1.
//
// Solidity: function slashingRefundRatio(uint256 validatorID) view returns(uint256 refundRatio)
func (_Contract *ContractSession) SlashingRefundRatio(validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.SlashingRefundRatio(&_Contract.CallOpts, validatorID)
}

// SlashingRefundRatio is a free data retrieval call binding the contract method 0xc65ee0e1.
//
// Solidity: function slashingRefundRatio(uint256 validatorID) view returns(uint256 refundRatio)
func (_Contract *ContractCallerSession) SlashingRefundRatio(validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.SlashingRefundRatio(&_Contract.CallOpts, validatorID)
}

// StakeSubscriberAddress is a free data retrieval call binding the contract method 0x093b41d0.
//
// Solidity: function stakeSubscriberAddress() view returns(address)
func (_Contract *ContractCaller) StakeSubscriberAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "stakeSubscriberAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// StakeSubscriberAddress is a free data retrieval call binding the contract method 0x093b41d0.
//
// Solidity: function stakeSubscriberAddress() view returns(address)
func (_Contract *ContractSession) StakeSubscriberAddress() (common.Address, error) {
	return _Contract.Contract.StakeSubscriberAddress(&_Contract.CallOpts)
}

// StakeSubscriberAddress is a free data retrieval call binding the contract method 0x093b41d0.
//
// Solidity: function stakeSubscriberAddress() view returns(address)
func (_Contract *ContractCallerSession) StakeSubscriberAddress() (common.Address, error) {
	return _Contract.Contract.StakeSubscriberAddress(&_Contract.CallOpts)
}

// StashedRewardsUntilEpoch is a free data retrieval call binding the contract method 0xa86a056f.
//
// Solidity: function stashedRewardsUntilEpoch(address delegator, uint256 validatorID) view returns(uint256 epoch)
func (_Contract *ContractCaller) StashedRewardsUntilEpoch(opts *bind.CallOpts, delegator common.Address, validatorID *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "stashedRewardsUntilEpoch", delegator, validatorID)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StashedRewardsUntilEpoch is a free data retrieval call binding the contract method 0xa86a056f.
//
// Solidity: function stashedRewardsUntilEpoch(address delegator, uint256 validatorID) view returns(uint256 epoch)
func (_Contract *ContractSession) StashedRewardsUntilEpoch(delegator common.Address, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.StashedRewardsUntilEpoch(&_Contract.CallOpts, delegator, validatorID)
}

// StashedRewardsUntilEpoch is a free data retrieval call binding the contract method 0xa86a056f.
//
// Solidity: function stashedRewardsUntilEpoch(address delegator, uint256 validatorID) view returns(uint256 epoch)
func (_Contract *ContractCallerSession) StashedRewardsUntilEpoch(delegator common.Address, validatorID *big.Int) (*big.Int, error) {
	return _Contract.Contract.StashedRewardsUntilEpoch(&_Contract.CallOpts, delegator, validatorID)
}

// TotalActiveStake is a free data retrieval call binding the contract method 0x28f73148.
//
// Solidity: function totalActiveStake() view returns(uint256)
func (_Contract *ContractCaller) TotalActiveStake(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "totalActiveStake")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalActiveStake is a free data retrieval call binding the contract method 0x28f73148.
//
// Solidity: function totalActiveStake() view returns(uint256)
func (_Contract *ContractSession) TotalActiveStake() (*big.Int, error) {
	return _Contract.Contract.TotalActiveStake(&_Contract.CallOpts)
}

// TotalActiveStake is a free data retrieval call binding the contract method 0x28f73148.
//
// Solidity: function totalActiveStake() view returns(uint256)
func (_Contract *ContractCallerSession) TotalActiveStake() (*big.Int, error) {
	return _Contract.Contract.TotalActiveStake(&_Contract.CallOpts)
}

// TotalStake is a free data retrieval call binding the contract method 0x8b0e9f3f.
//
// Solidity: function totalStake() view returns(uint256)
func (_Contract *ContractCaller) TotalStake(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "totalStake")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalStake is a free data retrieval call binding the contract method 0x8b0e9f3f.
//
// Solidity: function totalStake() view returns(uint256)
func (_Contract *ContractSession) TotalStake() (*big.Int, error) {
	return _Contract.Contract.TotalStake(&_Contract.CallOpts)
}

// TotalStake is a free data retrieval call binding the contract method 0x8b0e9f3f.
//
// Solidity: function totalStake() view returns(uint256)
func (_Contract *ContractCallerSession) TotalStake() (*big.Int, error) {
	return _Contract.Contract.TotalStake(&_Contract.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_Contract *ContractCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_Contract *ContractSession) TotalSupply() (*big.Int, error) {
	return _Contract.Contract.TotalSupply(&_Contract.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_Contract *ContractCallerSession) TotalSupply() (*big.Int, error) {
	return _Contract.Contract.TotalSupply(&_Contract.CallOpts)
}

// TreasuryAddress is a free data retrieval call binding the contract method 0xc5f956af.
//
// Solidity: function treasuryAddress() view returns(address)
func (_Contract *ContractCaller) TreasuryAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "treasuryAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// TreasuryAddress is a free data retrieval call binding the contract method 0xc5f956af.
//
// Solidity: function treasuryAddress() view returns(address)
func (_Contract *ContractSession) TreasuryAddress() (common.Address, error) {
	return _Contract.Contract.TreasuryAddress(&_Contract.CallOpts)
}

// TreasuryAddress is a free data retrieval call binding the contract method 0xc5f956af.
//
// Solidity: function treasuryAddress() view returns(address)
func (_Contract *ContractCallerSession) TreasuryAddress() (common.Address, error) {
	return _Contract.Contract.TreasuryAddress(&_Contract.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() pure returns(bytes3)
func (_Contract *ContractCaller) Version(opts *bind.CallOpts) ([3]byte, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "version")

	if err != nil {
		return *new([3]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([3]byte)).(*[3]byte)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() pure returns(bytes3)
func (_Contract *ContractSession) Version() ([3]byte, error) {
	return _Contract.Contract.Version(&_Contract.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() pure returns(bytes3)
func (_Contract *ContractCallerSession) Version() ([3]byte, error) {
	return _Contract.Contract.Version(&_Contract.CallOpts)
}

// SyncValidator is a paid mutator transaction binding the contract method 0xcc8343aa.
//
// Solidity: function _syncValidator(uint256 validatorID, bool syncPubkey) returns()
func (_Contract *ContractTransactor) SyncValidator(opts *bind.TransactOpts, validatorID *big.Int, syncPubkey bool) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "_syncValidator", validatorID, syncPubkey)
}

// SyncValidator is a paid mutator transaction binding the contract method 0xcc8343aa.
//
// Solidity: function _syncValidator(uint256 validatorID, bool syncPubkey) returns()
func (_Contract *ContractSession) SyncValidator(validatorID *big.Int, syncPubkey bool) (*types.Transaction, error) {
	return _Contract.Contract.SyncValidator(&_Contract.TransactOpts, validatorID, syncPubkey)
}

// SyncValidator is a paid mutator transaction binding the contract method 0xcc8343aa.
//
// Solidity: function _syncValidator(uint256 validatorID, bool syncPubkey) returns()
func (_Contract *ContractTransactorSession) SyncValidator(validatorID *big.Int, syncPubkey bool) (*types.Transaction, error) {
	return _Contract.Contract.SyncValidator(&_Contract.TransactOpts, validatorID, syncPubkey)
}

// AnnounceRedirection is a paid mutator transaction binding the contract method 0x46f1ca35.
//
// Solidity: function announceRedirection(address to) returns()
func (_Contract *ContractTransactor) AnnounceRedirection(opts *bind.TransactOpts, to common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "announceRedirection", to)
}

// AnnounceRedirection is a paid mutator transaction binding the contract method 0x46f1ca35.
//
// Solidity: function announceRedirection(address to) returns()
func (_Contract *ContractSession) AnnounceRedirection(to common.Address) (*types.Transaction, error) {
	return _Contract.Contract.AnnounceRedirection(&_Contract.TransactOpts, to)
}

// AnnounceRedirection is a paid mutator transaction binding the contract method 0x46f1ca35.
//
// Solidity: function announceRedirection(address to) returns()
func (_Contract *ContractTransactorSession) AnnounceRedirection(to common.Address) (*types.Transaction, error) {
	return _Contract.Contract.AnnounceRedirection(&_Contract.TransactOpts, to)
}

// BurnFTM is a paid mutator transaction binding the contract method 0x90a6c475.
//
// Solidity: function burnFTM(uint256 amount) returns()
func (_Contract *ContractTransactor) BurnFTM(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "burnFTM", amount)
}

// BurnFTM is a paid mutator transaction binding the contract method 0x90a6c475.
//
// Solidity: function burnFTM(uint256 amount) returns()
func (_Contract *ContractSession) BurnFTM(amount *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.BurnFTM(&_Contract.TransactOpts, amount)
}

// BurnFTM is a paid mutator transaction binding the contract method 0x90a6c475.
//
// Solidity: function burnFTM(uint256 amount) returns()
func (_Contract *ContractTransactorSession) BurnFTM(amount *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.BurnFTM(&_Contract.TransactOpts, amount)
}

// ClaimRewards is a paid mutator transaction binding the contract method 0x0962ef79.
//
// Solidity: function claimRewards(uint256 toValidatorID) returns()
func (_Contract *ContractTransactor) ClaimRewards(opts *bind.TransactOpts, toValidatorID *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "claimRewards", toValidatorID)
}

// ClaimRewards is a paid mutator transaction binding the contract method 0x0962ef79.
//
// Solidity: function claimRewards(uint256 toValidatorID) returns()
func (_Contract *ContractSession) ClaimRewards(toValidatorID *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.ClaimRewards(&_Contract.TransactOpts, toValidatorID)
}

// ClaimRewards is a paid mutator transaction binding the contract method 0x0962ef79.
//
// Solidity: function claimRewards(uint256 toValidatorID) returns()
func (_Contract *ContractTransactorSession) ClaimRewards(toValidatorID *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.ClaimRewards(&_Contract.TransactOpts, toValidatorID)
}

// CreateValidator is a paid mutator transaction binding the contract method 0xa5a470ad.
//
// Solidity: function createValidator(bytes pubkey) payable returns()
func (_Contract *ContractTransactor) CreateValidator(opts *bind.TransactOpts, pubkey []byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "createValidator", pubkey)
}

// CreateValidator is a paid mutator transaction binding the contract method 0xa5a470ad.
//
// Solidity: function createValidator(bytes pubkey) payable returns()
func (_Contract *ContractSession) CreateValidator(pubkey []byte) (*types.Transaction, error) {
	return _Contract.Contract.CreateValidator(&_Contract.TransactOpts, pubkey)
}

// CreateValidator is a paid mutator transaction binding the contract method 0xa5a470ad.
//
// Solidity: function createValidator(bytes pubkey) payable returns()
func (_Contract *ContractTransactorSession) CreateValidator(pubkey []byte) (*types.Transaction, error) {
	return _Contract.Contract.CreateValidator(&_Contract.TransactOpts, pubkey)
}

// DeactivateValidator is a paid mutator transaction binding the contract method 0x1e702f83.
//
// Solidity: function deactivateValidator(uint256 validatorID, uint256 status) returns()
func (_Contract *ContractTransactor) DeactivateValidator(opts *bind.TransactOpts, validatorID *big.Int, status *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "deactivateValidator", validatorID, status)
}

// DeactivateValidator is a paid mutator transaction binding the contract method 0x1e702f83.
//
// Solidity: function deactivateValidator(uint256 validatorID, uint256 status) returns()
func (_Contract *ContractSession) DeactivateValidator(validatorID *big.Int, status *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.DeactivateValidator(&_Contract.TransactOpts, validatorID, status)
}

// DeactivateValidator is a paid mutator transaction binding the contract method 0x1e702f83.
//
// Solidity: function deactivateValidator(uint256 validatorID, uint256 status) returns()
func (_Contract *ContractTransactorSession) DeactivateValidator(validatorID *big.Int, status *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.DeactivateValidator(&_Contract.TransactOpts, validatorID, status)
}

// Delegate is a paid mutator transaction binding the contract method 0x9fa6dd35.
//
// Solidity: function delegate(uint256 toValidatorID) payable returns()
func (_Contract *ContractTransactor) Delegate(opts *bind.TransactOpts, toValidatorID *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "delegate", toValidatorID)
}

// Delegate is a paid mutator transaction binding the contract method 0x9fa6dd35.
//
// Solidity: function delegate(uint256 toValidatorID) payable returns()
func (_Contract *ContractSession) Delegate(toValidatorID *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.Delegate(&_Contract.TransactOpts, toValidatorID)
}

// Delegate is a paid mutator transaction binding the contract method 0x9fa6dd35.
//
// Solidity: function delegate(uint256 toValidatorID) payable returns()
func (_Contract *ContractTransactorSession) Delegate(toValidatorID *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.Delegate(&_Contract.TransactOpts, toValidatorID)
}

// Initialize is a paid mutator transaction binding the contract method 0x3fbfd4df.
//
// Solidity: function initialize(uint256 sealedEpoch, uint256 _totalSupply, address nodeDriver, address _c, address owner) returns()
func (_Contract *ContractTransactor) Initialize(opts *bind.TransactOpts, sealedEpoch *big.Int, _totalSupply *big.Int, nodeDriver common.Address, _c common.Address, owner common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "initialize", sealedEpoch, _totalSupply, nodeDriver, _c, owner)
}

// Initialize is a paid mutator transaction binding the contract method 0x3fbfd4df.
//
// Solidity: function initialize(uint256 sealedEpoch, uint256 _totalSupply, address nodeDriver, address _c, address owner) returns()
func (_Contract *ContractSession) Initialize(sealedEpoch *big.Int, _totalSupply *big.Int, nodeDriver common.Address, _c common.Address, owner common.Address) (*types.Transaction, error) {
	return _Contract.Contract.Initialize(&_Contract.TransactOpts, sealedEpoch, _totalSupply, nodeDriver, _c, owner)
}

// Initialize is a paid mutator transaction binding the contract method 0x3fbfd4df.
//
// Solidity: function initialize(uint256 sealedEpoch, uint256 _totalSupply, address nodeDriver, address _c, address owner) returns()
func (_Contract *ContractTransactorSession) Initialize(sealedEpoch *big.Int, _totalSupply *big.Int, nodeDriver common.Address, _c common.Address, owner common.Address) (*types.Transaction, error) {
	return _Contract.Contract.Initialize(&_Contract.TransactOpts, sealedEpoch, _totalSupply, nodeDriver, _c, owner)
}

// InitiateRedirection is a paid mutator transaction binding the contract method 0xcc172784.
//
// Solidity: function initiateRedirection(address from, address to) returns()
func (_Contract *ContractTransactor) InitiateRedirection(opts *bind.TransactOpts, from common.Address, to common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "initiateRedirection", from, to)
}

// InitiateRedirection is a paid mutator transaction binding the contract method 0xcc172784.
//
// Solidity: function initiateRedirection(address from, address to) returns()
func (_Contract *ContractSession) InitiateRedirection(from common.Address, to common.Address) (*types.Transaction, error) {
	return _Contract.Contract.InitiateRedirection(&_Contract.TransactOpts, from, to)
}

// InitiateRedirection is a paid mutator transaction binding the contract method 0xcc172784.
//
// Solidity: function initiateRedirection(address from, address to) returns()
func (_Contract *ContractTransactorSession) InitiateRedirection(from common.Address, to common.Address) (*types.Transaction, error) {
	return _Contract.Contract.InitiateRedirection(&_Contract.TransactOpts, from, to)
}

// Redirect is a paid mutator transaction binding the contract method 0xd725e91f.
//
// Solidity: function redirect(address to) returns()
func (_Contract *ContractTransactor) Redirect(opts *bind.TransactOpts, to common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "redirect", to)
}

// Redirect is a paid mutator transaction binding the contract method 0xd725e91f.
//
// Solidity: function redirect(address to) returns()
func (_Contract *ContractSession) Redirect(to common.Address) (*types.Transaction, error) {
	return _Contract.Contract.Redirect(&_Contract.TransactOpts, to)
}

// Redirect is a paid mutator transaction binding the contract method 0xd725e91f.
//
// Solidity: function redirect(address to) returns()
func (_Contract *ContractTransactorSession) Redirect(to common.Address) (*types.Transaction, error) {
	return _Contract.Contract.Redirect(&_Contract.TransactOpts, to)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Contract *ContractTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Contract *ContractSession) RenounceOwnership() (*types.Transaction, error) {
	return _Contract.Contract.RenounceOwnership(&_Contract.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Contract *ContractTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Contract.Contract.RenounceOwnership(&_Contract.TransactOpts)
}

// RestakeRewards is a paid mutator transaction binding the contract method 0x08c36874.
//
// Solidity: function restakeRewards(uint256 toValidatorID) returns()
func (_Contract *ContractTransactor) RestakeRewards(opts *bind.TransactOpts, toValidatorID *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "restakeRewards", toValidatorID)
}

// RestakeRewards is a paid mutator transaction binding the contract method 0x08c36874.
//
// Solidity: function restakeRewards(uint256 toValidatorID) returns()
func (_Contract *ContractSession) RestakeRewards(toValidatorID *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.RestakeRewards(&_Contract.TransactOpts, toValidatorID)
}

// RestakeRewards is a paid mutator transaction binding the contract method 0x08c36874.
//
// Solidity: function restakeRewards(uint256 toValidatorID) returns()
func (_Contract *ContractTransactorSession) RestakeRewards(toValidatorID *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.RestakeRewards(&_Contract.TransactOpts, toValidatorID)
}

// SealEpoch is a paid mutator transaction binding the contract method 0xebdf104c.
//
// Solidity: function sealEpoch(uint256[] offlineTime, uint256[] offlineBlocks, uint256[] uptimes, uint256[] originatedTxsFee) returns()
func (_Contract *ContractTransactor) SealEpoch(opts *bind.TransactOpts, offlineTime []*big.Int, offlineBlocks []*big.Int, uptimes []*big.Int, originatedTxsFee []*big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "sealEpoch", offlineTime, offlineBlocks, uptimes, originatedTxsFee)
}

// SealEpoch is a paid mutator transaction binding the contract method 0xebdf104c.
//
// Solidity: function sealEpoch(uint256[] offlineTime, uint256[] offlineBlocks, uint256[] uptimes, uint256[] originatedTxsFee) returns()
func (_Contract *ContractSession) SealEpoch(offlineTime []*big.Int, offlineBlocks []*big.Int, uptimes []*big.Int, originatedTxsFee []*big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SealEpoch(&_Contract.TransactOpts, offlineTime, offlineBlocks, uptimes, originatedTxsFee)
}

// SealEpoch is a paid mutator transaction binding the contract method 0xebdf104c.
//
// Solidity: function sealEpoch(uint256[] offlineTime, uint256[] offlineBlocks, uint256[] uptimes, uint256[] originatedTxsFee) returns()
func (_Contract *ContractTransactorSession) SealEpoch(offlineTime []*big.Int, offlineBlocks []*big.Int, uptimes []*big.Int, originatedTxsFee []*big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SealEpoch(&_Contract.TransactOpts, offlineTime, offlineBlocks, uptimes, originatedTxsFee)
}

// SealEpochValidators is a paid mutator transaction binding the contract method 0xe08d7e66.
//
// Solidity: function sealEpochValidators(uint256[] nextValidatorIDs) returns()
func (_Contract *ContractTransactor) SealEpochValidators(opts *bind.TransactOpts, nextValidatorIDs []*big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "sealEpochValidators", nextValidatorIDs)
}

// SealEpochValidators is a paid mutator transaction binding the contract method 0xe08d7e66.
//
// Solidity: function sealEpochValidators(uint256[] nextValidatorIDs) returns()
func (_Contract *ContractSession) SealEpochValidators(nextValidatorIDs []*big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SealEpochValidators(&_Contract.TransactOpts, nextValidatorIDs)
}

// SealEpochValidators is a paid mutator transaction binding the contract method 0xe08d7e66.
//
// Solidity: function sealEpochValidators(uint256[] nextValidatorIDs) returns()
func (_Contract *ContractTransactorSession) SealEpochValidators(nextValidatorIDs []*big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SealEpochValidators(&_Contract.TransactOpts, nextValidatorIDs)
}

// SetGenesisDelegation is a paid mutator transaction binding the contract method 0xa8ab09ba.
//
// Solidity: function setGenesisDelegation(address delegator, uint256 toValidatorID, uint256 stake) returns()
func (_Contract *ContractTransactor) SetGenesisDelegation(opts *bind.TransactOpts, delegator common.Address, toValidatorID *big.Int, stake *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "setGenesisDelegation", delegator, toValidatorID, stake)
}

// SetGenesisDelegation is a paid mutator transaction binding the contract method 0xa8ab09ba.
//
// Solidity: function setGenesisDelegation(address delegator, uint256 toValidatorID, uint256 stake) returns()
func (_Contract *ContractSession) SetGenesisDelegation(delegator common.Address, toValidatorID *big.Int, stake *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SetGenesisDelegation(&_Contract.TransactOpts, delegator, toValidatorID, stake)
}

// SetGenesisDelegation is a paid mutator transaction binding the contract method 0xa8ab09ba.
//
// Solidity: function setGenesisDelegation(address delegator, uint256 toValidatorID, uint256 stake) returns()
func (_Contract *ContractTransactorSession) SetGenesisDelegation(delegator common.Address, toValidatorID *big.Int, stake *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SetGenesisDelegation(&_Contract.TransactOpts, delegator, toValidatorID, stake)
}

// SetGenesisValidator is a paid mutator transaction binding the contract method 0x76fed43a.
//
// Solidity: function setGenesisValidator(address auth, uint256 validatorID, bytes pubkey, uint256 createdTime) returns()
func (_Contract *ContractTransactor) SetGenesisValidator(opts *bind.TransactOpts, auth common.Address, validatorID *big.Int, pubkey []byte, createdTime *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "setGenesisValidator", auth, validatorID, pubkey, createdTime)
}

// SetGenesisValidator is a paid mutator transaction binding the contract method 0x76fed43a.
//
// Solidity: function setGenesisValidator(address auth, uint256 validatorID, bytes pubkey, uint256 createdTime) returns()
func (_Contract *ContractSession) SetGenesisValidator(auth common.Address, validatorID *big.Int, pubkey []byte, createdTime *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SetGenesisValidator(&_Contract.TransactOpts, auth, validatorID, pubkey, createdTime)
}

// SetGenesisValidator is a paid mutator transaction binding the contract method 0x76fed43a.
//
// Solidity: function setGenesisValidator(address auth, uint256 validatorID, bytes pubkey, uint256 createdTime) returns()
func (_Contract *ContractTransactorSession) SetGenesisValidator(auth common.Address, validatorID *big.Int, pubkey []byte, createdTime *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SetGenesisValidator(&_Contract.TransactOpts, auth, validatorID, pubkey, createdTime)
}

// SetRedirectionAuthorizer is a paid mutator transaction binding the contract method 0xb0ef386c.
//
// Solidity: function setRedirectionAuthorizer(address v) returns()
func (_Contract *ContractTransactor) SetRedirectionAuthorizer(opts *bind.TransactOpts, v common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "setRedirectionAuthorizer", v)
}

// SetRedirectionAuthorizer is a paid mutator transaction binding the contract method 0xb0ef386c.
//
// Solidity: function setRedirectionAuthorizer(address v) returns()
func (_Contract *ContractSession) SetRedirectionAuthorizer(v common.Address) (*types.Transaction, error) {
	return _Contract.Contract.SetRedirectionAuthorizer(&_Contract.TransactOpts, v)
}

// SetRedirectionAuthorizer is a paid mutator transaction binding the contract method 0xb0ef386c.
//
// Solidity: function setRedirectionAuthorizer(address v) returns()
func (_Contract *ContractTransactorSession) SetRedirectionAuthorizer(v common.Address) (*types.Transaction, error) {
	return _Contract.Contract.SetRedirectionAuthorizer(&_Contract.TransactOpts, v)
}

// StashRewards is a paid mutator transaction binding the contract method 0x8cddb015.
//
// Solidity: function stashRewards(address delegator, uint256 toValidatorID) returns()
func (_Contract *ContractTransactor) StashRewards(opts *bind.TransactOpts, delegator common.Address, toValidatorID *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "stashRewards", delegator, toValidatorID)
}

// StashRewards is a paid mutator transaction binding the contract method 0x8cddb015.
//
// Solidity: function stashRewards(address delegator, uint256 toValidatorID) returns()
func (_Contract *ContractSession) StashRewards(delegator common.Address, toValidatorID *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.StashRewards(&_Contract.TransactOpts, delegator, toValidatorID)
}

// StashRewards is a paid mutator transaction binding the contract method 0x8cddb015.
//
// Solidity: function stashRewards(address delegator, uint256 toValidatorID) returns()
func (_Contract *ContractTransactorSession) StashRewards(delegator common.Address, toValidatorID *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.StashRewards(&_Contract.TransactOpts, delegator, toValidatorID)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Contract *ContractTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Contract *ContractSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Contract.Contract.TransferOwnership(&_Contract.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Contract *ContractTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Contract.Contract.TransferOwnership(&_Contract.TransactOpts, newOwner)
}

// Undelegate is a paid mutator transaction binding the contract method 0x4f864df4.
//
// Solidity: function undelegate(uint256 toValidatorID, uint256 wrID, uint256 amount) returns()
func (_Contract *ContractTransactor) Undelegate(opts *bind.TransactOpts, toValidatorID *big.Int, wrID *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "undelegate", toValidatorID, wrID, amount)
}

// Undelegate is a paid mutator transaction binding the contract method 0x4f864df4.
//
// Solidity: function undelegate(uint256 toValidatorID, uint256 wrID, uint256 amount) returns()
func (_Contract *ContractSession) Undelegate(toValidatorID *big.Int, wrID *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.Undelegate(&_Contract.TransactOpts, toValidatorID, wrID, amount)
}

// Undelegate is a paid mutator transaction binding the contract method 0x4f864df4.
//
// Solidity: function undelegate(uint256 toValidatorID, uint256 wrID, uint256 amount) returns()
func (_Contract *ContractTransactorSession) Undelegate(toValidatorID *big.Int, wrID *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.Undelegate(&_Contract.TransactOpts, toValidatorID, wrID, amount)
}

// UpdateConstsAddress is a paid mutator transaction binding the contract method 0x860c2750.
//
// Solidity: function updateConstsAddress(address v) returns()
func (_Contract *ContractTransactor) UpdateConstsAddress(opts *bind.TransactOpts, v common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "updateConstsAddress", v)
}

// UpdateConstsAddress is a paid mutator transaction binding the contract method 0x860c2750.
//
// Solidity: function updateConstsAddress(address v) returns()
func (_Contract *ContractSession) UpdateConstsAddress(v common.Address) (*types.Transaction, error) {
	return _Contract.Contract.UpdateConstsAddress(&_Contract.TransactOpts, v)
}

// UpdateConstsAddress is a paid mutator transaction binding the contract method 0x860c2750.
//
// Solidity: function updateConstsAddress(address v) returns()
func (_Contract *ContractTransactorSession) UpdateConstsAddress(v common.Address) (*types.Transaction, error) {
	return _Contract.Contract.UpdateConstsAddress(&_Contract.TransactOpts, v)
}

// UpdateSlashingRefundRatio is a paid mutator transaction binding the contract method 0x4f7c4efb.
//
// Solidity: function updateSlashingRefundRatio(uint256 validatorID, uint256 refundRatio) returns()
func (_Contract *ContractTransactor) UpdateSlashingRefundRatio(opts *bind.TransactOpts, validatorID *big.Int, refundRatio *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "updateSlashingRefundRatio", validatorID, refundRatio)
}

// UpdateSlashingRefundRatio is a paid mutator transaction binding the contract method 0x4f7c4efb.
//
// Solidity: function updateSlashingRefundRatio(uint256 validatorID, uint256 refundRatio) returns()
func (_Contract *ContractSession) UpdateSlashingRefundRatio(validatorID *big.Int, refundRatio *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.UpdateSlashingRefundRatio(&_Contract.TransactOpts, validatorID, refundRatio)
}

// UpdateSlashingRefundRatio is a paid mutator transaction binding the contract method 0x4f7c4efb.
//
// Solidity: function updateSlashingRefundRatio(uint256 validatorID, uint256 refundRatio) returns()
func (_Contract *ContractTransactorSession) UpdateSlashingRefundRatio(validatorID *big.Int, refundRatio *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.UpdateSlashingRefundRatio(&_Contract.TransactOpts, validatorID, refundRatio)
}

// UpdateStakeSubscriberAddress is a paid mutator transaction binding the contract method 0xe880a159.
//
// Solidity: function updateStakeSubscriberAddress(address v) returns()
func (_Contract *ContractTransactor) UpdateStakeSubscriberAddress(opts *bind.TransactOpts, v common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "updateStakeSubscriberAddress", v)
}

// UpdateStakeSubscriberAddress is a paid mutator transaction binding the contract method 0xe880a159.
//
// Solidity: function updateStakeSubscriberAddress(address v) returns()
func (_Contract *ContractSession) UpdateStakeSubscriberAddress(v common.Address) (*types.Transaction, error) {
	return _Contract.Contract.UpdateStakeSubscriberAddress(&_Contract.TransactOpts, v)
}

// UpdateStakeSubscriberAddress is a paid mutator transaction binding the contract method 0xe880a159.
//
// Solidity: function updateStakeSubscriberAddress(address v) returns()
func (_Contract *ContractTransactorSession) UpdateStakeSubscriberAddress(v common.Address) (*types.Transaction, error) {
	return _Contract.Contract.UpdateStakeSubscriberAddress(&_Contract.TransactOpts, v)
}

// UpdateTreasuryAddress is a paid mutator transaction binding the contract method 0x841e4561.
//
// Solidity: function updateTreasuryAddress(address v) returns()
func (_Contract *ContractTransactor) UpdateTreasuryAddress(opts *bind.TransactOpts, v common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "updateTreasuryAddress", v)
}

// UpdateTreasuryAddress is a paid mutator transaction binding the contract method 0x841e4561.
//
// Solidity: function updateTreasuryAddress(address v) returns()
func (_Contract *ContractSession) UpdateTreasuryAddress(v common.Address) (*types.Transaction, error) {
	return _Contract.Contract.UpdateTreasuryAddress(&_Contract.TransactOpts, v)
}

// UpdateTreasuryAddress is a paid mutator transaction binding the contract method 0x841e4561.
//
// Solidity: function updateTreasuryAddress(address v) returns()
func (_Contract *ContractTransactorSession) UpdateTreasuryAddress(v common.Address) (*types.Transaction, error) {
	return _Contract.Contract.UpdateTreasuryAddress(&_Contract.TransactOpts, v)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_Contract *ContractTransactor) UpgradeToAndCall(opts *bind.TransactOpts, newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "upgradeToAndCall", newImplementation, data)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_Contract *ContractSession) UpgradeToAndCall(newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _Contract.Contract.UpgradeToAndCall(&_Contract.TransactOpts, newImplementation, data)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_Contract *ContractTransactorSession) UpgradeToAndCall(newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _Contract.Contract.UpgradeToAndCall(&_Contract.TransactOpts, newImplementation, data)
}

// Withdraw is a paid mutator transaction binding the contract method 0x441a3e70.
//
// Solidity: function withdraw(uint256 toValidatorID, uint256 wrID) returns()
func (_Contract *ContractTransactor) Withdraw(opts *bind.TransactOpts, toValidatorID *big.Int, wrID *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "withdraw", toValidatorID, wrID)
}

// Withdraw is a paid mutator transaction binding the contract method 0x441a3e70.
//
// Solidity: function withdraw(uint256 toValidatorID, uint256 wrID) returns()
func (_Contract *ContractSession) Withdraw(toValidatorID *big.Int, wrID *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.Withdraw(&_Contract.TransactOpts, toValidatorID, wrID)
}

// Withdraw is a paid mutator transaction binding the contract method 0x441a3e70.
//
// Solidity: function withdraw(uint256 toValidatorID, uint256 wrID) returns()
func (_Contract *ContractTransactorSession) Withdraw(toValidatorID *big.Int, wrID *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.Withdraw(&_Contract.TransactOpts, toValidatorID, wrID)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Contract *ContractTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contract.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Contract *ContractSession) Receive() (*types.Transaction, error) {
	return _Contract.Contract.Receive(&_Contract.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Contract *ContractTransactorSession) Receive() (*types.Transaction, error) {
	return _Contract.Contract.Receive(&_Contract.TransactOpts)
}

// ContractAnnouncedRedirectionIterator is returned from FilterAnnouncedRedirection and is used to iterate over the raw logs and unpacked data for AnnouncedRedirection events raised by the Contract contract.
type ContractAnnouncedRedirectionIterator struct {
	Event *ContractAnnouncedRedirection // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ContractAnnouncedRedirectionIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractAnnouncedRedirection)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ContractAnnouncedRedirection)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ContractAnnouncedRedirectionIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractAnnouncedRedirectionIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractAnnouncedRedirection represents a AnnouncedRedirection event raised by the Contract contract.
type ContractAnnouncedRedirection struct {
	From common.Address
	To   common.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterAnnouncedRedirection is a free log retrieval operation binding the contract event 0x857125196131cfcd709c738c6d1fd2701ce70f2a03785aeadae6f4b47fe73c1d.
//
// Solidity: event AnnouncedRedirection(address indexed from, address indexed to)
func (_Contract *ContractFilterer) FilterAnnouncedRedirection(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*ContractAnnouncedRedirectionIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "AnnouncedRedirection", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &ContractAnnouncedRedirectionIterator{contract: _Contract.contract, event: "AnnouncedRedirection", logs: logs, sub: sub}, nil
}

// WatchAnnouncedRedirection is a free log subscription operation binding the contract event 0x857125196131cfcd709c738c6d1fd2701ce70f2a03785aeadae6f4b47fe73c1d.
//
// Solidity: event AnnouncedRedirection(address indexed from, address indexed to)
func (_Contract *ContractFilterer) WatchAnnouncedRedirection(opts *bind.WatchOpts, sink chan<- *ContractAnnouncedRedirection, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "AnnouncedRedirection", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractAnnouncedRedirection)
				if err := _Contract.contract.UnpackLog(event, "AnnouncedRedirection", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAnnouncedRedirection is a log parse operation binding the contract event 0x857125196131cfcd709c738c6d1fd2701ce70f2a03785aeadae6f4b47fe73c1d.
//
// Solidity: event AnnouncedRedirection(address indexed from, address indexed to)
func (_Contract *ContractFilterer) ParseAnnouncedRedirection(log types.Log) (*ContractAnnouncedRedirection, error) {
	event := new(ContractAnnouncedRedirection)
	if err := _Contract.contract.UnpackLog(event, "AnnouncedRedirection", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractBurntFTMIterator is returned from FilterBurntFTM and is used to iterate over the raw logs and unpacked data for BurntFTM events raised by the Contract contract.
type ContractBurntFTMIterator struct {
	Event *ContractBurntFTM // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ContractBurntFTMIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractBurntFTM)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ContractBurntFTM)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ContractBurntFTMIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractBurntFTMIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractBurntFTM represents a BurntFTM event raised by the Contract contract.
type ContractBurntFTM struct {
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterBurntFTM is a free log retrieval operation binding the contract event 0x8918bd6046d08b314e457977f29562c5d76a7030d79b1edba66e8a5da0b77ae8.
//
// Solidity: event BurntFTM(uint256 amount)
func (_Contract *ContractFilterer) FilterBurntFTM(opts *bind.FilterOpts) (*ContractBurntFTMIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "BurntFTM")
	if err != nil {
		return nil, err
	}
	return &ContractBurntFTMIterator{contract: _Contract.contract, event: "BurntFTM", logs: logs, sub: sub}, nil
}

// WatchBurntFTM is a free log subscription operation binding the contract event 0x8918bd6046d08b314e457977f29562c5d76a7030d79b1edba66e8a5da0b77ae8.
//
// Solidity: event BurntFTM(uint256 amount)
func (_Contract *ContractFilterer) WatchBurntFTM(opts *bind.WatchOpts, sink chan<- *ContractBurntFTM) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "BurntFTM")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractBurntFTM)
				if err := _Contract.contract.UnpackLog(event, "BurntFTM", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBurntFTM is a log parse operation binding the contract event 0x8918bd6046d08b314e457977f29562c5d76a7030d79b1edba66e8a5da0b77ae8.
//
// Solidity: event BurntFTM(uint256 amount)
func (_Contract *ContractFilterer) ParseBurntFTM(log types.Log) (*ContractBurntFTM, error) {
	event := new(ContractBurntFTM)
	if err := _Contract.contract.UnpackLog(event, "BurntFTM", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractChangedValidatorStatusIterator is returned from FilterChangedValidatorStatus and is used to iterate over the raw logs and unpacked data for ChangedValidatorStatus events raised by the Contract contract.
type ContractChangedValidatorStatusIterator struct {
	Event *ContractChangedValidatorStatus // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ContractChangedValidatorStatusIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractChangedValidatorStatus)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ContractChangedValidatorStatus)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ContractChangedValidatorStatusIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractChangedValidatorStatusIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractChangedValidatorStatus represents a ChangedValidatorStatus event raised by the Contract contract.
type ContractChangedValidatorStatus struct {
	ValidatorID *big.Int
	Status      *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterChangedValidatorStatus is a free log retrieval operation binding the contract event 0xcd35267e7654194727477d6c78b541a553483cff7f92a055d17868d3da6e953e.
//
// Solidity: event ChangedValidatorStatus(uint256 indexed validatorID, uint256 status)
func (_Contract *ContractFilterer) FilterChangedValidatorStatus(opts *bind.FilterOpts, validatorID []*big.Int) (*ContractChangedValidatorStatusIterator, error) {

	var validatorIDRule []interface{}
	for _, validatorIDItem := range validatorID {
		validatorIDRule = append(validatorIDRule, validatorIDItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "ChangedValidatorStatus", validatorIDRule)
	if err != nil {
		return nil, err
	}
	return &ContractChangedValidatorStatusIterator{contract: _Contract.contract, event: "ChangedValidatorStatus", logs: logs, sub: sub}, nil
}

// WatchChangedValidatorStatus is a free log subscription operation binding the contract event 0xcd35267e7654194727477d6c78b541a553483cff7f92a055d17868d3da6e953e.
//
// Solidity: event ChangedValidatorStatus(uint256 indexed validatorID, uint256 status)
func (_Contract *ContractFilterer) WatchChangedValidatorStatus(opts *bind.WatchOpts, sink chan<- *ContractChangedValidatorStatus, validatorID []*big.Int) (event.Subscription, error) {

	var validatorIDRule []interface{}
	for _, validatorIDItem := range validatorID {
		validatorIDRule = append(validatorIDRule, validatorIDItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "ChangedValidatorStatus", validatorIDRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractChangedValidatorStatus)
				if err := _Contract.contract.UnpackLog(event, "ChangedValidatorStatus", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseChangedValidatorStatus is a log parse operation binding the contract event 0xcd35267e7654194727477d6c78b541a553483cff7f92a055d17868d3da6e953e.
//
// Solidity: event ChangedValidatorStatus(uint256 indexed validatorID, uint256 status)
func (_Contract *ContractFilterer) ParseChangedValidatorStatus(log types.Log) (*ContractChangedValidatorStatus, error) {
	event := new(ContractChangedValidatorStatus)
	if err := _Contract.contract.UnpackLog(event, "ChangedValidatorStatus", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractClaimedRewardsIterator is returned from FilterClaimedRewards and is used to iterate over the raw logs and unpacked data for ClaimedRewards events raised by the Contract contract.
type ContractClaimedRewardsIterator struct {
	Event *ContractClaimedRewards // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ContractClaimedRewardsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractClaimedRewards)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ContractClaimedRewards)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ContractClaimedRewardsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractClaimedRewardsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractClaimedRewards represents a ClaimedRewards event raised by the Contract contract.
type ContractClaimedRewards struct {
	Delegator     common.Address
	ToValidatorID *big.Int
	Rewards       *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterClaimedRewards is a free log retrieval operation binding the contract event 0x70de20a533702af05c8faf1637846c4586a021bbc71b6928b089b6d555e4fbc2.
//
// Solidity: event ClaimedRewards(address indexed delegator, uint256 indexed toValidatorID, uint256 rewards)
func (_Contract *ContractFilterer) FilterClaimedRewards(opts *bind.FilterOpts, delegator []common.Address, toValidatorID []*big.Int) (*ContractClaimedRewardsIterator, error) {

	var delegatorRule []interface{}
	for _, delegatorItem := range delegator {
		delegatorRule = append(delegatorRule, delegatorItem)
	}
	var toValidatorIDRule []interface{}
	for _, toValidatorIDItem := range toValidatorID {
		toValidatorIDRule = append(toValidatorIDRule, toValidatorIDItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "ClaimedRewards", delegatorRule, toValidatorIDRule)
	if err != nil {
		return nil, err
	}
	return &ContractClaimedRewardsIterator{contract: _Contract.contract, event: "ClaimedRewards", logs: logs, sub: sub}, nil
}

// WatchClaimedRewards is a free log subscription operation binding the contract event 0x70de20a533702af05c8faf1637846c4586a021bbc71b6928b089b6d555e4fbc2.
//
// Solidity: event ClaimedRewards(address indexed delegator, uint256 indexed toValidatorID, uint256 rewards)
func (_Contract *ContractFilterer) WatchClaimedRewards(opts *bind.WatchOpts, sink chan<- *ContractClaimedRewards, delegator []common.Address, toValidatorID []*big.Int) (event.Subscription, error) {

	var delegatorRule []interface{}
	for _, delegatorItem := range delegator {
		delegatorRule = append(delegatorRule, delegatorItem)
	}
	var toValidatorIDRule []interface{}
	for _, toValidatorIDItem := range toValidatorID {
		toValidatorIDRule = append(toValidatorIDRule, toValidatorIDItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "ClaimedRewards", delegatorRule, toValidatorIDRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractClaimedRewards)
				if err := _Contract.contract.UnpackLog(event, "ClaimedRewards", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseClaimedRewards is a log parse operation binding the contract event 0x70de20a533702af05c8faf1637846c4586a021bbc71b6928b089b6d555e4fbc2.
//
// Solidity: event ClaimedRewards(address indexed delegator, uint256 indexed toValidatorID, uint256 rewards)
func (_Contract *ContractFilterer) ParseClaimedRewards(log types.Log) (*ContractClaimedRewards, error) {
	event := new(ContractClaimedRewards)
	if err := _Contract.contract.UnpackLog(event, "ClaimedRewards", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractCreatedValidatorIterator is returned from FilterCreatedValidator and is used to iterate over the raw logs and unpacked data for CreatedValidator events raised by the Contract contract.
type ContractCreatedValidatorIterator struct {
	Event *ContractCreatedValidator // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ContractCreatedValidatorIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractCreatedValidator)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ContractCreatedValidator)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ContractCreatedValidatorIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractCreatedValidatorIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractCreatedValidator represents a CreatedValidator event raised by the Contract contract.
type ContractCreatedValidator struct {
	ValidatorID  *big.Int
	Auth         common.Address
	CreatedEpoch *big.Int
	CreatedTime  *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterCreatedValidator is a free log retrieval operation binding the contract event 0x49bca1ed2666922f9f1690c26a569e1299c2a715fe57647d77e81adfabbf25bf.
//
// Solidity: event CreatedValidator(uint256 indexed validatorID, address indexed auth, uint256 createdEpoch, uint256 createdTime)
func (_Contract *ContractFilterer) FilterCreatedValidator(opts *bind.FilterOpts, validatorID []*big.Int, auth []common.Address) (*ContractCreatedValidatorIterator, error) {

	var validatorIDRule []interface{}
	for _, validatorIDItem := range validatorID {
		validatorIDRule = append(validatorIDRule, validatorIDItem)
	}
	var authRule []interface{}
	for _, authItem := range auth {
		authRule = append(authRule, authItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "CreatedValidator", validatorIDRule, authRule)
	if err != nil {
		return nil, err
	}
	return &ContractCreatedValidatorIterator{contract: _Contract.contract, event: "CreatedValidator", logs: logs, sub: sub}, nil
}

// WatchCreatedValidator is a free log subscription operation binding the contract event 0x49bca1ed2666922f9f1690c26a569e1299c2a715fe57647d77e81adfabbf25bf.
//
// Solidity: event CreatedValidator(uint256 indexed validatorID, address indexed auth, uint256 createdEpoch, uint256 createdTime)
func (_Contract *ContractFilterer) WatchCreatedValidator(opts *bind.WatchOpts, sink chan<- *ContractCreatedValidator, validatorID []*big.Int, auth []common.Address) (event.Subscription, error) {

	var validatorIDRule []interface{}
	for _, validatorIDItem := range validatorID {
		validatorIDRule = append(validatorIDRule, validatorIDItem)
	}
	var authRule []interface{}
	for _, authItem := range auth {
		authRule = append(authRule, authItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "CreatedValidator", validatorIDRule, authRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractCreatedValidator)
				if err := _Contract.contract.UnpackLog(event, "CreatedValidator", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseCreatedValidator is a log parse operation binding the contract event 0x49bca1ed2666922f9f1690c26a569e1299c2a715fe57647d77e81adfabbf25bf.
//
// Solidity: event CreatedValidator(uint256 indexed validatorID, address indexed auth, uint256 createdEpoch, uint256 createdTime)
func (_Contract *ContractFilterer) ParseCreatedValidator(log types.Log) (*ContractCreatedValidator, error) {
	event := new(ContractCreatedValidator)
	if err := _Contract.contract.UnpackLog(event, "CreatedValidator", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractDeactivatedValidatorIterator is returned from FilterDeactivatedValidator and is used to iterate over the raw logs and unpacked data for DeactivatedValidator events raised by the Contract contract.
type ContractDeactivatedValidatorIterator struct {
	Event *ContractDeactivatedValidator // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ContractDeactivatedValidatorIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractDeactivatedValidator)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ContractDeactivatedValidator)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ContractDeactivatedValidatorIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractDeactivatedValidatorIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractDeactivatedValidator represents a DeactivatedValidator event raised by the Contract contract.
type ContractDeactivatedValidator struct {
	ValidatorID      *big.Int
	DeactivatedEpoch *big.Int
	DeactivatedTime  *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterDeactivatedValidator is a free log retrieval operation binding the contract event 0xac4801c32a6067ff757446524ee4e7a373797278ac3c883eac5c693b4ad72e47.
//
// Solidity: event DeactivatedValidator(uint256 indexed validatorID, uint256 deactivatedEpoch, uint256 deactivatedTime)
func (_Contract *ContractFilterer) FilterDeactivatedValidator(opts *bind.FilterOpts, validatorID []*big.Int) (*ContractDeactivatedValidatorIterator, error) {

	var validatorIDRule []interface{}
	for _, validatorIDItem := range validatorID {
		validatorIDRule = append(validatorIDRule, validatorIDItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "DeactivatedValidator", validatorIDRule)
	if err != nil {
		return nil, err
	}
	return &ContractDeactivatedValidatorIterator{contract: _Contract.contract, event: "DeactivatedValidator", logs: logs, sub: sub}, nil
}

// WatchDeactivatedValidator is a free log subscription operation binding the contract event 0xac4801c32a6067ff757446524ee4e7a373797278ac3c883eac5c693b4ad72e47.
//
// Solidity: event DeactivatedValidator(uint256 indexed validatorID, uint256 deactivatedEpoch, uint256 deactivatedTime)
func (_Contract *ContractFilterer) WatchDeactivatedValidator(opts *bind.WatchOpts, sink chan<- *ContractDeactivatedValidator, validatorID []*big.Int) (event.Subscription, error) {

	var validatorIDRule []interface{}
	for _, validatorIDItem := range validatorID {
		validatorIDRule = append(validatorIDRule, validatorIDItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "DeactivatedValidator", validatorIDRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractDeactivatedValidator)
				if err := _Contract.contract.UnpackLog(event, "DeactivatedValidator", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDeactivatedValidator is a log parse operation binding the contract event 0xac4801c32a6067ff757446524ee4e7a373797278ac3c883eac5c693b4ad72e47.
//
// Solidity: event DeactivatedValidator(uint256 indexed validatorID, uint256 deactivatedEpoch, uint256 deactivatedTime)
func (_Contract *ContractFilterer) ParseDeactivatedValidator(log types.Log) (*ContractDeactivatedValidator, error) {
	event := new(ContractDeactivatedValidator)
	if err := _Contract.contract.UnpackLog(event, "DeactivatedValidator", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractDelegatedIterator is returned from FilterDelegated and is used to iterate over the raw logs and unpacked data for Delegated events raised by the Contract contract.
type ContractDelegatedIterator struct {
	Event *ContractDelegated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ContractDelegatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractDelegated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ContractDelegated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ContractDelegatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractDelegatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractDelegated represents a Delegated event raised by the Contract contract.
type ContractDelegated struct {
	Delegator     common.Address
	ToValidatorID *big.Int
	Amount        *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterDelegated is a free log retrieval operation binding the contract event 0x9a8f44850296624dadfd9c246d17e47171d35727a181bd090aa14bbbe00238bb.
//
// Solidity: event Delegated(address indexed delegator, uint256 indexed toValidatorID, uint256 amount)
func (_Contract *ContractFilterer) FilterDelegated(opts *bind.FilterOpts, delegator []common.Address, toValidatorID []*big.Int) (*ContractDelegatedIterator, error) {

	var delegatorRule []interface{}
	for _, delegatorItem := range delegator {
		delegatorRule = append(delegatorRule, delegatorItem)
	}
	var toValidatorIDRule []interface{}
	for _, toValidatorIDItem := range toValidatorID {
		toValidatorIDRule = append(toValidatorIDRule, toValidatorIDItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "Delegated", delegatorRule, toValidatorIDRule)
	if err != nil {
		return nil, err
	}
	return &ContractDelegatedIterator{contract: _Contract.contract, event: "Delegated", logs: logs, sub: sub}, nil
}

// WatchDelegated is a free log subscription operation binding the contract event 0x9a8f44850296624dadfd9c246d17e47171d35727a181bd090aa14bbbe00238bb.
//
// Solidity: event Delegated(address indexed delegator, uint256 indexed toValidatorID, uint256 amount)
func (_Contract *ContractFilterer) WatchDelegated(opts *bind.WatchOpts, sink chan<- *ContractDelegated, delegator []common.Address, toValidatorID []*big.Int) (event.Subscription, error) {

	var delegatorRule []interface{}
	for _, delegatorItem := range delegator {
		delegatorRule = append(delegatorRule, delegatorItem)
	}
	var toValidatorIDRule []interface{}
	for _, toValidatorIDItem := range toValidatorID {
		toValidatorIDRule = append(toValidatorIDRule, toValidatorIDItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "Delegated", delegatorRule, toValidatorIDRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractDelegated)
				if err := _Contract.contract.UnpackLog(event, "Delegated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDelegated is a log parse operation binding the contract event 0x9a8f44850296624dadfd9c246d17e47171d35727a181bd090aa14bbbe00238bb.
//
// Solidity: event Delegated(address indexed delegator, uint256 indexed toValidatorID, uint256 amount)
func (_Contract *ContractFilterer) ParseDelegated(log types.Log) (*ContractDelegated, error) {
	event := new(ContractDelegated)
	if err := _Contract.contract.UnpackLog(event, "Delegated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the Contract contract.
type ContractInitializedIterator struct {
	Event *ContractInitialized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ContractInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractInitialized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ContractInitialized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ContractInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractInitialized represents a Initialized event raised by the Contract contract.
type ContractInitialized struct {
	Version uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Contract *ContractFilterer) FilterInitialized(opts *bind.FilterOpts) (*ContractInitializedIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &ContractInitializedIterator{contract: _Contract.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Contract *ContractFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *ContractInitialized) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractInitialized)
				if err := _Contract.contract.UnpackLog(event, "Initialized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseInitialized is a log parse operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Contract *ContractFilterer) ParseInitialized(log types.Log) (*ContractInitialized, error) {
	event := new(ContractInitialized)
	if err := _Contract.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Contract contract.
type ContractOwnershipTransferredIterator struct {
	Event *ContractOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ContractOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ContractOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ContractOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractOwnershipTransferred represents a OwnershipTransferred event raised by the Contract contract.
type ContractOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Contract *ContractFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*ContractOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ContractOwnershipTransferredIterator{contract: _Contract.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Contract *ContractFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *ContractOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractOwnershipTransferred)
				if err := _Contract.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Contract *ContractFilterer) ParseOwnershipTransferred(log types.Log) (*ContractOwnershipTransferred, error) {
	event := new(ContractOwnershipTransferred)
	if err := _Contract.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractRefundedSlashedLegacyDelegationIterator is returned from FilterRefundedSlashedLegacyDelegation and is used to iterate over the raw logs and unpacked data for RefundedSlashedLegacyDelegation events raised by the Contract contract.
type ContractRefundedSlashedLegacyDelegationIterator struct {
	Event *ContractRefundedSlashedLegacyDelegation // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ContractRefundedSlashedLegacyDelegationIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractRefundedSlashedLegacyDelegation)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ContractRefundedSlashedLegacyDelegation)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ContractRefundedSlashedLegacyDelegationIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractRefundedSlashedLegacyDelegationIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractRefundedSlashedLegacyDelegation represents a RefundedSlashedLegacyDelegation event raised by the Contract contract.
type ContractRefundedSlashedLegacyDelegation struct {
	Delegator   common.Address
	ValidatorID *big.Int
	Amount      *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterRefundedSlashedLegacyDelegation is a free log retrieval operation binding the contract event 0x172fdfaf5222519d28d2794b7617be033f46d954f9b6c3896e7d2611ff444252.
//
// Solidity: event RefundedSlashedLegacyDelegation(address indexed delegator, uint256 indexed validatorID, uint256 amount)
func (_Contract *ContractFilterer) FilterRefundedSlashedLegacyDelegation(opts *bind.FilterOpts, delegator []common.Address, validatorID []*big.Int) (*ContractRefundedSlashedLegacyDelegationIterator, error) {

	var delegatorRule []interface{}
	for _, delegatorItem := range delegator {
		delegatorRule = append(delegatorRule, delegatorItem)
	}
	var validatorIDRule []interface{}
	for _, validatorIDItem := range validatorID {
		validatorIDRule = append(validatorIDRule, validatorIDItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "RefundedSlashedLegacyDelegation", delegatorRule, validatorIDRule)
	if err != nil {
		return nil, err
	}
	return &ContractRefundedSlashedLegacyDelegationIterator{contract: _Contract.contract, event: "RefundedSlashedLegacyDelegation", logs: logs, sub: sub}, nil
}

// WatchRefundedSlashedLegacyDelegation is a free log subscription operation binding the contract event 0x172fdfaf5222519d28d2794b7617be033f46d954f9b6c3896e7d2611ff444252.
//
// Solidity: event RefundedSlashedLegacyDelegation(address indexed delegator, uint256 indexed validatorID, uint256 amount)
func (_Contract *ContractFilterer) WatchRefundedSlashedLegacyDelegation(opts *bind.WatchOpts, sink chan<- *ContractRefundedSlashedLegacyDelegation, delegator []common.Address, validatorID []*big.Int) (event.Subscription, error) {

	var delegatorRule []interface{}
	for _, delegatorItem := range delegator {
		delegatorRule = append(delegatorRule, delegatorItem)
	}
	var validatorIDRule []interface{}
	for _, validatorIDItem := range validatorID {
		validatorIDRule = append(validatorIDRule, validatorIDItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "RefundedSlashedLegacyDelegation", delegatorRule, validatorIDRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractRefundedSlashedLegacyDelegation)
				if err := _Contract.contract.UnpackLog(event, "RefundedSlashedLegacyDelegation", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRefundedSlashedLegacyDelegation is a log parse operation binding the contract event 0x172fdfaf5222519d28d2794b7617be033f46d954f9b6c3896e7d2611ff444252.
//
// Solidity: event RefundedSlashedLegacyDelegation(address indexed delegator, uint256 indexed validatorID, uint256 amount)
func (_Contract *ContractFilterer) ParseRefundedSlashedLegacyDelegation(log types.Log) (*ContractRefundedSlashedLegacyDelegation, error) {
	event := new(ContractRefundedSlashedLegacyDelegation)
	if err := _Contract.contract.UnpackLog(event, "RefundedSlashedLegacyDelegation", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractRestakedRewardsIterator is returned from FilterRestakedRewards and is used to iterate over the raw logs and unpacked data for RestakedRewards events raised by the Contract contract.
type ContractRestakedRewardsIterator struct {
	Event *ContractRestakedRewards // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ContractRestakedRewardsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractRestakedRewards)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ContractRestakedRewards)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ContractRestakedRewardsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractRestakedRewardsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractRestakedRewards represents a RestakedRewards event raised by the Contract contract.
type ContractRestakedRewards struct {
	Delegator     common.Address
	ToValidatorID *big.Int
	Rewards       *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterRestakedRewards is a free log retrieval operation binding the contract event 0x663e0f63f4fc6b01be195c4b56111fd6f14b947d6264497119b08daf77e26da5.
//
// Solidity: event RestakedRewards(address indexed delegator, uint256 indexed toValidatorID, uint256 rewards)
func (_Contract *ContractFilterer) FilterRestakedRewards(opts *bind.FilterOpts, delegator []common.Address, toValidatorID []*big.Int) (*ContractRestakedRewardsIterator, error) {

	var delegatorRule []interface{}
	for _, delegatorItem := range delegator {
		delegatorRule = append(delegatorRule, delegatorItem)
	}
	var toValidatorIDRule []interface{}
	for _, toValidatorIDItem := range toValidatorID {
		toValidatorIDRule = append(toValidatorIDRule, toValidatorIDItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "RestakedRewards", delegatorRule, toValidatorIDRule)
	if err != nil {
		return nil, err
	}
	return &ContractRestakedRewardsIterator{contract: _Contract.contract, event: "RestakedRewards", logs: logs, sub: sub}, nil
}

// WatchRestakedRewards is a free log subscription operation binding the contract event 0x663e0f63f4fc6b01be195c4b56111fd6f14b947d6264497119b08daf77e26da5.
//
// Solidity: event RestakedRewards(address indexed delegator, uint256 indexed toValidatorID, uint256 rewards)
func (_Contract *ContractFilterer) WatchRestakedRewards(opts *bind.WatchOpts, sink chan<- *ContractRestakedRewards, delegator []common.Address, toValidatorID []*big.Int) (event.Subscription, error) {

	var delegatorRule []interface{}
	for _, delegatorItem := range delegator {
		delegatorRule = append(delegatorRule, delegatorItem)
	}
	var toValidatorIDRule []interface{}
	for _, toValidatorIDItem := range toValidatorID {
		toValidatorIDRule = append(toValidatorIDRule, toValidatorIDItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "RestakedRewards", delegatorRule, toValidatorIDRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractRestakedRewards)
				if err := _Contract.contract.UnpackLog(event, "RestakedRewards", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRestakedRewards is a log parse operation binding the contract event 0x663e0f63f4fc6b01be195c4b56111fd6f14b947d6264497119b08daf77e26da5.
//
// Solidity: event RestakedRewards(address indexed delegator, uint256 indexed toValidatorID, uint256 rewards)
func (_Contract *ContractFilterer) ParseRestakedRewards(log types.Log) (*ContractRestakedRewards, error) {
	event := new(ContractRestakedRewards)
	if err := _Contract.contract.UnpackLog(event, "RestakedRewards", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractUndelegatedIterator is returned from FilterUndelegated and is used to iterate over the raw logs and unpacked data for Undelegated events raised by the Contract contract.
type ContractUndelegatedIterator struct {
	Event *ContractUndelegated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ContractUndelegatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractUndelegated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ContractUndelegated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ContractUndelegatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractUndelegatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractUndelegated represents a Undelegated event raised by the Contract contract.
type ContractUndelegated struct {
	Delegator     common.Address
	ToValidatorID *big.Int
	WrID          *big.Int
	Amount        *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterUndelegated is a free log retrieval operation binding the contract event 0xd3bb4e423fbea695d16b982f9f682dc5f35152e5411646a8a5a79a6b02ba8d57.
//
// Solidity: event Undelegated(address indexed delegator, uint256 indexed toValidatorID, uint256 indexed wrID, uint256 amount)
func (_Contract *ContractFilterer) FilterUndelegated(opts *bind.FilterOpts, delegator []common.Address, toValidatorID []*big.Int, wrID []*big.Int) (*ContractUndelegatedIterator, error) {

	var delegatorRule []interface{}
	for _, delegatorItem := range delegator {
		delegatorRule = append(delegatorRule, delegatorItem)
	}
	var toValidatorIDRule []interface{}
	for _, toValidatorIDItem := range toValidatorID {
		toValidatorIDRule = append(toValidatorIDRule, toValidatorIDItem)
	}
	var wrIDRule []interface{}
	for _, wrIDItem := range wrID {
		wrIDRule = append(wrIDRule, wrIDItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "Undelegated", delegatorRule, toValidatorIDRule, wrIDRule)
	if err != nil {
		return nil, err
	}
	return &ContractUndelegatedIterator{contract: _Contract.contract, event: "Undelegated", logs: logs, sub: sub}, nil
}

// WatchUndelegated is a free log subscription operation binding the contract event 0xd3bb4e423fbea695d16b982f9f682dc5f35152e5411646a8a5a79a6b02ba8d57.
//
// Solidity: event Undelegated(address indexed delegator, uint256 indexed toValidatorID, uint256 indexed wrID, uint256 amount)
func (_Contract *ContractFilterer) WatchUndelegated(opts *bind.WatchOpts, sink chan<- *ContractUndelegated, delegator []common.Address, toValidatorID []*big.Int, wrID []*big.Int) (event.Subscription, error) {

	var delegatorRule []interface{}
	for _, delegatorItem := range delegator {
		delegatorRule = append(delegatorRule, delegatorItem)
	}
	var toValidatorIDRule []interface{}
	for _, toValidatorIDItem := range toValidatorID {
		toValidatorIDRule = append(toValidatorIDRule, toValidatorIDItem)
	}
	var wrIDRule []interface{}
	for _, wrIDItem := range wrID {
		wrIDRule = append(wrIDRule, wrIDItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "Undelegated", delegatorRule, toValidatorIDRule, wrIDRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractUndelegated)
				if err := _Contract.contract.UnpackLog(event, "Undelegated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUndelegated is a log parse operation binding the contract event 0xd3bb4e423fbea695d16b982f9f682dc5f35152e5411646a8a5a79a6b02ba8d57.
//
// Solidity: event Undelegated(address indexed delegator, uint256 indexed toValidatorID, uint256 indexed wrID, uint256 amount)
func (_Contract *ContractFilterer) ParseUndelegated(log types.Log) (*ContractUndelegated, error) {
	event := new(ContractUndelegated)
	if err := _Contract.contract.UnpackLog(event, "Undelegated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractUpdatedSlashingRefundRatioIterator is returned from FilterUpdatedSlashingRefundRatio and is used to iterate over the raw logs and unpacked data for UpdatedSlashingRefundRatio events raised by the Contract contract.
type ContractUpdatedSlashingRefundRatioIterator struct {
	Event *ContractUpdatedSlashingRefundRatio // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ContractUpdatedSlashingRefundRatioIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractUpdatedSlashingRefundRatio)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ContractUpdatedSlashingRefundRatio)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ContractUpdatedSlashingRefundRatioIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractUpdatedSlashingRefundRatioIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractUpdatedSlashingRefundRatio represents a UpdatedSlashingRefundRatio event raised by the Contract contract.
type ContractUpdatedSlashingRefundRatio struct {
	ValidatorID *big.Int
	RefundRatio *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterUpdatedSlashingRefundRatio is a free log retrieval operation binding the contract event 0x047575f43f09a7a093d94ec483064acfc61b7e25c0de28017da442abf99cb917.
//
// Solidity: event UpdatedSlashingRefundRatio(uint256 indexed validatorID, uint256 refundRatio)
func (_Contract *ContractFilterer) FilterUpdatedSlashingRefundRatio(opts *bind.FilterOpts, validatorID []*big.Int) (*ContractUpdatedSlashingRefundRatioIterator, error) {

	var validatorIDRule []interface{}
	for _, validatorIDItem := range validatorID {
		validatorIDRule = append(validatorIDRule, validatorIDItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "UpdatedSlashingRefundRatio", validatorIDRule)
	if err != nil {
		return nil, err
	}
	return &ContractUpdatedSlashingRefundRatioIterator{contract: _Contract.contract, event: "UpdatedSlashingRefundRatio", logs: logs, sub: sub}, nil
}

// WatchUpdatedSlashingRefundRatio is a free log subscription operation binding the contract event 0x047575f43f09a7a093d94ec483064acfc61b7e25c0de28017da442abf99cb917.
//
// Solidity: event UpdatedSlashingRefundRatio(uint256 indexed validatorID, uint256 refundRatio)
func (_Contract *ContractFilterer) WatchUpdatedSlashingRefundRatio(opts *bind.WatchOpts, sink chan<- *ContractUpdatedSlashingRefundRatio, validatorID []*big.Int) (event.Subscription, error) {

	var validatorIDRule []interface{}
	for _, validatorIDItem := range validatorID {
		validatorIDRule = append(validatorIDRule, validatorIDItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "UpdatedSlashingRefundRatio", validatorIDRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractUpdatedSlashingRefundRatio)
				if err := _Contract.contract.UnpackLog(event, "UpdatedSlashingRefundRatio", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUpdatedSlashingRefundRatio is a log parse operation binding the contract event 0x047575f43f09a7a093d94ec483064acfc61b7e25c0de28017da442abf99cb917.
//
// Solidity: event UpdatedSlashingRefundRatio(uint256 indexed validatorID, uint256 refundRatio)
func (_Contract *ContractFilterer) ParseUpdatedSlashingRefundRatio(log types.Log) (*ContractUpdatedSlashingRefundRatio, error) {
	event := new(ContractUpdatedSlashingRefundRatio)
	if err := _Contract.contract.UnpackLog(event, "UpdatedSlashingRefundRatio", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractUpgradedIterator is returned from FilterUpgraded and is used to iterate over the raw logs and unpacked data for Upgraded events raised by the Contract contract.
type ContractUpgradedIterator struct {
	Event *ContractUpgraded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ContractUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractUpgraded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ContractUpgraded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ContractUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractUpgraded represents a Upgraded event raised by the Contract contract.
type ContractUpgraded struct {
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpgraded is a free log retrieval operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_Contract *ContractFilterer) FilterUpgraded(opts *bind.FilterOpts, implementation []common.Address) (*ContractUpgradedIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return &ContractUpgradedIterator{contract: _Contract.contract, event: "Upgraded", logs: logs, sub: sub}, nil
}

// WatchUpgraded is a free log subscription operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_Contract *ContractFilterer) WatchUpgraded(opts *bind.WatchOpts, sink chan<- *ContractUpgraded, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractUpgraded)
				if err := _Contract.contract.UnpackLog(event, "Upgraded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUpgraded is a log parse operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_Contract *ContractFilterer) ParseUpgraded(log types.Log) (*ContractUpgraded, error) {
	event := new(ContractUpgraded)
	if err := _Contract.contract.UnpackLog(event, "Upgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractWithdrawnIterator is returned from FilterWithdrawn and is used to iterate over the raw logs and unpacked data for Withdrawn events raised by the Contract contract.
type ContractWithdrawnIterator struct {
	Event *ContractWithdrawn // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ContractWithdrawnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractWithdrawn)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ContractWithdrawn)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ContractWithdrawnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractWithdrawnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractWithdrawn represents a Withdrawn event raised by the Contract contract.
type ContractWithdrawn struct {
	Delegator     common.Address
	ToValidatorID *big.Int
	WrID          *big.Int
	Amount        *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterWithdrawn is a free log retrieval operation binding the contract event 0x75e161b3e824b114fc1a33274bd7091918dd4e639cede50b78b15a4eea956a21.
//
// Solidity: event Withdrawn(address indexed delegator, uint256 indexed toValidatorID, uint256 indexed wrID, uint256 amount)
func (_Contract *ContractFilterer) FilterWithdrawn(opts *bind.FilterOpts, delegator []common.Address, toValidatorID []*big.Int, wrID []*big.Int) (*ContractWithdrawnIterator, error) {

	var delegatorRule []interface{}
	for _, delegatorItem := range delegator {
		delegatorRule = append(delegatorRule, delegatorItem)
	}
	var toValidatorIDRule []interface{}
	for _, toValidatorIDItem := range toValidatorID {
		toValidatorIDRule = append(toValidatorIDRule, toValidatorIDItem)
	}
	var wrIDRule []interface{}
	for _, wrIDItem := range wrID {
		wrIDRule = append(wrIDRule, wrIDItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "Withdrawn", delegatorRule, toValidatorIDRule, wrIDRule)
	if err != nil {
		return nil, err
	}
	return &ContractWithdrawnIterator{contract: _Contract.contract, event: "Withdrawn", logs: logs, sub: sub}, nil
}

// WatchWithdrawn is a free log subscription operation binding the contract event 0x75e161b3e824b114fc1a33274bd7091918dd4e639cede50b78b15a4eea956a21.
//
// Solidity: event Withdrawn(address indexed delegator, uint256 indexed toValidatorID, uint256 indexed wrID, uint256 amount)
func (_Contract *ContractFilterer) WatchWithdrawn(opts *bind.WatchOpts, sink chan<- *ContractWithdrawn, delegator []common.Address, toValidatorID []*big.Int, wrID []*big.Int) (event.Subscription, error) {

	var delegatorRule []interface{}
	for _, delegatorItem := range delegator {
		delegatorRule = append(delegatorRule, delegatorItem)
	}
	var toValidatorIDRule []interface{}
	for _, toValidatorIDItem := range toValidatorID {
		toValidatorIDRule = append(toValidatorIDRule, toValidatorIDItem)
	}
	var wrIDRule []interface{}
	for _, wrIDItem := range wrID {
		wrIDRule = append(wrIDRule, wrIDItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "Withdrawn", delegatorRule, toValidatorIDRule, wrIDRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractWithdrawn)
				if err := _Contract.contract.UnpackLog(event, "Withdrawn", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseWithdrawn is a log parse operation binding the contract event 0x75e161b3e824b114fc1a33274bd7091918dd4e639cede50b78b15a4eea956a21.
//
// Solidity: event Withdrawn(address indexed delegator, uint256 indexed toValidatorID, uint256 indexed wrID, uint256 amount)
func (_Contract *ContractFilterer) ParseWithdrawn(log types.Log) (*ContractWithdrawn, error) {
	event := new(ContractWithdrawn)
	if err := _Contract.contract.UnpackLog(event, "Withdrawn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

var ContractBinRuntime = "0x6080604052600436106103b6575f3560e01c80638cddb015116101e9578063c7be95de11610108578063df00c9221161009d578063e9a505a71161006d578063e9a505a714610dc6578063ebdf104c14610de5578063f2fde38b14610e04578063fb36025f14610e23575f5ffd5b8063df00c92214610d14578063e08d7e6614610d4e578063e261641a14610d6d578063e880a15914610da7575f5ffd5b8063d46fa518116100d8578063d46fa51814610c70578063d725e91f14610c8d578063db5ca3e514610cac578063dc31e1af14610cda575f5ffd5b8063c7be95de14610be7578063cc17278414610bfc578063cc8343aa14610c1b578063cfd4766314610c3a575f5ffd5b8063aa5d82921161017e578063b88a37e21161014e578063b88a37e214610b31578063c3de580e14610b5d578063c5f956af14610b9d578063c65ee0e114610bbc575f5ffd5b8063aa5d8292146109e4578063ad3cb1cc14610a3f578063b0ef386c14610a6f578063b5d8962714610a8e575f5ffd5b8063a198d229116101b9578063a198d22914610942578063a5a470ad1461097c578063a86a056f1461098f578063a8ab09ba146109c5575f5ffd5b80638cddb015146108b55780638da5cb5b146108d457806390a6c475146109105780639fa6dd351461092f575f5ffd5b806352d1902d116102d5578063736de9ae1161026a578063841e45611161023a578063841e456114610836578063854873e114610855578063860c2750146108815780638b0e9f3f146108a0575f5ffd5b8063736de9ae146107ba57806376671808146107ee57806376fed43a146108025780637cacb1d614610821575f5ffd5b80636099ecb2116102a55780636099ecb21461070c57806361e53fcc1461072b5780636f49866314610765578063715018a6146107a6575f5ffd5b806352d1902d1461069c57806354fd4d50146106b05780635601fe01146106ce57806358f95b80146106ed575f5ffd5b806339b80c001161034b57806346f1ca351161031b57806346f1ca351461062c5780634f1ef2861461064b5780634f7c4efb1461065e5780634f864df41461067d575f5ffd5b806339b80c001461053c5780633fbfd4df146105ba578063441a3e70146105d9578063468f35ee146105f8575f5ffd5b806318160ddd1161038657806318160ddd1461048d5780631e702f83146104a25780631f270152146104c157806328f7314814610527575f5ffd5b80630135b1db146103d857806308c3687414610416578063093b41d0146104375780630962ef791461046e575f5ffd5b366103d45760405163ab064ad360e01b815260040160405180910390fd5b5f5ffd5b3480156103e3575f5ffd5b506104036103f2366004614299565b60036020525f908152604090205481565b6040519081526020015b60405180910390f35b348015610421575f5ffd5b506104356104303660046142b2565b610e4e565b005b348015610442575f5ffd5b50601154610456906001600160a01b031681565b6040516001600160a01b03909116815260200161040d565b348015610479575f5ffd5b506104356104883660046142b2565b610eb0565b348015610498575f5ffd5b50610403600c5481565b3480156104ad575f5ffd5b506104356104bc3660046142c9565b610f80565b3480156104cc575f5ffd5b5061050c6104db3660046142e9565b600a60209081525f938452604080852082529284528284209052825290208054600182015460029092015490919083565b6040805193845260208401929092529082015260600161040d565b348015610532575f5ffd5b5061040360075481565b348015610547575f5ffd5b5061058d6105563660046142b2565b600d60208190525f918252604090912060088101546009820154600a830154600b840154600c850154949095015492949193909286565b604080519687526020870195909552938501929092526060840152608083015260a082015260c00161040d565b3480156105c5575f5ffd5b506104356105d4366004614319565b611009565b3480156105e4575f5ffd5b506104356105f33660046142c9565b611172565b348015610603575f5ffd5b50610456610612366004614299565b60146020525f90815260409020546001600160a01b031681565b348015610637575f5ffd5b50610435610646366004614299565b61118a565b610435610659366004614380565b6111c2565b348015610669575f5ffd5b506104356106783660046142c9565b6111dd565b348015610688575f5ffd5b50610435610697366004614441565b61128c565b3480156106a7575f5ffd5b506104036113d9565b3480156106bb575f5ffd5b50604051600160fa1b815260200161040d565b3480156106d9575f5ffd5b506104036106e83660046142b2565b6113f4565b3480156106f8575f5ffd5b506104036107073660046142c9565b611426565b348015610717575f5ffd5b5061040361072636600461446a565b611446565b348015610736575f5ffd5b506104036107453660046142c9565b5f918252600d602090815260408084209284526001909201905290205490565b348015610770575f5ffd5b5061040361077f36600461446a565b6001600160a01b03919091165f908152600860209081526040808320938352929052205490565b3480156107b1575f5ffd5b5061043561148b565b3480156107c5575f5ffd5b506104566107d4366004614299565b60156020525f90815260409020546001600160a01b031681565b3480156107f9575f5ffd5b5061040361149e565b34801561080d575f5ffd5b5061043561081c3660046144d6565b6114b3565b34801561082c575f5ffd5b5061040360015481565b348015610841575f5ffd5b50610435610850366004614299565b611505565b348015610860575f5ffd5b5061087461086f3660046142b2565b61152f565b60405161040d9190614560565b34801561088c575f5ffd5b5061043561089b366004614299565b6115c6565b3480156108ab575f5ffd5b5061040360065481565b3480156108c0575f5ffd5b506104356108cf36600461446a565b6115f0565b3480156108df575f5ffd5b507f9016d09d72d40fdae2fd8ceac6b6234c7706214fd39c1cd1e609a0528c199300546001600160a01b0316610456565b34801561091b575f5ffd5b5061043561092a3660046142b2565b611617565b61043561093d3660046142b2565b61162b565b34801561094d575f5ffd5b5061040361095c3660046142c9565b5f918252600d602090815260408084209284526006909201905290205490565b61043561098a366004614572565b611636565b34801561099a575f5ffd5b506104036109a936600461446a565b600960209081525f928352604080842090915290825290205481565b3480156109d0575f5ffd5b506104356109df3660046142e9565b61177d565b3480156109ef575f5ffd5b50610a276109fe3660046142c9565b5f918252600d60209081526040808420928452600390920190529020546001600160401b031690565b6040516001600160401b03909116815260200161040d565b348015610a4a575f5ffd5b50610874604051806040016040528060058152602001640352e302e360dc1b81525081565b348015610a7a575f5ffd5b50610435610a89366004614299565b6117bc565b348015610a99575f5ffd5b50610aee610aa83660046142b2565b600260208190525f918252604090912080546001820154928201546003830154600484015460058501546006909501549395946001600160a01b03909316939192909187565b6040805197885260208801969096526001600160a01b03909416948601949094526060850191909152608084015260a083019190915260c082015260e00161040d565b348015610b3c575f5ffd5b50610b50610b4b3660046142b2565b611814565b60405161040d91906145b0565b348015610b68575f5ffd5b50610b8d610b773660046142b2565b5f90815260026020526040902054608016151590565b604051901515815260200161040d565b348015610ba8575f5ffd5b50600f54610456906001600160a01b031681565b348015610bc7575f5ffd5b50610403610bd63660046142b2565b600e6020525f908152604090205481565b348015610bf2575f5ffd5b5061040360055481565b348015610c07575f5ffd5b50610435610c163660046145f2565b611876565b348015610c26575f5ffd5b50610435610c35366004614623565b61193c565b348015610c45575f5ffd5b50610403610c5436600461446a565b600b60209081525f928352604080842090915290825290205481565b348015610c7b575f5ffd5b506010546001600160a01b0316610456565b348015610c98575f5ffd5b50610435610ca7366004614299565b611a66565b348015610cb7575f5ffd5b50610403610cc63660046142b2565b5f908152600d602052604090206009015490565b348015610ce5575f5ffd5b50610403610cf43660046142c9565b5f918252600d602090815260408084209284526004909201905290205490565b348015610d1f575f5ffd5b50610403610d2e3660046142c9565b5f918252600d602090815260408084209284526002909201905290205490565b348015610d59575f5ffd5b50610435610d68366004614695565b611b0c565b348015610d78575f5ffd5b50610403610d873660046142c9565b5f918252600d602090815260408084209284526005909201905290205490565b348015610db2575f5ffd5b50610435610dc1366004614299565b611bcd565b348015610dd1575f5ffd5b50601354610456906001600160a01b031681565b348015610df0575f5ffd5b50610435610dff3660046146c7565b611bf7565b348015610e0f575f5ffd5b50610435610e1e366004614299565b611e9d565b348015610e2e575f5ffd5b50610403610e3d366004614299565b60126020525f908152604090205481565b335f610e5a8284611edc565b9050610e67828483611f5f565b82826001600160a01b03167f663e0f63f4fc6b01be195c4b56111fd6f14b947d6264497119b08daf77e26da583604051610ea391815260200190565b60405180910390a3505050565b335f610ebc8284611edc565b90505f610ec883611fec565b6001600160a01b0316826040515f6040518083038185875af1925050503d805f8114610f0f576040519150601f19603f3d011682016040523d82523d5f602084013e610f14565b606091505b5050905080610f36576040516312171d8360e31b815260040160405180910390fd5b83836001600160a01b03167f70de20a533702af05c8faf1637846c4586a021bbc71b6928b089b6d555e4fbc284604051610f7291815260200190565b60405180910390a350505050565b5f546001600160a01b03163314610faa5760405163c78d372960e01b815260040160405180910390fd5b80610fc85760405163396bd83560e21b815260040160405180910390fd5b610fd28282612014565b610fdc825f61193c565b5f828152600260208190526040822001546001600160a01b0316906110049082908190612116565b505050565b7ff0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a008054600160401b810460ff1615906001600160401b03165f8115801561104d5750825b90505f826001600160401b031660011480156110685750303b155b905081158015611076575080155b156110945760405163f92ee8a960e01b815260040160405180910390fd5b845467ffffffffffffffff1916600117855583156110be57845460ff60401b1916600160401b1785555b6110c7866121f2565b6110cf612203565b60018a90555f80546001600160a01b03808b166001600160a01b03199283161790925560108054928a1692909116919091179055600c89905561110f4290565b5f8b8152600d6020526040902060080155831561116657845460ff60401b19168555604051600181527fc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d29060200160405180910390a15b50505050505050505050565b61118633838361118133611fec565b61220b565b5050565b6040516001600160a01b0382169033907f857125196131cfcd709c738c6d1fd2701ce70f2a03785aeadae6f4b47fe73c1d905f90a350565b6111ca61259c565b6111d382612640565b6111868282612648565b6111e5612704565b5f82815260026020526040902054608016611213576040516321b6a8f960e11b815260040160405180910390fd5b670de0b6b3a764000081111561123c576040516357c70d6360e01b815260040160405180910390fd5b5f828152600e6020526040908190208290555182907f047575f43f09a7a093d94ec483064acfc61b7e25c0de28017da442abf99cb917906112809084815260200190565b60405180910390a25050565b33611297818561275f565b50815f036112b857604051631f2a200560e01b815260040160405180910390fd5b6001600160a01b0381165f908152600a602090815260408083208784528252808320868452909152902060020154156113045760405163756f5c2d60e11b815260040160405180910390fd5b61131481858460015f60016127cd565b6001600160a01b0381165f908152600a602090815260408083208784528252808320868452909152902060020182905561134c61149e565b6001600160a01b0382165f908152600a60209081526040808320888452825280832087845290915281209182554260019092019190915561138e90859061193c565b8284826001600160a01b03167fd3bb4e423fbea695d16b982f9f682dc5f35152e5411646a8a5a79a6b02ba8d57856040516113cb91815260200190565b60405180910390a450505050565b5f6113e261299e565b505f516020614b825f395f51905f5290565b5f818152600260208181526040808420909201546001600160a01b03168352600b815281832093835292909252205490565b5f828152600d602090815260408083208484529091529020545b92915050565b5f5f61145284846129e7565b6001600160a01b0385165f9081526008602090815260408083208784529091529020549091506114839082906147a5565b949350505050565b611493612704565b61149c5f612a51565b565b5f60015460016114ae91906147a5565b905090565b5f546001600160a01b031633146114dd5760405163c78d372960e01b815260040160405180910390fd5b6114ee858585855f5f875f5f612ac1565b6005548411156114fe5760058490555b5050505050565b61150d612704565b600f80546001600160a01b0319166001600160a01b0392909216919091179055565b60046020525f908152604090208054611547906147b8565b80601f0160208091040260200160405190810160405280929190818152602001828054611573906147b8565b80156115be5780601f10611595576101008083540402835291602001916115be565b820191905f5260205f20905b8154815290600101906020018083116115a157829003601f168201915b505050505081565b6115ce612704565b601080546001600160a01b0319166001600160a01b0392909216919091179055565b6115fa828261275f565b6111865760405163208e0a4160e11b815260040160405180910390fd5b61161f612704565b61162881612c6c565b50565b611628338234611f5f565b60105f9054906101000a90046001600160a01b03166001600160a01b031663c5f530af6040518163ffffffff1660e01b8152600401602060405180830381865afa158015611686573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906116aa91906147f0565b3410156116ca5760405163047447a360e11b815260040160405180910390fd5b604281141580611703575081815f8181106116e7576116e7614807565b9050013560f81c60f81b6001600160f81b03191660c060f81b14155b15611721576040516338497f4960e11b815260040160405180910390fd5b60125f61172e8484612cd3565b6001600160a01b0316815260208101919091526040015f2054156117655760405163028aeb6760e21b815260040160405180910390fd5b611770338383612cff565b6111863360055434611f5f565b5f546001600160a01b031633146117a75760405163c78d372960e01b815260040160405180910390fd5b6117b38383835f612d2d565b61100481612e7b565b6117c4612704565b6013546001600160a01b038083169116036117f257604051639b92bed360e01b815260040160405180910390fd5b601380546001600160a01b0319166001600160a01b0392909216919091179055565b5f818152600d602090815260409182902060070180548351818402810184019094528084526060939283018282801561186a57602002820191905f5260205f20905b815481526020019060010190808311611856575b50505050509050919050565b6013546001600160a01b031633146118a15760405163ea8e4eb560e01b815260040160405180910390fd5b6001600160a01b038281165f908152601560205260409020548183169116036118dd5760405163eb81e1a360e01b815260040160405180910390fd5b806001600160a01b0316826001600160a01b03160361190f5760405163367558c360e01b815260040160405180910390fd5b6001600160a01b039182165f90815260146020526040902080546001600160a01b03191691909216179055565b5f8281526002602052604090206004015461196a57604051635926e0c360e01b815260040160405180910390fd5b5f828152600260205260409020600181015490541561198657505f5b5f5460405163520337df60e11b815260048101859052602481018390526001600160a01b039091169063a4066fbe906044015f604051808303815f87803b1580156119cf575f5ffd5b505af11580156119e1573d5f5f3e3d5ffd5b505050508180156119f157508015155b15611004575f805484825260046020819052604092839020925163242a6e3f60e01b81526001600160a01b039092169263242a6e3f92611a34928892910161481b565b5f604051808303815f87803b158015611a4b575f5ffd5b505af1158015611a5d573d5f5f3e3d5ffd5b50505050505050565b336001600160a01b038216611a8e5760405163d92e233d60e01b815260040160405180910390fd5b6001600160a01b038181165f90815260146020526040902054811690831614611aca57604051630fe3b3c160e31b815260040160405180910390fd5b6001600160a01b039081165f9081526015602090815260408083208054949095166001600160a01b031994851617909455601490529190912080549091169055565b5f546001600160a01b03163314611b365760405163c78d372960e01b815260040160405180910390fd5b5f600d5f611b4261149e565b81526020019081526020015f2090505f5f90505b82811015611bb8575f848483818110611b7157611b71614807565b602090810292909201355f81815260028452604080822060010154948890529020839055600c860154909350611ba9915082906147a5565b600c8501555050600101611b56565b50611bc7600782018484614221565b50505050565b611bd5612704565b601180546001600160a01b0319166001600160a01b0392909216919091179055565b5f546001600160a01b03163314611c215760405163c78d372960e01b815260040160405180910390fd5b5f600d5f611c2d61149e565b81526020019081526020015f2090505f81600701805480602002602001604051908101604052809291908181526020018280548015611c8957602002820191905f5260205f20905b815481526020019060010190808311611c75575b50505050509050611d0e82828c8c808060200260200160405190810160405280939291908181526020018383602002808284375f81840152601f19601f820116905080830192505050505050508b8b808060200260200160405190810160405280939291908181526020018383602002808284375f92019190915250612eed92505050565b600180545f908152600d602052604090206008810154909190421115611d40576008820154611d3d90426148a9565b90505b611dc0818584868c8c808060200260200160405190810160405280939291908181526020018383602002808284375f81840152601f19601f820116905080830192505050505050508b8b808060200260200160405190810160405280939291908181526020018383602002808284375f9201919091525061310792505050565b611dff818584868c8c808060200260200160405190810160405280939291908181526020018383602002808284375f9201919091525061382592505050565b5050611e0961149e565b6001554260088301554360098301556010546040805163d9a7c1f960e01b815290516001600160a01b039092169163d9a7c1f9916004808201926020929091908290030181865afa158015611e60573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190611e8491906147f0565b600b83015550600c54600d909101555050505050505050565b611ea5612704565b6001600160a01b038116611ed357604051631e4fbdf760e01b81525f60048201526024015b60405180910390fd5b61162881612a51565b5f611ee7838361275f565b506001600160a01b0383165f90815260086020908152604080832085845290915281205490819003611f2c5760405163899aaa9d60e01b815260040160405180910390fd5b6001600160a01b0384165f908152600860209081526040808320868452909152812055611f5881612e7b565b9392505050565b5f82815260026020526040902060040154611f8d57604051635926e0c360e01b815260040160405180910390fd5b5f8281526002602052604090205415611fb9576040516353670afb60e11b815260040160405180910390fd5b611fc68383836001612d2d565b611fcf82613aa3565b6110045760405163c2eb4ead60e01b815260040160405180910390fd5b6001600160a01b038082165f9081526015602052604081205490911680611440575090919050565b5f8281526002602052604090205415801561202e57508015155b15612055575f8281526002602052604090206001015460075461205191906148a9565b6007555b5f82815260026020526040902054811115611186575f8281526002602052604081208281556006015490036120e45761208c61149e565b5f8381526002602090815260409182902060068101849055426005909101819055825193845290830152805184927fac4801c32a6067ff757446524ee4e7a373797278ac3c883eac5c693b4ad72e4792908290030190a25b817fcd35267e7654194727477d6c78b541a553483cff7f92a055d17868d3da6e953e8260405161128091815260200190565b6011546001600160a01b031615611004576011546040516001600160a01b03858116602483015284811660448301525f921690627a12009060640160408051601f198184030181529181526020820180516001600160e01b0316631fbcb08360e11b1790525161218691906148bc565b5f604051808303815f8787f1925050503d805f81146121c0576040519150601f19603f3d011682016040523d82523d5f602084013e6121c5565b606091505b50509050801580156121d45750815b15611bc7576040516347b4be6960e11b815260040160405180910390fd5b6121fa613b5a565b61162881613ba3565b61149c613b5a565b6001600160a01b0384165f908152600a60209081526040808320868452825280832085845282528083208151606081018352815480825260018301549482019490945260029091015491810191909152910361227a57604051630fe3b3c160e31b815260040160405180910390fd5b60208082015182515f8781526002909352604090922060050154909190158015906122b457505f8681526002602052604090206005015482115b156122d45750505f84815260026020526040902060058101546006909101545b60105f9054906101000a90046001600160a01b03166001600160a01b031663b82b84276040518163ffffffff1660e01b8152600401602060405180830381865afa158015612324573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061234891906147f0565b61235290836147a5565b42101561237257604051635ada9a9960e01b815260040160405180910390fd5b60105f9054906101000a90046001600160a01b03166001600160a01b031663650acd666040518163ffffffff1660e01b8152600401602060405180830381865afa1580156123c2573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906123e691906147f0565b6123f090826147a5565b6123f861149e565b1015612417576040516323ea994d60e01b815260040160405180910390fd5b6001600160a01b0387165f908152600a60209081526040808320898452825280832088845282528083206002908101548a855290835281842054600e9093529083205490926080909216151591906124729084908490613bab565b6001600160a01b038b165f908152600a602090815260408083208d845282528083208c84529091528120818155600181018290556002015590508083116124cc576040516318f967fb60e01b815260040160405180910390fd5b5f6001600160a01b0388166124e183866148a9565b6040515f81818185875af1925050503d805f811461251a576040519150601f19603f3d011682016040523d82523d5f602084013e61251f565b606091505b5050905080612541576040516312171d8360e31b815260040160405180910390fd5b61254a82612c6c565b888a8c6001600160a01b03167f75e161b3e824b114fc1a33274bd7091918dd4e639cede50b78b15a4eea956a218760405161258791815260200190565b60405180910390a45050505050505050505050565b306001600160a01b037f0000000000000000000000002685751d3c7a49ebf485e823079ac65e2a35a3dd16148061262257507f0000000000000000000000002685751d3c7a49ebf485e823079ac65e2a35a3dd6001600160a01b03166126165f516020614b825f395f51905f52546001600160a01b031690565b6001600160a01b031614155b1561149c5760405163703e46dd60e11b815260040160405180910390fd5b611628612704565b816001600160a01b03166352d1902d6040518163ffffffff1660e01b8152600401602060405180830381865afa9250505080156126a2575060408051601f3d908101601f1916820190925261269f918101906147f0565b60015b6126ca57604051634c9c8ce360e01b81526001600160a01b0383166004820152602401611eca565b5f516020614b825f395f51905f5281146126fa57604051632a87526960e21b815260048101829052602401611eca565b6110048383613c11565b336127367f9016d09d72d40fdae2fd8ceac6b6234c7706214fd39c1cd1e609a0528c199300546001600160a01b031690565b6001600160a01b03161461149c5760405163118cdaa760e01b8152336004820152602401611eca565b5f5f61276b84846129e7565b905061277683613c66565b6001600160a01b0385165f8181526009602090815260408083208884528252808320949094559181526008825282812086825290915290812080548392906127bf9084906147a5565b909155505015159392505050565b6001600160a01b0386165f908152600b60209081526040808320888452909152812080548692906127ff9084906148a9565b90915550505f858152600260205260409020600101546128209085906148a9565b5f8681526002602052604090206001015560065461283f9085906148a9565b6006555f85815260026020526040902054612866578360075461286291906148a9565b6007555b5f612870866113f4565b9050801580159061288c57505f86815260026020526040902054155b1561296c5760105f9054906101000a90046001600160a01b03166001600160a01b031663c5f530af6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156128e1573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061290591906147f0565b81101561293657821561292b5760405163047447a360e11b815260040160405180910390fd5b612936866001612014565b818015612949575061294786613aa3565b155b156129675760405163c2eb4ead60e01b815260040160405180910390fd5b612977565b612977866001612014565b5f8681526002602081905260409091200154611a5d9088906001600160a01b031686612116565b306001600160a01b037f0000000000000000000000002685751d3c7a49ebf485e823079ac65e2a35a3dd161461149c5760405163703e46dd60e11b815260040160405180910390fd5b6001600160a01b0382165f90815260096020908152604080832084845290915281205481612a1484613c66565b6001600160a01b0386165f908152600b60209081526040808320888452909152812054919250612a4682878686613cbb565b979650505050505050565b7f9016d09d72d40fdae2fd8ceac6b6234c7706214fd39c1cd1e609a0528c19930080546001600160a01b031981166001600160a01b03848116918217845560405192169182907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0905f90a3505050565b6001600160a01b0389165f9081526003602052604090205415612af757604051633f4dc7d360e11b815260040160405180910390fd5b6001600160a01b0389165f8181526003602081815260408084208d90558c845260028083528185208b81559384018a905560048085018a90556005850188905560068501899055930180546001600160a01b0319169095179094555220612b5f878983614916565b508760125f612b6e8a8a612cd3565b6001600160a01b03166001600160a01b031681526020019081526020015f2081905550886001600160a01b0316887f49bca1ed2666922f9f1690c26a569e1299c2a715fe57647d77e81adfabbf25bf8686604051612bd6929190918252602082015260400190565b60405180910390a38115612c2057604080518381526020810183905289917fac4801c32a6067ff757446524ee4e7a373797278ac3c883eac5c693b4ad72e47910160405180910390a25b8415612c6157877fcd35267e7654194727477d6c78b541a553483cff7f92a055d17868d3da6e953e86604051612c5891815260200190565b60405180910390a25b505050505050505050565b8015611628576040515f9082156108fc0290839083818181858288f19350505050158015612c9c573d5f5f3e3d5ffd5b506040518181527f8918bd6046d08b314e457977f29562c5d76a7030d79b1edba66e8a5da0b77ae89060200160405180910390a150565b5f612ce182600281866149cf565b604051612cef9291906149f6565b6040519081900390209392505050565b5f60055f8154612d0e90614a05565b91829055509050611bc7848285855f612d2561149e565b425f5f612ac1565b815f03612d4d57604051631f2a200560e01b815260040160405180910390fd5b612d57848461275f565b506001600160a01b0384165f908152600b60209081526040808320868452909152902054612d869083906147a5565b6001600160a01b0385165f908152600b60209081526040808320878452825280832093909355600290522060010154612dbf83826147a5565b5f85815260026020526040902060010155600654612dde9084906147a5565b6006555f84815260026020526040902054612e055782600754612e0191906147a5565b6007555b612e1084821561193c565b83856001600160a01b03167f9a8f44850296624dadfd9c246d17e47171d35727a181bd090aa14bbbe00238bb85604051612e4c91815260200190565b60405180910390a35f84815260026020819052604090912001546114fe9086906001600160a01b031684612116565b5f546040516366e7ea0f60e01b8152306004820152602481018390526001600160a01b03909116906366e7ea0f906044015f604051808303815f87803b158015612ec3575f5ffd5b505af1158015612ed5573d5f5f3e3d5ffd5b5050505080600c54612ee791906147a5565b600c5550565b5f5b83518110156114fe5760105f9054906101000a90046001600160a01b03166001600160a01b0316635a68f01a6040518163ffffffff1660e01b8152600401602060405180830381865afa158015612f48573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190612f6c91906147f0565b828281518110612f7e57612f7e614807565b602002602001015111801561301e575060105f9054906101000a90046001600160a01b03166001600160a01b031662cc7f836040518163ffffffff1660e01b8152600401602060405180830381865afa158015612fdd573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061300191906147f0565b83828151811061301357613013614807565b602002602001015110155b1561306a5761304784828151811061303857613038614807565b60200260200101516008612014565b61306a84828151811061305c5761305c614807565b60200260200101515f61193c565b82818151811061307c5761307c614807565b6020026020010151856005015f86848151811061309b5761309b614807565b602002602001015181526020019081526020015f20819055508181815181106130c6576130c6614807565b6020026020010151856006015f8684815181106130e5576130e5614807565b60209081029190910181015182528101919091526040015f2055600101612eef565b5f6040518060a0016040528085516001600160401b0381111561312c5761312c61436c565b604051908082528060200260200182016040528015613155578160200160208202803683370190505b5081526020015f815260200185516001600160401b0381111561317a5761317a61436c565b6040519080825280602002602001820160405280156131a3578160200160208202803683370190505b5081526020015f81526020015f81525090505f5f90505b84518110156132e3575f866004015f8784815181106131db576131db614807565b602002602001015181526020019081526020015f205490505f5f90508185848151811061320a5761320a614807565b60200260200101511115613240578185848151811061322b5761322b614807565b602002602001015161323d91906148a9565b90505b8986848151811061325357613253614807565b6020026020010151826132669190614a1d565b6132709190614a48565b8460400151848151811061328657613286614807565b602002602001018181525050836040015183815181106132a8576132a8614807565b602002602001015184606001516132bf91906147a5565b606085015260808401516132d49082906147a5565b608085015250506001016131ba565b505f5b84518110156133d0578784828151811061330257613302614807565b60200260200101518986848151811061331d5761331d614807565b60200260200101518a5f015f8a878151811061333b5761333b614807565b602002602001015181526020019081526020015f205461335b9190614a1d565b6133659190614a48565b61336f9190614a1d565b6133799190614a48565b825180518390811061338d5761338d614807565b602090810291909101015281518051829081106133ac576133ac614807565b602002602001015182602001516133c391906147a5565b60208301526001016132e6565b505f5b84518110156136d4575f61347b8960105f9054906101000a90046001600160a01b03166001600160a01b031663d9a7c1f96040518163ffffffff1660e01b8152600401602060405180830381865afa158015613431573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061345591906147f0565b855180518690811061346957613469614807565b60200260200101518660200151613d24565b90506134ad83608001518460400151848151811061349b5761349b614807565b60200260200101518560600151613d5f565b6134b790826147a5565b90505f8683815181106134cc576134cc614807565b6020908102919091018101515f8181526002808452604080832090910154601054825163a778651560e01b815292519496506001600160a01b03918216959394613564948994929093169263a778651592600480820193918290030181865afa15801561353b573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061355f91906147f0565b613eb0565b6001600160a01b0383165f908152600b6020908152604080832087845290915290205490915080156135c8576001600160a01b0383165f908152600860209081526040808320878452909152812080548492906135c29084906147a5565b90915550505b5f6135d383876148a9565b5f86815260026020526040812060010154919250811561360d5781613600670de0b6b3a764000085614a1d565b61360a9190614a48565b90505b5f87815260018f0160205260409020546136289082906147a5565b8f6001015f8981526020019081526020015f20819055508a898151811061365157613651614807565b60200260200101518f6004015f8981526020019081526020015f20819055508b898151811061368257613682614807565b60200260200101518e6002015f8981526020019081526020015f20546136a891906147a5565b8f6002015f8981526020019081526020015f2081905550505050505050505080806001019150506133d3565b506080810151600a8701819055600c54111561370a5785600a0154600c5f8282546136ff91906148a9565b9091555061370f9050565b5f600c555b600f546001600160a01b031615611a5d575f670de0b6b3a764000060105f9054906101000a90046001600160a01b03166001600160a01b03166394c3e9146040518163ffffffff1660e01b8152600401602060405180830381865afa15801561377a573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061379e91906147f0565b83608001516137ad9190614a1d565b6137b79190614a48565b90506137c281612e7b565b600f546040515f916001600160a01b031690620f424090849084818181858888f193505050503d805f8114613812576040519150601f19603f3d011682016040523d82523d5f602084013e613817565b606091505b505050505050505050505050565b5f5b8251811015613a9b575f83828151811061384357613843614807565b602002602001015190505f8761385e670de0b6b3a764000090565b85858151811061387057613870614807565b60200260200101516138829190614a1d565b61388c9190614a48565b9050670de0b6b3a76400008111156138a95750670de0b6b3a76400005b5f8281526003870160209081526040808320815160608101835290546001600160401b038116825263ffffffff600160401b8204811694830194909452600160601b900490921690820152906138ff8383613ece565b5f85815260038b0160209081526040918290208351815485840151868601516001600160401b039093166bffffffffffffffffffffffff1990921691909117600160401b63ffffffff928316021763ffffffff60601b1916600160601b91909216021790556010548251631c25433760e01b815292519394506001600160a01b031692631c2543379260048082019392918290030181865afa1580156139a7573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906139cb9190614a5b565b6001600160401b0316815f01516001600160401b0316108015613a71575060105f9054906101000a90046001600160a01b03166001600160a01b0316633fa225486040518163ffffffff1660e01b8152600401602060405180830381865afa158015613a39573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190613a5d9190614a81565b63ffffffff16816040015163ffffffff1610155b15613a8b57613a81846010612014565b613a8b845f61193c565b5050600190920191506138279050565b505050505050565b5f670de0b6b3a764000060105f9054906101000a90046001600160a01b03166001600160a01b0316632265f2846040518163ffffffff1660e01b8152600401602060405180830381865afa158015613afd573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190613b2191906147f0565b613b2a846113f4565b613b349190614a1d565b613b3e9190614a48565b5f92835260026020526040909220600101549190911115919050565b7ff0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a0054600160401b900460ff1661149c57604051631afcd79f60e31b815260040160405180910390fd5b611ea5613b5a565b5f821580613bc15750670de0b6b3a76400008210155b15613bcd57505f611f58565b670de0b6b3a7640000613be083826148a9565b613bea9086614a1d565b613bf49190614a48565b613bff9060016147a5565b905083811115611f5857509192915050565b613c1a826140b1565b6040516001600160a01b038316907fbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b905f90a2805115613c5e576110048282614114565b61118661417d565b5f8181526002602052604081206006015415613cb3575f828152600260205260409020600601546001541015613c9e57505060015490565b505f9081526002602052604090206006015490565b505060015490565b5f818310613cca57505f611483565b5f838152600d6020818152604080842088855260019081018352818520548786529383528185208986520190915290912054670de0b6b3a764000087613d1084846148a9565b613d1a9190614a1d565b612a469190614a48565b5f825f03613d3357505f611483565b5f613d3e8587614a1d565b905082613d4b8583614a1d565b613d559190614a48565b9695505050505050565b5f825f03613d6e57505f611f58565b5f82613d7a8587614a1d565b613d849190614a48565b9050670de0b6b3a764000060105f9054906101000a90046001600160a01b03166001600160a01b03166394c3e9146040518163ffffffff1660e01b8152600401602060405180830381865afa158015613ddf573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190613e0391906147f0565b60105f9054906101000a90046001600160a01b03166001600160a01b031663c74dd6216040518163ffffffff1660e01b8152600401602060405180830381865afa158015613e53573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190613e7791906147f0565b613e8990670de0b6b3a76400006148a9565b613e9391906148a9565b613e9d9083614a1d565b613ea79190614a48565b95945050505050565b5f670de0b6b3a7640000613ec48385614a1d565b611f589190614a48565b604080516060810182525f8082526020820181905291810191909152604080516060810182525f8082526020820181905291810191909152826040015163ffffffff165f03613f31576001600160401b0384168152600160408201529050611440565b5f83604001516001613f439190614aa4565b63ffffffff1690505f846020015163ffffffff16866001600160401b0316865f01516001600160401b0316600185613f7b9190614ac0565b613f859190614adf565b613f8f9190614b08565b613f999190614b08565b9050613fa58282614b27565b6001600160401b03168352613fba8282614b54565b63ffffffff166020840152670de0b6b3a764000083516001600160401b03161115613feb57670de0b6b3a764000083525b60105f9054906101000a90046001600160a01b03166001600160a01b0316633fa225486040518163ffffffff1660e01b8152600401602060405180830381865afa15801561403b573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061405f9190614a81565b63ffffffff16856040015163ffffffff161015614096576040850151614086906001614aa4565b63ffffffff1660408401526140a7565b60408086015163ffffffff16908401525b5090949350505050565b806001600160a01b03163b5f036140e657604051634c9c8ce360e01b81526001600160a01b0382166004820152602401611eca565b5f516020614b825f395f51905f5280546001600160a01b0319166001600160a01b0392909216919091179055565b60605f5f846001600160a01b03168460405161413091906148bc565b5f60405180830381855af49150503d805f8114614168576040519150601f19603f3d011682016040523d82523d5f602084013e61416d565b606091505b5091509150613ea785838361419c565b341561149c5760405163b398979f60e01b815260040160405180910390fd5b6060826141b1576141ac826141f8565b611f58565b81511580156141c857506001600160a01b0384163b155b156141f157604051639996b31560e01b81526001600160a01b0385166004820152602401611eca565b5080611f58565b8051156142085780518082602001fd5b60405163d6bda27560e01b815260040160405180910390fd5b828054828255905f5260205f2090810192821561425a579160200282015b8281111561425a57823582559160200191906001019061423f565b5061426692915061426a565b5090565b5b80821115614266575f815560010161426b565b80356001600160a01b0381168114614294575f5ffd5b919050565b5f602082840312156142a9575f5ffd5b611f588261427e565b5f602082840312156142c2575f5ffd5b5035919050565b5f5f604083850312156142da575f5ffd5b50508035926020909101359150565b5f5f5f606084860312156142fb575f5ffd5b6143048461427e565b95602085013595506040909401359392505050565b5f5f5f5f5f60a0868803121561432d575f5ffd5b85359450602086013593506143446040870161427e565b92506143526060870161427e565b91506143606080870161427e565b90509295509295909350565b634e487b7160e01b5f52604160045260245ffd5b5f5f60408385031215614391575f5ffd5b61439a8361427e565b915060208301356001600160401b038111156143b4575f5ffd5b8301601f810185136143c4575f5ffd5b80356001600160401b038111156143dd576143dd61436c565b604051601f8201601f19908116603f011681016001600160401b038111828210171561440b5761440b61436c565b604052818152828201602001871015614422575f5ffd5b816020840160208301375f602083830101528093505050509250929050565b5f5f5f60608486031215614453575f5ffd5b505081359360208301359350604090920135919050565b5f5f6040838503121561447b575f5ffd5b6144848361427e565b946020939093013593505050565b5f5f83601f8401126144a2575f5ffd5b5081356001600160401b038111156144b8575f5ffd5b6020830191508360208285010111156144cf575f5ffd5b9250929050565b5f5f5f5f5f608086880312156144ea575f5ffd5b6144f38661427e565b94506020860135935060408601356001600160401b03811115614514575f5ffd5b61452088828901614492565b96999598509660600135949350505050565b5f81518084528060208401602086015e5f602082860101526020601f19601f83011685010191505092915050565b602081525f611f586020830184614532565b5f5f60208385031215614583575f5ffd5b82356001600160401b03811115614598575f5ffd5b6145a485828601614492565b90969095509350505050565b602080825282518282018190525f918401906040840190835b818110156145e75783518352602093840193909201916001016145c9565b509095945050505050565b5f5f60408385031215614603575f5ffd5b61460c8361427e565b915061461a6020840161427e565b90509250929050565b5f5f60408385031215614634575f5ffd5b823591506020830135801515811461464a575f5ffd5b809150509250929050565b5f5f83601f840112614665575f5ffd5b5081356001600160401b0381111561467b575f5ffd5b6020830191508360208260051b85010111156144cf575f5ffd5b5f5f602083850312156146a6575f5ffd5b82356001600160401b038111156146bb575f5ffd5b6145a485828601614655565b5f5f5f5f5f5f5f5f6080898b0312156146de575f5ffd5b88356001600160401b038111156146f3575f5ffd5b6146ff8b828c01614655565b90995097505060208901356001600160401b0381111561471d575f5ffd5b6147298b828c01614655565b90975095505060408901356001600160401b03811115614747575f5ffd5b6147538b828c01614655565b90955093505060608901356001600160401b03811115614771575f5ffd5b61477d8b828c01614655565b999c989b5096995094979396929594505050565b634e487b7160e01b5f52601160045260245ffd5b8082018082111561144057611440614791565b600181811c908216806147cc57607f821691505b6020821081036147ea57634e487b7160e01b5f52602260045260245ffd5b50919050565b5f60208284031215614800575f5ffd5b5051919050565b634e487b7160e01b5f52603260045260245ffd5b828152604060208201525f5f8354614832816147b8565b806040860152600182165f8114614850576001811461486c5761489d565b60ff1983166060870152606082151560051b870101935061489d565b865f5260205f205f5b8381101561489457815488820160600152600190910190602001614875565b87016060019450505b50919695505050505050565b8181038181111561144057611440614791565b5f82518060208501845e5f920191825250919050565b601f82111561100457805f5260205f20601f840160051c810160208510156148f75750805b601f840160051c820191505b818110156114fe575f8155600101614903565b6001600160401b0383111561492d5761492d61436c565b6149418361493b83546147b8565b836148d2565b5f601f841160018114614972575f851561495b5750838201355b5f19600387901b1c1916600186901b1783556114fe565b5f83815260208120601f198716915b828110156149a15786850135825560209485019460019092019101614981565b50868210156149bd575f1960f88860031b161c19848701351681555b505060018560011b0183555050505050565b5f5f858511156149dd575f5ffd5b838611156149e9575f5ffd5b5050820193919092039150565b818382375f9101908152919050565b5f60018201614a1657614a16614791565b5060010190565b808202811582820484141761144057611440614791565b634e487b7160e01b5f52601260045260245ffd5b5f82614a5657614a56614a34565b500490565b5f60208284031215614a6b575f5ffd5b81516001600160401b0381168114611f58575f5ffd5b5f60208284031215614a91575f5ffd5b815163ffffffff81168114611f58575f5ffd5b63ffffffff818116838216019081111561144057611440614791565b6001600160801b03828116828216039081111561144057611440614791565b6001600160801b038181168382160290811690818114614b0157614b01614791565b5092915050565b6001600160801b03818116838216019081111561144057611440614791565b5f6001600160801b03831680614b3f57614b3f614a34565b806001600160801b0384160491505092915050565b5f6001600160801b03831680614b6c57614b6c614a34565b806001600160801b038416069150509291505056fe360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbca2646970667358221220b85ce566241cb74ec11834f70798de9be5c9b0494cacd0dd034911314e5aff8364736f6c634300081b0033"
