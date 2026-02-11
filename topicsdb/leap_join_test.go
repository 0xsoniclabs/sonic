// Copyright 2025 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package topicsdb

import (
	"encoding/binary"
	"errors"
	"testing"

	"github.com/0xsoniclabs/sonic/utils/leap"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

//go:generate mockgen -source=leap_join_test.go -destination=leap_join_test_mock.go -package=topicsdb

var _ leap.Iterator[logrec] = (*indexIterator)(nil)

func TestIndexIterator_Default_IsEmpty(t *testing.T) {
	it := &indexIterator{}
	// is empty
	require.False(t, it.Next())
	// remains empty
	require.False(t, it.Next())
	// current does not panic
	require.Equal(t, logrec{}, it.Current())
}

func TestIndexIterator_Default_CanBeReleased(t *testing.T) {
	it := &indexIterator{}
	it.Release()
}

func TestIndexIterator_newIndexIterator_DoesNotCreateUnderlyingIterator(t *testing.T) {
	iter := newIndexIterator(nil, common.Hash{}, 0, 0, 0)
	require.NotNil(t, iter)
	require.Nil(t, iter.iter)
}

func TestIndexIterator_Next_CreatesUnderlyingIteratorLazily(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	table := NewMock_table(ctrl)
	iter := NewMock_iterator(ctrl)

	topic := common.Hash{1, 2, 3}
	pos := 5
	from := idx.Block(10)
	to := idx.Block(20)

	it := newIndexIterator(table, topic, pos, from, to)
	require.NotNil(it)
	require.Nil(it.iter)

	// Expect the iterator to be created with the correct prefix and start key
	// when calling Next for the first time.
	prefix := append(topic[:], byte(pos))
	start := binary.BigEndian.AppendUint64(nil, uint64(from))
	table.EXPECT().NewIterator(prefix, start).Return(iter)

	id := NewID(uint64(from)+1, common.Hash{}, 0)
	key := topicKey(topic, uint8(pos), id)
	gomock.InOrder(
		iter.EXPECT().Next().Return(true),
		iter.EXPECT().Key().Return(key),
		iter.EXPECT().Value().Return([]byte{5}),
		iter.EXPECT().Error().Return(nil),
	)

	require.True(it.Next())
	require.Equal(iter, it.iter)
	require.True(ctrl.Satisfied())

	// Subsequent calls to Next should not create a new iterator.
	gomock.InOrder(
		iter.EXPECT().Next().Return(true),
		iter.EXPECT().Key().Return(key),
		iter.EXPECT().Value().Return([]byte{5}),
		iter.EXPECT().Error().Return(nil),
	)
	require.True(it.Next())
}

func TestIndexIterator_Next_AbortsIfIteratorCreationFails(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	table := NewMock_table(ctrl)

	topic := common.Hash{1, 2, 3}
	pos := 5
	from := idx.Block(10)
	to := idx.Block(20)

	it := newIndexIterator(table, topic, pos, from, to)
	require.NotNil(it)
	require.Nil(it.iter)

	// Let the iterator creation fail by returning nil.
	table.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(nil)

	require.False(it.Next())
	require.ErrorContains(it.Error(), "failed to create iterator")
	require.Nil(it.iter)
	require.True(ctrl.Satisfied())

	// Subsequent calls to Next should not create a new iterator.
	require.False(it.Next())
}

func TestIndexIterator_Next_AbortsIfIteratorHasError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	iter := NewMock_iterator(ctrl)

	topic := common.Hash{1, 2, 3}
	from := idx.Block(10)
	pos := 5
	id := NewID(uint64(from)+1, common.Hash{}, 0)
	key := topicKey(topic, uint8(pos), id)

	it := &indexIterator{
		table: NewMock_table(ctrl),
		iter:  iter,
	}

	issue := errors.New("injected issue")
	iter.EXPECT().Next().Return(true)
	iter.EXPECT().Key().Return(key)
	iter.EXPECT().Value().Return([]byte{5})
	iter.EXPECT().Error().Return(issue)

	require.False(it.Next())
	require.ErrorIs(it.Error(), issue)
	require.Equal(iter, it.iter)
	require.True(ctrl.Satisfied())

	// Subsequent calls to Next should not produce new elements.
	require.False(it.Next())
	require.ErrorIs(it.Error(), issue)
}

