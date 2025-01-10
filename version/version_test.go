package version

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

type testcase struct {
	vMajor, vMinor, vPatch uint16
	result                 uint64
	str                    string
}

func TestAsBigInt(t *testing.T) {
	require := require.New(t)

	prev := testcase{0, 0, 0, 0, "0.0.0"}
	for _, next := range []testcase{
		{0, 0, 1, 1, "0.0.1"},
		{0, 0, 2, 2, "0.0.2"},
		{0, 1, 0, 1000000, "0.1.0"},
		{0, 1, math.MaxUint16, 1065535, "0.1.65535"},
		{1, 0, 0, 1000000000000, "1.0.0"},
		{1, 0, math.MaxUint16, 1000000065535, "1.0.65535"},
		{1, 1, 0, 1000001000000, "1.1.0"},
		{2, 9, 9, 2000009000009, "2.9.9"},
		{3, 1, 0, 3000001000000, "3.1.0"},
		{math.MaxUint16, math.MaxUint16, math.MaxUint16, 65535065535065535, "65535.65535.65535"},
	} {
		a := ToU64(prev.vMajor, prev.vMinor, prev.vPatch)
		b := ToU64(next.vMajor, next.vMinor, next.vPatch)
		require.Equal(a, prev.result)
		require.Equal(b, next.result)
		require.Equal(U64ToString(a), prev.str)
		require.Equal(U64ToString(b), next.str)
		require.Greater(b, a)
		prev = next
	}
}

func TestVersion_parseVersion(t *testing.T) {
	require := require.New(t)

	tests := map[string]struct {
		major int
		minor int
		patch int
		meta  string
	}{
		"v1.2.3":                        {major: 1, minor: 2, patch: 3},
		"v1.2.3-alpha":                  {major: 1, minor: 2, patch: 3, meta: "alpha"},
		"v1.2.3-alpha-dirty":            {major: 1, minor: 2, patch: 3, meta: "alpha-dirty"},
		"some-non.stan-dard.12tag":      {},
		"!`@#$%^&*()_+{}|:<>?[]\\;',./": {},
		"myTestTag":                     {},
	}

	originalGitTag := GitTag
	for tag, want := range tests {
		versionMajor = 0
		versionMinor = 0
		versionPatch = 0
		versionMeta = ""

		GitTag = tag
		parseVersion()

		require.Equal(want.major, versionMajor, "major version mismatch")
		require.Equal(want.minor, versionMinor, "minor version mismatch")
		require.Equal(want.patch, versionPatch, "patch version mismatch")
		require.Equal(want.meta, versionMeta, "meta version mismatch")
	}

	GitTag = originalGitTag
}
