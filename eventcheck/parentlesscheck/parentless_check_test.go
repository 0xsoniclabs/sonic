package parentlesscheck

import (
	"errors"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/dag"
)

type fakeEvent struct {
	dag.Event
}

type fakeLightCheck struct {
	err error
}

func (f *fakeLightCheck) check(e dag.Event) error {
	return f.err
}

type fakeHeavyCheck struct {
	enqueued bool
	err      error
}

func (f *fakeHeavyCheck) Enqueue(e dag.Event, checked func(error)) error {
	f.enqueued = true
	if f.err != nil {
		return f.err
	}
	checked(nil)
	return nil
}

func TestChecker_Enqueue_LightCheckFails(t *testing.T) {
	lightErr := errors.New("light check failed")
	light := &fakeLightCheck{err: lightErr}
	heavy := &fakeHeavyCheck{}

	c := &Checker{
		HeavyCheck: heavy,
		LightCheck: light.check,
	}

	var gotErr error
	c.Enqueue(&fakeEvent{}, func(err error) {
		gotErr = err
	})

	if gotErr != lightErr {
		t.Fatalf("expected light check error, got %v", gotErr)
	}
	if heavy.enqueued {
		t.Fatal("heavy check should not be called when light check fails")
	}
}

func TestChecker_Enqueue_LightCheckPasses(t *testing.T) {
	light := &fakeLightCheck{err: nil}
	heavy := &fakeHeavyCheck{}

	c := &Checker{
		HeavyCheck: heavy,
		LightCheck: light.check,
	}

	var gotErr error
	c.Enqueue(&fakeEvent{}, func(err error) {
		gotErr = err
	})

	if gotErr != nil {
		t.Fatalf("expected nil error, got %v", gotErr)
	}
	if !heavy.enqueued {
		t.Fatal("heavy check should be called when light check passes")
	}
}
