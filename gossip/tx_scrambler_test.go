package gossip

import (
	"cmp"
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/emitter/mock"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
)

func TestTxScrambler_AnalyseEntryList_RemovesDuplicateTransactions(t *testing.T) {
	entries := []ScramblerEntry{
		&dummyScramblerEntry{hash: common.Hash{1}},
		&dummyScramblerEntry{hash: common.Hash{2}},
		&dummyScramblerEntry{hash: common.Hash{3}},
		&dummyScramblerEntry{hash: common.Hash{2}},
		&dummyScramblerEntry{hash: common.Hash{1}},
	}

	shuffleEntries(entries)
	result, _, _ := analyseEntryList(entries)
	if len(result) != 3 {
		t.Fatalf("unexpected length of result list, wanted 3, got %d", len(result))
	}

	seen := map[common.Hash]struct{}{}
	for _, entry := range result {
		if _, seen := seen[entry.Hash()]; seen {
			t.Fatalf("duplicate hash %v", entry.Hash())
		}
		seen[entry.Hash()] = struct{}{}
	}
}

func TestTxScrambler_UnifyEntries_SaltCreationIsDeterministic(t *testing.T) {
	entries := []ScramblerEntry{
		&dummyScramblerEntry{hash: common.Hash{1}},
		&dummyScramblerEntry{hash: common.Hash{2}},
		&dummyScramblerEntry{hash: common.Hash{3}},
		&dummyScramblerEntry{hash: common.Hash{2}},
		&dummyScramblerEntry{hash: common.Hash{1}},
	}

	_, wantedSalt, _ := analyseEntryList(entries)
	for range 10 {
		shuffleEntries(entries)
		_, gotSalt, _ := analyseEntryList(entries)
		if gotSalt != wantedSalt {
			t.Fatal("incorrect salt - salt creation is not deterministic")
		}
	}

}

func TestTxScrambler_AnalyseEntryList_ReportsDuplicateAddresses(t *testing.T) {
	tests := []struct {
		name         string
		input        []ScramblerEntry
		hasDuplicate bool
	}{
		{
			name: "has duplicate address",
			input: []ScramblerEntry{
				&dummyScramblerEntry{
					hash:   common.Hash{1},
					sender: common.Address{1},
				},
				&dummyScramblerEntry{
					hash:   common.Hash{2},
					sender: common.Address{3},
				},
				&dummyScramblerEntry{
					hash:   common.Hash{3},
					sender: common.Address{2},
				},
				&dummyScramblerEntry{
					hash:   common.Hash{4},
					sender: common.Address{3},
				},
			},
			hasDuplicate: true,
		},
		{
			name: "has no duplicate address",
			input: []ScramblerEntry{
				&dummyScramblerEntry{
					hash:   common.Hash{1},
					sender: common.Address{1},
				},
				&dummyScramblerEntry{
					hash:   common.Hash{2},
					sender: common.Address{2},
				},
				&dummyScramblerEntry{
					hash:   common.Hash{3},
					sender: common.Address{3},
				},
			},
			hasDuplicate: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, hasDuplicateAddresses := analyseEntryList(test.input)
			if hasDuplicateAddresses != test.hasDuplicate {
				t.Error("wrongly reported duplicate address")
			}
		})
	}

}

func TestTxScrambler_ScrambleTransactions_ScrambleIsDeterministic(t *testing.T) {
	res1 := []ScramblerEntry{
		&dummyScramblerEntry{hash: common.Hash{1}},
		&dummyScramblerEntry{hash: common.Hash{2}},
		&dummyScramblerEntry{hash: common.Hash{3}},
		&dummyScramblerEntry{hash: common.Hash{2}},
		&dummyScramblerEntry{hash: common.Hash{1}},
	}

	res2 := slices.Clone(res1)

	for i := 0; i < 10; i++ {
		salt := createRandomSalt()
		scrambleTransactions(res1, salt)
		for j := 0; j < 10; j++ {
			shuffleEntries(res2)
			scrambleTransactions(res2, salt)
			if slices.CompareFunc(res1, res2, compareFunc) != 0 {
				t.Error("scramble is not deterministic")
			}
		}
	}
}

