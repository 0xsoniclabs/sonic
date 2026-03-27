package dagstreamseeder

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig(cachescale.Identity)
	if cfg.SenderThreads != 8 {
		t.Fatalf("expected SenderThreads 8, got %d", cfg.SenderThreads)
	}
	if cfg.MaxSenderTasks != 128 {
		t.Fatalf("expected MaxSenderTasks 128, got %d", cfg.MaxSenderTasks)
	}
	if cfg.MaxResponsePayloadNum != 16384 {
		t.Fatalf("expected MaxResponsePayloadNum 16384, got %d", cfg.MaxResponsePayloadNum)
	}
	if cfg.MaxResponsePayloadSize != 8*1024*1024 {
		t.Fatalf("expected MaxResponsePayloadSize %d, got %d", 8*1024*1024, cfg.MaxResponsePayloadSize)
	}
	if cfg.MaxResponseChunks != 12 {
		t.Fatalf("expected MaxResponseChunks 12, got %d", cfg.MaxResponseChunks)
	}
	if cfg.MaxPendingResponsesSize <= 0 {
		t.Fatal("expected positive MaxPendingResponsesSize")
	}
}

func TestDefaultConfig_WithScale(t *testing.T) {
	scale := cachescale.Ratio{Base: 100, Target: 50}
	cfg := DefaultConfig(scale)
	// With 50% scale, MaxPendingResponsesSize should be smaller
	cfgFull := DefaultConfig(cachescale.Identity)
	if cfg.MaxPendingResponsesSize >= cfgFull.MaxPendingResponsesSize {
		t.Fatal("expected scaled MaxPendingResponsesSize to be smaller")
	}
}
