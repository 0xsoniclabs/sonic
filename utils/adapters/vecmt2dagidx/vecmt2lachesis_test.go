package vecmt2dagidx

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/abft"
	"github.com/Fantom-foundation/lachesis-base/vecfc"
)

func TestBranchSeq_Seq(t *testing.T) {
	bs := &BranchSeq{vecfc.BranchSeq{Seq: 42, MinSeq: 10}}
	if got := bs.Seq(); got != 42 {
		t.Errorf("expected Seq=42, got %d", got)
	}
}

func TestBranchSeq_MinSeq(t *testing.T) {
	bs := &BranchSeq{vecfc.BranchSeq{Seq: 42, MinSeq: 10}}
	if got := bs.MinSeq(); got != 10 {
		t.Errorf("expected MinSeq=10, got %d", got)
	}
}

func TestAdapter_ImplementsDagIndex(t *testing.T) {
	// Verify at compile time that *Adapter implements abft.DagIndex.
	var _ abft.DagIndex = (*Adapter)(nil)
}

func TestWrap_NilInput(t *testing.T) {
	// Wrap should not panic with nil; the adapter is just a thin wrapper.
	adapter := Wrap(nil)
	if adapter == nil {
		t.Fatal("Wrap returned nil")
	}
	if adapter.Index != nil {
		t.Error("expected nil Index in adapter")
	}
}
