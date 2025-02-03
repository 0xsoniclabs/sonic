package version

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersion_parseVersion(t *testing.T) {
	require := require.New(t)

	tests := map[string]Version{
		"":                          {Major: versionMajor, Minor: versionMinor, Meta: "dev"},
		"::":                        {Major: versionMajor, Minor: versionMinor, Meta: "dev"},
		"v1.2.3::":                  {Major: 1, Minor: 2, Patch: 3},
		"v1.2.3::dirty":             {Major: 1, Minor: 2, Patch: 3, Dirty: true},
		"v1.2.3:dev:dirty":          {Major: 1, Minor: 3, Meta: "dev"},
		"v1.2.3-rc1::":              {Major: 1, Minor: 2, Patch: 3, Meta: "rc1"},
		"v1.2.3-rc1-beta-12::":      {Major: 1, Minor: 2, Patch: 3, Meta: "rc1-beta-12"},
		"v17.1258.3478-rc52-beta::": {Major: 17, Minor: 1258, Patch: 3478, Meta: "rc52-beta"},
	}

	for tag, want := range tests {
		version, err := parseVersion(tag)
		require.NoError(err, "failed to parse version")
		require.Equal(want, version, "version mismatch")
	}
}

func TestVersion_parseVersion_shouldFailInvalidVersionString(t *testing.T) {
	require := require.New(t)

	tests := []string{
		// missing or too many parts
		"v1.2.3",
		"v1.2.3:",
		"v1.2.3:::",

		// invalid version parts
		"some-non.stan-#.12tag::",
		"!`@#$%^&*()_{}|<>?[]\\;',./::",

		// invalid meta part
		"v1.2.3-ü::",

		// invalid dev part
		"v1.2.3:dev-:",

		// invalid dirty part
		"v1.2.3::dirty-",
	}

	for _, tag := range tests {
		_, err := parseVersion(tag)
		require.Error(err, "expected parsing to fail")
	}
}
