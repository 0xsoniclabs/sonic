package verwatcher

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"

	"github.com/0xsoniclabs/sonic/opera/contracts/driver"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver/driverpos"
)

func newTestWatcher(t *testing.T) *VersionWatcher {
	t.Helper()
	db := memorydb.New()
	store := NewStore(db)
	return New(store)
}

func TestNew_VersionWatcher(t *testing.T) {
	w := newTestWatcher(t)
	if w == nil {
		t.Fatal("New returned nil")
	}
}

func TestPause_NoUpgradeNeeded(t *testing.T) {
	w := newTestWatcher(t)
	// With default (zero) network version, current version should be fine.
	err := w.Pause()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPause_MissedVersion(t *testing.T) {
	w := newTestWatcher(t)
	w.store.SetMissedVersion(1)

	err := w.Pause()
	if err == nil {
		t.Fatal("expected error for missed version")
	}
}

func TestOnNewLog_WrongAddress(t *testing.T) {
	w := newTestWatcher(t)

	wrongAddr := common.HexToAddress("0x1234")
	l := &types.Log{
		Address: wrongAddr,
		Topics:  []common.Hash{driverpos.Topics.UpdateNetworkVersion},
		Data:    make([]byte, 32),
	}
	w.OnNewLog(l)

	// Network version should remain 0.
	if got := w.store.GetNetworkVersion(); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestOnNewLog_UpdateNetworkVersion(t *testing.T) {
	w := newTestWatcher(t)

	// Build log data with version number in last 8 bytes.
	data := make([]byte, 32)
	v := new(big.Int).SetUint64(42)
	vBytes := v.Bytes()
	copy(data[32-len(vBytes):], vBytes)

	l := &types.Log{
		Address: driver.ContractAddress,
		Topics:  []common.Hash{driverpos.Topics.UpdateNetworkVersion},
		Data:    data,
	}
	w.OnNewLog(l)

	if got := w.store.GetNetworkVersion(); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestOnNewLog_WrongTopic(t *testing.T) {
	w := newTestWatcher(t)

	l := &types.Log{
		Address: driver.ContractAddress,
		Topics:  []common.Hash{common.HexToHash("0xdeadbeef")},
		Data:    make([]byte, 32),
	}
	w.OnNewLog(l)

	if got := w.store.GetNetworkVersion(); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestOnNewLog_DataTooShort(t *testing.T) {
	w := newTestWatcher(t)

	l := &types.Log{
		Address: driver.ContractAddress,
		Topics:  []common.Hash{driverpos.Topics.UpdateNetworkVersion},
		Data:    make([]byte, 10), // too short (< 32)
	}
	w.OnNewLog(l)

	if got := w.store.GetNetworkVersion(); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestStartStop(t *testing.T) {
	w := newTestWatcher(t)
	w.Start()
	w.Stop()
}
