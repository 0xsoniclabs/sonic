package utils

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToFtm_doesNotOverflow256Ints(t *testing.T) {
	ftm := ToFtm(math.MaxUint64)
	require.Less(t, ftm.BitLen(), 256, "ToFtm should overflow for large input")
}