func TestTxScrambler_SortTransactionsWithSameSender_SortsByNonce(t *testing.T) {
	entries := []ScramblerEntry{
		&dummyScramblerEntry{
			hash:   common.Hash{1},
			sender: common.Address{1},
			nonce:  1,
		},
		&dummyScramblerEntry{
			hash:   common.Hash{2},
			sender: common.Address{2},
			nonce:  1,
		},
		&dummyScramblerEntry{
			hash:   common.Hash{3},
			sender: common.Address{3},
			nonce:  1,
		},
		&dummyScramblerEntry{
			hash:   common.Hash{4},
			sender: common.Address{2},
			nonce:  2,
		},
		&dummyScramblerEntry{
			hash:   common.Hash{5},
			sender: common.Address{1},
			nonce:  2,
		},
	}

	shuffleEntries(entries)
	sortTransactionsWithSameSender(entries)
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].Sender() == entries[j].Sender() {
				if entries[i].Nonce() > entries[j].Nonce() {
					t.Errorf("incorrect nonce order %d must be before %d", entries[j].Nonce(), entries[i].Nonce())
				}
			}
		}
	}
}

func TestTxScrambler_SortTransactionsWithSameSender_SortsByGasIfNonceIsSame(t *testing.T) {
	entries := []ScramblerEntry{
		&dummyScramblerEntry{
			hash:     common.Hash{1},
			sender:   common.Address{1},
			nonce:    1,
			gasPrice: big.NewInt(1),
		},
		&dummyScramblerEntry{
			hash:     common.Hash{2},
			sender:   common.Address{1},
			nonce:    1,
			gasPrice: big.NewInt(2),
		},
		&dummyScramblerEntry{
			hash:     common.Hash{3},
			sender:   common.Address{2},
			nonce:    1,
			gasPrice: big.NewInt(3),
		},
		&dummyScramblerEntry{
			hash:     common.Hash{4},
			sender:   common.Address{2},
			nonce:    1,
			gasPrice: big.NewInt(4),
		},
	}

	shuffleEntries(entries)
	sortTransactionsWithSameSender(entries)
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].Sender() == entries[j].Sender() {
				if entries[i].Nonce() == entries[j].Nonce() && entries[i].GasPrice().Uint64() < entries[j].GasPrice().Uint64() {
					t.Errorf("incorrect gas price order %d must be before %d", entries[i].GasPrice(), entries[j].GasPrice())
				}
			}
		}
	}
}

func TestTxScrambler_SortTransactionsWithSameSender_SortsByHashIfNonceAndGasIsSame(t *testing.T) {
	entries := []ScramblerEntry{
		&dummyScramblerEntry{
			hash:     common.Hash{0},
			sender:   common.Address{1},
			nonce:    1,
			gasPrice: big.NewInt(1),
		},
		&dummyScramblerEntry{
			hash:     common.Hash{1},
			sender:   common.Address{1},
			nonce:    1,
			gasPrice: big.NewInt(1),
		},
		&dummyScramblerEntry{
			hash:     common.Hash{2},
			sender:   common.Address{1},
			nonce:    1,
			gasPrice: big.NewInt(1),
		},
		&dummyScramblerEntry{
			hash:     common.Hash{3},
			sender:   common.Address{1},
			nonce:    1,
			gasPrice: big.NewInt(1),
		},
	}

	shuffleEntries(entries)
	sortTransactionsWithSameSender(entries)
	// addrs, nonces and gas prices is same for every entry
	// we expect that entries are sorted by hash ascending
	for i := 0; i < len(entries); i++ {
		if got, want := entries[i].Hash(), (common.Hash{byte(i)}); got != want {
			t.Fatalf("wrong order, got: %s, want: %s", got, want)
		}
	}
}

