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

package migration

import (
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/log"
)

// KvdbIDStore stores id
type KvdbIDStore struct {
	table kvdb.Store
	key   []byte
}

// NewKvdbIDStore constructor
func NewKvdbIDStore(table kvdb.Store) *KvdbIDStore {
	return &KvdbIDStore{
		table: table,
		key:   []byte("id"),
	}
}

// GetID is a getter
func (p *KvdbIDStore) GetID() string {
	id, err := p.table.Get(p.key)
	if err != nil {
		log.Crit("Failed to get key-value", "err", err)
	}

	if id == nil {
		return ""
	}
	return string(id)
}

// SetID is a setter
func (p *KvdbIDStore) SetID(id string) {
	err := p.table.Put(p.key, []byte(id))
	if err != nil {
		log.Crit("Failed to put key-value", "err", err)
	}
}
