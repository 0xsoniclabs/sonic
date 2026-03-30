package rpctest

import (
	"fmt"
	"os"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
)

type testState struct {
	*evmstore.CarmenStateDB
}

func newTestState() *testState {
	carmenDir, err := os.MkdirTemp("", "test.carmen")
	// TODO: pass testing.T and use t.TempDir() instead of os.MkdirTemp,
	// which will automatically clean up the directory after the test finishes.
	if err != nil {
		panic(fmt.Errorf("failed to create temporary dir for GenesisBuilder: %v", err))
	}
	carmenState, err := carmen.NewState(carmen.Parameters{
		Variant:      "go-file",
		Schema:       carmen.Schema(5),
		Archive:      carmen.S5Archive,
		Directory:    carmenDir,
		LiveCache:    1, // use minimum cache (not default)
		ArchiveCache: 1, // use minimum cache (not default)
	})
	if err != nil {
		panic(fmt.Errorf("failed to create carmen state; %s", err))
	}
	// Set cache size to lowest value possible
	carmenStateDb := carmen.CreateCustomStateDBUsing(carmenState, 1024)
	return &testState{evmstore.CreateCarmenStateDb(carmenStateDb)}
}

func (t *testState) Copy() state.StateDB {
	// FIXME: do copy
	panic("not implemented")
}

func (t *testState) setAccount(addr common.Address, acc TestAccount) {
	panic("not implemented")
}
