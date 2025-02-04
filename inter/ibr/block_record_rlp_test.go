package ibr_test

import (
	"bytes"
	"github.com/0xsoniclabs/sonic/inter/ibr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"testing"
)

// Verify that [] is equivalent to nil for log's Topics in the genesis file
func TestNilTopicsDoesNotMatterInGenesis(t *testing.T) {
	rlp1, err := rlp.EncodeToBytes(ibr.LlrIdxFullBlockRecord{
		LlrFullBlockRecord: ibr.LlrFullBlockRecord{
			Receipts: []*types.ReceiptForStorage{
				{
					Logs: []*types.Log{
						{
							Topics: nil,
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	rlp2, err := rlp.EncodeToBytes(ibr.LlrIdxFullBlockRecord{
		LlrFullBlockRecord: ibr.LlrFullBlockRecord{
			Receipts: []*types.ReceiptForStorage{
				{
					Logs: []*types.Log{
						{
							Topics: []common.Hash{},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rlp1, rlp2) {
		t.Errorf("serialized byte slices does not match: %x != %x", rlp1, rlp2)
	}
}
