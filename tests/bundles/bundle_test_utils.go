package bundles

import (
	"context"
	"errors"
	"slices"

	"github.com/0xsoniclabs/sonic/api/sonicapi"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

// GetBundleInfo calls the sonic_getBundleInfo RPC method to retrieve
// information about the execution of a transaction bundle.
func GetBundleInfo(
	ctxt context.Context,
	client *rpc.Client,
	executionPlanHash common.Hash,
) (*sonicapi.RPCBundleInfo, error) {
	var info *sonicapi.RPCBundleInfo
	err := client.CallContext(
		ctxt,
		&info,
		"sonic_getBundleInfo",
		executionPlanHash,
	)
	if err == nil && info == nil {
		return nil, ethereum.NotFound
	}
	return info, err
}

// WaitForBundleExecution waits until the bundle execution information of a
// transaction bundle becomes available through the sonic_getBundleInfo RPC
// method. The waiting time can be limited by the provided context.
func WaitForBundleExecution(
	ctxt context.Context,
	client *rpc.Client,
	executionPlanHash common.Hash,
) (*sonicapi.RPCBundleInfo, error) {
	infos, err := WaitForBundleExecutions(
		ctxt, client,
		[]common.Hash{executionPlanHash},
	)
	return infos[0], err
}

// WaitForBundleExecutions waits until the bundle execution information of a
// list of execution plans becomes available through the sonic_getBundleInfo RPC
// method. The waiting time can be limited by the provided context.
func WaitForBundleExecutions(
	ctxt context.Context,
	client *rpc.Client,
	executionPlanHashes []common.Hash,
) ([]*sonicapi.RPCBundleInfo, error) {

	infos := make([]*sonicapi.RPCBundleInfo, len(executionPlanHashes))
	err := tests.WaitFor(ctxt, func(innerCtx context.Context) (bool, error) {
		for i, plan := range executionPlanHashes {
			if infos[i] != nil {
				continue
			}

			info, err := GetBundleInfo(innerCtx, client, plan)
			if err != nil {
				if errors.Is(err, ethereum.NotFound) {
					continue
				}
				return false, err
			}

			if info != nil {
				infos[i] = info
			}
		}
		return !slices.Contains(infos, nil), nil
	})
	return infos, err
}
