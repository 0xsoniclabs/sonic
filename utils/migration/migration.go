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
