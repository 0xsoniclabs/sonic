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

package main

import (
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/p2p/networks"
)

// TestDemoStatus_BlockHeight_AdvancesWithClock checks the faked chain advances at
// one block per second and includes the per-node drift offset.
func TestDemoStatus_BlockHeight_AdvancesWithClock(t *testing.T) {
	base := genesisEpoch.Add(1000 * time.Second)
	current := base
	status := demoStatus{
		role:         networks.RoleValidator,
		heightOffset: 2,
		now:          func() time.Time { return current },
	}

	first := status.Status().BlockHeight
	if want := uint64(1000 + 2); first != want {
		t.Fatalf("initial height = %d, want %d", first, want)
	}
	current = base.Add(5 * time.Second)
	if second := status.Status().BlockHeight; second != first+5 {
		t.Fatalf("height did not advance by 5: %d then %d", first, second)
	}
}

// TestDemoStatus_Status_ReportsRoleAndVersion checks the reported role and client
// version.
func TestDemoStatus_Status_ReportsRoleAndVersion(t *testing.T) {
	status := newDemoStatus(networks.RoleArchive, "archive-seed")
	got := status.Status()
	if got.Role != networks.RoleArchive {
		t.Fatalf("reported role = %v, want archive", got.Role)
	}
	if got.ClientVersion != clientVersion {
		t.Fatalf("reported client version = %q, want %q", got.ClientVersion, clientVersion)
	}
}
