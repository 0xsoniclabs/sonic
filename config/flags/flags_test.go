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

package flags

import (
	"testing"
)

func TestFlags_HaveNames(t *testing.T) {
	// Verify that key flags have non-empty names set.
	flagsToCheck := []struct {
		name     string
		flagName string
	}{
		{"DataDirFlag", DataDirFlag.GetName()},
		{"MinFreeDiskSpaceFlag", MinFreeDiskSpaceFlag.GetName()},
		{"KeyStoreDirFlag", KeyStoreDirFlag.GetName()},
		{"USBFlag", USBFlag.GetName()},
		{"IdentityFlag", IdentityFlag.GetName()},
		{"LightKDFFlag", LightKDFFlag.GetName()},
		{"TxPoolLocalsFlag", TxPoolLocalsFlag.GetName()},
		{"TxPoolNoLocalsFlag", TxPoolNoLocalsFlag.GetName()},
		{"UnlockedAccountFlag", UnlockedAccountFlag.GetName()},
		{"PasswordFileFlag", PasswordFileFlag.GetName()},
		{"IPCDisabledFlag", IPCDisabledFlag.GetName()},
		{"HTTPEnabledFlag", HTTPEnabledFlag.GetName()},
		{"HTTPListenAddrFlag", HTTPListenAddrFlag.GetName()},
		{"HTTPPortFlag", HTTPPortFlag.GetName()},
		{"WSEnabledFlag", WSEnabledFlag.GetName()},
		{"MaxPeersFlag", MaxPeersFlag.GetName()},
		{"BootnodesFlag", BootnodesFlag.GetName()},
		{"CacheFlag", CacheFlag.GetName()},
		{"ValidatorIDFlag", ValidatorIDFlag.GetName()},
		{"ValidatorPubkeyFlag", ValidatorPubkeyFlag.GetName()},
		{"ValidatorPasswordFlag", ValidatorPasswordFlag.GetName()},
		{"ConfigFileFlag", ConfigFileFlag.GetName()},
	}

	for _, tc := range flagsToCheck {
		if tc.flagName == "" {
			t.Errorf("%s has an empty name", tc.name)
		}
	}
}

func TestFlags_UniqueNames(t *testing.T) {
	// Collect all flag names and verify they're unique.
	names := []string{
		DataDirFlag.GetName(),
		MinFreeDiskSpaceFlag.GetName(),
		KeyStoreDirFlag.GetName(),
		USBFlag.GetName(),
		IdentityFlag.GetName(),
		LightKDFFlag.GetName(),
		TxPoolLocalsFlag.GetName(),
		TxPoolNoLocalsFlag.GetName(),
		TxPoolJournalFlag.GetName(),
		TxPoolRejournalFlag.GetName(),
		TxPoolPriceLimitFlag.GetName(),
		TxPoolMinTipFlag.GetName(),
		TxPoolPriceBumpFlag.GetName(),
		TxPoolAccountSlotsFlag.GetName(),
		TxPoolGlobalSlotsFlag.GetName(),
		TxPoolAccountQueueFlag.GetName(),
		TxPoolGlobalQueueFlag.GetName(),
		TxPoolLifetimeFlag.GetName(),
		UnlockedAccountFlag.GetName(),
		PasswordFileFlag.GetName(),
		ExternalSignerFlag.GetName(),
		InsecureUnlockAllowedFlag.GetName(),
		IPCDisabledFlag.GetName(),
		IPCPathFlag.GetName(),
		HTTPEnabledFlag.GetName(),
		HTTPListenAddrFlag.GetName(),
		HTTPPortFlag.GetName(),
		HTTPCORSDomainFlag.GetName(),
		HTTPVirtualHostsFlag.GetName(),
		WSEnabledFlag.GetName(),
		WSListenAddrFlag.GetName(),
		WSPortFlag.GetName(),
		MaxPeersFlag.GetName(),
		BootnodesFlag.GetName(),
		CacheFlag.GetName(),
		ValidatorIDFlag.GetName(),
		ValidatorPubkeyFlag.GetName(),
		ValidatorPasswordFlag.GetName(),
		ConfigFileFlag.GetName(),
	}

	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if seen[name] {
			t.Errorf("duplicate flag name: %q", name)
		}
		seen[name] = true
	}
}

func TestFlags_DataDirFlag(t *testing.T) {
	if DataDirFlag.GetName() != "datadir" {
		t.Errorf("expected DataDirFlag name 'datadir', got %q", DataDirFlag.GetName())
	}
}

func TestFlags_ValidatorIDFlag(t *testing.T) {
	if ValidatorIDFlag.GetName() != "validator.id" {
		t.Errorf("expected ValidatorIDFlag name 'validator.id', got %q", ValidatorIDFlag.GetName())
	}
}
