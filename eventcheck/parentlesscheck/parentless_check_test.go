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
