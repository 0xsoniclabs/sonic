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
