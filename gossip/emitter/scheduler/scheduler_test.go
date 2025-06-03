package scheduler

import (
	"context"
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestScheduler_Schedule_ForwardsBlockInfoToTheProcessor(t *testing.T) {
	ctrl := gomock.NewController(t)
	factory := NewMockprocessorFactory(ctrl)
	processor := NewMockprocessor(ctrl)
	txs := NewMockPrioritizedTransactions(ctrl)

	number := idx.Block(2)
	time := inter.Timestamp(345)
	gasLimit := uint64(67)
	coinbase := common.Address{0, 8, 15}
	mixHash := common.Hash{42, 73}
	baseFee := *uint256.NewInt(100)
	blobBaseFee := *uint256.NewInt(200)

	factory.EXPECT().beginBlock(&evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number:      big.NewInt(int64(number)),
			Time:        time,
			GasLimit:    gasLimit,
			Coinbase:    coinbase,
			PrevRandao:  mixHash,
			BaseFee:     baseFee.ToBig(),
			BlobBaseFee: blobBaseFee.ToBig(),
		},
	}).Return(processor)
	processor.EXPECT().release()

	txs.EXPECT().Current().Return(nil)

	newScheduler(factory).Schedule(
		t.Context(),
		&BlockInfo{
			Number:      idx.Block(number),
			Time:        time,
			GasLimit:    gasLimit,
			Coinbase:    coinbase,
			MixHash:     mixHash,
			BaseFee:     baseFee,
			BlobBaseFee: blobBaseFee,
		},
		txs,
		Limits{},
	)
}

func TestScheduler_Schedule_TransactionsAreSignaledAsAcceptedOrSkipped(t *testing.T) {
	ctrl := gomock.NewController(t)
	factory := NewMockprocessorFactory(ctrl)
	processor := NewMockprocessor(ctrl)
	factory.EXPECT().beginBlock(gomock.Any()).Return(processor).AnyTimes()
	processor.EXPECT().release().AnyTimes()

	txs := NewMockPrioritizedTransactions(ctrl)

	tx1 := types.NewTx(&types.LegacyTx{Nonce: 1})
	tx2 := types.NewTx(&types.LegacyTx{Nonce: 2})
	tx3 := types.NewTx(&types.LegacyTx{Nonce: 3})

	gomock.InOrder(
		txs.EXPECT().Current().Return(tx1),
		processor.EXPECT().run(tx1, gomock.Any()).Return(true, uint64(0)),
		txs.EXPECT().Accept(),
		txs.EXPECT().Current().Return(tx2),
		processor.EXPECT().run(tx2, gomock.Any()).Return(false, uint64(0)),
		txs.EXPECT().Skip(),
		txs.EXPECT().Current().Return(tx3),
		processor.EXPECT().run(tx3, gomock.Any()).Return(true, uint64(0)),
		txs.EXPECT().Accept(),
		txs.EXPECT().Current().Return(nil),
	)

	scheduler := newScheduler(factory)
	result := scheduler.Schedule(
		t.Context(),
		&BlockInfo{},
		txs,
		Limits{
			Gas:  uint64(100) * params.TxGas,
			Size: math.MaxUint64, // no size limit
		},
	)

	require.Equal(t, []*types.Transaction{tx1, tx3}, result)
}

