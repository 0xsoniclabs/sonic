package evmstore

import (
	"context"
	"fmt"
	"path/filepath"

	cc "github.com/0xsoniclabs/carmen/go/common"
	"github.com/0xsoniclabs/carmen/go/database/mpt"
	"github.com/0xsoniclabs/carmen/go/database/mpt/io"
	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

func (s *Store) VerifyWorldState(expectedBlockNum uint64, expectedHash common.Hash) error {
	if s.carmenState != nil {
		return fmt.Errorf("carmen state must be closed for the world state verification")
	}

	observer := verificationObserver{s.Log}

	// check hash of the live state / last state in the archive
	if err := verifyLastState(s.parameters, expectedBlockNum, expectedHash); err != nil {
		return fmt.Errorf("verification of the last block failed: %w", err)
	}
	s.Log.Info("State hash matches the last block state root.")

	// verify the live world state
	liveDir := filepath.Join(s.parameters.Directory, "live")
	info, err := io.CheckMptDirectoryAndGetInfo(liveDir)
	if err != nil {
		return fmt.Errorf("failed to check live state dir: %w", err)
	}
	if err := mpt.VerifyFileLiveTrie(context.Background(), liveDir, info.Config, observer); err != nil {
		return fmt.Errorf("live state verification failed: %w", err)
	}
	s.Log.Info("Live state verified successfully.")

	// verify the archive
	if s.parameters.Archive != carmen.S5Archive {
		return nil // skip archive checks when S5 archive is not used
	}
	archiveDir := filepath.Join(s.parameters.Directory, "archive")
	archiveInfo, err := io.CheckMptDirectoryAndGetInfo(archiveDir)
	if err != nil {
		return fmt.Errorf("failed to check archive dir: %w", err)
	}
	if err := mpt.VerifyArchiveTrie(context.Background(), archiveDir, archiveInfo.Config, observer); err != nil {
		return fmt.Errorf("archive verification failed: %w", err)
	}
	s.Log.Info("Archive verified successfully.")
	return nil
}

func verifyLastState(params carmen.Parameters, expectedBlockNum uint64, expectedHash common.Hash) error {
	liveState, err := carmen.NewState(params)
	if err != nil {
		return fmt.Errorf("failed to open carmen live state in %s: %w", params.Directory, err)
	}
	defer liveState.Close()
	if err := checkStateHash(liveState, expectedHash); err != nil {
		return fmt.Errorf("live state check failed; %w", err)
	}

	lastArchiveBlock, _, err := liveState.GetArchiveBlockHeight()
	if err != nil {
		return fmt.Errorf("failed to get last archive block height; %w", err)
	}
	if lastArchiveBlock != expectedBlockNum {
		return fmt.Errorf("the last archive block height does not match (%d != %d)", lastArchiveBlock, expectedBlockNum)
	}

	if params.Archive == carmen.NoArchive {
		return nil // skip archive checks when archive is not enabled
	}
	archiveState, err := liveState.GetArchiveState(lastArchiveBlock)
	if err != nil {
		return fmt.Errorf("failed to get carmen archive state; %w", err)
	}
	defer archiveState.Close()
	if err := checkStateHash(archiveState, expectedHash); err != nil {
		return fmt.Errorf("archive state check failed; %w", err)
	}
	return nil
}

func checkStateHash(state carmen.State, expectedHash common.Hash) error {
	stateHash, err := state.GetHash()
	if err != nil {
		return fmt.Errorf("failed to get state hash; %w", err)
	}
	if stateHash != cc.Hash(expectedHash) {
		return fmt.Errorf("state hash does not match (%x != %x)", stateHash, expectedHash)
	}
	return nil
}

type verificationObserver struct {
	log.Logger
}

func (o verificationObserver) StartVerification() {}

func (o verificationObserver) Progress(msg string) {
	o.Info(msg)
}

func (o verificationObserver) EndVerification(res error) {}
