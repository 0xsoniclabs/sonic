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

package dagstreamleecher

import (
	"time"

	"github.com/Fantom-foundation/lachesis-base/gossip/basestream/basestreamleecher/basepeerleecher"
)

type Config struct {
	Session              basepeerleecher.EpochDownloaderConfig
	RecheckInterval      time.Duration
	BaseProgressWatchdog time.Duration
	BaseSessionWatchdog  time.Duration
	MinSessionRestart    time.Duration
	MaxSessionRestart    time.Duration
}

// DefaultConfig returns default leecher config
func DefaultConfig() Config {
	return Config{
		Session: basepeerleecher.EpochDownloaderConfig{
			DefaultChunkItemsNum:   500,
			DefaultChunkItemsSize:  512 * 1024,
			ParallelChunksDownload: 6,
			RecheckInterval:        10 * time.Millisecond,
		},
		RecheckInterval:      time.Second,
		BaseProgressWatchdog: time.Second * 5,
		BaseSessionWatchdog:  time.Second * 30 * 5,
		MinSessionRestart:    time.Second * 5,
		MaxSessionRestart:    time.Minute * 5,
	}
}

// LiteConfig returns default leecher config for tests
func LiteConfig() Config {
	cfg := DefaultConfig()
	cfg.Session.DefaultChunkItemsSize /= 10
	cfg.Session.DefaultChunkItemsNum /= 10
	cfg.Session.ParallelChunksDownload = cfg.Session.ParallelChunksDownload/2 + 1
	return cfg
}