func TestTxScrambler_FilterAndOrderTransactions_SortIsDeterministic_IdenticalData(t *testing.T) {
	tests := []struct {
		name    string
		entries []ScramblerEntry
	}{
		{
			name: "identical hashes",
			entries: []ScramblerEntry{
				&dummyScramblerEntry{
					hash:   common.Hash{1},
					sender: common.Address{1},
					nonce:  1,
				},
				&dummyScramblerEntry{
					hash:   common.Hash{1},
					sender: common.Address{1},
					nonce:  1,
				},
				&dummyScramblerEntry{
					hash:   common.Hash{1},
					sender: common.Address{1},
					nonce:  1,
				},
			},
		},
		{
			name: "identical addresses",
			entries: []ScramblerEntry{
				&dummyScramblerEntry{
					hash:   common.Hash{1},
					sender: common.Address{1},
					nonce:  1,
				},
				&dummyScramblerEntry{
					hash:   common.Hash{2},
					sender: common.Address{1},
					nonce:  2,
				},
				&dummyScramblerEntry{
					hash:   common.Hash{3},
					sender: common.Address{1},
					nonce:  3,
				},
			},
		},
		{
			name: "identical addresses and nonces",
			entries: []ScramblerEntry{
				&dummyScramblerEntry{
					hash:     common.Hash{1},
					sender:   common.Address{1},
					nonce:    1,
					gasPrice: big.NewInt(1),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{2},
					sender:   common.Address{1},
					nonce:    1,
					gasPrice: big.NewInt(2),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{3},
					sender:   common.Address{1},
					nonce:    1,
					gasPrice: big.NewInt(3),
				},
			},
		},
		{
			name: "identical addresses, nonces and gas prices",
			entries: []ScramblerEntry{
				&dummyScramblerEntry{
					hash:     common.Hash{1},
					sender:   common.Address{1},
					nonce:    1,
					gasPrice: big.NewInt(1),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{2},
					sender:   common.Address{1},
					nonce:    1,
					gasPrice: big.NewInt(1),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{3},
					sender:   common.Address{1},
					nonce:    1,
					gasPrice: big.NewInt(1),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res1 := test.entries
			res2 := slices.Clone(res1)
			// shuffle one array
			shuffleEntries(res2)

			res1 = filterAndOrderTransactions(res1)
			res2 = filterAndOrderTransactions(res2)
			if slices.CompareFunc(res1, res2, compareFunc) != 0 {
				t.Error("slices have different order - algorithm is not deterministic")
			}
		})
	}
}

func TestTxScrambler_FilterAndOrderTransactions_SortIsDeterministic_RepeatedData(t *testing.T) {
	tests := []struct {
		name    string
		entries []ScramblerEntry
	}{
		{
			name: "repeated hashes",
			entries: []ScramblerEntry{
				&dummyScramblerEntry{
					hash:     common.Hash{1},
					sender:   common.Address{1},
					nonce:    1,
					gasPrice: big.NewInt(1),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{2},
					sender:   common.Address{2},
					nonce:    2,
					gasPrice: big.NewInt(2),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{3},
					sender:   common.Address{3},
					nonce:    3,
					gasPrice: big.NewInt(3),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{2},
					sender:   common.Address{2},
					nonce:    2,
					gasPrice: big.NewInt(2),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{1},
					sender:   common.Address{1},
					nonce:    1,
					gasPrice: big.NewInt(1),
				},
			},
		},
		{
			name: "repeated addresses",
			entries: []ScramblerEntry{
				&dummyScramblerEntry{
					hash:   common.Hash{1},
					sender: common.Address{1},
					nonce:  1,
				},
				&dummyScramblerEntry{
					hash:   common.Hash{2},
					sender: common.Address{2},
					nonce:  2,
				},
				&dummyScramblerEntry{
					hash:   common.Hash{3},
					sender: common.Address{3},
					nonce:  3,
				},
				&dummyScramblerEntry{
					hash:   common.Hash{4},
					sender: common.Address{2},
					nonce:  4,
				},
				&dummyScramblerEntry{
					hash:   common.Hash{5},
					sender: common.Address{1},
					nonce:  5,
				},
			},
		},
		{
			name: "repeated addresses and nonces",
			entries: []ScramblerEntry{
				&dummyScramblerEntry{
					hash:     common.Hash{1},
					sender:   common.Address{1},
					nonce:    1,
					gasPrice: big.NewInt(1),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{2},
					sender:   common.Address{2},
					nonce:    2,
					gasPrice: big.NewInt(2),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{3},
					sender:   common.Address{3},
					nonce:    3,
					gasPrice: big.NewInt(3),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{4},
					sender:   common.Address{2},
					nonce:    2,
					gasPrice: big.NewInt(4),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{5},
					sender:   common.Address{1},
					nonce:    1,
					gasPrice: big.NewInt(5),
				},
			},
		},
		{
			name: "repeated addresses, nonces and gas prices",
			entries: []ScramblerEntry{
				&dummyScramblerEntry{
					hash:     common.Hash{1},
					sender:   common.Address{1},
					nonce:    1,
					gasPrice: big.NewInt(1),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{2},
					sender:   common.Address{2},
					nonce:    2,
					gasPrice: big.NewInt(2),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{3},
					sender:   common.Address{3},
					nonce:    3,
					gasPrice: big.NewInt(3),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{4},
					sender:   common.Address{2},
					nonce:    2,
					gasPrice: big.NewInt(2),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{5},
					sender:   common.Address{1},
					nonce:    1,
					gasPrice: big.NewInt(1),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res1 := test.entries
			res2 := slices.Clone(res1)
			// shuffle one array
			shuffleEntries(res2)

			res1 = filterAndOrderTransactions(res1)
			res2 = filterAndOrderTransactions(res2)
			if slices.CompareFunc(res1, res2, compareFunc) != 0 {
				t.Error("slices have different order - algorithm is not deterministic")
			}
		})
	}
}

func TestTxScrambler_FilterAndOrderTransactions_SortRemovesDuplicateHashes(t *testing.T) {
	entries := []ScramblerEntry{
		&dummyScramblerEntry{hash: common.Hash{1}},
		&dummyScramblerEntry{hash: common.Hash{2}},
		&dummyScramblerEntry{hash: common.Hash{3}},
		&dummyScramblerEntry{hash: common.Hash{2}},
		&dummyScramblerEntry{hash: common.Hash{1}},
	}
	shuffleEntries(entries)
	entries = filterAndOrderTransactions(entries)

	checkDuplicateHashes(t, entries)
}

func TestTxScrambler_FilterAndOrderTransactions_SortsSameSenderByNonceAndGas(t *testing.T) {
	entries := []ScramblerEntry{
		&dummyScramblerEntry{
			hash:   common.Hash{1},
			sender: common.Address{1},
			nonce:  1,
		},
		&dummyScramblerEntry{
			hash:   common.Hash{2},
			sender: common.Address{1},
			nonce:  2,
		},
		&dummyScramblerEntry{
			hash:     common.Hash{3},
			sender:   common.Address{1},
			nonce:    3,
			gasPrice: big.NewInt(1),
		},
		&dummyScramblerEntry{
			hash:     common.Hash{4},
			sender:   common.Address{1},
			nonce:    3,
			gasPrice: big.NewInt(2),
		},
		&dummyScramblerEntry{
			hash:   common.Hash{5},
			sender: common.Address{1},
			nonce:  4,
		},
	}
	shuffleEntries(entries)
	entries = filterAndOrderTransactions(entries)

	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].Sender() == entries[j].Sender() {
				if entries[i].Nonce() > entries[j].Nonce() {
					t.Errorf("incorrect nonce order %d must be before %d", entries[j].Nonce(), entries[i].Nonce())
				}
				if entries[i].Nonce() == entries[j].Nonce() && entries[i].GasPrice().Uint64() < entries[j].GasPrice().Uint64() {
					t.Errorf("incorrect gas price order %d must be before %d", entries[j].GasPrice(), entries[i].GasPrice())
				}
			}
		}
	}
}

