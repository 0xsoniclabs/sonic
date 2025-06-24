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
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type (
	logrec struct {
		ID          ID
		topicsCount uint8
		result      *types.Log
		err         error

		matched int
	}
)

func newLogrec(rec ID, topicCount uint8) *logrec {
	return &logrec{
		ID:          rec,
		topicsCount: topicCount,
	}
}

// fetch record's data.
func (rec *logrec) fetch(
	logrecTable kvdb.Reader,
) {
	r := &types.Log{
		BlockNumber: rec.ID.BlockNumber(),
		TxHash:      rec.ID.TxHash(),
		Index:       rec.ID.Index(),
		Topics:      make([]common.Hash, rec.topicsCount),
	}

	var (
		buf    []byte
		offset int
	)
	buf, rec.err = logrecTable.Get(rec.ID.Bytes())
	if rec.err != nil {
		return
	}

	// topics
	for i := 0; i < len(r.Topics); i++ {
		r.Topics[i] = common.BytesToHash(buf[offset : offset+common.HashLength])
		offset += common.HashLength
	}

	// fields
	r.BlockHash = common.BytesToHash(buf[offset : offset+common.HashLength])
	offset += common.HashLength
	r.Address = common.BytesToAddress(buf[offset : offset+common.AddressLength])
	offset += common.AddressLength
	r.Data = buf[offset:]

	rec.result = r
}
