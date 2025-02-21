package serialization

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHexBytes_MarshalJSON(t *testing.T) {
	h := HexBytes([]byte{0x01, 0x2a, 0xbc})
	data, err := h.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, []byte(`"0x012abc"`), data)
	h = nil
	data, err = h.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, []byte("null"), data)
}

func TestHexBytes_UnmarshalJSON_ValidHexString(t *testing.T) {
	var h HexBytes
	data := []byte(`"0x012abc"`)
	require.NoError(t, h.UnmarshalJSON(data))
	data = []byte(`"null"`)
	require.NoError(t, h.UnmarshalJSON(data))
	data = []byte(`"0x12abc"`)
	require.NoError(t, h.UnmarshalJSON(data))
	require.Equal(t, HexBytes([]byte{0x1, 0x2a, 0xbc}), h)
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

func TestHexBytes48_MarshalJSON(t *testing.T) {
	p := HexBytes48([48]byte{0x01})
	data, err := p.MarshalJSON()
	require.NoError(t, err)
	expected := `"0x010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"`
	require.Equal(t, []byte(expected), data)
}

func TestHexBytes48_UnmarshalJSON_ShortHexString(t *testing.T) {
	var p HexBytes48
	data := []byte(`"0x1234567"`)
	err := p.UnmarshalJSON(data)
	require.Error(t, err)
}

func TestHexBytes48_UnmarshalJSON_ValidHexString(t *testing.T) {
	var p HexBytes48
	data := []byte(`"0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"`)
	err := p.UnmarshalJSON(data)
	require.NoError(t, err)
}

func TestHexBytes48_String(t *testing.T) {
	byteString := "0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"
	p := HexBytes48([]byte{47: 0x01})
	require.Equal(t, byteString, p.String())
}

func TestHexBytes96_MarshalJSON(t *testing.T) {
	s := HexBytes96([96]byte{0x01})
	data, err := s.MarshalJSON()
	require.NoError(t, err)
	expected := []byte(`"0x010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"`)
	require.Equal(t, len(expected), len(data))
	require.Equal(t, []byte(expected), data)
}

func TestHexBytes96_UnmarshallJSON_ShortHexString(t *testing.T) {
	var s HexBytes96
	data := []byte(`"0x1234567"`)
	require.Error(t, s.UnmarshalJSON(data))
	data = []byte(`"0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"`)
	require.Error(t, s.UnmarshalJSON(data))
}

func TestHexBytes96_UnmarshallJSON_ValidHexString(t *testing.T) {
	var s HexBytes96
	data := []byte(`"0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"`)
	require.NoError(t, s.UnmarshalJSON(data))
}

func TestHexBytes96_UnmarshalJSON_String(t *testing.T) {
	byteString := "0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"
	S := HexBytes96([]byte{95: 0x01})
	require.Equal(t, byteString, S.String())
}
