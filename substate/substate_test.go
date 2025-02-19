package substate

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/substate/substate"
	"github.com/0xsoniclabs/substate/types"
)

// legacyTxType
var txType int32 = 0x00

var testSubstate = &substate.Substate{
	InputSubstate:  substate.NewWorldState(),
	OutputSubstate: substate.NewWorldState(),
	Env: &substate.Env{
		Coinbase:    types.Address{1},
		Difficulty:  new(big.Int).SetUint64(1),
		GasLimit:    1,
		Number:      1,
		Timestamp:   1,
		BlockHashes: make(map[uint64]types.Hash),
		BaseFee:     new(big.Int).SetUint64(1),
	},
	Message:     substate.NewMessage(1, true, new(big.Int).SetUint64(1), 1, types.Address{1}, new(types.Address), new(big.Int).SetUint64(1), []byte{1}, nil, &txType, types.AccessList{}, new(big.Int).SetUint64(1), new(big.Int).SetUint64(1), new(big.Int).SetUint64(1), []types.Hash{}),
	Result:      substate.NewResult(1, types.Bloom{}, []*types.Log{}, types.Address{0}, 1),
	Block:       37_534_834,
	Transaction: 1,
}

func TestSubstateDB_PutAndGetSubstate(t *testing.T) {
	dbPath := t.TempDir() + "test-db"
	err := NewSubstateDB(dbPath, "pb")
	defer staticSubstateDB.Close()

	h1 := types.Hash{}
	h1.SetBytes(nil)

	h2 := types.Hash{}
	h2.SetBytes(nil)

	testSubstate.InputSubstate[types.Address{1}] = substate.NewAccount(1, new(big.Int).SetUint64(1), h1[:])
	testSubstate.OutputSubstate[types.Address{2}] = substate.NewAccount(2, new(big.Int).SetUint64(2), h2[:])
	testSubstate.Env.BlockHashes[1] = types.BytesToHash([]byte{1})

	err = staticSubstateDB.PutSubstate(testSubstate)
	if err != nil {
		t.Fatalf("unable to put substate to staticSubstateDB: %v", err)
	}

	ss, err := staticSubstateDB.GetSubstate(37_534_834, 1)
	if err != nil {
		t.Fatalf("get substate returned error; %v", err)
	}

	if ss == nil {
		t.Fatal("substate is nil")
	}

	if err = ss.Equal(testSubstate); err != nil {
		t.Fatalf("substates are different; %v", err)
	}
}
