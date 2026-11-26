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

// Package freeze implements Sonic's account freeze: accounts the on-chain
// registry lists as frozen may not move native S or tokens, except tokens whose
// contracts the registry lists as opted out. Freeze status is read directly
// from the registry's storage, never by calling the contract.
package freeze

import (
	"github.com/0xsoniclabs/sonic/gossip/blockproc/freeze/registry"
	priorities "github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	subsidies "github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/0xsoniclabs/sonic/opera/contracts/evmwriter"
	"github.com/0xsoniclabs/sonic/opera/contracts/sfc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

//go:generate go tool mockgen -source=freeze.go -destination=freeze_mock.go -package=freeze

// Checker reports the freeze status of addresses.
type Checker interface {
	// IsEnabled reports whether the freeze applies at all.
	IsEnabled() bool
	// IsFrozen reports whether account is frozen.
	IsFrozen(account common.Address) bool
	// IsOptedOut reports whether token is opted out of the freeze: transfers
	// of this token contract are exempt from the freezing logic. It is meant
	// to exclude token contracts, not to keep accounts from being frozen.
	IsOptedOut(token common.Address) bool
}

// StateReader is the part of the state needed to read the registry.
type StateReader interface {
	GetState(common.Address, common.Hash) common.Hash
}

// RegistryChecker is a Checker reading the registry's storage slots from a
// state. The zero address, params.SystemAddress and system contracts are never
// frozen, whatever the registry says, so that internal transactions and system
// calls are never blocked.
type RegistryChecker struct {
	enabled bool // feature flag `AccountFreeze` is enabled
	db      StateReader
}

// NewRegistryChecker creates a RegistryChecker reading the registry from db,
// reporting nothing as frozen or opted out unless enabled.
func NewRegistryChecker(enabled bool, db StateReader) *RegistryChecker {
	return &RegistryChecker{enabled: enabled, db: db}
}

// IsEnabled implements Checker.
func (c *RegistryChecker) IsEnabled() bool {
	return c.enabled
}

// IsFrozen implements Checker.
func (c *RegistryChecker) IsFrozen(account common.Address) bool {
	if !c.enabled || account == (common.Address{}) || account == params.SystemAddress || isSystemContract(account) {
		return false
	}
	return c.db.GetState(registry.GetAddress(), registry.GetFrozenSlot(account)) != (common.Hash{})
}

// IsOptedOut implements Checker.
func (c *RegistryChecker) IsOptedOut(token common.Address) bool {
	if !c.enabled {
		return false
	}
	return c.db.GetState(registry.GetAddress(), registry.GetOptedOutSlot(token)) != (common.Hash{})
}

func isSystemContract(addr common.Address) bool {
	switch addr {
	case registry.GetAddress(),
		sfc.ContractAddress,
		driver.ContractAddress,
		driverauth.ContractAddress,
		evmwriter.ContractAddress,
		subsidies.GetAddress(),
		priorities.GetAddress():
		return true
	}
	return false
}
