package gossip

import (
	"fmt"

	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/sonic/opera"

	"github.com/0xsoniclabs/consensus/kvdb"
	"github.com/0xsoniclabs/sonic/utils/migration"
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
