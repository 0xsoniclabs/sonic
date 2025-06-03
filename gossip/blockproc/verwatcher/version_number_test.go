package verwatcher

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/version"
)

func TestVersionNumber_AreOrderedFollowingSemanticVersioningRules(t *testing.T) {
	cur := version.Version{}
	versions := []version.Version{}
	for major := range 4 {
		cur.Major = major
		for minor := range 4 {
			cur.Minor = minor
			for patch := range 4 {
				cur.Patch = patch

				// The develop version is the lowest in the version order.
				cur.ReleaseCandidate = 0
				cur.IsDevelopment = true
				versions = append(versions, cur)

				// Release candidates are the next higher in the version order.
				for rc := range 4 {
					cur.ReleaseCandidate = uint8(rc + 1)
					cur.IsDevelopment = false
					versions = append(versions, cur)
				}

				// Release versions are the highest in the version order.
				cur.ReleaseCandidate = 0
				versions = append(versions, cur)
			}
		}
	}

	for i := range len(versions) - 1 {
		a := toVersionNumber(versions[i])
		b := toVersionNumber(versions[i+1])
		if a >= b {
			t.Errorf("unexpected result, %s < %s (%d.%d.%d.%d < %d.%d.%d.%d) does not hold",
				versions[i], versions[i+1],
				(a>>48)&0xffff, (a>>32)&0xffff, (a>>16)&0xffff, (a>>0)&0xffff,
				(b>>48)&0xffff, (b>>32)&0xffff, (b>>16)&0xffff, (b>>0)&0xffff,
			)
		}
	}
}

func TestVersionNumber_PrintedInHumanReadableFormat(t *testing.T) {
	tests := map[versionNumber]string{
		0:                           "0.0.0-dev",
		1:                           "0.0.0-rc.1",
		2:                           "0.0.0-rc.2",
		255:                         "0.0.0-rc.255",
		256:                         "0.0.0",
		257:                         "0.0.0",
		0xffff:                      "0.0.0",
		1 << 48:                     "1.0.0-dev",
		1<<48 | 5:                   "1.0.0-rc.5",
		1<<48 | 256:                 "1.0.0",
		1<<48 | 2<<32 | 3<<16:       "1.2.3-dev",
		1<<48 | 2<<32 | 3<<16 | 17:  "1.2.3-rc.17",
		1<<48 | 2<<32 | 3<<16 | 256: "1.2.3",
	}

	for v, want := range tests {
		if got := v.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
		if got := fmt.Sprintf("%v", v); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}
