package provider

//go:generate mockgen -source=client.go -package=provider -destination=client_mock.go

// RpcClient is an interface for making RPC calls.
type RpcClient interface {
	// Call makes an RPC call to the given method with the given arguments.
	// The result of the call is stored in the result parameter.
	// The result parameter must be a pointer to the expected result type.
	Call(result any, method string, args ...any) error

	// Close closes the client.
	Close()
}
