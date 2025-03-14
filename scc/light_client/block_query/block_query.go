package blockquery

import (
	"fmt"

	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/sonic/scc/light_client/provider"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
)

//go:generate mockgen -source=block_query.go -package=blockquery -destination=block_query_mock.go

// BlockQueryI defines the interface for querying block information.
type BlockQueryI interface {
	// GetAddressInfo returns the state root hash, balance and nonce of the
	// given address at the given height.
	//
	// Parameters:
	// - address: The address to query.
	// - height: The block height to query.
	//
	// Returns:
	// - ProofQuery: The proof of the state root, balance, and nonce.
	// - error: An error if the query fails.
	GetAddressInfo(address common.Address, height idx.Block) (ProofQuery, error)

	// Close closes the BlockQuery.
	Close()
}

// ProofQuery represents proof data for an account's state.
// It includes the state root, account balance, and nonce.
//
// Fields:
// - stateRoot: The root hash corresponding to the state.
// - balance: The account's balance in Wei.
// - nonce: The nonce of the related account.
type ProofQuery struct {
	StorageHash common.Hash
	Balance     *uint256.Int
	Nonce       hexutil.Uint64
}

// BlockQuery implements the BlockQueryI interface and provides methods
// for querying the balance and nonce of an address at a given height.
type BlockQuery struct {
	client provider.RpcClient
}

// NewBlockQuery creates a new BlockQuery with a new RPC client
// connected to the given URL.
// The resulting BlockQuery must be closed after use.
//
// Parameters:
// - url: The URL of the RPC node to connect to.
//
// Returns:
// - *BlockQuery: A new instance of BlockQuery.
// - error: An error if the connection fails.
func NewBlockQuery(url string) (*BlockQuery, error) {
	client, err := rpc.Dial(url)
	if err != nil {
		return nil, err
	}
	return &BlockQuery{
		client: client,
	}, nil
}

// Close closes the BlockQuery.
// Closing an already closed instance has no effect.
func (b *BlockQuery) Close() {
	if b.client == nil {
		return
	}
	b.client.Close()
	b.client = nil
}

// GetAddressInfo returns the balance and nonce of the given address at the
// given height, with a proof of the state root.
//
// Parameters:
// - address: The address to query.
// - height: The height to query.
//
// Returns:
// - ProofQuery: The proof of the state root, balance, and nonce.
// - error: An error if the query fails.
func (b *BlockQuery) GetAddressInfo(address common.Address, height idx.Block) (ProofQuery, error) {
	result := ProofQuery{}
	heightString := fmt.Sprintf("0x%x", height)
	if height == provider.LatestBlock {
		heightString = "latest"
	}
	err := b.client.Call(
		&result,
		"eth_getProof",
		fmt.Sprintf("%v", address),
		[]string{fmt.Sprintf("%v", common.Hash{})},
		heightString,
	)
	if err != nil {
		return ProofQuery{}, err
	}
	return result, nil
}