func TestIndexIterator_Next_SkipsKeysWithInvalidLength(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	iter := NewMock_iterator(ctrl)

	it := &indexIterator{
		table: NewMock_table(ctrl),
		iter:  iter,
	}

	// Simulate an iterator that returns a key with invalid length.
	gomock.InOrder(
		iter.EXPECT().Next().Return(true),
		iter.EXPECT().Key().Return([]byte{1, 2, 3}), // too short => skipped
		// The iterator is exhausted after skipping the invalid key.
		iter.EXPECT().Next().Return(false),
		iter.EXPECT().Error().Return(nil),
		iter.EXPECT().Release(),
	)

	require.False(it.Next())
}

func TestIndexIterator_Next_AbortsWithValueLengthMismatch(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	iter := NewMock_iterator(ctrl)

	it := &indexIterator{
		table: NewMock_table(ctrl),
		iter:  iter,
	}

	// Simulate an iterator that returns a key with invalid length.
	gomock.InOrder(
		iter.EXPECT().Next().Return(true),
		iter.EXPECT().Key().Return(make([]byte, topicKeySize)),
		iter.EXPECT().Value().Return(make([]byte, 2)), // invalid length => error
	)

	require.False(it.Next())
	require.ErrorContains(it.Error(), "invalid value length")
}

func TestIndexIterator_Next_IteratorDoesNotRestartAfterExhaustion(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	iter := NewMock_iterator(ctrl)

	it := &indexIterator{
		table: NewMock_table(ctrl),
		iter:  iter,
	}

	gomock.InOrder(
		// Simulate iterator exhaustion.
		iter.EXPECT().Next().Return(false),
		// The iterator is also released since there are no more elements.
		iter.EXPECT().Error(),
		iter.EXPECT().Release(),
	)

	require.False(it.Next())
	require.True(ctrl.Satisfied())

	// Subsequent calls to Next should not create a new iterator.
	require.False(it.Next())
}

func TestIndexIterator_Next_AbortsIfBlockRangeIsExceeded(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	iter := NewMock_iterator(ctrl)

	to := idx.Block(20)
	it := &indexIterator{
		table: NewMock_table(ctrl),
		iter:  iter,
		to:    to,
	}

	// Simulate a key with a block number greater than 'to'.
	id := NewID(uint64(to)+1, common.Hash{}, 0)
	key := topicKey(common.Hash{}, 0, id)
	value := []byte{5}
	gomock.InOrder(
		iter.EXPECT().Next().Return(true),
		iter.EXPECT().Key().Return(key),
		iter.EXPECT().Value().Return(value),
		iter.EXPECT().Error(),
		// The iterator is also released since there are no more elements.
		iter.EXPECT().Error(),
		iter.EXPECT().Release(),
	)

	require.False(it.Next())
}

func TestIndexIterator_Current_DefaultReturnsZeroValue(t *testing.T) {
	it := &indexIterator{}
	require.Zero(t, it.Current())
}

func TestIndexIterator_Current_ReturnsCachedLogRecord(t *testing.T) {
	require := require.New(t)

	block := uint64(10)
	txHash := common.Hash{1, 2, 3}
	logIndex := uint(5)
	id := NewID(block, txHash, logIndex)

	topicCount := uint8(3)
	logRec := *newLogrec(id, topicCount)

	it := &indexIterator{
		current: logRec,
	}

	require.Equal(logRec, it.Current())
}

func TestIndexIterator_Seek_DefaultOrExhaustedIteratorReturnsFalse(t *testing.T) {
	it := &indexIterator{}
	require.False(t, it.Seek(logrec{}))
}

