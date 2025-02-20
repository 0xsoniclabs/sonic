package serialization

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHexBytes_UnmarshalJSON(t *testing.T) {
	var h HexBytes

	data := []byte(`"0x012abc"`)
	err := h.UnmarshalJSON(data)
	if err != nil {
		panic(err)
	}
}

func TestHexBytes_InvalidUnmarshallJSON(t *testing.T) {
	var h HexBytes
	data := []byte(`"0xg"`)
	err := h.UnmarshalJSON(data)
	require.Error(t, err)
	data = []byte(`"01"`)
	err = h.UnmarshalJSON(data)
	require.Error(t, err)

}

func TestPublicKey_UnmarshalJSON(t *testing.T) {
	var p PublicKey

	data := []byte(`"0xdeadbeef"`)
	err := p.UnmarshalJSON(data)
	if err != nil {
		panic(err)
	}
}
