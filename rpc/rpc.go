package sonicrpc

import (
	"github.com/0xsoniclabs/sonic/ethapi"
	sonicapi "github.com/0xsoniclabs/sonic/rpc/sonic"
	rpctypes "github.com/0xsoniclabs/sonic/rpc/types"
	"github.com/ethereum/go-ethereum/rpc"
)

func GetAPIs(apiBackend rpctypes.Backend) []rpc.API {
	nonceLock := new(ethapi.AddrLocker)
	return []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   ethapi.NewPublicEthereumAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   ethapi.NewPublicBlockChainAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "dag",
			Version:   "1.0",
			Service:   ethapi.NewPublicDAGChainAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   ethapi.NewPublicTransactionPoolAPI(apiBackend, nonceLock),
			Public:    true,
		}, {
			Namespace: "txpool",
			Version:   "1.0",
			Service:   ethapi.NewPublicTxPoolAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   ethapi.NewPrivateDebugAPI(apiBackend),
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   ethapi.NewPublicAccountAPI(apiBackend.AccountManager()),
			Public:    true,
		}, {
			Namespace: "personal",
			Version:   "1.0",
			Service:   ethapi.NewPrivateAccountAPI(apiBackend, nonceLock),
			Public:    false,
		}, {
			Namespace: "abft",
			Version:   "1.0",
			Service:   ethapi.NewPublicAbftAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "sonic",
			Version:   "1.0",
			Service:   ethapi.NewPublicSccApi(apiBackend),
			Public:    true,
		}, {
			Namespace: "sonic",
			Version:   "1.0",
			Service:   sonicapi.NewPublicBundleAPI(apiBackend),
			Public:    true,
		},
	}
}
