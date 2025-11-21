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

package ethapi

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"math/big"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

// makeConfigFromUpgrade constructs the config that was active for the
// given block number based on the upgrade heights.
func makeConfigFromUpgrade(
	ctx context.Context,
	b Backend,
	upgradeHeight opera.UpgradeHeight,
) (*config, error) {

	chainID := b.ChainID()
	chainCfg := b.ChainConfig(upgradeHeight.Height)

	precompiled := make(contractRegistry)
	chainCfgRules := chainCfg.Rules(big.NewInt(int64(upgradeHeight.Height)), true, uint64(0))
	for addr, c := range vm.ActivePrecompiledContracts(chainCfgRules) {
		precompiled[c.Name()] = addr
	}

	forkId, err := MakeForkId(upgradeHeight, b.GetGenesisID())
	if err != nil {
		// this can only fail if RLP encoding fails, which is unexpected
		return nil, fmt.Errorf("could not make fork id, %v", err)
	}

	block, err := b.BlockByNumber(ctx, rpc.BlockNumber(int64(upgradeHeight.Height)))
	if err != nil {
		return nil, fmt.Errorf("could not get block %d to determine activation time, %v", upgradeHeight.Height, err)
	}

	if block == nil {
		return nil, fmt.Errorf("block %d not found to determine activation time", upgradeHeight.Height)
	}

	return &config{
		// block time needs to be converted to unix timestamp as it is done in
		// evmcore/dummy_block.go in method EvmHeader.EthHeader()
		ActivationTime:  uint64(block.Time.Unix()),
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
		sysContracts["GAS_SUBSIDY_REGISTRY_ADDRESS"] = registry.GetAddress()
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
