package jsonhex

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBytes_MarshalJSON_HandlesAllCases(t *testing.T) {
	h := Bytes([]byte{0x01, 0x2a, 0xbc})
	data, err := h.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, []byte(`"0x012abc"`), data)
	h = nil
	data, err = h.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, []byte(`"null"`), data)
	h = Bytes([]byte{0x1, 0x2a, 0xbc})
	data, err = h.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, []byte(`"0x012abc"`), data)
}

func TestBytes_UnmarshalJSON_ValidHexString_DoesNotProduceError(t *testing.T) {
	var h Bytes
	data := []byte(`"0x012abc"`)
	require.NoError(t, h.UnmarshalJSON(data))
	data = []byte(`"null"`)
	require.NoError(t, h.UnmarshalJSON(data))
	data = []byte(`"0x12abc"`)
	require.NoError(t, h.UnmarshalJSON(data))
	require.Equal(t, Bytes([]byte{0x1, 0x2a, 0xbc}), h)
}

func TestBytes_UnmarshalJSON_InvalidHexString_ProducesError(t *testing.T) {
	var h Bytes
	data := []byte(`"0xg"`)
	err := h.UnmarshalJSON(data)
	require.Error(t, err)
	data = []byte(`"01"`)
	err = h.UnmarshalJSON(data)
	require.Error(t, err)
}

func TestBytes_String_IsCorrectlyProduced(t *testing.T) {
	h := Bytes([]byte{0x01, 0x2a, 0xbc})
	require.Equal(t, `"0x012abc"`, h.String())
	h = nil
	require.Equal(t, `"null"`, h.String())
}

func TestBytes_CanBeJSONEncodedAndDecoded(t *testing.T) {
	h := Bytes([]byte{0x01, 0x2a, 0xbc})
	data, err := json.Marshal(h)
	require.NoError(t, err)
	var h2 Bytes
	err = json.Unmarshal(data, &h2)
	require.NoError(t, err)
	require.Equal(t, h, h2)
}

func TestBytes48_MarshalJSON_StringIsCorrectlyProduced(t *testing.T) {
	p := Bytes48([48]byte{0x01})
	data, err := p.MarshalJSON()
	require.NoError(t, err)
	expected := `"0x010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"`
	require.Equal(t, []byte(expected), data)
}

func TestBytes48_MarshalJSON_ZeroValue(t *testing.T) {
	var p Bytes48
	data, err := p.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, []byte(`"0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"`), data)
}

func TestBytes48_UnmarshalJSON_TooShortHexStringIsRejected(t *testing.T) {
	var p Bytes48
	data := []byte(`"0x1234567"`)
	err := p.UnmarshalJSON(data)
	require.Error(t, err)
}

func TestBytes48_UnmarshalJSON_ValidHexString(t *testing.T) {
	var p Bytes48
	data := []byte(`"0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"`)
	err := p.UnmarshalJSON(data)
	require.NoError(t, err)
	want := Bytes48{0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	require.Equal(t, want, p)
}

func TestBytes48_String(t *testing.T) {
	byteString := `"0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"`
	p := Bytes48([]byte{47: 0x01})
	require.Equal(t, byteString, p.String())
}

func TestBytes48_CanBeJsonEncodedAndDecoded(t *testing.T) {
	p := Bytes48([]byte{47: 0x01})
	data, err := json.Marshal(p)
	require.NoError(t, err)
	var p2 Bytes48
	err = json.Unmarshal(data, &p2)
	require.NoError(t, err)
	require.Equal(t, p, p2)
}

func TestBytes96_MarshalJSON(t *testing.T) {
	s := Bytes96([96]byte{0x01})
	data, err := s.MarshalJSON()
	require.NoError(t, err)
	expected := []byte(`"0x010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"`)
	require.Equal(t, len(expected), len(data))
	require.Equal(t, []byte(expected), data)
}

func TestBytes96_UnmarshalJSON_ShortHexString(t *testing.T) {
	var s Bytes96
	data := []byte(`0x1234567`)
	require.Error(t, s.UnmarshalJSON(data))
	data = []byte(`0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef`)
	require.Error(t, s.UnmarshalJSON(data))
}

func TestBytes96_UnmarshalJSON_ValidHexString(t *testing.T) {
	var s Bytes96
	data := []byte(`"0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"`)
	require.NoError(t, s.UnmarshalJSON(data))
	want := Bytes96{0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	require.Equal(t, want, s)
}

func TestBytes96_UnmarshalJSON_String(t *testing.T) {
	byteString := `"0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"`
	S := Bytes96([]byte{95: 0x01})
	require.Equal(t, byteString, S.String())
}

func TestBytes96_CanBeJsonEncodedAndDecoded(t *testing.T) {
	s := Bytes96([]byte{95: 0x01})
	data, err := json.Marshal(s)
	require.NoError(t, err)
	var s2 Bytes96
	err = json.Unmarshal(data, &s2)
	require.NoError(t, err)
	require.Equal(t, s, s2)
}