func TestScheduler_Schedule_RetrievalOfTransactionsStopsWhenGasLimitIsReached(t *testing.T) {
	ctrl := gomock.NewController(t)
	factory := NewMockprocessorFactory(ctrl)
	processor := NewMockprocessor(ctrl)
	factory.EXPECT().beginBlock(gomock.Any()).Return(processor).AnyTimes()
	processor.EXPECT().release().AnyTimes()

	txs := NewMockPrioritizedTransactions(ctrl)

	tx1 := types.NewTx(&types.LegacyTx{Nonce: 1})
	tx2 := types.NewTx(&types.LegacyTx{Nonce: 2})
	tx3 := types.NewTx(&types.LegacyTx{Nonce: 2})
	tx4 := types.NewTx(&types.LegacyTx{Nonce: 2})

	gomock.InOrder(
		txs.EXPECT().Current().Return(tx1),
		processor.EXPECT().run(tx1, gomock.Any()).Return(true, uint64(5)*params.TxGas),
		txs.EXPECT().Accept(),
		txs.EXPECT().Current().Return(tx2),
		processor.EXPECT().run(tx2, gomock.Any()).Return(true, uint64(14)*params.TxGas-1),
		txs.EXPECT().Accept(),
		// remaining gas: params.TxGas + 1
		txs.EXPECT().Current().Return(tx3),
		processor.EXPECT().run(tx3, gomock.Any()).Return(true, uint64(1)),
		txs.EXPECT().Accept(),
		// remaining gas: params.TxGas
		txs.EXPECT().Current().Return(tx4),
		processor.EXPECT().run(tx4, gomock.Any()).Return(true, uint64(1)),
		txs.EXPECT().Accept(),
		// remaining gas: params.TxGas - 1
		// No more Current() after this point
	)

	scheduler := newScheduler(factory)
	result := scheduler.Schedule(
		t.Context(),
		&BlockInfo{},
		txs,
		Limits{
			Gas:  uint64(20) * params.TxGas,
			Size: math.MaxUint64, // no size limit
		},
	)

	require.Equal(t, []*types.Transaction{tx1, tx2, tx3, tx4}, result)
}

func TestScheduler_Schedule_TooLargeTransactionsAreSkipped(t *testing.T) {
	ctrl := gomock.NewController(t)
	factory := NewMockprocessorFactory(ctrl)
	processor := NewMockprocessor(ctrl)
	factory.EXPECT().beginBlock(gomock.Any()).Return(processor).AnyTimes()
	processor.EXPECT().release().AnyTimes()

	txs := NewMockPrioritizedTransactions(ctrl)

	tx1 := types.NewTx(&types.LegacyTx{Nonce: 1})
	tx2 := types.NewTx(&types.LegacyTx{Nonce: 2, Data: make([]byte, 1000)})
	tx3 := types.NewTx(&types.LegacyTx{Nonce: 2})
	tx4 := types.NewTx(&types.LegacyTx{Nonce: 2})

	small := tx1.Size()
	large := tx2.Size()
	require.Less(t, small, large)
	require.Less(t, uint64(10), large)

	gomock.InOrder(
		txs.EXPECT().Current().Return(tx1),
		processor.EXPECT().run(tx1, gomock.Any()).Return(true, uint64(5)*params.TxGas),
		txs.EXPECT().Accept(),
		txs.EXPECT().Current().Return(tx2),
		// tx2 is too large, so it is not executed, just skipped
		txs.EXPECT().Skip(),
		txs.EXPECT().Current().Return(tx3),
		processor.EXPECT().run(tx3, gomock.Any()).Return(true, uint64(1)),
		txs.EXPECT().Accept(),
		txs.EXPECT().Current().Return(tx4),
		processor.EXPECT().run(tx4, gomock.Any()).Return(true, uint64(1)),
		txs.EXPECT().Accept(),
		txs.EXPECT().Current().Return(nil),
	)

	scheduler := newScheduler(factory)
	result := scheduler.Schedule(
		t.Context(),
		&BlockInfo{},
		txs,
		Limits{
			Gas:  math.MaxInt64,
			Size: 3*small + 10,
		},
	)

	require.Equal(t, []*types.Transaction{tx1, tx3, tx4}, result)
}

