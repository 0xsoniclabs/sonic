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

package heavycheck

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxQueuedTasks != 1024 {
		t.Fatalf("expected MaxQueuedTasks 1024, got %d", cfg.MaxQueuedTasks)
	}
	if cfg.Threads != 0 {
		t.Fatalf("expected Threads 0, got %d", cfg.Threads)
	}
}

func TestConfig_CustomValues(t *testing.T) {
	cfg := Config{
		MaxQueuedTasks: 512,
		Threads:        4,
	}
	if cfg.MaxQueuedTasks != 512 {
		t.Fatalf("expected MaxQueuedTasks 512, got %d", cfg.MaxQueuedTasks)
	}
	if cfg.Threads != 4 {
		t.Fatalf("expected Threads 4, got %d", cfg.Threads)
	}
}
