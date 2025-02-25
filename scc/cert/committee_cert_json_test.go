package cert

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/utils/jsonhex"
	"github.com/stretchr/testify/require"
)

func TestCommitteeCertificateJson_String(t *testing.T) {
	var c committeeCertificateJson
	want := fmt.Sprintf(`{"chainId":0,"period":0,"members":[],"signers":"%v","signature":"%v"}`,
		jsonhex.Bytes{}, jsonhex.Bytes96{})
	require.Equal(t, want, c.String())
}