func TestScheduler_Schedule_SizeLimitIsEnforced(t *testing.T) {
	ctrl := gomock.NewController(t)
	factory := NewMockprocessorFactory(ctrl)
	processor := NewMockprocessor(ctrl)
	factory.EXPECT().beginBlock(gomock.Any()).Return(processor).AnyTimes()
	processor.EXPECT().run(gomock.Any(), gomock.Any()).Return(true, uint64(10)).AnyTimes()
	processor.EXPECT().release().AnyTimes()

	txs := []*types.Transaction{}
	for i := range uint64(5) {
		txs = append(txs, types.NewTx(&types.LegacyTx{
			Nonce: i,
			Data:  make([]byte, 10*i),
		}))
	}

	totalSize := uint64(0)
	for _, tx := range txs {
		totalSize += tx.Size()
	}

	scheduler := newScheduler(factory)
	for limit := range totalSize + 10 {
		for input := range utils.Permute(txs) {
			result := scheduler.Schedule(
				t.Context(),
				&BlockInfo{},
				&fakeTxCollection{input},
				Limits{
					Gas:  math.MaxUint64, // no gas limit
					Size: limit,
				},
			)

			want := uint64(0)
			for _, tx := range input {
				size := tx.Size()
				if want+size <= limit {
					want += size
				}
			}
			got := uint64(0)
			for _, tx := range result {
				got += tx.Size()
			}
			require.LessOrEqual(t, got, limit)
			require.Equal(t, want, got)
		}
	}

}

func TestScheduler_Schedule_StopsWhenContextIsCancelled(t *testing.T) {
	ctrl := gomock.NewController(t)
	factory := NewMockprocessorFactory(ctrl)
	processor := NewMockprocessor(ctrl)
	factory.EXPECT().beginBlock(gomock.Any()).Return(processor).AnyTimes()
	processor.EXPECT().release().AnyTimes()

	txs := NewMockPrioritizedTransactions(ctrl)

	tx1 := types.NewTx(&types.LegacyTx{Nonce: 1})
	tx2 := types.NewTx(&types.LegacyTx{Nonce: 2})

	ctxt, cancel := context.WithCancel(t.Context())

	gomock.InOrder(
		txs.EXPECT().Current().Return(tx1),
		processor.EXPECT().run(tx1, gomock.Any()).Return(true, params.TxGas),
		txs.EXPECT().Accept(),
		txs.EXPECT().Current().Return(tx2),
		processor.EXPECT().run(tx2, gomock.Any()).DoAndReturn(
			func(tx *types.Transaction, remainingGas uint64) (bool, uint64) {
				cancel() // cancel the scheduler
				return true, params.TxGas
			},
		),
		txs.EXPECT().Accept(),
		// No more Current() after this point
	)

	scheduler := newScheduler(factory)
	result := scheduler.Schedule(
		ctxt,
		&BlockInfo{},
		txs,
		Limits{
			Gas:  uint64(20) * params.TxGas,
			Size: math.MaxUint64, // no size limit
		},
	)

	require.Equal(t, []*types.Transaction{tx1, tx2}, result)
}

func TestScheduler_Schedule_IgnoresFailedTransactions(t *testing.T) {

	tests := map[string]struct {
		txSuccess []bool // < in order of priority
		selected  []int  // < expected selected transactions
	}{
		"all passing": {
			txSuccess: []bool{true, true, true, true},
			selected:  []int{0, 1, 2, 3},
		},
		"one failing": {
			txSuccess: []bool{true, false, true, true},
			selected:  []int{0, 2, 3},
		},
		"two failing": {
			txSuccess: []bool{true, false, false, true},
			selected:  []int{0, 3},
		},
		"all failing": {
			txSuccess: []bool{false, false, false, false},
			selected:  []int{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			factory := NewMockprocessorFactory(ctrl)
			processor := NewMockprocessor(ctrl)
			factory.EXPECT().beginBlock(gomock.Any()).Return(processor).AnyTimes()
			processor.EXPECT().release().AnyTimes()
			processor.EXPECT().run(gomock.Any(), gomock.Any()).DoAndReturn(
				func(tx *types.Transaction, remainingGas uint64) (bool, uint64) {
					return test.txSuccess[tx.Nonce()], 0
				},
			).AnyTimes()

			txs := []*types.Transaction{}
			for i := range uint64(len(test.txSuccess)) {
				txs = append(txs, types.NewTx(&types.LegacyTx{Nonce: i}))
			}

			scheduler := newScheduler(factory)
			result := scheduler.Schedule(
				t.Context(),
				&BlockInfo{},
				&fakeTxCollection{txs},
				Limits{
					Gas:  uint64(100) * params.TxGas,
					Size: math.MaxUint64, // no size limit
				},
			)

			got := []int{}
			for _, tx := range result {
				got = append(got, int(tx.Nonce()))
			}
			require.ElementsMatch(t, test.selected, got)
		})
	}
}

