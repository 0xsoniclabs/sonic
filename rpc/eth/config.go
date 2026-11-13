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
	"math"

	"github.com/0xsoniclabs/sonic/evmcore"
	priorityRegistry "github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	subsidiesRegistry "github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// configResponse is the answer of eth_config, as EIP-7910 describes it. The
// network activates an upgrade by a block, not at a time, so there is never a
// next configuration.
type configResponse struct {
	Current *networkConfig `json:"current"`
	Next    *networkConfig `json:"next"`
	// Last is the configuration before the current one; nil with only one.
	Last *networkConfig `json:"last"`
}

// networkConfig is one configuration of EIP-7910.
type networkConfig struct {
	// ActivationTime is the time of the block that activated it.
	ActivationTime uint64 `json:"activationTime"`
	// BlockHeight is the block that activated it, which EIP-7910 has no field
	// for, describing upgrades activated by time.
	BlockHeight *hexutil.Big `json:"blockHeight"`
	// BlobSchedule is zero: the network supports no blobs, and EIP-7910
	// requires an object.
	BlobSchedule params.BlobConfig `json:"blobSchedule"`
	ChainId      *hexutil.Big      `json:"chainId"`
	// ForkId is CRC32(genesisId || bigEndian(upgrade.Height) || Rlp(upgrade)).
	ForkId          hexutil.Bytes             `json:"forkId"`
	Precompiles     map[string]common.Address `json:"precompiles"`
	SystemContracts map[string]common.Address `json:"systemContracts"`
}

// config answers the configuration active at the head and the one before it.
// An upgrade whose block is above the head is not active yet.
func (h *handlers) config(context.Context, struct{}) (*configResponse, error) {
	head := h.environment.Head()
	info := h.environment.ChainInfo()
	var heights []opera.UpgradeHeight
	for _, height := range info.UpgradeHeights {
		if uint64(height.Height) <= head.NumberU64() {
			heights = append(heights, height)
		}
	}
	if len(heights) == 0 {
		return nil, errors.New("no configs found")
	}
	current, err := h.networkConfig(head, info.GenesisID, heights[len(heights)-1])
	if err != nil {
		return nil, fmt.Errorf("failed to get current config, %w", err)
	}
	response := &configResponse{Current: current}
	if len(heights) > 1 {
		if response.Last, err = h.networkConfig(head, info.GenesisID, heights[len(heights)-2]); err != nil {
			return nil, fmt.Errorf("failed to get previous config, %w", err)
		}
	}
	return response, nil
}

func (h *handlers) networkConfig(head *evmcore.EvmBlock, genesisID common.Hash, upgrade opera.UpgradeHeight) (*networkConfig, error) {
	forkID, err := makeForkID(upgrade, genesisID)
	if err != nil {
		return nil, fmt.Errorf("could not make fork id, %v", err)
	}
	height := uint64(upgrade.Height)
	if height > math.MaxInt64 {
		return nil, fmt.Errorf("upgrade height %d exceeds the largest addressable block number", height)
	}
	header, err := h.environment.HeaderAt(head, height)
	if errors.Is(err, core.ErrNotFound) {
		return nil, fmt.Errorf("block %d not found to determine activation time", height)
	}
	if err != nil {
		return nil, fmt.Errorf("could not get block %d to determine activation time, %v", height, err)
	}
	activationTime := uint64(header.Time.Unix())
	rules := h.environment.ChainConfig(height).Rules(header.Number, true, activationTime)
	precompiles := map[string]common.Address{}
	for address, contract := range vm.ActivePrecompiledContracts(rules) {
		precompiles[contract.Name()] = address
	}
	return &networkConfig{
		ActivationTime:  activationTime,
		BlockHeight:     (*hexutil.Big)(header.Number),
		ChainId:         (*hexutil.Big)(h.environment.ChainID()),
		ForkId:          forkID[:],
		Precompiles:     precompiles,
		SystemContracts: systemContracts(upgrade.Upgrades),
	}, nil
}

// systemContracts names the system contracts the upgrades make active.
func systemContracts(upgrades opera.Upgrades) map[string]common.Address {
	contracts := map[string]common.Address{}
	if upgrades.Allegro {
		contracts["HISTORY_STORAGE_ADDRESS"] = params.HistoryStorageAddress
	}
	if upgrades.GasSubsidies {
		contracts["GAS_SUBSIDY_REGISTRY_ADDRESS"] = subsidiesRegistry.GetAddress()
	}
	if upgrades.TransactionPriorities {
		contracts["TRANSACTION_PRIORITY_REGISTRY_ADDRESS"] = priorityRegistry.GetAddress()
	}
	return contracts
}

// makeForkID is the CRC32 checksum of the genesis id, the big-endian height
// of the upgrade and its RLP encoding.
func makeForkID(upgrade opera.UpgradeHeight, genesisID common.Hash) ([4]byte, error) {
	encoded, err := rlp.EncodeToBytes(upgrade.Upgrades)
	if err != nil {
		return [4]byte{}, fmt.Errorf("could not encode upgrade to RLP, %v", err)
	}
	checksum := crc32.ChecksumIEEE(genesisID.Bytes())
	checksum = crc32.Update(checksum, crc32.IEEETable, binary.BigEndian.AppendUint64(nil, uint64(upgrade.Height)))
	checksum = crc32.Update(checksum, crc32.IEEETable, encoded)
	return [4]byte(binary.BigEndian.AppendUint32(nil, checksum)), nil
}
