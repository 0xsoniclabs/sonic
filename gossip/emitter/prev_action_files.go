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

package emitter

import (
	"io"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/utils"
)

var openPrevActionFile = utils.OpenFile

func (em *Emitter) writeLastEmittedEventID(id hash.Event) {
	if em.emittedEventFile == nil {
		return
	}
	_, err := em.emittedEventFile.WriteAt(id.Bytes(), 0)
	if err != nil {
		log.Crit("Failed to write event file", "file", em.config.PrevEmittedEventFile.Path, "err", err)
	}
}

func (em *Emitter) readLastEmittedEventID() *hash.Event {
	if em.emittedEventFile == nil {
		return nil
	}
	buf := make([]byte, 32)
	_, err := em.emittedEventFile.ReadAt(buf, 0)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		log.Crit("Failed to read event file", "file", em.config.PrevEmittedEventFile.Path, "err", err)
	}
	v := hash.BytesToEvent(buf)
	return &v
}
