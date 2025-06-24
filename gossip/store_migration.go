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
	"fmt"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"

	"github.com/0xsoniclabs/sonic/utils/migration"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
)

func isEmptyDB(db kvdb.Iteratee) bool {
	it := db.NewIterator(nil, nil)
	defer it.Release()
	return !it.Next()
}

func (s *Store) migrateData() error {
	versions := migration.NewKvdbIDStore(s.table.Version)
	if isEmptyDB(s.table.Version) {
		// short circuit if empty DB
		versions.SetID(s.migrations().ID())
		return nil
	}

	err := s.migrations().Exec(versions, s.flushDBs)
	return err
}

func (s *Store) migrations() *migration.Migration {
	return migration.
		Begin("opera-gossip-store").
		Next("used gas recovery", unsupportedMigration).
		Next("tx hashes recovery", unsupportedMigration).
		Next("DAG heads recovery", unsupportedMigration).
		Next("DAG last events recovery", unsupportedMigration).
		Next("BlockState recovery", unsupportedMigration).
		Next("LlrState recovery", unsupportedMigration).
		Next("erase gossip-async db", unsupportedMigration).
		Next("erase SFC API table", unsupportedMigration).
		Next("erase legacy genesis DB", unsupportedMigration).
		Next("calculate upgrade heights", unsupportedMigration).
		Next("add time into upgrade heights", s.addTimeIntoUpgradeHeights)
}

func unsupportedMigration() error {
	return fmt.Errorf("DB version isn't supported, please restart from scratch")
}

type legacyUpgradeHeight struct {
	Upgrades opera.Upgrades
	Height   idx.Block
}

func (s *Store) addTimeIntoUpgradeHeights() error {
	oldHeights, ok := s.rlp.Get(s.table.UpgradeHeights, []byte{}, &[]legacyUpgradeHeight{}).(*[]legacyUpgradeHeight)
	if !ok {
		return fmt.Errorf("failed to decode old UpgradeHeights, please restart from scratch")
	}
	newHeights := make([]opera.UpgradeHeight, 0, len(*oldHeights))
	for _, height := range *oldHeights {
		block := s.GetBlock(height.Height - 1)
		if block == nil {
			return fmt.Errorf("failed to get block by UpgradeHeights, please restart from scratch")
		}
		newHeights = append(newHeights, opera.UpgradeHeight{
			Upgrades: height.Upgrades,
			Height:   height.Height,
			Time:     block.Time + 1,
		})
	}
	s.rlp.Set(s.table.UpgradeHeights, []byte{}, newHeights)
	return nil
}
