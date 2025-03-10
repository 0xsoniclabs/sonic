package utils

import (
	"github.com/0xsoniclabs/kvdb"
	"github.com/0xsoniclabs/kvdb/table"
)

func NewTableOrSelf(db kvdb.Store, prefix []byte) kvdb.Store {
	if len(prefix) == 0 {
		return db
	}
	return table.New(db, prefix)
}
