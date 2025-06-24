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
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
)

// Migration is a migration step.
type Migration struct {
	name string
	exec func() error
	prev *Migration
}

// Begin with empty unique migration step.
func Begin(appName string) *Migration {
	return &Migration{
		name: appName,
	}
}

// Next creates next migration.
func (m *Migration) Next(name string, exec func() error) *Migration {
	if name == "" {
		panic("empty name")
	}

	if exec == nil {
		panic("empty exec")
	}

	return &Migration{
		name: name,
		exec: exec,
		prev: m,
	}
}

func idOf(name string) string {
	digest := sha256.New()
	digest.Write([]byte(name))

	bytes := digest.Sum(nil)
	return fmt.Sprintf("%x", bytes)
}

// ID is an uniq migration's id.
func (m *Migration) ID() string {
	return idOf(m.name)
}

// Exec method run migrations chain in order
func (m *Migration) Exec(curr IDStore, flush func() error) error {
	currID := curr.GetID()
	myID := m.ID()

	if m.veryFirst() {
		if currID != myID {
			return errors.New("unknown version: " + currID)
		}
		return nil
	}

	if currID == myID {
		return nil
	}

	err := m.prev.Exec(curr, flush)
	if err != nil {
		return err
	}

	log.Warn("Applying migration", "name", m.name)
	err = m.exec()
	if err != nil {
		log.Error("'"+m.name+"' migration failed", "err", err)
		return err
	}

	curr.SetID(myID)

	return flush()
}

func (m *Migration) veryFirst() bool {
	return m.exec == nil
}