func TestTxScrambler_FilterAndOrderTransactions_RandomInput(t *testing.T) {
	// this tests these input sizes:
	// 1, 4, 16, 64, 256, 1024
	for i := int64(1); i <= 1024; i = i * 4 {
		input := createRandomScramblerTestInput(i)
		cpy := slices.Clone(input)
		shuffleEntries(cpy)
		input = filterAndOrderTransactions(input)
		cpy = filterAndOrderTransactions(cpy)
		if slices.CompareFunc(input, cpy, compareFunc) != 0 {
			t.Error("slices have different order - algorithm is not deterministic")
		}
	}
}

func TestGetExecutionOrder_ScramblerIsUsedOnlyForSonic(t *testing.T) {
	tests := []struct {
		name        string
		isSonic     bool
		expectedLen int
	}{
		{
			name:        "SonicIsDisabled-InputStaysUnchanged",
			isSonic:     false,
			expectedLen: 4,
		},
		{
			name:        "SonicIsEnabled-InputIsChanged",
			isSonic:     true,
			expectedLen: 3,
		},
	}
	input := types.Transactions{
		types.NewTx(&types.LegacyTx{
			Nonce: 0,
			Gas:   0,
		}),
		types.NewTx(&types.LegacyTx{
			Nonce: 1,
			Gas:   0,
		}),
		types.NewTx(&types.LegacyTx{
			Nonce: 2,
			Gas:   0,
		}),
		types.NewTx(&types.LegacyTx{
			Nonce: 3,
			Gas:   0,
		}),
	}

	ctrl := gomock.NewController(t)
	signer := mock.NewMockTxSigner(ctrl)

	// Only one loop is expected because the scrambler is disabled if Sonic is not enabled.
	gomock.InOrder(
		signer.EXPECT().Sender(input[0]).Return(common.Address{1}, nil),
		signer.EXPECT().Sender(input[1]).Return(common.Address{2}, nil),
		signer.EXPECT().Sender(input[2]).Return(common.Address{}, errors.New("error")),
		signer.EXPECT().Sender(input[3]).Return(common.Address{3}, nil),
	)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			txs := getExecutionOrder(input, signer, test.isSonic)
			// one transaction is removed from the list
			if got, want := len(txs), test.expectedLen; got != want {
				t.Errorf("unexpected number of transasctions, got: %d, want: %d", got, want)
			}
		})
	}
}

