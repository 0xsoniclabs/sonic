package ethapi

import (
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestNewInvalidParamsError_ReportsTheMessageAndTheInvalidParamsCode(t *testing.T) {
	err := NewInvalidParamsError("some message")

	require.EqualError(t, err, "some message")

	var rpcErr rpc.Error
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, errCodeInvalidParams, rpcErr.ErrorCode())
	require.Equal(t, -32602, rpcErr.ErrorCode())
}