func TestIndexIterator_Seek_FailsIfThereIsAnExistingIssue(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	issue := errors.New("injected issue")
	it := &indexIterator{
		table: NewMock_table(ctrl),
		err:   issue,
	}

	require.False(it.Seek(logrec{}))
}

func TestIndexIterator_Seek_ReleasesOldIteratorAndCreatesNewOne(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	table := NewMock_table(ctrl)
	oldIter := NewMock_iterator(ctrl)
	newIter := NewMock_iterator(ctrl)

	prefix := []byte{1, 2, 3}

	it := &indexIterator{
		table:  table,
		iter:   oldIter,
		prefix: prefix,
		from:   idx.Block(0),
		to:     idx.Block(100),
	}

	block := uint64(10)
	id := NewID(block, common.Hash{4, 5, 6}, 7)
	target := *newLogrec(id, 12)

	// Expect the old iterator to be released and a new iterator to be created
	// with the correct prefix and start key.
	gomock.InOrder(
		oldIter.EXPECT().Error().Return(nil),
		oldIter.EXPECT().Release(),
		table.EXPECT().NewIterator(prefix, target.ID.Bytes()).Return(newIter),
		newIter.EXPECT().Next().Return(true),
		newIter.EXPECT().Key().Return(topicKey(common.Hash{}, 0, id)),
		newIter.EXPECT().Value().Return([]byte{12}),
		newIter.EXPECT().Error().Return(nil),
	)

	require.True(it.Seek(target))
	require.Equal(newIter, it.iter)
}

func TestIndexIterator_Seek_FailsIfNewIteratorCreationFails(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	table := NewMock_table(ctrl)
	oldIter := NewMock_iterator(ctrl)

	it := &indexIterator{
		table: table,
		iter:  oldIter,
	}

	block := uint64(10)
	id := NewID(block, common.Hash{4, 5, 6}, 7)
	target := *newLogrec(id, 12)

	gomock.InOrder(
		oldIter.EXPECT().Error().Return(nil),
		oldIter.EXPECT().Release(),
		table.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(nil),
	)

	require.False(it.Seek(target))
	require.Nil(it.iter)
	require.ErrorContains(it.Error(), "failed to create iterator")
}

func TestIndexIterator_Error_DefaultHasNoError(t *testing.T) {
	it := &indexIterator{}
	require.NoError(t, it.Error())
}

func TestIndexIterator_Error_ReturnsCollectedError(t *testing.T) {
	require := require.New(t)
	issue := errors.New("injected issue")
	it := &indexIterator{
		err: issue,
	}
	require.ErrorIs(it.Error(), issue)
}

func TestIndexIterator_Release_DefaultDoesNotPanic(t *testing.T) {
	it := &indexIterator{}
	it.Release()
}

func TestIndexIterator_Release_ReleasesNestedIterator(t *testing.T) {
	ctrl := gomock.NewController(t)
	iter := NewMock_iterator(ctrl)

	gomock.InOrder(
		iter.EXPECT().Error().Return(nil),
		iter.EXPECT().Release(),
	)

	it := &indexIterator{
		iter: iter,
	}

	it.Release()
}

func TestIndexIterator_Release_PreservesIteratorError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	iter := NewMock_iterator(ctrl)

	issue := errors.New("injected issue")
	iter.EXPECT().Error().Return(issue).MinTimes(1)
	iter.EXPECT().Release()

	it := &indexIterator{
		iter: iter,
	}

	it.Release()
	require.ErrorIs(it.Error(), issue)
}

// used for mock generation

type _table interface {
	kvdb.Iteratee
}

var _ _table = (*Mock_table)(nil) // to avoid an unused warning for the _table interface

type _iterator interface {
	kvdb.Iterator
}

var _ _iterator = (*Mock_iterator)(nil) // to avoid an unused warning for the _iterator interface
