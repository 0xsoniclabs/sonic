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

package core_types

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestCoreLogFromGethLog_NilRemainsNil(t *testing.T) {
	require.Nil(t, CoreLogFromGethLog(nil))
}

func TestCoreLogFromGethLog_CoversAddress(t *testing.T) {
	addresses := []common.Address{
		{},
		{1, 2, 3},
	}

	for _, address := range addresses {
		gethLog := &types.Log{
			Address: address,
		}
		coreLog := CoreLogFromGethLog(gethLog)
		require.Equal(t, address, coreLog.Address)
	}
}

func TestCoreLogFromGethLog_CoversTopics(t *testing.T) {
	topics := [][]common.Hash{
		{},
		{{1, 2, 3}},
		{{4, 5}, {6, 7, 8}},
	}

	for _, topicList := range topics {
		gethLog := &types.Log{
			Topics: topicList,
		}
		coreLog := CoreLogFromGethLog(gethLog)
		require.Equal(t, topicList, coreLog.Topics)
	}
}

func TestCoreLogFromGethLog_CoversData(t *testing.T) {
	datas := [][]byte{
		{},
		{0x00},
		{0x01, 0x02, 0x03},
	}

	for _, data := range datas {
		gethLog := &types.Log{
			Data: data,
		}
		coreLog := CoreLogFromGethLog(gethLog)
		require.Equal(t, data, coreLog.Data)
	}
}
