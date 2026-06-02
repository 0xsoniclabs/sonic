package execution_api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"
)

// Dispatcher wraps an in-process JSON-RPC server for testing.
// It registers API services and dispatches raw JSON-RPC requests.
type Dispatcher struct {
	server *rpc.Server
	client *rpc.Client
}

// NewDispatcher creates a new Dispatcher with the given RPC APIs registered.
func NewDispatcher(apis []rpc.API) (*Dispatcher, error) {
	server := rpc.NewServer()

	for _, api := range apis {
		if err := server.RegisterName(api.Namespace, api.Service); err != nil {
			return nil, fmt.Errorf("registering %s API: %w", api.Namespace, err)
		}
	}

	client := rpc.DialInProc(server)

	return &Dispatcher{
		server: server,
		client: client,
	}, nil
}

// Call sends a raw JSON-RPC request and returns the raw response.
// The request should be a valid JSON-RPC 2.0 request object.
func (d *Dispatcher) Call(ctx context.Context, request json.RawMessage) (json.RawMessage, error) {
	var req struct {
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
		ID     json.RawMessage   `json:"id"`
	}

	if err := json.Unmarshal(request, &req); err != nil {
		return nil, fmt.Errorf("parsing request: %w", err)
	}

	// Make the call through the client
	var result json.RawMessage
	err := d.client.CallContext(ctx, &result, req.Method, rawParams(req.Params)...)

	// Build a JSON-RPC response matching the expected format
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      json.RawMessage(req.ID),
	}

	if err != nil {
		// Check if it's an RPC error with a code
		if rpcErr, ok := err.(rpc.Error); ok {
			errObj := map[string]any{
				"code":    rpcErr.ErrorCode(),
				"message": rpcErr.Error(),
			}
			// Check for data field
			if dataErr, ok := err.(rpc.DataError); ok {
				errObj["data"] = dataErr.ErrorData()
			}
			response["error"] = errObj
		} else {
			response["error"] = map[string]any{
				"code":    -32000,
				"message": err.Error(),
			}
		}
	} else {
		response["result"] = result
	}

	return json.Marshal(response)
}

// Close shuts down the dispatcher.
func (d *Dispatcher) Close() {
	d.client.Close()
	d.server.Stop()
}

// rawParams converts a slice of json.RawMessage to []interface{} for use with CallContext.
func rawParams(params []json.RawMessage) []interface{} {
	result := make([]interface{}, len(params))
	for i, p := range params {
		result[i] = p
	}
	return result
}
