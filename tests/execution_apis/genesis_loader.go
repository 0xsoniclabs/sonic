package execution_apis

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	rpctest "github.com/0xsoniclabs/sonic/api/rpc_test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

// LoadGenesis loads a genesis.json file and returns the parsed genesis object.
func LoadGenesis(path string) (*core.Genesis, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading genesis file: %w", err)
	}

	var genesis core.Genesis
	if err := json.Unmarshal(data, &genesis); err != nil {
		return nil, fmt.Errorf("parsing genesis JSON: %w", err)
	}

	return &genesis, nil
}

// GenesisBlock returns a rpctest.Block representing the genesis block (block 0)
// derived from the genesis specification.
func GenesisBlock(genesis *core.Genesis) rpctest.Block {
	var baseFee *big.Int
	if genesis.BaseFee != nil {
		baseFee = genesis.BaseFee
	}

	return rpctest.Block{
		Number:  0,
		Hash:    common.Hash{}, // will be derived from first chain block's parent
		BaseFee: baseFee,
	}
}
