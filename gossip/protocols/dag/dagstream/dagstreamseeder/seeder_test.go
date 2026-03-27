package dagstreamseeder

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/sonic/gossip/protocols/dag/dagstream"
)

func TestErrors(t *testing.T) {
	if ErrWrongType == nil {
		t.Fatal("ErrWrongType should not be nil")
	}
	if ErrWrongSelectorLen == nil {
		t.Fatal("ErrWrongSelectorLen should not be nil")
	}
}

func TestNew(t *testing.T) {
	cfg := DefaultConfig(cachescale.Identity)
	callbacks := Callbacks{
		ForEachEvent: func(start []byte, onEvent func(key hash.Event, eventB rlp.RawValue) bool) {
		},
	}
	s := New(cfg, callbacks)
	if s == nil {
		t.Fatal("expected non-nil Seeder")
	}
	if s.BaseSeeder == nil {
		t.Fatal("expected non-nil BaseSeeder")
	}
}

func TestNotifyRequestReceived_WrongSelectorLen(t *testing.T) {
	cfg := DefaultConfig(cachescale.Identity)
	callbacks := Callbacks{
		ForEachEvent: func(start []byte, onEvent func(key hash.Event, eventB rlp.RawValue) bool) {},
	}
	s := New(cfg, callbacks)
	s.Start()
	defer s.Stop()

	peer := Peer{
		ID:           "test-peer",
		SendChunk:    func(dagstream.Response, hash.Events) error { return nil },
		Misbehaviour: func(error) {},
	}

	// Use a locator that's too long
	tooLong := make(dagstream.Locator, 100)
	r := dagstream.Request{
		Session: dagstream.Session{
			ID:    1,
			Start: tooLong,
			Stop:  dagstream.Locator{},
		},
		Type: dagstream.RequestEvents,
	}

	_, peerErr := s.NotifyRequestReceived(peer, r)
	if peerErr != ErrWrongSelectorLen {
		t.Fatalf("expected ErrWrongSelectorLen, got %v", peerErr)
	}
}

func TestNotifyRequestReceived_WrongType(t *testing.T) {
	cfg := DefaultConfig(cachescale.Identity)
	callbacks := Callbacks{
		ForEachEvent: func(start []byte, onEvent func(key hash.Event, eventB rlp.RawValue) bool) {},
	}
	s := New(cfg, callbacks)
	s.Start()
	defer s.Stop()

	peer := Peer{
		ID:           "test-peer",
		SendChunk:    func(dagstream.Response, hash.Events) error { return nil },
		Misbehaviour: func(error) {},
	}

	r := dagstream.Request{
		Session: dagstream.Session{
			ID:    1,
			Start: dagstream.Locator([]byte{0x01}),
			Stop:  dagstream.Locator([]byte{0xff}),
		},
		Type: 99, // invalid type
	}

	_, peerErr := s.NotifyRequestReceived(peer, r)
	if peerErr != ErrWrongType {
		t.Fatalf("expected ErrWrongType, got %v", peerErr)
	}
}
