package serialization

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHexBytes_UnmarshalJSON_ValidHexString(t *testing.T) {
	var h HexBytes
	data := []byte(`"0x012abc"`)
	err := h.UnmarshalJSON(data)
	require.NoError(t, err)
}

func TestHexBytes_UnmarshallJSON_Invalid(t *testing.T) {
	var h HexBytes
	data := []byte(`"0xg"`)
	err := h.UnmarshalJSON(data)
	require.Error(t, err)
	data = []byte(`"01"`)
	err = h.UnmarshalJSON(data)
	require.Error(t, err)

}

func TestPublicKey_UnmarshalJSON_ShortHexString(t *testing.T) {
	var p PublicKey
	data := []byte(`"0x1234567"`)
	err := p.UnmarshalJSON(data)
	require.Error(t, err)
}

func TestPublicKey_UnmarshalJSON_ValidHexString(t *testing.T) {
	var p PublicKey
	data := []byte(`"0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"`)
	err := p.UnmarshalJSON(data)
	require.NoError(t, err)
}

func TestSignature_UnmarshallJSON_ShortHexString(t *testing.T) {
	var s Signature
	data := []byte(`"0x1234567"`)
	require.Error(t, s.UnmarshalJSON(data))
	data = []byte(`"0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"`)
	require.Error(t, s.UnmarshalJSON(data))
}

func TestSignature_UnmarshallJSON_ValidHexString(t *testing.T) {
	var s Signature
	data := []byte(`"0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"`)
	require.NoError(t, s.UnmarshalJSON(data))
}
