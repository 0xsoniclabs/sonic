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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"

	priorities "github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	subsidies "github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// configResponse is the answer of eth_config (EIP-7910). Sonic activates its
// upgrades at block heights, not at times, so there is never a next one.
type configResponse struct {
	Current *forkConfig `json:"current"`
	Next    *forkConfig `json:"next"`
	Last    *forkConfig `json:"last"`
}

// forkConfig is the configuration of one upgrade (EIP-7910).
type forkConfig struct {
	ActivationTime uint64 `json:"activationTime"`
	// BlockHeight is the block that activated the configuration, an extension
	// of EIP-7910: Sonic activates an upgrade with a transaction, so its block
	// identifies the fork.
	BlockHeight *hexutil.Big `json:"blockHeight"`
	// BlobSchedule reports no blob capacity; EIP-7910 requires an object.
	BlobSchedule    params.BlobConfig `json:"blobSchedule"`
	ChainID         *hexutil.Big      `json:"chainId"`
	ForkID          hexutil.Bytes     `json:"forkId"`
	Precompiles     contractRegistry  `json:"precompiles"`
	SystemContracts contractRegistry  `json:"systemContracts"`
}

type contractRegistry map[string]common.Address

func (handler *handlers) config(ctx context.Context, _ dispatch.Arguments) (any, error) {
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	heights := handler.environment.ChainInfo().UpgradeHeights
	if len(heights) == 0 {
		return nil, errors.New("no configs found")
	}
	current, err := handler.forkConfig(head, heights[len(heights)-1])
	if err != nil {
		return nil, fmt.Errorf("failed to get current config, %w", err)
	}
	response := &configResponse{Current: current}
	if len(heights) > 1 {
		response.Last, err = handler.forkConfig(head, heights[len(heights)-2])
		if err != nil {
			return nil, fmt.Errorf("failed to get previous config, %w", err)
		}
	}
	return response, nil
}

// forkConfig returns the configuration an upgrade activated.
func (handler *handlers) forkConfig(head exec.Head, upgrade opera.UpgradeHeight) (*forkConfig, error) {
	info := handler.environment.ChainInfo()
	forkID, err := makeForkID(upgrade, info.GenesisID)
	if err != nil {
		return nil, fmt.Errorf("could not make fork id, %v", err)
	}
	height := uint64(upgrade.Height)
	if height > math.MaxInt64 {
		return nil, fmt.Errorf("upgrade height %d exceeds the largest addressable block number", height)
	}
	header, err := handler.environment.HeaderAt(head, height)
	if exec.Absent(err) {
		return nil, fmt.Errorf("block %d not found to determine activation time", height)
	}
	if err != nil {
		return nil, fmt.Errorf("could not get block %d to determine activation time, %v", height, err)
	}
	activationTime := uint64(header.Time.Unix())
	rules := handler.environment.ChainConfig(height).Rules(header.Number, true, activationTime)
	precompiles := contractRegistry{}
	for address, contract := range vm.ActivePrecompiledContracts(rules) {
		precompiles[contract.Name()] = address
	}
	return &forkConfig{
		ActivationTime:  activationTime,
		BlockHeight:     (*hexutil.Big)(new(big.Int).Set(header.Number)),
		ChainID:         (*hexutil.Big)(handler.environment.ChainID()),
		ForkID:          forkID[:],
		Precompiles:     precompiles,
		SystemContracts: activeSystemContracts(upgrade.Upgrades),
	}, nil
}

// activeSystemContracts names the system contracts an upgrade activates.
func activeSystemContracts(upgrades opera.Upgrades) contractRegistry {
	contracts := contractRegistry{}
	if upgrades.Allegro {
		contracts["HISTORY_STORAGE_ADDRESS"] = params.HistoryStorageAddress
	}
	if upgrades.GasSubsidies {
		contracts["GAS_SUBSIDY_REGISTRY_ADDRESS"] = subsidies.GetAddress()
	}
	if upgrades.TransactionPriorities {
		contracts["TRANSACTION_PRIORITY_REGISTRY_ADDRESS"] = priorities.GetAddress()
	}
	return contracts
}

// makeForkID derives the fork id of an upgrade:
// CRC32(genesisId || bigEndian(upgrade.Height) || Rlp(upgrade)).
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
