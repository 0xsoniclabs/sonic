package opera

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultRulesAreValid(t *testing.T) {
	rules := map[string]Rules{
		"mainnet": MainNetRules(),
		"fakenet": FakeNetRules(),
	}
	for name, r := range rules {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, r.Validate())
		})
	}
}
