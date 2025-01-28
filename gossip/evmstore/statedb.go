package evmstore

import (
	"fmt"
	cc "github.com/0xsoniclabs/carmen/go/common"
	carmen "github.com/0xsoniclabs/carmen/go/state"
	_ "github.com/0xsoniclabs/carmen/go/state/gostate"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// GetLiveStateDb obtains StateDB for block processing - the live writable state
func (s *Store) GetLiveStateDb(stateRoot hash.Hash) (state.StateDB, error) {
	if s.liveStateDb == nil {
		return nil, fmt.Errorf("unable to get live StateDb - EvmStore is not open")
	}
	if s.liveStateDb.GetHash() != cc.Hash(stateRoot) {
		return nil, fmt.Errorf("unable to get Carmen live StateDB - unexpected state root (%x != %x)", s.liveStateDb.GetHash(), stateRoot)
	}
	return CreateCarmenStateDb(s.liveStateDb), nil
}

// GetTxPoolStateDB obtains StateDB for TxPool evaluation - the latest finalized, read-only.
// It is also used in emitter for emitterdriver contract reading at the start of an epoch.
func (s *Store) GetTxPoolStateDB() (state.StateDB, error) {
	// for TxPool and emitter it is ok to provide the newest state (and ignore the expected hash)
	if s.carmenState == nil {
		return nil, fmt.Errorf("unable to get TxPool StateDb - EvmStore is not open")
	}
	stateDb := carmen.CreateNonCommittableStateDBUsing(s.carmenState)
	return CreateCarmenStateDb(stateDb), nil
}

// GetArchiveBlockHeight provides the last block number available in the archive. Returns 0 if not known.
func (s *Store) GetArchiveBlockHeight() (height uint64, empty bool, err error) {
	if s.liveStateDb == nil {
		return 0, true, fmt.Errorf("unable to get archive block height - EvmStore is not open")
	}
	return s.liveStateDb.GetArchiveBlockHeight()
}

// GetRpcStateDb obtains archive StateDB for RPC requests evaluation
func (s *Store) GetRpcStateDb(blockNum *big.Int, stateRoot common.Hash) (state.StateDB, error) {
	// always use archive state (live state may mix data from various block heights)
	if s.liveStateDb == nil {
		return nil, fmt.Errorf("unable to get RPC StateDb - EvmStore is not open")
	}
	stateDb, err := s.liveStateDb.GetArchiveStateDB(blockNum.Uint64())
	if err != nil {
		return nil, err
	}
	if stateDb.GetHash() != cc.Hash(stateRoot) && blockNum.Sign() != 0 {
		return nil, fmt.Errorf("unable to get Carmen archive StateDB - unexpected state root (%x != %x)", stateDb.GetHash(), stateRoot)
	}
	return CreateCarmenStateDb(stateDb), nil
}

// CheckLiveStateHash returns if the hash of the current live StateDB hash matches (and fullsync is possible)
func (s *Store) CheckLiveStateHash(blockNum idx.Block, root hash.Hash) error {
	if s.liveStateDb == nil {
		return fmt.Errorf("unable to get live state - EvmStore is not open")
	}
	stateHash := s.liveStateDb.GetHash()
	if cc.Hash(root) != stateHash {
		return fmt.Errorf("hash of the EVM state is incorrect: blockNum: %d expected: %x reproducedHash: %x", blockNum, root, stateHash)
	}
	return nil
}

// CheckArchiveStateHash returns if the hash of the given archive StateDB hash matches
func (s *Store) CheckArchiveStateHash(blockNum idx.Block, root hash.Hash) error {
	if s.carmenState == nil {
		return fmt.Errorf("unable to get live state - EvmStore is not open")
	}
	archiveState, err := s.carmenState.GetArchiveState(uint64(blockNum))
	if err != nil {
		return fmt.Errorf("unable to get archive state: %w", err)
	}
	defer archiveState.Close()

	stateHash, err := archiveState.GetHash()
	if err != nil {
		return fmt.Errorf("unable to get archive state hash: %w", err)
	}
	if cc.Hash(root) != stateHash {
		return fmt.Errorf("hash of the archive EVM state is incorrect: blockNum: %d expected: %x actual: %x", blockNum, root, stateHash)
	}
	return nil
}