func TestTxScrambler_AuthorizationsAreRespectedInOrdering(t *testing.T) {
	entries := []ScramblerEntry{
		&dummyScramblerEntry{
			hash:   common.Hash{1},
			sender: common.Address{1},
			nonce:  1,
			authorizations: []Authorization{
				{
					address: common.Address{5},
					nonce:   11,
				},
			},
		},
		&dummyScramblerEntry{
			hash:   common.Hash{2},
			sender: common.Address{2},
			nonce:  2,
			authorizations: []Authorization{
				{
					address: common.Address{5},
					nonce:   10,
				},
			},
		},
	}

	original := slices.Clone(entries)
	sortTransactionsWithSameSender(entries)
	if entries[0] != original[1] || entries[1] != original[0] {
		t.Error("unexpected order")
	}
}

func TestTxScrambler_SetCodeAuthorizationTransaction(t *testing.T) {
	address := common.Address{10}
	tests := map[string]struct {
		first  ScramblerEntry
		second ScramblerEntry
		third  ScramblerEntry
	}{
		"authorization before transactions": {
			first: &dummyScramblerEntry{
				hash:   common.Hash{7},
				sender: common.Address{100},
				nonce:  10,
				authorizations: []Authorization{
					{
						address: address,
						nonce:   1,
					},
				},
			},
			second: &dummyScramblerEntry{
				hash:   common.Hash{6},
				sender: address,
				nonce:  10,
			},
			third: &dummyScramblerEntry{
				hash:   common.Hash{5},
				sender: address,
				nonce:  20,
			},
		},
		"authorization between transactions": {
			first: &dummyScramblerEntry{
				hash:   common.Hash{5},
				sender: address,
				nonce:  10,
			},
			third: &dummyScramblerEntry{
				hash:   common.Hash{6},
				sender: address,
				nonce:  30,
			},
			second: &dummyScramblerEntry{
				hash:   common.Hash{7},
				sender: common.Address{100},
				nonce:  100,
				authorizations: []Authorization{
					{
						address: address,
						nonce:   20,
					},
				},
			},
		},
		"authorization after transactions": {
			first: &dummyScramblerEntry{
				hash:   common.Hash{7},
				sender: address,
				nonce:  5,
			},
			second: &dummyScramblerEntry{
				hash:   common.Hash{6},
				sender: address,
				nonce:  6,
			},
			third: &dummyScramblerEntry{
				hash:   common.Hash{5},
				sender: common.Address{1},
				nonce:  1,
				authorizations: []Authorization{
					{
						address: address,
						nonce:   7,
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// add entries in reverse order
			entries := []ScramblerEntry{test.third, test.second, test.first}

			sortTransactionsWithSameSender(entries)
			if got, want := len(entries), 3; got != want {
				t.Fatalf("unexpected number of transactions, got: %d, want: %d", got, want)
			}
			if entries[0] != test.first || entries[1] != test.second || entries[2] != test.third {
				t.Errorf("unexpected order")
			}
		})
	}
}

func TestTxScrambler_TransitiveDependenciesAreRespected(t *testing.T) {
	address1 := common.Address{10}
	address2 := common.Address{20}
	address3 := common.Address{30}
	transactions := []ScramblerEntry{
		&dummyScramblerEntry{
			hash:   common.Hash{7},
			sender: address1,
			nonce:  30,
		},
		&dummyScramblerEntry{
			hash:   common.Hash{6},
			sender: address2,
			nonce:  20,
			authorizations: []Authorization{
				{
					address: address1,
					nonce:   20,
				},
			},
		},
		&dummyScramblerEntry{
			hash:   common.Hash{5},
			sender: address3,
			nonce:  110,
			authorizations: []Authorization{
				{
					address: address2,
					nonce:   10,
				},
			},
		},
		&dummyScramblerEntry{
			hash:   common.Hash{4},
			sender: address3,
			nonce:  100,
		},
	}

	entries := slices.Clone(transactions)
	sortTransactionsWithSameSender(entries)
	if got, want := len(entries), 4; got != want {
		t.Fatalf("unexpected number of transactions, got: %d, want: %d", got, want)
	}
	if entries[0] != transactions[3] || entries[1] != transactions[2] ||
		entries[2] != transactions[1] || entries[3] != transactions[0] {
		t.Errorf("unexpected order %v, %v, %v, %v", entries[0].Hash(), entries[1].Hash(), entries[2].Hash(), entries[3].Hash())
	}
}

func TestTxScrambler_SortTransactionsWithSameSenderDoesNotReorderOtherTransactions(t *testing.T) {
	address1 := common.Address{10}
	address2 := common.Address{20}
	transactions := []ScramblerEntry{
		&dummyScramblerEntry{
			hash:   common.Hash{7},
			sender: address1,
			nonce:  20,
		},
		&dummyScramblerEntry{
			hash:   common.Hash{6},
			sender: address2,
			nonce:  20,
		},
		&dummyScramblerEntry{
			hash:   common.Hash{5},
			sender: address1,
			nonce:  10,
		},
		&dummyScramblerEntry{
			hash:   common.Hash{4},
			sender: address2,
			nonce:  10,
		},
	}

	entries := slices.Clone(transactions)
	sortTransactionsWithSameSender(entries)
	if got, want := len(entries), 4; got != want {
		t.Fatalf("unexpected number of transactions, got: %d, want: %d", got, want)
	}
	if entries[0] != transactions[2] || entries[1] != transactions[3] || entries[2] != transactions[0] || entries[3] != transactions[1] {
		t.Errorf("unexpected order")
	}
}

func TestTopSort_CycleBetweenTransactionAndAuthorization(t *testing.T) {
	entries := []ScramblerEntry{
		&dummyScramblerEntry{
			hash:   common.Hash{1},
			sender: common.Address{1},
			nonce:  11,
			authorizations: []Authorization{
				{
					address: common.Address{5},
					nonce:   10,
				},
			},
		},
		&dummyScramblerEntry{
			hash:   common.Hash{2},
			sender: common.Address{1},
			nonce:  10,
			authorizations: []Authorization{
				{
					address: common.Address{5},
					nonce:   11,
				},
			},
		},
	}

	original := slices.Clone(entries)
	sortTransactionsWithSameSender(entries)
	if len(entries) != 2 {
		t.Fatalf("unexpected number of transactions, got: %d, want: %d", len(entries), 2)
	}
	if entries[0] != original[0] || entries[1] != original[1] {
		t.Error("unexpected order")
	}
}

func TestTopSort_CyclicDependencyWithSortableSubset(t *testing.T) {
	// T1 -> T2 -> T3 -> T4
	// T4 -> T3
	// ensure T1 and T2 are still sorted correctly
	entries := []ScramblerEntry{
		&dummyScramblerEntry{
			hash:   common.Hash{1},
			sender: common.Address{1},
			nonce:  10,
			authorizations: []Authorization{
				{
					address: common.Address{100},
					nonce:   10,
				},
			},
		},
		&dummyScramblerEntry{
			hash:   common.Hash{2},
			sender: common.Address{2},
			nonce:  10,
			authorizations: []Authorization{
				{
					address: common.Address{100},
					nonce:   11,
				},
				{
					address: common.Address{101},
					nonce:   10,
				},
			},
		},
		&dummyScramblerEntry{
			hash:   common.Hash{3},
			sender: common.Address{3},
			nonce:  10,
			authorizations: []Authorization{
				{
					address: common.Address{101},
					nonce:   11,
				},
				{
					address: common.Address{102},
					nonce:   10,
				},
				{
					address: common.Address{103},
					nonce:   11,
				},
			},
		},
		&dummyScramblerEntry{
			hash:   common.Hash{4},
			sender: common.Address{4},
			nonce:  10,
			authorizations: []Authorization{
				{
					address: common.Address{102},
					nonce:   11,
				},
				{
					address: common.Address{103},
					nonce:   10,
				},
			},
		},
	}

	original := slices.Clone(entries)
	shuffleEntries(entries)
	sortTransactionsWithSameSender(entries)
	if len(entries) != 4 {
		t.Fatalf("unexpected number of transactions, got: %d, want: %d", len(entries), 4)
	}
	first := true
	for _, entry := range entries {
		if entry == original[0] {
			first = false
			continue
		}
		if first && entry == original[1] {
			t.Error("unexpected order")
		}
	}
}

func compareFunc(a ScramblerEntry, b ScramblerEntry) int {
	addrCmp := a.Sender().Cmp(b.Sender())
	if addrCmp != 0 {
		return addrCmp
	}
	res := cmp.Compare(a.Nonce(), b.Nonce())
	if res != 0 {
		return res
	}
	res = a.GasPrice().Cmp(b.GasPrice())
	if res != 0 {
		return res
	}
	return a.Hash().Cmp(b.Hash())
}

func BenchmarkTxScrambler(b *testing.B) {
	for size := int64(10); size < 100_000; size *= 10 {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			for i := 1; i <= b.N; i++ {
				b.StopTimer()
				entries := createRandomScramblerTestInput(size)
				b.StartTimer()
				filterAndOrderTransactions(entries)
			}
		})
	}
}

