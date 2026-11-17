// Copyright 2026 Sonic Operations Ltd
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

package freeze

import (
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/freeze/registry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestIsFrozenTokenLog_DetectsErc20EventsNamingFrozenAccounts(t *testing.T) {
	db := newFakeState(frozenAccount)
	db.slots[registry.GetOptedOutSlot(optedOutToken)] = common.Hash{31: 1}
	topic := func(a common.Address) common.Hash { return common.BytesToHash(a.Bytes()) }

	tests := map[string]struct {
		log  types.Log
		want bool
	}{
		"transfer from frozen": {
			types.Log{Address: token, Topics: []common.Hash{transferTopic, topic(frozenAccount), topic(otherAccount)}},
			true,
		},
		"approval by frozen": {
			types.Log{Address: token, Topics: []common.Hash{approvalTopic, topic(frozenAccount), topic(otherAccount)}},
			true,
		},
		"transfer to frozen": {
			types.Log{Address: token, Topics: []common.Hash{transferTopic, topic(otherAccount), topic(frozenAccount)}},
			true,
		},
		"approval of frozen spender": {
			types.Log{Address: token, Topics: []common.Hash{approvalTopic, topic(otherAccount), topic(frozenAccount)}},
			true,
		},
		"transfer between others": {
			types.Log{Address: token, Topics: []common.Hash{transferTopic, topic(otherAccount), topic(otherAccount)}},
			false,
		},
		"erc-721 transfer": {
			types.Log{Address: token, Topics: []common.Hash{transferTopic, topic(frozenAccount), topic(otherAccount), {}}},
			false,
		},
		"opted-out token": {
			types.Log{Address: optedOutToken, Topics: []common.Hash{transferTopic, topic(frozenAccount), topic(otherAccount)}},
			false,
		},
		"other event": {
			types.Log{Address: token, Topics: []common.Hash{{0x1}, topic(frozenAccount), topic(otherAccount)}},
			false,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.want, isFrozenTokenLog(db, &test.log))
		})
	}
}

func TestMovedSince_TracksFrozenLogsPerSnapshot(t *testing.T) {
	db := Wrap(newFakeState(frozenAccount))
	topic := func(a common.Address) common.Hash { return common.BytesToHash(a.Bytes()) }
	frozenTransfer := &types.Log{Address: token, Topics: []common.Hash{transferTopic, topic(frozenAccount), topic(otherAccount)}}
	otherTransfer := &types.Log{Address: token, Topics: []common.Hash{transferTopic, topic(otherAccount), topic(otherAccount)}}

	tokenCall := db.Snapshot()
	db.AddLog(otherTransfer)
	require.False(t, db.movedSince(tokenCall), "transfer between others")

	db.AddLog(frozenTransfer)
	laterCall := db.Snapshot()
	require.False(t, db.movedSince(laterCall), "call started after the log")
	require.True(t, db.movedSince(tokenCall))

	db.RevertToSnapshot(tokenCall)
	require.False(t, db.movedSince(tokenCall), "log reverted")

	db.AddLog(frozenTransfer)
	db.Prepare(params.Rules{}, otherAccount, common.Address{}, nil, nil, nil)
	require.False(t, db.movedSince(db.Snapshot()), "next transaction")
}
