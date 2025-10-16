package substate

import (
	"math"

	"github.com/0xsoniclabs/substate/db"
	"github.com/0xsoniclabs/substate/substate"
)

// record-replay - global variables tracking number of transactions in a block
var (
	TxLastIndex    int
	OldBlockNumber uint64 = math.MaxUint64

	staticSubstateDB  db.SubstateDB
	staticExceptionDB db.ExceptionDB
	//staticStateHashDB db.StateHashDB
)

func NewSubstateDB(path string) error {
	var err error
	baseDB, err := db.NewDefaultBaseDB(path)
	if err != nil {
		return err
	}
	staticSubstateDB = db.MakeDefaultSubstateDBFromBaseDB(baseDB)
	staticExceptionDB = db.MakeDefaultExceptionDBFromBaseDB(baseDB)
	//staticStateHashDB = db.MakeDefaultStateHashDBFromBaseDB(baseDB)

	return nil
}

func CloseSubstateDB() error {
	err := WriteUnprocessedSkippedTxToDatabase()
	if err != nil {
		return err
	}

	//return errors.Join(staticSubstateDB.Close(), staticExceptionDB.Close(), staticStateHashDB.Close())
	return nil
}

func PutSubstate(ss *substate.Substate) error {
	return staticSubstateDB.PutSubstate(ss)
}

func PutStateHash(blockNumber uint64, stateHash []byte) error {
	//if staticStateHashDB == nil {
	//	return errors.New("state hash db is not initialized")
	//}
	//
	//if err := staticStateHashDB.PutStateHash(blockNumber, stateHash); err != nil {
	//	return errors.New(fmt.Sprintf("unable to put state hash for block %d: %v", blockNumber, err))
	//}
	return nil
}