func createRandomScramblerTestInput(size int64) []ScramblerEntry {
	var entries []ScramblerEntry
	for i := int64(0); i < size; i++ {
		// same hashes must have same data
		r := rand.IntN(100 - 1)
		entries = append(entries, &dummyScramblerEntry{
			hash:     common.Hash{byte(r)},
			sender:   common.Address{byte(r)},
			nonce:    uint64(r),
			gasPrice: big.NewInt(int64(r)),
		})
	}

	return entries
}

// shuffleEntries shuffles given entries randomly.
func shuffleEntries(entries []ScramblerEntry) {
	rand.Shuffle(len(entries), func(i, j int) {
		entries[i], entries[j] = entries[j], entries[i]
	})
}

// checkDuplicateHashes checks hash of each entry and fails test if duplicate hash is found.
func checkDuplicateHashes(t *testing.T, entries []ScramblerEntry) {
	seenHashes := make(map[common.Hash]struct{})
	for _, entry := range entries {
		if _, found := seenHashes[entry.Hash()]; found {
			t.Fatalf("found duplicate hash in entries: %s", entry.Hash())
		}
		seenHashes[entry.Hash()] = struct{}{}
	}
}

func createRandomSalt() [32]byte {
	var salt = [32]byte{}
	for i := 0; i < 32; i++ {
		salt[i] = byte(rand.IntN(256))
	}
	return salt
}

// dummyScramblerEntry represents scramblery entry data used for testing
type dummyScramblerEntry struct {
	hash           common.Hash    // transaction hash
	sender         common.Address // sender of the transaction
	nonce          uint64         // transaction nonce
	gasPrice       *big.Int       // transaction gasPrice
	authorizations []Authorization
}

func (s *dummyScramblerEntry) Hash() common.Hash {
	return s.hash
}

func (s *dummyScramblerEntry) Sender() common.Address {
	return s.sender
}

func (s *dummyScramblerEntry) Nonce() uint64 {
	return s.nonce
}

func (s *dummyScramblerEntry) GasPrice() *big.Int {
	return s.gasPrice
}

func (s *dummyScramblerEntry) Authorizations() []Authorization {
	return s.authorizations
}
