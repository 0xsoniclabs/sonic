package substate

import (
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/0xsoniclabs/substate/db"
	"github.com/0xsoniclabs/substate/substate"
)

var (
	staticSubstateDB db.SubstateDB
)

func NewSubstateDB(path, encoding string) error {
	var err error
	staticSubstateDB, err = db.NewSubstateDB(path, &opt.Options{ReadOnly: false}, nil, nil)
	if err != nil {
		return err
	}
	staticSubstateDB, err = staticSubstateDB.SetSubstateEncoding(encoding)
	return err
}

func CloseSubstateDB() error {
	return staticSubstateDB.Close()
}

func PutSubstate(ss *substate.Substate) error {
	return staticSubstateDB.PutSubstate(ss)
}
