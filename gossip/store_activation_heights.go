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

package gossip

import (
	"github.com/0xsoniclabs/sonic/opera"
)

func (s *Store) AddUpgradeHeight(h opera.UpgradeHeight) {
	orig := s.GetUpgradeHeights()
	// allocate new memory to avoid race condition in cache
	cp := make([]opera.UpgradeHeight, 0, len(orig)+1)
	cp = append(append(cp, orig...), h)

	s.rlp.Set(s.table.UpgradeHeights, []byte{}, cp)
	s.cache.UpgradeHeights.Store(cp)
}

func (s *Store) GetUpgradeHeights() []opera.UpgradeHeight {
	if v := s.cache.UpgradeHeights.Load(); v != nil {
		return v.([]opera.UpgradeHeight)
	}
	hh, ok := s.rlp.Get(s.table.UpgradeHeights, []byte{}, &[]opera.UpgradeHeight{}).(*[]opera.UpgradeHeight)
	if !ok {
		return []opera.UpgradeHeight{}
	}
	s.cache.UpgradeHeights.Store(*hh)
	return *hh
}
