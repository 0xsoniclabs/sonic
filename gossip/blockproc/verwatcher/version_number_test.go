package verwatcher

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/version"
	"github.com/stretchr/testify/require"
)

func TestVersionNumber_AreOrderedFollowingSemanticVersioningRules(t *testing.T) {
	versions := []version.Version{}
	for major := range []int{0, 1, 2, 3} {
		for minor := range []int{0, 1, 2, 3} {
			for patch := range []int{0, 1, 2, 3} {
				version := version.Version{Major: major, Minor: minor, Patch: patch, Meta: "pre"}
				versions = append(versions, version)
				version.Meta = ""
				versions = append(versions, version)
			}
		}
	}

	for i := range len(versions) - 1 {
		if toVersionNumber(versions[i]) > toVersionNumber(versions[i+1]) {
			t.Errorf("%s > %s", versions[i], versions[i+1])
		}
	}
}

func TestVersionNumber_toVersionNumber_AnyMetaTagIsTreatedEquivalent(t *testing.T) {
	version1 := version.Version{Meta: "alpha"}
	version2 := version.Version{Meta: "beta"}
	require.Equal(t, toVersionNumber(version1), toVersionNumber(version2))
}

func TestVersionNumber_ReleasesAreHigherThanVersionsWithMetaData(t *testing.T) {
	release := version.Version{}
	prerelease := version.Version{Meta: "some-meta"}
	require.Less(t, toVersionNumber(prerelease), toVersionNumber(release))
}

func TestVersionNumber_PrintedInHumanReadableFormat(t *testing.T) {
	tests := map[versionNumber]string{
		0:                "v0.0.0-pre",
		0x0001 << 48:     "v1.0.0-pre",
		0x0001 << 32:     "v0.1.0-pre",
		0x0001 << 16:     "v0.0.1-pre",
		1:                "v0.0.0-pre",
		255:              "v0.0.0-pre",
		256:              "v0.0.0",
		257:              "v0.0.0",
		0x0001<<48 | 256: "v1.0.0",
		0x0001<<32 | 256: "v0.1.0",
		0x0001<<16 | 256: "v0.0.1",
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
