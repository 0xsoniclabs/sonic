package genesisstore

import (
	"io"
	"testing"

	"github.com/0xsoniclabs/sonic/opera/genesis"
)

func TestSectionNames(t *testing.T) {
	tests := []struct {
		fn       func(int) string
		idx      int
		expected string
	}{
		{BlocksSection, 0, "brs"},
		{BlocksSection, 1, "brs-1"},
		{EpochsSection, 0, "ers"},
		{EpochsSection, 2, "ers-2"},
		{EvmSection, 0, "evm"},
		{EvmSection, 3, "evm-3"},
		{FwsLiveSection, 0, "fws"},
		{FwsLiveSection, 1, "fws-1"},
		{FwsArchiveSection, 0, "fwa"},
		{FwsArchiveSection, 1, "fwa-1"},
		{SccCommitteeSection, 0, "scc_cc"},
		{SccCommitteeSection, 1, "scc_cc-1"},
		{SccBlockSection, 0, "scc_bc"},
		{SccBlockSection, 1, "scc_bc-1"},
	}

	for _, tt := range tests {
		got := tt.fn(tt.idx)
		if got != tt.expected {
			t.Fatalf("expected %q, got %q", tt.expected, got)
		}
	}
}

func TestNewStore(t *testing.T) {
	header := genesis.Header{
		NetworkID:   1,
		NetworkName: "test",
	}
	closed := false
	s := NewStore(
		func(name string) (io.Reader, error) {
			return nil, nil
		},
		header,
		func() error {
			closed = true
			return nil
		},
	)
	if s == nil {
		t.Fatal("expected non-nil Store")
	}

	err := s.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !closed {
		t.Fatal("expected close function to be called")
	}
}
