package provider

//go:generate mockgen -source=client.go -package=provider -destination=client_mock.go

type RpcClient interface {
	Call(result any, method string, args ...any) error
	Close()
}
