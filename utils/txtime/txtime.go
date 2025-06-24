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

package txtime

import (
	"sync/atomic"
	"time"

	"github.com/Fantom-foundation/lachesis-base/utils/wlru"
	"github.com/ethereum/go-ethereum/common"
)

var (
	globalFinalized, _    = wlru.New(30000, 300000)
	globalNonFinalized, _ = wlru.New(5000, 50000)
	Enabled               = atomic.Bool{}
)

func Saw(txid common.Hash, t time.Time) {
	if !Enabled.Load() {
		return
	}
	globalNonFinalized.ContainsOrAdd(txid, t, 1)
}

func Validated(txid common.Hash, t time.Time) {
	if !Enabled.Load() {
		return
	}
	v, has := globalNonFinalized.Peek(txid)
	if has {
		t = v.(time.Time)
	}
	globalFinalized.ContainsOrAdd(txid, t, 1)
}

func Of(txid common.Hash) time.Time {
	if !Enabled.Load() {
		return time.Time{}
	}
	v, has := globalFinalized.Get(txid)
	if has {
		return v.(time.Time)
	}
	v, has = globalNonFinalized.Get(txid)
	if has {
		return v.(time.Time)
	}
	now := time.Now()
	Saw(txid, now)
	return now
}

func Get(txid common.Hash) time.Time {
	if !Enabled.Load() {
		return time.Time{}
	}
	v, has := globalFinalized.Get(txid)
	if has {
		return v.(time.Time)
	}
	v, has = globalNonFinalized.Get(txid)
	if has {
		return v.(time.Time)
	}
	return time.Time{}
}