func TestScheduler_Schedule_OrderOfInputTransactionsIsPreserved(t *testing.T) {
	ctrl := gomock.NewController(t)
	factory := NewMockprocessorFactory(ctrl)
	processor := NewMockprocessor(ctrl)
	factory.EXPECT().beginBlock(gomock.Any()).Return(processor).AnyTimes()
	processor.EXPECT().release().AnyTimes()

	txResults := []struct {
		costs   int
		success bool
	}{
		{costs: 1, success: true},
		{costs: 2, success: true},
		{costs: 1, success: false},
		{costs: 4, success: true},
		{costs: 2, success: false},
		{costs: 1, success: true},
	}

	processor.EXPECT().run(gomock.Any(), gomock.Any()).DoAndReturn(
		func(tx *types.Transaction, remainingGas uint64) (bool, uint64) {
			res := txResults[tx.Nonce()]
			return res.success, uint64(res.costs) * params.TxGas
		},
	).AnyTimes()

	txs := []*types.Transaction{}
	for i := range uint64(len(txResults)) {
		txs = append(txs, types.NewTx(&types.LegacyTx{Nonce: i}))
	}

	for limit := range uint64(15) {

		scheduler := newScheduler(factory)
		result := scheduler.Schedule(
			t.Context(),
			&BlockInfo{},
			&fakeTxCollection{txs},
			Limits{
				Gas:  limit * params.TxGas,
				Size: math.MaxUint64, // no size limit
			},
		)

		got := []int{}
		for _, tx := range result {
			got = append(got, int(tx.Nonce()))
		}

		want := []int{}
		sum := uint64(0)
		for i, res := range txResults {
			if !res.success {
				continue
			}
			cost := uint64(res.costs)
			if sum+cost <= limit {
				want = append(want, i)
				sum += cost
			}
		}

		require.Equal(t, want, got, "for limit %d", limit)
	}
}

func TestScheduler_Schedule_GetsOptimalPrefixIfAllTransactionsArePassing(t *testing.T) {

	tests := map[string]struct {
		txCosts  []int // < in order of priority, all passing
		limit    int   // < gas limit for the block
		selected []int // < expected selected transactions
	}{
		"enough for all": {
			txCosts:  []int{1, 1, 1, 1},
			limit:    4,
			selected: []int{0, 1, 2, 3},
		},
		"more then enough for all": {
			txCosts:  []int{1, 1, 1, 1},
			limit:    6,
			selected: []int{0, 1, 2, 3},
		},
		"enough for all but one low-priority transaction": {
			txCosts:  []int{1, 1, 1, 2},
			limit:    4,
			selected: []int{0, 1, 2},
		},
		"enough for all but two low-priority": {
			txCosts:  []int{1, 1, 2, 2},
			limit:    3,
			selected: []int{0, 1},
		},
		"a high priority tx to be skipped": {
			txCosts:  []int{6, 1, 1, 1},
			limit:    3,
			selected: []int{1, 2, 3},
		},
		"a medium priority tx to be skipped": {
			txCosts:  []int{2, 6, 1, 1, 1},
			limit:    4,
			selected: []int{0, 2, 3},
		},
		"a low priority tx to be skipped": {
			txCosts:  []int{1, 1, 6, 1, 1},
			limit:    3,
			selected: []int{0, 1, 3},
		},
		"a high and low priority tx to be skipped": {
			txCosts:  []int{6, 1, 1, 1, 2},
			limit:    4,
			selected: []int{1, 2, 3},
		},
		"zero limit produces no transactions": {
			txCosts:  []int{1, 2},
			limit:    0,
			selected: []int{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			factory := NewMockprocessorFactory(ctrl)
			processor := NewMockprocessor(ctrl)
			factory.EXPECT().beginBlock(gomock.Any()).Return(processor).AnyTimes()
			processor.EXPECT().release().AnyTimes()
			processor.EXPECT().run(gomock.Any(), gomock.Any()).DoAndReturn(
				func(tx *types.Transaction, remainingGas uint64) (bool, uint64) {
					needed := uint64(test.txCosts[tx.Nonce()]) * params.TxGas
					return remainingGas >= needed, needed
				},
			).AnyTimes()

			txs := []*types.Transaction{}
			for i := range uint64(len(test.txCosts)) {
				txs = append(txs, types.NewTx(&types.LegacyTx{Nonce: i}))
			}

			scheduler := newScheduler(factory)
			result := scheduler.Schedule(
				t.Context(),
				&BlockInfo{},
				&fakeTxCollection{txs},
				Limits{
					Gas:  uint64(test.limit) * params.TxGas,
					Size: math.MaxUint64, // no size limit
				},
			)

			got := []int{}
			for _, tx := range result {
				got = append(got, int(tx.Nonce()))
			}
			require.ElementsMatch(t, test.selected, got)
		})
	}
}

