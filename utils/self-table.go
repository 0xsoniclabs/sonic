package utils

import (
	"github.com/0xsoniclabs/consensus/kvdb"
	"github.com/0xsoniclabs/consensus/kvdb/table"
)

func NewTableOrSelf(db kvdb.Store, prefix []byte) kvdb.Store {
	if len(prefix) == 0 {
		return db
	}
	return table.New(db, prefix)
}
