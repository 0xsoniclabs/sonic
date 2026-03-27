package autocompact

import (
	"bytes"
	"errors"
	"testing"
)

func TestDevnullContainer_Add(t *testing.T) {
	d := DevnullContainer{}
	d.Add([]byte("key"), 100)
	// should be no-op
}

func TestDevnullContainer_Merge(t *testing.T) {
	d := DevnullContainer{}
	other := NewForwardCont()
	other.Add([]byte("key"), 100)
	d.Merge(other)
	// should be no-op
}

func TestDevnullContainer_Error(t *testing.T) {
	d := DevnullContainer{}
	if d.Error() != nil {
		t.Fatal("expected nil error")
	}
}

func TestDevnullContainer_Reset(t *testing.T) {
	d := DevnullContainer{}
	d.Reset()
	// should be no-op
}

func TestDevnullContainer_Size(t *testing.T) {
	d := DevnullContainer{}
	if d.Size() != 0 {
		t.Fatalf("expected size 0, got %d", d.Size())
	}
}

func TestDevnullContainer_Ranges(t *testing.T) {
	d := DevnullContainer{}
	r := d.Ranges()
	if len(r) != 0 {
		t.Fatalf("expected 0 ranges, got %d", len(r))
	}
}

func TestNewForwardCont(t *testing.T) {
	c := NewForwardCont()
	if c == nil {
		t.Fatal("expected non-nil container")
	}
	if c.Size() != 0 {
		t.Fatal("expected size 0")
	}
}

func TestNewBackwardsCont(t *testing.T) {
	c := NewBackwardsCont()
	if c == nil {
		t.Fatal("expected non-nil container")
	}
}

func TestNewDevnullCont(t *testing.T) {
	c := NewDevnullCont()
	if _, ok := c.(DevnullContainer); !ok {
		t.Fatal("expected DevnullContainer")
	}
}

func TestForwardContainer_MonotonicAdd(t *testing.T) {
	c := NewForwardCont()
	c.Add([]byte{0x01}, 10)
	c.Add([]byte{0x02}, 20)
	c.Add([]byte{0x03}, 30)

	if c.Size() != 60 {
		t.Fatalf("expected size 60, got %d", c.Size())
	}
	ranges := c.Ranges()
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range for monotonic forward, got %d", len(ranges))
	}
	if !bytes.Equal(ranges[0].minKey, []byte{0x01}) {
		t.Fatalf("expected minKey 0x01, got %v", ranges[0].minKey)
	}
	if !bytes.Equal(ranges[0].maxKey, []byte{0x03}) {
		t.Fatalf("expected maxKey 0x03, got %v", ranges[0].maxKey)
	}
	if c.Error() != nil {
		t.Fatalf("expected no error, got %v", c.Error())
	}
}

func TestForwardContainer_NonMonotonicAdd(t *testing.T) {
	c := NewForwardCont()
	c.Add([]byte{0x03}, 10)
	c.Add([]byte{0x01}, 20) // non-monotonic: creates new range

	ranges := c.Ranges()
	if len(ranges) != 2 {
		t.Fatalf("expected 2 ranges, got %d", len(ranges))
	}
}

func TestForwardContainer_TooManyRanges(t *testing.T) {
	c := NewForwardCont()
	c.Add([]byte{0x03}, 10)
	c.Add([]byte{0x01}, 10) // new range
	c.Add([]byte{0x05}, 10)
	c.Add([]byte{0x02}, 10) // new range (3rd)

	if c.Error() == nil {
		t.Fatal("expected error for too many ranges")
	}
}

func TestBackwardsContainer_MonotonicAdd(t *testing.T) {
	c := NewBackwardsCont()
	c.Add([]byte{0x03}, 10)
	c.Add([]byte{0x02}, 20)
	c.Add([]byte{0x01}, 30)

	if c.Size() != 60 {
		t.Fatalf("expected size 60, got %d", c.Size())
	}
	ranges := c.Ranges()
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range for monotonic backwards, got %d", len(ranges))
	}
	if !bytes.Equal(ranges[0].minKey, []byte{0x01}) {
		t.Fatalf("expected minKey 0x01, got %v", ranges[0].minKey)
	}
	if !bytes.Equal(ranges[0].maxKey, []byte{0x03}) {
		t.Fatalf("expected maxKey 0x03, got %v", ranges[0].maxKey)
	}
}

func TestBackwardsContainer_NonMonotonicAdd(t *testing.T) {
	c := NewBackwardsCont()
	c.Add([]byte{0x01}, 10)
	c.Add([]byte{0x03}, 20) // non-monotonic for backwards

	ranges := c.Ranges()
	if len(ranges) != 2 {
		t.Fatalf("expected 2 ranges, got %d", len(ranges))
	}
}

func TestForwardContainer_EqualKeys(t *testing.T) {
	c := NewForwardCont()
	c.Add([]byte{0x01}, 10)
	c.Add([]byte{0x01}, 10) // equal key, should extend same range

	ranges := c.Ranges()
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range for equal keys, got %d", len(ranges))
	}
}

func TestBackwardsContainer_EqualKeys(t *testing.T) {
	c := NewBackwardsCont()
	c.Add([]byte{0x01}, 10)
	c.Add([]byte{0x01}, 10)

	ranges := c.Ranges()
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range for equal keys, got %d", len(ranges))
	}
}

func TestContainer_Reset(t *testing.T) {
	c := NewForwardCont()
	c.Add([]byte{0x01}, 10)
	c.Add([]byte{0x02}, 20)
	c.Reset()

	if c.Size() != 0 {
		t.Fatalf("expected size 0 after reset, got %d", c.Size())
	}
	if len(c.Ranges()) != 0 {
		t.Fatalf("expected 0 ranges after reset, got %d", len(c.Ranges()))
	}
}

func TestForwardContainer_Merge(t *testing.T) {
	c1 := NewForwardCont()
	c1.Add([]byte{0x01}, 10)
	c1.Add([]byte{0x02}, 20)

	c2 := NewForwardCont()
	c2.Add([]byte{0x03}, 30)
	c2.Add([]byte{0x04}, 40)

	c1.Merge(c2)
	if c1.Size() != 100 {
		t.Fatalf("expected size 100 after merge, got %d", c1.Size())
	}
}

func TestContainer_MergeWithError(t *testing.T) {
	c1 := NewForwardCont()
	c2 := &MonotonicContainer{
		forward: true,
		err:     errors.New("test error"),
	}

	c1.Merge(c2)
	if c1.Error() == nil {
		t.Fatal("expected error to propagate through Merge")
	}
}

func TestContainer_ErrorWithPresetErr(t *testing.T) {
	c := &MonotonicContainer{
		forward: true,
		err:     errors.New("preset error"),
	}
	if c.Error() == nil || c.Error().Error() != "preset error" {
		t.Fatal("expected preset error")
	}
}
