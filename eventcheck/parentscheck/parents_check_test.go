package parentscheck

import (
	"testing"
)

func TestNew(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("expected non-nil Checker")
	}
	if c.base == nil {
		t.Fatal("expected non-nil base checker")
	}
}

func TestErrPastTime(t *testing.T) {
	if ErrPastTime == nil {
		t.Fatal("ErrPastTime should not be nil")
	}
	if ErrPastTime.Error() != "event has lower claimed time than self-parent" {
		t.Fatalf("unexpected error message: %s", ErrPastTime.Error())
	}
}
