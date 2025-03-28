package topicsdb

import (
	"context"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// dummyIndex is empty implementation of Index
type dummyIndex struct{}

func (n dummyIndex) FindInBlocks(ctx context.Context, from, to consensus.BlockID, pattern [][]common.Hash) (logs []*types.Log, err error) {
	return nil, ErrLogsNotRecorded
}

func (n dummyIndex) Push(recs ...*types.Log) error {
	return nil
}

func (n dummyIndex) Close() {}

func (n dummyIndex) WrapTablesAsBatched() (unwrap func()) {
	return func() {}
}
