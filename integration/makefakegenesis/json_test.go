package makefakegenesis

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHexBytes48_MarshalJson_ProducesHexString(t *testing.T) {
	data := hexBytes48{}
	for i := range data {
		data[i] = byte(i + 1)
	}
	expected := fmt.Sprintf("\"0x%x\"", data[:])
	actual, err := json.Marshal(data)
	require.NoError(t, err)
	require.Equal(t, expected, string(actual))
}

func TestHexBytes48_UnmarshalJson_AcceptsHexString(t *testing.T) {
	want := hexBytes48{}
	for i := range want {
		want[i] = byte(i + 1)
	}
	input := fmt.Sprintf("\"0x%x\"", want[:])
	fmt.Printf("input: %s\n", input)

	got := hexBytes48{}
	require.NoError(t, json.Unmarshal([]byte(input), &got))
	require.Equal(t, want, got)
}

func TestHexBytes48_UnmarshalAndMarshalJson_RoundTrip(t *testing.T) {
	want := hexBytes48{}
	for i := range want {
		want[i] = byte(i + 1)
	}

	data, err := json.Marshal(want)
	require.NoError(t, err)

	got := hexBytes48{}
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got)
}
