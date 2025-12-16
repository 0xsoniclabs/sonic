package evmcore

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewEVMBlockContext_DifficultyIsOne(t *testing.T) {
	header := &EvmHeader{
		Number: big.NewInt(12),
	}
	context := NewEVMBlockContext(header, nil, nil)
	require.Equal(t, big.NewInt(1), context.Difficulty)
}

func TestNewEVMBlockContextWithDifficulty_UsesProvidedDifficulty(t *testing.T) {
	header := &EvmHeader{
		Number: big.NewInt(12),
	}
	for i := range int64(10) {
		difficulty := big.NewInt(i)
		context := NewEVMBlockContextWithDifficulty(header, nil, nil, difficulty)
		require.Equal(t, difficulty, context.Difficulty)
	}
}
