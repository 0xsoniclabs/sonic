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

// config as described by https://eips.ethereum.org/EIPS/eip-7910
type config struct {
	// ActivationTime will remain 0 because in Sonic this is not relevant
	ActivationTime uint64 `json:"activationTime"`
	// BlobSchedule will remain nil because in Sonic this is not relevant
	BlobSchedule *params.BlobConfig `json:"blobSchedule"`

	ChainId *hexutil.Big `json:"chainId"`

	// ForkId in sonic is a checksum derived from the json marshall of the corresponding upgrade
	ForkId hexutil.Bytes `json:"forkId"`

	Precompiles     contractRegistry `json:"precompiles"`
	SystemContracts contractRegistry `json:"systemContracts"`
}

// helper types to improve readability of the returned structure.
type contractRegistry map[string]common.Address
type forkId [4]byte

// makeConfigFromUpgrade constructs the config that was active for the
// given block number based on the upgrade heights.
func makeConfigFromUpgrade(
	ctx context.Context,
	b Backend,
	upgradeHeight opera.UpgradeHeight,
) (*config, error) {

	chainID := b.ChainID()
	chainCfg := opera.CreateTransientEvmChainConfig(
		chainID.Uint64(),
		[]opera.UpgradeHeight{upgradeHeight},
		upgradeHeight.Height,
	)

	precompiled := make(contractRegistry)
	chainCfgRules := chainCfg.Rules(big.NewInt(int64(upgradeHeight.Height)), true, uint64(0))
	for addr, c := range vm.ActivePrecompiledContracts(chainCfgRules) {
		precompiled[c.Name()] = addr
	}

	forkId, err := MakeForkId(upgradeHeight, b.GetGenesisID())
	if err != nil {
		// this can only fail if json.Marshal fails, which is unexpected
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
		sysContracts["HISTORY_STORAGE"] = params.HistoryStorageAddress
	}
	if upgrade.GasSubsidies {
		sysContracts["GAS_SUBSIDY_REGISTRY"] = registry.GetAddress()
	}
	return sysContracts
}

// MakeForkId creates a fork ID from the given upgrade.
func MakeForkId(upgrade opera.UpgradeHeight, genesisId *common.Hash) (forkId, error) {

	upgradeRlp, err := rlp.EncodeToBytes(upgrade.Upgrades)
	if err != nil {
		return forkId{}, fmt.Errorf("could not encode upgrade to json, %v", err)
	}
	upgradeHash := crc32.ChecksumIEEE(upgradeRlp)
	// update hash with block number of last change in upgrade.
	BlockNumberHash := uint64(upgrade.Height)
	forkId := checksumUpdate(upgradeHash, BlockNumberHash)
	// update with genesis ID
	genesisHash := crc32.ChecksumIEEE(genesisId.Bytes())
	forkId = checksumUpdate(forkId, genesisHash)

	return checksumToBytes(forkId), nil
}

type u32u64 interface{ ~uint32 | ~uint64 }

// checksumUpdate calculates the next IEEE CRC32 checksum based on the previous
// one and a fork block number (equivalent to CRC32(original-blob || fork)).
func checksumUpdate[U u32u64](hash uint32, fork U) uint32 {
	var blob [8]byte
	binary.BigEndian.PutUint64(blob[:], uint64(fork))
	return crc32.Update(hash, crc32.IEEETable, blob[:])
}

// checksumToBytes converts a uint32 checksum into a [4]byte array.
func checksumToBytes(hash uint32) [4]byte {
	var blob [4]byte
	binary.BigEndian.PutUint32(blob[:], hash)
	return blob
}
