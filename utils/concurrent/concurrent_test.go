package concurrent

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

func TestWrapValidatorEventsSet(t *testing.T) {
	val := map[idx.ValidatorID]hash.Event{
		idx.ValidatorID(1): hash.ZeroEvent,
	}
	wrapped := WrapValidatorEventsSet(val)
	if wrapped == nil {
		t.Fatal("expected non-nil ValidatorEventsSet")
	}
	if len(wrapped.Val) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(wrapped.Val))
	}
	if wrapped.Val[idx.ValidatorID(1)] != hash.ZeroEvent {
		t.Fatal("unexpected value")
	}
}

func TestWrapValidatorEventsSet_Nil(t *testing.T) {
	wrapped := WrapValidatorEventsSet(nil)
	if wrapped == nil {
		t.Fatal("expected non-nil ValidatorEventsSet")
	}
	if wrapped.Val != nil {
		t.Fatal("expected nil Val")
	}
}

func TestWrapValidatorEventsSet_Concurrency(t *testing.T) {
	val := map[idx.ValidatorID]hash.Event{}
	wrapped := WrapValidatorEventsSet(val)

	done := make(chan struct{})
	go func() {
		wrapped.Lock()
		wrapped.Val[idx.ValidatorID(1)] = hash.ZeroEvent
		wrapped.Unlock()
		close(done)
	}()
	<-done

	wrapped.RLock()
	defer wrapped.RUnlock()
	if len(wrapped.Val) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(wrapped.Val))
	}
}

func TestWrapEventsSet(t *testing.T) {
	val := make(hash.EventsSet)
	val.Add(hash.ZeroEvent)
	wrapped := WrapEventsSet(val)
	if wrapped == nil {
		t.Fatal("expected non-nil EventsSet")
	}
	if !wrapped.Val.Contains(hash.ZeroEvent) {
		t.Fatal("expected ZeroEvent in set")
	}
}

func TestWrapEventsSet_Nil(t *testing.T) {
	wrapped := WrapEventsSet(nil)
	if wrapped == nil {
		t.Fatal("expected non-nil EventsSet")
	}
}

func TestWrapEventsSet_Concurrency(t *testing.T) {
	val := make(hash.EventsSet)
	wrapped := WrapEventsSet(val)

	done := make(chan struct{})
	go func() {
		wrapped.Lock()
		wrapped.Val.Add(hash.ZeroEvent)
		wrapped.Unlock()
		close(done)
	}()
	<-done

	wrapped.RLock()
	defer wrapped.RUnlock()
	if !wrapped.Val.Contains(hash.ZeroEvent) {
		t.Fatal("expected ZeroEvent in set")
	}
}
