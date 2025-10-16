package substate

import (
	"errors"
	"fmt"
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
	staticStateHashDB db.StateHashDB
	staticBlockHashDB db.BlockHashDB
)

func NewSubstateDB(path string) error {
	var err error
	staticSubstateDB, err = db.NewDefaultSubstateDB(path)
	if err != nil {
		return err
	}
	staticExceptionDB = db.MakeDefaultExceptionDBFromBaseDB(staticSubstateDB)
	staticStateHashDB = db.MakeDefaultStateHashDBFromBaseDB(staticSubstateDB)
	staticBlockHashDB = db.MakeDefaultBlockHashDBFromBaseDB(staticSubstateDB)
	return nil
}

func CloseSubstateDB() error {
	err := WriteUnprocessedSkippedTxToDatabase()
	if err != nil {
		return err
	}

	return errors.Join(staticSubstateDB.Close(), staticExceptionDB.Close(), staticStateHashDB.Close(), staticBlockHashDB.Close())
}

func PutSubstate(ss *substate.Substate) error {
	return staticSubstateDB.PutSubstate(ss)
}

func PutStateHash(blockNumber uint64, stateHash []byte) error {
	if staticStateHashDB == nil {
		return errors.New("state hash db is not initialized")
	}

	if err := staticStateHashDB.PutStateHash(blockNumber, stateHash); err != nil {
		return errors.New(fmt.Sprintf("unable to put state hash for block %d: %v", blockNumber, err))
	}
	return nil
}

func PutBlockHash(blockNumber uint64, blockHash []byte) error {
	if staticBlockHashDB == nil {
		return errors.New("blockHash db is not initialized")
	}

	if err := staticBlockHashDB.PutBlockHash(blockNumber, blockHash); err != nil {
		return errors.New(fmt.Sprintf("unable to put blockHash for block %d: %v", blockNumber, err))
	}
	return nil
}
