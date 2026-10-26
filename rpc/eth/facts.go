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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"

	priorityRegistry "github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	subsidiesRegistry "github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// syncedAge is the age of the newest block below which the node reports it
// is not syncing; it is far above the longest emission interval.
const syncedAge = 90 * time.Minute

func (a *api) chainID(ctx context.Context, _ dispatch.None) (any, error) {
	return (*hexutil.Big)(a.env.ChainID()), nil
}

func (a *api) blockNumber(ctx context.Context, _ dispatch.None) (any, error) {
	return hexutil.Uint64(a.env.Latest(ctx).Number()), nil
}

func (a *api) syncing(ctx context.Context, _ dispatch.None) (any, error) {
	progress := a.node.Progress()
	if time.Since(progress.CurrentBlockTime.Time()) <= syncedAge {
		return false, nil
	}
	return map[string]any{
		"startingBlock":    hexutil.Uint64(0), // back-compatibility
		"currentEpoch":     hexutil.Uint64(progress.CurrentEpoch),
		"currentBlock":     hexutil.Uint64(progress.CurrentBlock),
		"currentBlockHash": progress.CurrentBlockHash.Hex(),
		"currentBlockTime": hexutil.Uint64(progress.CurrentBlockTime),
		"highestBlock":     hexutil.Uint64(progress.HighestBlock),
		"highestEpoch":     hexutil.Uint64(progress.HighestEpoch),
		"pulledStates":     hexutil.Uint64(0), // back-compatibility
		"knownStates":      hexutil.Uint64(0), // back-compatibility
	}, nil
}

func (a *api) currentEpoch(ctx context.Context, _ dispatch.None) (any, error) {
	return hexutil.Uint64(a.node.Progress().CurrentEpoch), nil
}

type blockNumberArg struct {
	Block chain.BlockNumber `rpc:"block"`
}

// getRules answers the rules in force at a block. The tags that select the
// head answer the rules the next block is processed with, as the open epoch
// did in the reference; a block the node does not have is null.
func (a *api) getRules(ctx context.Context, p blockNumberArg) (any, error) {
	head := a.env.Latest(ctx)
	switch p.Block {
	case chain.LatestBlockNumber, chain.PendingBlockNumber, chain.SafeBlockNumber, chain.FinalizedBlockNumber:
		return exec.NullIfNotFound(a.env.OpenRules(head))
	}
	n, err := a.env.Chain().ResolveNumber(head.Head, p.Block)
	if err != nil {
		return nil, err
	}
	if n > head.Number() {
		return nil, nil
	}
	return exec.NullIfNotFound(a.env.Rules(ctx, n))
}

func zeroAddress(context.Context, dispatch.None) (any, error) {
	return common.Address{}, nil
}

func zeroHashrate(context.Context, dispatch.None) (any, error) {
	return hexutil.Uint64(0), nil
}

// configResponse is the answer of eth_config, as EIP-7910 describes it.
// Sonic activates its configurations by block height, not time, so Next is
// always null.
type configResponse struct {
	Current *forkConfig `json:"current"`
	Next    *forkConfig `json:"next"`
	Last    *forkConfig `json:"last"`
}

type forkConfig struct {
	ActivationTime  uint64             `json:"activationTime"`
	BlockHeight     *hexutil.Big       `json:"blockHeight"`
	BlobSchedule    *params.BlobConfig `json:"blobSchedule"`
	ChainId         *hexutil.Big       `json:"chainId"`
	ForkId          hexutil.Bytes      `json:"forkId"`
	Precompiles     contractRegistry   `json:"precompiles"`
	SystemContracts contractRegistry   `json:"systemContracts"`
}

type contractRegistry map[string]common.Address

// config answers the configuration in force and the one before it.
func (a *api) config(ctx context.Context, _ dispatch.None) (any, error) {
	head := a.env.Latest(ctx)
	info := a.env.ChainInfo()
	heights := info.UpgradeHeights
	if len(heights) == 0 {
		return nil, errors.New("no configs found")
	}
	current, err := a.forkConfig(head, info.ChainID, info.GenesisID, heights[len(heights)-1])
	if err != nil {
		return nil, fmt.Errorf("failed to get current config, %w", err)
	}
	var last *forkConfig
	if len(heights) > 1 {
		if last, err = a.forkConfig(head, info.ChainID, info.GenesisID, heights[len(heights)-2]); err != nil {
			return nil, fmt.Errorf("failed to get previous config, %w", err)
		}
	}
	return &configResponse{Current: current, Last: last}, nil
}

func (a *api) forkConfig(head *exec.Head, chainID *big.Int, genesisID common.Hash, upgrade opera.UpgradeHeight) (*forkConfig, error) {
	height := uint64(upgrade.Height)
	chainCfg := a.env.ChainConfig(height)
	precompiles := make(contractRegistry)
	rules := chainCfg.Rules(new(big.Int).SetUint64(height), true, 0)
	for addr, c := range vm.ActivePrecompiledContracts(rules) {
		precompiles[c.Name()] = addr
	}
	forkID, err := MakeForkId(upgrade, genesisID)
	if err != nil {
		return nil, fmt.Errorf("could not make fork id, %v", err)
	}
	header, err := a.env.HeaderAt(head, height)
	if exec.IsMissing(err) {
		return nil, fmt.Errorf("block %d not found to determine activation time", height)
	}
	if err != nil {
		return nil, fmt.Errorf("could not get block %d to determine activation time, %v", height, err)
	}
	return &forkConfig{
		ActivationTime:  uint64(header.Time.Unix()),
		BlockHeight:     (*hexutil.Big)(header.Number),
		ChainId:         (*hexutil.Big)(chainID),
		ForkId:          forkID[:],
		Precompiles:     precompiles,
		SystemContracts: activeSystemContracts(upgrade.Upgrades),
	}, nil
}

func activeSystemContracts(upgrade opera.Upgrades) contractRegistry {
	contracts := make(contractRegistry)
	if upgrade.Allegro {
		contracts["HISTORY_STORAGE_ADDRESS"] = params.HistoryStorageAddress
	}
	if upgrade.GasSubsidies {
		contracts["GAS_SUBSIDY_REGISTRY_ADDRESS"] = subsidiesRegistry.GetAddress()
	}
	if upgrade.TransactionPriorities {
		contracts["TRANSACTION_PRIORITY_REGISTRY_ADDRESS"] = priorityRegistry.GetAddress()
	}
	return contracts
}

// MakeForkId is the fork id of an upgrade, as EIP-7910 defines it for Sonic:
// CRC32(genesisId || bigEndian(upgrade.Height) || Rlp(upgrade)).
func MakeForkId(upgrade opera.UpgradeHeight, genesisID common.Hash) ([4]byte, error) {
	upgradeRlp, err := rlp.EncodeToBytes(upgrade.Upgrades)
	if err != nil {
		return [4]byte{}, fmt.Errorf("could not encode upgrade to RLP, %v", err)
	}
	id := crc32.ChecksumIEEE(genesisID.Bytes())
	id = crc32.Update(id, crc32.IEEETable, binary.BigEndian.AppendUint64(nil, uint64(upgrade.Height)))
	id = crc32.Update(id, crc32.IEEETable, upgradeRlp)
	return [4]byte(binary.BigEndian.AppendUint32(nil, id)), nil
}
