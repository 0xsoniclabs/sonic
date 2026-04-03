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

package itemsfetcher

import (
	"time"

	"github.com/0xsoniclabs/cacheutils/cachescale"
)

type Config struct {
	ForgetTimeout time.Duration // Time before an announced event is forgotten
	ArriveTimeout time.Duration // Time allowance before an announced event is explicitly requested
	GatherSlack   time.Duration // Interval used to collate almost-expired announces with fetches
	HashLimit     int           // Maximum number of unique events a peer may have announced

	MaxBatch int // Maximum number of hashes in an announce batch (batch is divided if exceeded)

	MaxParallelRequests int // Maximum number of parallel requests

	// MaxQueuedHashesBatches is the maximum number of announce batches to queue up before
	// dropping incoming hashes.
	MaxQueuedBatches int
}

func DefaultConfig(scale cachescale.Func) Config {
	return Config{
		ForgetTimeout:       1 * time.Minute,
		ArriveTimeout:       1000 * time.Millisecond,
		GatherSlack:         100 * time.Millisecond,
		HashLimit:           20000,
		MaxBatch:            scale.I(512),
		MaxQueuedBatches:    scale.I(32),
		MaxParallelRequests: 256,
	}
}
