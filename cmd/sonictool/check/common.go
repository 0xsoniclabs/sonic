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

package check

import (
	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/integration"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/log"
)

func createGdb(dataDir string, cacheRatio cachescale.Func, archive carmen.ArchiveType, skipArchiveCheck bool) (*gossip.Store, kvdb.FullDBProducer, error) {
	return integration.OpenStore(dataDir, cacheRatio, func(cfg *gossip.StoreConfig) {
		cfg.EVM.StateDb.Archive = archive
		cfg.EVM.SkipArchiveCheck = skipArchiveCheck // skip archive mode check (allow "check live" to run with archive enabled)
	})
}

type verificationObserver struct{}

func (o verificationObserver) StartVerification() {}

func (o verificationObserver) Progress(msg string) {
	log.Info(msg)
}

func (o verificationObserver) EndVerification(res error) {}
