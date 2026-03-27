package dagstream

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestLocator_Compare(t *testing.T) {
	a := Locator([]byte{0x01, 0x02})
	b := Locator([]byte{0x01, 0x03})
	c := Locator([]byte{0x01, 0x02})

	if a.Compare(b) >= 0 {
		t.Fatal("expected a < b")
	}
	if b.Compare(a) <= 0 {
		t.Fatal("expected b > a")
	}
	if a.Compare(c) != 0 {
		t.Fatal("expected a == c")
	}
}

func TestLocator_Inc(t *testing.T) {
	a := Locator([]byte{0x00, 0x01})
	b := a.Inc().(Locator)
	if len(b) != 2 {
		t.Fatalf("expected length 2, got %d", len(b))
	}
	if b[0] != 0x00 || b[1] != 0x02 {
		t.Fatalf("expected [0x00, 0x02], got %v", b)
	}
}

func TestLocator_Inc_Overflow(t *testing.T) {
	a := Locator([]byte{0x00, 0xff})
	b := a.Inc().(Locator)
	if len(b) != 2 {
		t.Fatalf("expected length 2, got %d", len(b))
	}
	if b[0] != 0x01 || b[1] != 0x00 {
		t.Fatalf("expected [0x01, 0x00], got %v", b)
	}
}

func TestPayload_AddEvent(t *testing.T) {
	p := &Payload{}
	id := hash.FakeEvent()
	eventB := rlp.RawValue([]byte{0x01, 0x02, 0x03})

	p.AddEvent(id, eventB)

	if p.Len() != 1 {
		t.Fatalf("expected length 1, got %d", p.Len())
	}
	if p.TotalSize() != 3 {
		t.Fatalf("expected size 3, got %d", p.TotalSize())
	}
	if p.IDs[0] != id {
		t.Fatal("unexpected ID")
	}
}

func TestPayload_AddID(t *testing.T) {
	p := &Payload{}
	id := hash.FakeEvent()

	p.AddID(id, 100)

	if p.Len() != 1 {
		t.Fatalf("expected length 1, got %d", p.Len())
	}
	if p.TotalSize() != 100 {
		t.Fatalf("expected size 100, got %d", p.TotalSize())
	}
	if len(p.Events) != 0 {
		t.Fatal("expected no events")
	}
}

func TestPayload_TotalMemSize_WithEvents(t *testing.T) {
	p := &Payload{}
	p.AddEvent(hash.FakeEvent(), rlp.RawValue([]byte{0x01, 0x02}))
	p.AddEvent(hash.FakeEvent(), rlp.RawValue([]byte{0x03, 0x04}))

	memSize := p.TotalMemSize()
	if memSize <= 0 {
		t.Fatal("expected positive memory size")
	}
}

func TestPayload_TotalMemSize_IDsOnly(t *testing.T) {
	p := &Payload{}
	p.AddID(hash.FakeEvent(), 50)
	p.AddID(hash.FakeEvent(), 50)

	memSize := p.TotalMemSize()
	expected := 2 * 128
	if memSize != expected {
		t.Fatalf("expected memSize %d, got %d", expected, memSize)
	}
}

func TestPayload_Empty(t *testing.T) {
	p := Payload{}
	if p.Len() != 0 {
		t.Fatalf("expected length 0, got %d", p.Len())
	}
	if p.TotalSize() != 0 {
		t.Fatalf("expected size 0, got %d", p.TotalSize())
	}
	if p.TotalMemSize() != 0 {
		t.Fatalf("expected memSize 0, got %d", p.TotalMemSize())
	}
}

func TestRequest_Fields(t *testing.T) {
	r := Request{
		Session: Session{
			ID:    1,
			Start: Locator([]byte{0x01}),
			Stop:  Locator([]byte{0xff}),
		},
		Type:      RequestEvents,
		MaxChunks: 10,
	}
	if r.Session.ID != 1 {
		t.Fatal("unexpected session ID")
	}
	if r.Type != RequestEvents {
		t.Fatal("unexpected request type")
	}
}

func TestResponse_Fields(t *testing.T) {
	r := Response{
		SessionID: 42,
		Done:      true,
		IDs:       hash.Events{hash.FakeEvent()},
		Events:    nil,
	}
	if r.SessionID != 42 {
		t.Fatal("unexpected session ID")
	}
	if !r.Done {
		t.Fatal("expected Done to be true")
	}
}

func TestConstants(t *testing.T) {
	if RequestIDs != 0 {
		t.Fatalf("expected RequestIDs == 0, got %d", RequestIDs)
	}
	if RequestEvents != 2 {
		t.Fatalf("expected RequestEvents == 2, got %d", RequestEvents)
	}
}
