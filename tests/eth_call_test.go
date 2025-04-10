package tests

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEthCall(t *testing.T) {
	net := StartIntegrationTestNet(t)

	client, err := net.GetClient()
	if err != nil {
		t.Fatalf("Failed to connect to the integration test network: %v", err)
	}
	defer client.Close()

	var dataFile jsonrpcMessage
	json.Unmarshal([]byte(data), &dataFile)

	c := client.Client()

	var res interface{}
	err = c.Call(&res, "eth_call", dataFile.Params[0], dataFile.Params[1], dataFile.Params[2])
	require.NoError(t, err)

	t.Logf("result: %v", res)

	t.Logf("end")
}

func TestParseJsonEthCall(t *testing.T) {
	//t.Logf("data", strings.Split(data, "\"")[1])

	var res jsonrpcMessage
	json.Unmarshal([]byte(data), &res)

	t.Logf("method: %v", res.Method)

	t.Logf("end")
}

type jsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  []interface{}   `json:"params,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
