package tests

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/transientstorage"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

func TestTransientStorage_TransientStorageIsValidInTransaction(t *testing.T) {
	net := StartIntegrationTestNet(t)

	// Deploy the transient storage contract
	contract, _, err := DeployContract(net, transientstorage.DeployTransientstorage)
	if err != nil {
		t.Fatalf("failed to deploy contract; %v", err)
	}

	// Get the value from the contract before changing it
	valueBefore, err := contract.GetValue(nil)
	if err != nil {
		t.Fatalf("failed to get value; %v", err)
	}

	// Store the value in transient storage value
	receipt, err := net.Apply(contract.StoreValue)
	if err != nil {
		t.Fatalf("failed to store value; %v", err)
	}

	// Check that the value was stored during transaction and emitted to logs
	if len(receipt.Logs) != 1 {
		t.Fatalf("unexpected number of logs; expected 1, got %d", len(receipt.Logs))
	}

	// Get the value from the log
	logValue, err := contract.ParseStoredValue(*receipt.Logs[0])
	if err != nil {
		t.Fatalf("failed to parse log; %v", err)
	}
	fromLog := logValue.Value

	// Get the value from the archive at time of store transaction
	fromArchive, err := contract.GetValue(&bind.CallOpts{BlockNumber: receipt.BlockNumber})
	if err != nil {
		t.Fatalf("failed to get transient value from archive; %v", err)
	}

	// Get the value from the archive from head
	fromArchiveHead, err := contract.GetValue(nil)
	if err != nil {
		t.Fatalf("failed to get transient value from archive at head time; %v", err)
	}

	// Check that all non log values are zero
	if valueBefore.Sign() != 0 || fromArchive.Sign() != 0 || fromArchiveHead.Sign() != 0 {
		t.Fatalf("unexpected value; expected 0, got valueBefore %v, fromArchive %v, FromArchiveHead %v", valueBefore, fromArchive, fromArchiveHead)
	}

	// Check that the log value is the same as set in contract
	if fromLog.Cmp(big.NewInt(42)) != 0 {
		t.Fatalf("unexpected log value; expected non-zero, got %v", fromLog)
	}
}
