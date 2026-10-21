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

package eth

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"

	priorityRegistry "github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	subsidiesRegistry "github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

type configResponse struct {
	// current is the config active at the latest block number.
	Current *config `json:"current"`
	// Next will remain nil since Sonic config activation does not depend on time.
	Next *config `json:"next"`
	// Last is the config active before the current one and could be nil if only one upgrades heights exists.
	Last *config `json:"last"`
}

// config as described by https://eips.ethereum.org/EIPS/eip-7910
type config struct {
	// ActivationTime is the timestamp of the first block where this config is active.
	ActivationTime uint64       `json:"activationTime"`
	BlockHeight    *hexutil.Big `json:"blockHeight"`

	// BlobSchedule will remain nil because in Sonic this is not relevant
	BlobSchedule *params.BlobConfig `json:"blobSchedule"`

	ChainId *hexutil.Big `json:"chainId"`

	// ForkId provides an identity scheme to both precisely and concisely
	// summarize the chain's current status. Both regarding its genesis
	// configuration and the last upgrade in use.
	// ForkId is derived with this formula
	// CRC32(genesisId || bigEndian(upgrade.Height) || Rlp(upgrade))
	ForkId hexutil.Bytes `json:"forkId"`

	Precompiles     contractRegistry `json:"precompiles"`
	SystemContracts contractRegistry `json:"systemContracts"`
}

// helper types to improve readability of the returned structure.
type contractRegistry map[string]common.Address

// config implements eth_config: the current and previous (if any) network
// configs following the structure of https://eips.ethereum.org/EIPS/eip-7910.
//
// In Sonic, config changes are based on block heights (upgrade heights) rather
// than activation times. Therefore, the "Next" config is always nil. The
// "Current" config corresponds to the config active at the current block, and
// the "Last" config (if available) to the one before it. BlobSchedule is not
// relevant in Sonic and always nil.
func (a *API) config(ctx context.Context) (*configResponse, error) {
	head := a.chain.Head()
	if head.Header == nil {
		return nil, errors.New("current block header not found")
	}

	info := a.core.ChainInfo()
	updateHeights := info.UpgradeHeights
	if len(updateHeights) == 0 {
		return nil, errors.New("no configs found")
	}

	currentUpgradeHeight := updateHeights[len(updateHeights)-1]
	current, err := a.makeConfigFromUpgrade(info, currentUpgradeHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to get current config, %w", err)
	}

	// if there is an older config, make it too
	var last *config
	if n := len(updateHeights); n > 1 {
		last, err = a.makeConfigFromUpgrade(info, updateHeights[n-2])
		if err != nil {
			return nil, fmt.Errorf("failed to get previous config, %w", err)
		}
	}

	return &configResponse{
		Current: current,
		Last:    last,
	}, nil
}

// makeConfigFromUpgrade constructs the config that was active for the given
// upgrade height.
func (a *API) makeConfigFromUpgrade(info core.ChainInfo, upgradeHeight opera.UpgradeHeight) (*config, error) {
	chainID := new(big.Int).SetUint64(info.ChainID)
	chainCfg := exec.ChainConfig(info, uint64(upgradeHeight.Height))

	precompiled := make(contractRegistry)
	chainCfgRules := chainCfg.Rules(big.NewInt(int64(upgradeHeight.Height)), true, uint64(0))
	for addr, c := range vm.ActivePrecompiledContracts(chainCfgRules) {
		precompiled[c.Name()] = addr
	}

	forkId, err := MakeForkId(upgradeHeight, info.GenesisID)
	if err != nil {
		// this can only fail if RLP encoding fails, which is unexpected
		return nil, fmt.Errorf("could not make fork id, %v", err)
	}

	header, err := a.env.HeaderAt(a.chain.Head(), uint64(upgradeHeight.Height))
	if err != nil {
		return nil, fmt.Errorf("could not get block %d to determine activation time, %v", upgradeHeight.Height, err)
	}
	if header == nil {
		return nil, fmt.Errorf("block %d not found to determine activation time", upgradeHeight.Height)
	}

	return &config{
		// block time needs to be converted to unix timestamp as it is done in
		// evmcore/dummy_block.go in method EvmHeader.EthHeader()
		ActivationTime:  uint64(header.Time.Unix()),
		BlockHeight:     (*hexutil.Big)(header.Number),
		ChainId:         (*hexutil.Big)(chainID),
		ForkId:          forkId[:],
		Precompiles:     precompiled,
		SystemContracts: activeSystemContracts(upgradeHeight.Upgrades),
	}, nil
}

// activeSystemContracts returns a map of system contract names to their addresses
// based on the active upgrade.
func activeSystemContracts(upgrade opera.Upgrades) contractRegistry {
	sysContracts := make(contractRegistry)
	if upgrade.Allegro {
		sysContracts["HISTORY_STORAGE_ADDRESS"] = params.HistoryStorageAddress
	}
	if upgrade.GasSubsidies {
		sysContracts["GAS_SUBSIDY_REGISTRY_ADDRESS"] = subsidiesRegistry.GetAddress()
	}
	if upgrade.TransactionPriorities {
		sysContracts["TRANSACTION_PRIORITY_REGISTRY_ADDRESS"] = priorityRegistry.GetAddress()
	}
	return sysContracts
}

type forkId [4]byte

// MakeForkId creates a fork ID from the given upgrade.
// The Fork ID is calculated as the CRC32 checksum of the RLP encoding of the upgrade,
// the block number, and the genesis ID.
//
// CRC32(genesisId || bigEndian(upgrade.Height) || Rlp(upgrade))
func MakeForkId(upgrade opera.UpgradeHeight, genesisId common.Hash) (forkId, error) {
	upgradeRlp, err := rlp.EncodeToBytes(upgrade.Upgrades)
	if err != nil {
		return forkId{}, fmt.Errorf("could not encode upgrade to RLP, %v", err)
	}

	forkId := crc32.ChecksumIEEE(genesisId.Bytes())
	blockNumberBytes := binary.BigEndian.AppendUint64(nil, uint64(upgrade.Height))
	forkId = crc32.Update(forkId, crc32.IEEETable, blockNumberBytes)
	forkId = crc32.Update(forkId, crc32.IEEETable, upgradeRlp)

	return [4]byte(binary.BigEndian.AppendUint32(nil, forkId)), nil
}
