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

// Package contracts puts the applications of tests/contracts on the network under test and draws
// calls to them: a valid call to a method of a deployed contract, or a mutation of one that is
// malformed in a way calldata can be. It is what makes a generated transaction reach the EVM with
// something to do, rather than a payload no code ever looks at.
//
// It knows nothing of the harness that uses it -- a call is plain data, encoded from the bindings
// alone -- so that core can depend on it and it on nothing of core's.
package contracts

import (
	"fmt"
	"slices"
	"strings"

	access_cost "github.com/0xsoniclabs/sonic/tests/contracts/access_cost"
	add "github.com/0xsoniclabs/sonic/tests/contracts/add"
	basefee "github.com/0xsoniclabs/sonic/tests/contracts/basefee"
	batch "github.com/0xsoniclabs/sonic/tests/contracts/batch"
	blobbasefee "github.com/0xsoniclabs/sonic/tests/contracts/blobbasefee"
	block_hash "github.com/0xsoniclabs/sonic/tests/contracts/block_hash"
	block_parameters "github.com/0xsoniclabs/sonic/tests/contracts/block_parameters"
	blockoverride "github.com/0xsoniclabs/sonic/tests/contracts/blockoverride"
	blsContracts "github.com/0xsoniclabs/sonic/tests/contracts/blsContracts"
	contractcreator "github.com/0xsoniclabs/sonic/tests/contracts/contractcreator"
	counter "github.com/0xsoniclabs/sonic/tests/contracts/counter"
	counter_event_emitter "github.com/0xsoniclabs/sonic/tests/contracts/counter_event_emitter"
	data_reader "github.com/0xsoniclabs/sonic/tests/contracts/data_reader"
	failing_post_tx_registry "github.com/0xsoniclabs/sonic/tests/contracts/failing_post_tx_registry"
	indexed_logs "github.com/0xsoniclabs/sonic/tests/contracts/indexed_logs"
	invalidstart "github.com/0xsoniclabs/sonic/tests/contracts/invalidstart"
	legacy_registry "github.com/0xsoniclabs/sonic/tests/contracts/legacy_registry"
	magic_value_priority "github.com/0xsoniclabs/sonic/tests/contracts/magic_value_priority"
	misbehaving_priority_registry "github.com/0xsoniclabs/sonic/tests/contracts/misbehaving_priority_registry"
	network_sponsor "github.com/0xsoniclabs/sonic/tests/contracts/network_sponsor"
	network_sponsor_configurable "github.com/0xsoniclabs/sonic/tests/contracts/network_sponsor_configurable"
	network_sponsor_tracking "github.com/0xsoniclabs/sonic/tests/contracts/network_sponsor_tracking"
	nonce_priority_registry "github.com/0xsoniclabs/sonic/tests/contracts/nonce_priority_registry"
	prevrandao "github.com/0xsoniclabs/sonic/tests/contracts/prevrandao"
	privilege_deescalation "github.com/0xsoniclabs/sonic/tests/contracts/privilege_deescalation"
	read_history_storage "github.com/0xsoniclabs/sonic/tests/contracts/read_history_storage"
	revert "github.com/0xsoniclabs/sonic/tests/contracts/revert"
	selfdestruct "github.com/0xsoniclabs/sonic/tests/contracts/selfdestruct"
	sponsor_everything "github.com/0xsoniclabs/sonic/tests/contracts/sponsor_everything"
	sponsoring "github.com/0xsoniclabs/sonic/tests/contracts/sponsoring"
	storage "github.com/0xsoniclabs/sonic/tests/contracts/storage"
	store "github.com/0xsoniclabs/sonic/tests/contracts/store"
	transientstorage "github.com/0xsoniclabs/sonic/tests/contracts/transientstorage"
	transitive_call "github.com/0xsoniclabs/sonic/tests/contracts/transitive_call"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// Contract is one deployable application, as the registry knows it before anything is on chain.
type Contract struct {
	Name string
	ABI  abi.ABI

	// Code is the creation code, constructor arguments included.
	Code []byte

	// Methods are what a call may target, in a fixed order so that a drawn index means the same
	// thing on every run -- abi.ABI holds them in a map, whose order is not.
	Methods []abi.Method
}

// Registry is every contract this package deploys, in a fixed order. A drawn call names one by its
// index here, so entries may be appended but not reordered without invalidating a recorded seed.
var Registry = mustRegister([]source{
	{"AccessCost", access_cost.AccessCostMetaData, access_cost.AccessCostBin, nil},
	{"Add", add.AddMetaData, add.AddBin, nil},
	{"Basefee", basefee.BasefeeMetaData, basefee.BasefeeBin, nil},
	{"Batch", batch.BatchMetaData, batch.BatchBin, nil},
	{"BigNumbers", blsContracts.BigNumbersMetaData, blsContracts.BigNumbersBin, nil},
	{"Blobbasefee", blobbasefee.BlobbasefeeMetaData, blobbasefee.BlobbasefeeBin, nil},
	{"BlockHash", block_hash.BlockHashMetaData, block_hash.BlockHashBin, nil},
	{"BlockOverride", blockoverride.BlockOverrideMetaData, blockoverride.BlockOverrideBin, nil},
	{"BlockParameters", block_parameters.BlockParametersMetaData, block_parameters.BlockParametersBin, nil},
	{"BLS", blsContracts.BLSMetaData, blsContracts.BLSBin, nil},
	{"BLSLibrary", blsContracts.BLSLibraryMetaData, blsContracts.BLSLibraryBin, nil},
	{"Contractcreator", contractcreator.ContractcreatorMetaData, contractcreator.ContractcreatorBin, nil},
	{"Counter", counter.CounterMetaData, counter.CounterBin, nil},
	{"CounterEventEmitter", counter_event_emitter.CounterEventEmitterMetaData, counter_event_emitter.CounterEventEmitterBin, nil},
	{"DataReader", data_reader.DataReaderMetaData, data_reader.DataReaderBin, nil},
	{"Elements", blsContracts.ElementsMetaData, blsContracts.ElementsBin, nil},
	{"FailingPostTxRegistry", failing_post_tx_registry.FailingPostTxRegistryMetaData, failing_post_tx_registry.FailingPostTxRegistryBin, nil},
	{"IndexedLogs", indexed_logs.IndexedLogsMetaData, indexed_logs.IndexedLogsBin, nil},
	{"Invalidstart", invalidstart.InvalidstartMetaData, invalidstart.InvalidstartBin, nil},
	{"LegacyRegistry", legacy_registry.LegacyRegistryMetaData, legacy_registry.LegacyRegistryBin, nil},
	{"MagicValuePriority", magic_value_priority.MagicValuePriorityMetaData, magic_value_priority.MagicValuePriorityBin, nil},
	{"MisbehavingPriorityRegistry", misbehaving_priority_registry.MisbehavingPriorityRegistryMetaData, misbehaving_priority_registry.MisbehavingPriorityRegistryBin, nil},
	{"NetworkSponsorConfigurable", network_sponsor_configurable.NetworkSponsorConfigurableMetaData, network_sponsor_configurable.NetworkSponsorConfigurableBin, nil},
	{"NetworkSponsor", network_sponsor.NetworkSponsorMetaData, network_sponsor.NetworkSponsorBin, nil},
	{"NetworkSponsorTracking", network_sponsor_tracking.NetworkSponsorTrackingMetaData, network_sponsor_tracking.NetworkSponsorTrackingBin, nil},
	{"NoncePriorityRegistry", nonce_priority_registry.NoncePriorityRegistryMetaData, nonce_priority_registry.NoncePriorityRegistryBin, nil},
	{"Prevrandao", prevrandao.PrevrandaoMetaData, prevrandao.PrevrandaoBin, nil},
	{"PrivilegeDeescalation", privilege_deescalation.PrivilegeDeescalationMetaData, privilege_deescalation.PrivilegeDeescalationBin, nil},
	{"ReadHistoryStorage", read_history_storage.ReadHistoryStorageMetaData, read_history_storage.ReadHistoryStorageBin, nil},
	{"Revert", revert.RevertMetaData, revert.RevertBin, nil},
	{"SelfDestructFactory", selfdestruct.SelfDestructFactoryMetaData, selfdestruct.SelfDestructFactoryBin, nil},
	{"SelfDestruct", selfdestruct.SelfDestructMetaData, selfdestruct.SelfDestructBin, []any{false, false, common.Address{}}},
	{"SponsorEverything", sponsor_everything.SponsorEverythingMetaData, sponsor_everything.SponsorEverythingBin, nil},
	{"Sponsoring", sponsoring.SponsoringMetaData, sponsoring.SponsoringBin, nil},
	{"Storage", storage.StorageMetaData, storage.StorageBin, nil},
	{"Store", store.StoreMetaData, store.StoreBin, nil},
	{"Transientstorage", transientstorage.TransientstorageMetaData, transientstorage.TransientstorageBin, nil},
	{"TransitiveCall", transitive_call.TransitiveCallMetaData, transitive_call.TransitiveCallBin, nil},
})

// source is a registry entry as the generated bindings give it, before the ABI is parsed and the
// constructor arguments are packed onto the creation code.
type source struct {
	name string
	meta *bind.MetaData
	bin  string
	ctor []any
}

// mustRegister resolves the bindings into contracts, panicking on anything wrong with them: the
// bindings are checked in alongside this table, so a mismatch is a build-time mistake rather than
// something a run should discover.
func mustRegister(sources []source) []Contract {
	out := make([]Contract, 0, len(sources))
	for _, s := range sources {
		parsed, err := s.meta.GetAbi()
		if err != nil {
			panic(fmt.Sprintf("contract %s has an unparsable ABI: %v", s.name, err))
		}
		args, err := parsed.Pack("", s.ctor...)
		if err != nil {
			panic(fmt.Sprintf("contract %s rejects its constructor arguments: %v", s.name, err))
		}

		methods := make([]abi.Method, 0, len(parsed.Methods))
		for _, method := range parsed.Methods {
			methods = append(methods, method)
		}
		slices.SortFunc(methods, func(a, b abi.Method) int { return strings.Compare(a.Name, b.Name) })

		out = append(out, Contract{
			Name:    s.name,
			ABI:     *parsed,
			Code:    append(common.FromHex(s.bin), args...),
			Methods: methods,
		})
	}
	return out
}