func FuzzScheduler_Schedule_PicksLongestPrefixOfAcceptedTransactions(f *testing.F) {
	// the txResults slice encodes the number of transactions, their
	// success/failure through the least significant bit, and the gas
	// cost of each transaction in the remaining bits.
	f.Fuzz(func(t *testing.T, limit uint64, txResults []byte) {

		getCosts := func(b byte) uint64 {
			return uint64(b>>1) * params.TxGas
		}
		isSuccess := func(b byte) bool {
			return b%2 == 0 && getCosts(b) > 0
		}

		ctrl := gomock.NewController(t)

		factory := NewMockprocessorFactory(ctrl)
		processor := NewMockprocessor(ctrl)
		factory.EXPECT().beginBlock(gomock.Any()).Return(processor).AnyTimes()
		processor.EXPECT().release().AnyTimes()
		processor.EXPECT().run(gomock.Any(), gomock.Any()).DoAndReturn(
			func(tx *types.Transaction, remainingGas uint64) (bool, uint64) {
				res := txResults[tx.Nonce()]
				cost := getCosts(res)
				return isSuccess(res) && cost <= remainingGas, cost
			},
		).AnyTimes()

		txs := []*types.Transaction{}
		for i := range uint64(len(txResults)) {
			txs = append(txs, types.NewTx(&types.LegacyTx{Nonce: i}))
		}

		scheduler := newScheduler(factory)
		result := scheduler.Schedule(
			t.Context(),
			&BlockInfo{},
			&fakeTxCollection{txs},
			Limits{
				Gas:  limit * params.TxGas,
				Size: math.MaxUint64, // no size limit
			},
		)

		// Get the expected schedule by taking all elements from the
		// beginning that still fit in the gas limit.
		want := []int{}
		sum := uint64(0)
		for i, res := range txResults {
			if !isSuccess(res) {
				continue
			}
			cost := getCosts(res)
			if sum+cost <= limit*params.TxGas {
				want = append(want, i)
				sum += cost
			}
		}

		got := []int{}
		for _, tx := range result {
			got = append(got, int(tx.Nonce()))
		}
		require.Equal(t, want, got)
	})
}

type fakeTxCollection struct {
	transactions []*types.Transaction
}

func (c *fakeTxCollection) Current() *types.Transaction {
	if len(c.transactions) == 0 {
		return nil
	}
	return c.transactions[0]
}

func (c *fakeTxCollection) Accept() {
	if len(c.transactions) == 0 {
		return
	}
	c.transactions = c.transactions[1:]
}

func (c *fakeTxCollection) Skip() {
	c.Accept()
}
