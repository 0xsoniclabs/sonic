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

package verwatcher

import (
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
)

const (
	nvKey = "v"
	mvKey = "m"
)

// SetNetworkVersion stores network version.
func (s *Store) SetNetworkVersion(v uint64) {
	s.cache.networkVersion.Store(v)
	err := s.mainDB.Put([]byte(nvKey), bigendian.Uint64ToBytes(v))
	if err != nil {
		s.Log.Crit("Failed to put key", "err", err)
	}
}

// GetNetworkVersion returns stored network version.
func (s *Store) GetNetworkVersion() uint64 {
	if v := s.cache.networkVersion.Load(); v != nil {
		return v.(uint64)
	}
	valBytes, err := s.mainDB.Get([]byte(nvKey))
	if err != nil {
		s.Log.Crit("Failed to get key", "err", err)
	}
	v := uint64(0)
	if valBytes != nil {
		v = bigendian.BytesToUint64(valBytes)
	}
	s.cache.networkVersion.Store(v)
	return v
}

// SetMissedVersion stores non-supported network upgrade.
func (s *Store) SetMissedVersion(v uint64) {
	s.cache.missedVersion.Store(v)
	err := s.mainDB.Put([]byte(mvKey), bigendian.Uint64ToBytes(v))
	if err != nil {
		s.Log.Crit("Failed to put key", "err", err)
	}
}

// GetMissedVersion returns stored non-supported network upgrade.
func (s *Store) GetMissedVersion() uint64 {
	if v := s.cache.missedVersion.Load(); v != nil {
		return v.(uint64)
	}
	valBytes, err := s.mainDB.Get([]byte(mvKey))
	if err != nil {
		s.Log.Crit("Failed to get key", "err", err)
	}
	v := uint64(0)
	if valBytes != nil {
		v = bigendian.BytesToUint64(valBytes)
	}
	s.cache.missedVersion.Store(v)
	return v
}
