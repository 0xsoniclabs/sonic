package drivermodule

import (
	"math"
	"math/big"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/drivertype"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver/driverpos"
)

// --- helpers ----------------------------------------------------------------

func testValidators(ids ...idx.ValidatorID) *pos.Validators {
	b := pos.NewBuilder()
	for _, id := range ids {
		b.Set(id, 1)
	}
	return b.Build()
}

func testBlockState(vals *pos.Validators) iblockproc.BlockState {
	states := make([]iblockproc.ValidatorBlockState, vals.Len())
	for i := range states {
		states[i].Originated = new(big.Int)
	}
	profiles := make(iblockproc.ValidatorProfiles)
	for _, id := range vals.IDs() {
		profiles[id] = drivertype.Validator{
			Weight: big.NewInt(1),
			PubKey: validatorpk.PubKey{Type: 0, Raw: []byte{}},
		}
	}
	return iblockproc.BlockState{
		ValidatorStates:       states,
		NextValidatorProfiles: profiles,
	}
}

func testEpochState(vals *pos.Validators) iblockproc.EpochState {
	return iblockproc.EpochState{
		Epoch:      1,
		EpochStart: inter.Timestamp(1000),
		Validators: vals,
		ValidatorStates: make(
			[]iblockproc.ValidatorEpochState, vals.Len(),
		),
		Rules: opera.FakeNetRules(opera.Upgrades{London: true}),
	}
}

// --- NewDriverTxListenerModule / Start / Finalize / Update -----------------

func TestNewDriverTxListenerModule(t *testing.T) {
	m := NewDriverTxListenerModule()
	require.NotNil(t, m)
}

func TestDriverTxListenerModule_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	vals := testValidators(1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)
	require.NotNil(t, listener)
}

func TestDriverTxListener_Finalize_ReturnsBlockState(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	vals := testValidators(1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)
	result := listener.Finalize()
	require.Equal(t, bs.NextValidatorProfiles, result.NextValidatorProfiles)
}

func TestDriverTxListener_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	vals := testValidators(1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)

	bs2 := testBlockState(vals)
	es2 := testEpochState(vals)
	es2.Epoch = 5
	listener.Update(bs2, es2)

	// After update, Finalize should reflect new state.
	result := listener.Finalize()
	require.Equal(t, bs2.NextValidatorProfiles, result.NextValidatorProfiles)
}

// --- OnNewLog: UpdateValidatorWeight ---------------------------------------

func TestOnNewLog_UpdateValidatorWeight_SetsWeight(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	v1 := idx.ValidatorID(1)
	vals := testValidators(v1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)

	// Build log: UpdateValidatorWeight(validatorID=1, weight=500)
	topic1 := common.Hash{}
	big.NewInt(int64(v1)).FillBytes(topic1[:])
	data := make([]byte, 32)
	big.NewInt(500).FillBytes(data)

	l := &types.Log{
		Address: driver.ContractAddress,
		Topics:  []common.Hash{driverpos.Topics.UpdateValidatorWeight, topic1},
		Data:    data,
	}
	listener.OnNewLog(l)

	result := listener.Finalize()
	profile := result.NextValidatorProfiles[v1]
	require.Equal(t, big.NewInt(500), profile.Weight)
}

func TestOnNewLog_UpdateValidatorWeight_ZeroWeight_Deletes(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	v1 := idx.ValidatorID(1)
	vals := testValidators(v1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)

	topic1 := common.Hash{}
	big.NewInt(int64(v1)).FillBytes(topic1[:])
	data := make([]byte, 32) // all zeros = weight 0

	l := &types.Log{
		Address: driver.ContractAddress,
		Topics:  []common.Hash{driverpos.Topics.UpdateValidatorWeight, topic1},
		Data:    data,
	}
	listener.OnNewLog(l)

	result := listener.Finalize()
	_, exists := result.NextValidatorProfiles[v1]
	require.False(t, exists, "validator should be deleted when weight is 0")
}

func TestOnNewLog_UpdateValidatorWeight_NewValidator(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	v1 := idx.ValidatorID(1)
	vals := testValidators(v1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)

	// Add a new validator (ID=99) via weight update.
	newVal := idx.ValidatorID(99)
	topic1 := common.Hash{}
	big.NewInt(int64(newVal)).FillBytes(topic1[:])
	data := make([]byte, 32)
	big.NewInt(200).FillBytes(data)

	l := &types.Log{
		Address: driver.ContractAddress,
		Topics:  []common.Hash{driverpos.Topics.UpdateValidatorWeight, topic1},
		Data:    data,
	}
	listener.OnNewLog(l)

	result := listener.Finalize()
	profile, exists := result.NextValidatorProfiles[newVal]
	require.True(t, exists)
	require.Equal(t, big.NewInt(200), profile.Weight)
	require.True(t, profile.PubKey.Empty(), "new validator should have empty pubkey initially")
}

// --- OnNewLog: UpdateValidatorPubkey ---------------------------------------

func TestOnNewLog_UpdateValidatorPubkey(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	v1 := idx.ValidatorID(1)
	vals := testValidators(v1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)

	// Encode dynamic bytes per the ABI encoding used in decodeDataBytes:
	// first 32 bytes = offset, then at offset: 32 bytes length + data.
	rawPubKey := validatorpk.PubKey{
		Type: validatorpk.Types.Secp256k1,
		Raw:  make([]byte, 33),
	}
	rawPubKey.Raw[0] = 0x02 // compressed secp256k1 prefix
	pubkeyBytes := rawPubKey.Bytes()

	// ABI-encode the bytes parameter.
	offset := make([]byte, 32)
	big.NewInt(32).FillBytes(offset)
	length := make([]byte, 32)
	big.NewInt(int64(len(pubkeyBytes))).FillBytes(length)
	data := append(offset, length...)
	data = append(data, pubkeyBytes...)
	// Pad to 32-byte boundary.
	if rem := len(pubkeyBytes) % 32; rem != 0 {
		data = append(data, make([]byte, 32-rem)...)
	}

	topic1 := common.Hash{}
	big.NewInt(int64(v1)).FillBytes(topic1[:])

	l := &types.Log{
		Address: driver.ContractAddress,
		Topics:  []common.Hash{driverpos.Topics.UpdateValidatorPubkey, topic1},
		Data:    data,
	}
	listener.OnNewLog(l)

	result := listener.Finalize()
	profile := result.NextValidatorProfiles[v1]
	require.Equal(t, rawPubKey.Type, profile.PubKey.Type)
	require.Equal(t, rawPubKey.Raw, profile.PubKey.Raw)
}

func TestOnNewLog_UpdateValidatorPubkey_UnknownValidator_Ignored(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	v1 := idx.ValidatorID(1)
	vals := testValidators(v1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)

	unknownVal := idx.ValidatorID(99)
	topic1 := common.Hash{}
	big.NewInt(int64(unknownVal)).FillBytes(topic1[:])

	// Valid ABI-encoded pubkey data.
	offset := make([]byte, 32)
	big.NewInt(32).FillBytes(offset)
	length := make([]byte, 32)
	big.NewInt(2).FillBytes(length)
	data := append(offset, length...)
	data = append(data, 0xc0, 0x01) // type + 1 byte
	data = append(data, make([]byte, 30)...)

	l := &types.Log{
		Address: driver.ContractAddress,
		Topics:  []common.Hash{driverpos.Topics.UpdateValidatorPubkey, topic1},
		Data:    data,
	}
	listener.OnNewLog(l)

	result := listener.Finalize()
	_, exists := result.NextValidatorProfiles[unknownVal]
	require.False(t, exists, "pubkey update for unknown validator should be ignored")
}

// --- OnNewLog: AdvanceEpochs -----------------------------------------------

func TestOnNewLog_AdvanceEpochs(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	vals := testValidators(1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)

	data := make([]byte, 32)
	big.NewInt(3).FillBytes(data)

	l := &types.Log{
		Address: driver.ContractAddress,
		Topics:  []common.Hash{driverpos.Topics.AdvanceEpochs},
		Data:    data,
	}
	listener.OnNewLog(l)

	result := listener.Finalize()
	require.Equal(t, idx.Epoch(3), result.AdvanceEpochs)
}

func TestOnNewLog_AdvanceEpochs_CappedAtMax(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	vals := testValidators(1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)

	// Send a huge number; should be capped at maxAdvanceEpochs.
	data := make([]byte, 32)
	big.NewInt(int64(maxAdvanceEpochs + 100)).FillBytes(data)

	l := &types.Log{
		Address: driver.ContractAddress,
		Topics:  []common.Hash{driverpos.Topics.AdvanceEpochs},
		Data:    data,
	}
	listener.OnNewLog(l)

	result := listener.Finalize()
	require.Equal(t, idx.Epoch(maxAdvanceEpochs), result.AdvanceEpochs)
}

func TestOnNewLog_AdvanceEpochs_Accumulates(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	vals := testValidators(1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)

	for _, n := range []int64{2, 5} {
		data := make([]byte, 32)
		big.NewInt(n).FillBytes(data)
		l := &types.Log{
			Address: driver.ContractAddress,
			Topics:  []common.Hash{driverpos.Topics.AdvanceEpochs},
			Data:    data,
		}
		listener.OnNewLog(l)
	}

	result := listener.Finalize()
	require.Equal(t, idx.Epoch(7), result.AdvanceEpochs)
}

// --- OnNewLog: UpdateNetworkRules ------------------------------------------

func TestOnNewLog_UpdateNetworkRules(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	vals := testValidators(1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)

	// Build a valid JSON diff for network rules.
	diff := []byte(`{"Blocks":{"MaxBlockGas":999999}}`)

	// ABI-encode bytes: offset(32) + length(32) + data(padded).
	offset := make([]byte, 32)
	big.NewInt(32).FillBytes(offset)
	length := make([]byte, 32)
	big.NewInt(int64(len(diff))).FillBytes(length)
	data := append(offset, length...)
	data = append(data, diff...)
	if rem := len(diff) % 32; rem != 0 {
		data = append(data, make([]byte, 32-rem)...)
	}

	l := &types.Log{
		Address: driver.ContractAddress,
		Topics:  []common.Hash{driverpos.Topics.UpdateNetworkRules},
		Data:    data,
	}
	listener.OnNewLog(l)

	result := listener.Finalize()
	require.NotNil(t, result.DirtyRules)
	require.Equal(t, uint64(999999), result.DirtyRules.Blocks.MaxBlockGas)
}

// --- OnNewLog: wrong address / wrong topic ---------------------------------

func TestOnNewLog_WrongAddress_Ignored(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	vals := testValidators(1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)

	l := &types.Log{
		Address: common.HexToAddress("0x1234"),
		Topics:  []common.Hash{driverpos.Topics.AdvanceEpochs},
		Data:    make([]byte, 32),
	}
	listener.OnNewLog(l)

	result := listener.Finalize()
	require.Equal(t, idx.Epoch(0), result.AdvanceEpochs)
}

func TestOnNewLog_UnknownTopic_Ignored(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	vals := testValidators(1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)

	l := &types.Log{
		Address: driver.ContractAddress,
		Topics:  []common.Hash{common.HexToHash("0xdeadbeef")},
		Data:    make([]byte, 64),
	}
	listener.OnNewLog(l)

	result := listener.Finalize()
	require.Nil(t, result.DirtyRules)
	require.Equal(t, idx.Epoch(0), result.AdvanceEpochs)
}

// --- OnNewReceipt: zero originator -----------------------------------------

func TestOnNewReceipt_ZeroOriginator_Ignored(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	vals := testValidators(1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)

	tx := types.NewTransaction(0, common.Address{}, nil, 100000, big.NewInt(100), nil)
	receipt := &types.Receipt{GasUsed: 21000}

	// originator=0 means the transaction has no validator originator.
	listener.OnNewReceipt(tx, receipt, 0, big.NewInt(50), big.NewInt(1))

	result := listener.Finalize()
	require.Equal(t, big.NewInt(0), result.ValidatorStates[0].Originated)
}

// --- OnNewReceipt: gas refund tracking -------------------------------------

func TestOnNewReceipt_TracksGasRefund(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	v1 := idx.ValidatorID(1)
	vals := testValidators(v1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)

	gasLimit := uint64(100000)
	gasUsed := uint64(21000)
	tx := types.NewTransaction(0, common.Address{}, nil, gasLimit, big.NewInt(100), nil)
	receipt := &types.Receipt{GasUsed: gasUsed}

	listener.OnNewReceipt(tx, receipt, v1, big.NewInt(50), big.NewInt(1))

	result := listener.Finalize()
	vidx := es.Validators.GetIdx(v1)
	require.Equal(t, gasLimit-gasUsed, result.ValidatorStates[vidx].DirtyGasRefund)
}

// --- NewDriverTxTransactor / NewDriverTxPreTransactor ----------------------

func TestNewDriverTxTransactor(t *testing.T) {
	require.NotNil(t, NewDriverTxTransactor())
}

func TestNewDriverTxPreTransactor(t *testing.T) {
	require.NotNil(t, NewDriverTxPreTransactor())
}

// --- PopInternalTxs: DriverTxPreTransactor ---------------------------------

func TestDriverTxPreTransactor_PopInternalTxs_NotSealing_NoCheaters(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	vals := testValidators(1)
	bs := testBlockState(vals)
	es := testEpochState(vals)
	block := iblockproc.BlockCtx{Idx: 1, Time: inter.Timestamp(2000)}

	pre := NewDriverTxPreTransactor()
	txs := pre.PopInternalTxs(block, bs, es, false, statedb)
	require.Empty(t, txs)
}

func TestDriverTxPreTransactor_PopInternalTxs_WithCheaters(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)
	statedb.EXPECT().GetNonce(common.Address{}).Return(uint64(0))

	v1 := idx.ValidatorID(1)
	v2 := idx.ValidatorID(2)
	vals := testValidators(v1, v2)
	bs := testBlockState(vals)
	bs.EpochCheaters = append(bs.EpochCheaters, v1, v2)
	bs.CheatersWritten = 0
	es := testEpochState(vals)
	block := iblockproc.BlockCtx{Idx: 1, Time: inter.Timestamp(2000)}

	pre := NewDriverTxPreTransactor()
	txs := pre.PopInternalTxs(block, bs, es, false, statedb)
	// One deactivation tx per cheater.
	require.Len(t, txs, 2)
}

func TestDriverTxPreTransactor_PopInternalTxs_Sealing(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)
	statedb.EXPECT().GetNonce(common.Address{}).Return(uint64(0))

	v1 := idx.ValidatorID(1)
	vals := testValidators(v1)
	bs := testBlockState(vals)
	es := testEpochState(vals)
	block := iblockproc.BlockCtx{Idx: 1, Time: inter.Timestamp(2000)}

	pre := NewDriverTxPreTransactor()
	txs := pre.PopInternalTxs(block, bs, es, true, statedb)
	// Sealing produces a SealEpoch tx.
	require.Len(t, txs, 1)
}

func TestDriverTxPreTransactor_PopInternalTxs_SealingWithCheaters(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)
	statedb.EXPECT().GetNonce(common.Address{}).Return(uint64(0))

	v1 := idx.ValidatorID(1)
	vals := testValidators(v1)
	bs := testBlockState(vals)
	bs.EpochCheaters = append(bs.EpochCheaters, v1)
	bs.CheatersWritten = 0
	es := testEpochState(vals)
	block := iblockproc.BlockCtx{Idx: 1, Time: inter.Timestamp(2000)}

	pre := NewDriverTxPreTransactor()
	txs := pre.PopInternalTxs(block, bs, es, true, statedb)
	// 1 cheater deactivation + 1 SealEpoch = 2.
	require.Len(t, txs, 2)
}

func TestDriverTxPreTransactor_PopInternalTxs_PartialCheatersWritten(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)
	statedb.EXPECT().GetNonce(common.Address{}).Return(uint64(0))

	v1 := idx.ValidatorID(1)
	v2 := idx.ValidatorID(2)
	vals := testValidators(v1, v2)
	bs := testBlockState(vals)
	bs.EpochCheaters = append(bs.EpochCheaters, v1, v2)
	bs.CheatersWritten = 1 // first cheater already written
	es := testEpochState(vals)
	block := iblockproc.BlockCtx{Idx: 1, Time: inter.Timestamp(2000)}

	pre := NewDriverTxPreTransactor()
	txs := pre.PopInternalTxs(block, bs, es, false, statedb)
	// Only the unwritten cheater (v2) should produce a tx.
	require.Len(t, txs, 1)
}

// --- PopInternalTxs: DriverTxTransactor ------------------------------------

func TestDriverTxTransactor_PopInternalTxs_NotSealing(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	vals := testValidators(1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	post := NewDriverTxTransactor()
	txs := post.PopInternalTxs(iblockproc.BlockCtx{}, bs, es, false, statedb)
	require.Empty(t, txs)
}

func TestDriverTxTransactor_PopInternalTxs_Sealing(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)
	statedb.EXPECT().GetNonce(common.Address{}).Return(uint64(0))

	vals := testValidators(1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	post := NewDriverTxTransactor()
	txs := post.PopInternalTxs(iblockproc.BlockCtx{}, bs, es, true, statedb)
	// Sealing produces a SealEpochValidators tx.
	require.Len(t, txs, 1)
}

// --- InternalTxBuilder -----------------------------------------------------

func TestInternalTxBuilder_IncrementsNonce(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)
	statedb.EXPECT().GetNonce(common.Address{}).Return(uint64(10))

	buildTx := InternalTxBuilder(statedb)

	tx1 := buildTx([]byte{0x01}, common.Address{0x01})
	require.Equal(t, uint64(10), tx1.Nonce())

	tx2 := buildTx([]byte{0x02}, common.Address{0x02})
	require.Equal(t, uint64(11), tx2.Nonce())
}

func TestInternalTxBuilder_TransactionProperties(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)
	statedb.EXPECT().GetNonce(common.Address{}).Return(uint64(0))

	buildTx := InternalTxBuilder(statedb)
	addr := common.HexToAddress("0xabcdef")
	calldata := []byte{0xaa, 0xbb, 0xcc}

	tx := buildTx(calldata, addr)
	require.Equal(t, uint64(0), tx.Nonce())
	require.Equal(t, &addr, tx.To())
	require.Equal(t, common.Big0, tx.Value())
	require.Equal(t, uint64(internalTransactionsGasLimit), tx.Gas())
	require.Equal(t, common.Big0, tx.GasPrice())
	require.Equal(t, calldata, tx.Data())
}

// --- maxBlockIdx -----------------------------------------------------------

func TestMaxBlockIdx(t *testing.T) {
	require.Equal(t, idx.Block(5), maxBlockIdx(5, 3))
	require.Equal(t, idx.Block(5), maxBlockIdx(3, 5))
	require.Equal(t, idx.Block(5), maxBlockIdx(5, 5))
	require.Equal(t, idx.Block(0), maxBlockIdx(0, 0))
}

// --- effectiveGasPrice -----------------------------------------------------

func TestEffectiveGasPrice_NilBaseFee(t *testing.T) {
	tx := types.NewTransaction(0, common.Address{}, nil, 21000, big.NewInt(100), nil)
	result := effectiveGasPrice(tx, nil)
	require.Equal(t, big.NewInt(100), result)
}

func TestEffectiveGasPrice_WithBaseFee(t *testing.T) {
	tx := types.NewTx(&types.DynamicFeeTx{
		GasTipCap: big.NewInt(10),
		GasFeeCap: big.NewInt(100),
	})
	baseFee := big.NewInt(50)
	result := effectiveGasPrice(tx, baseFee)
	// effective gas price = baseFee + min(gasTipCap, gasFeeCap - baseFee)
	// = 50 + min(10, 100-50) = 50 + 10 = 60
	require.Equal(t, big.NewInt(60), result)
}

func TestEffectiveGasPrice_TipCappedByFeeCap(t *testing.T) {
	tx := types.NewTx(&types.DynamicFeeTx{
		GasTipCap: big.NewInt(100),
		GasFeeCap: big.NewInt(60),
	})
	baseFee := big.NewInt(50)
	result := effectiveGasPrice(tx, baseFee)
	// effective gas price = baseFee + min(gasTipCap, gasFeeCap - baseFee)
	// = 50 + min(100, 60-50) = 50 + 10 = 60
	require.Equal(t, big.NewInt(60), result)
}

// --- decodeDataBytes -------------------------------------------------------

func TestDecodeDataBytes_Valid(t *testing.T) {
	// Standard ABI encoding: offset at 32, then length + data.
	payload := []byte("hello world")

	offset := make([]byte, 32)
	big.NewInt(32).FillBytes(offset)
	length := make([]byte, 32)
	big.NewInt(int64(len(payload))).FillBytes(length)

	data := append(offset, length...)
	data = append(data, payload...)

	result, err := decodeDataBytes(&types.Log{Data: data})
	require.NoError(t, err)
	require.Equal(t, payload, result)
}

func TestDecodeDataBytes_DataTooShort(t *testing.T) {
	_, err := decodeDataBytes(&types.Log{Data: make([]byte, 31)})
	require.Error(t, err)
}

func TestDecodeDataBytes_OffsetOutOfBounds(t *testing.T) {
	data := make([]byte, 32)
	// Set offset to point beyond the data.
	big.NewInt(100).FillBytes(data[24:32])

	_, err := decodeDataBytes(&types.Log{Data: data})
	require.Error(t, err)
}

func TestDecodeDataBytes_SizeExceedsData(t *testing.T) {
	offset := make([]byte, 32)
	big.NewInt(32).FillBytes(offset)
	length := make([]byte, 32)
	big.NewInt(999).FillBytes(length) // size way too large

	data := append(offset, length...)

	_, err := decodeDataBytes(&types.Log{Data: data})
	require.Error(t, err)
}

// --- OnNewLog: edge cases for data length ----------------------------------

func TestOnNewLog_UpdateValidatorWeight_DataTooShort_Ignored(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	v1 := idx.ValidatorID(1)
	vals := testValidators(v1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)

	topic1 := common.Hash{}
	big.NewInt(int64(v1)).FillBytes(topic1[:])

	l := &types.Log{
		Address: driver.ContractAddress,
		Topics:  []common.Hash{driverpos.Topics.UpdateValidatorWeight, topic1},
		Data:    make([]byte, 31), // < 32, should be ignored
	}
	listener.OnNewLog(l)

	result := listener.Finalize()
	// Original profile should remain unchanged.
	_, exists := result.NextValidatorProfiles[v1]
	require.True(t, exists)
}

func TestOnNewLog_UpdateValidatorWeight_NoTopicForID_Ignored(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	vals := testValidators(1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)

	l := &types.Log{
		Address: driver.ContractAddress,
		Topics:  []common.Hash{driverpos.Topics.UpdateValidatorWeight}, // only 1 topic
		Data:    make([]byte, 32),
	}
	listener.OnNewLog(l)

	// Should not panic; no changes.
	result := listener.Finalize()
	require.NotNil(t, result.NextValidatorProfiles)
}

func TestOnNewLog_AdvanceEpochs_DataTooShort_Ignored(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	vals := testValidators(1)
	bs := testBlockState(vals)
	es := testEpochState(vals)

	m := NewDriverTxListenerModule()
	listener := m.Start(iblockproc.BlockCtx{}, bs, es, statedb)

	l := &types.Log{
		Address: driver.ContractAddress,
		Topics:  []common.Hash{driverpos.Topics.AdvanceEpochs},
		Data:    make([]byte, 31), // < 32, should be ignored
	}
	listener.OnNewLog(l)

	result := listener.Finalize()
	require.Equal(t, idx.Epoch(0), result.AdvanceEpochs)
}

// --- PopInternalTxs: sealing metrics correctness ---------------------------

func TestDriverTxPreTransactor_SealingMetrics_ForgiveDowntime(t *testing.T) {
	// When a validator is within BlockMissedSlack, downtime should be
	// forgiven in the sealing metrics. This test verifies the function
	// doesn't panic and produces the correct number of transactions.
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)
	statedb.EXPECT().GetNonce(common.Address{}).Return(uint64(0))

	v1 := idx.ValidatorID(1)
	vals := testValidators(v1)
	bs := testBlockState(vals)
	es := testEpochState(vals)
	es.Rules.Economy.BlockMissedSlack = 100

	// Validator has been active recently.
	vidx := es.Validators.GetIdx(v1)
	bs.ValidatorStates[vidx].LastBlock = 5
	bs.ValidatorStates[vidx].LastOnlineTime = inter.Timestamp(1500)

	block := iblockproc.BlockCtx{Idx: 10, Time: inter.Timestamp(2000)}

	pre := NewDriverTxPreTransactor()
	txs := pre.PopInternalTxs(block, bs, es, true, statedb)
	require.Len(t, txs, 1) // SealEpoch only
}

// --- InternalTxBuilder: lazy nonce initialization --------------------------

func TestInternalTxBuilder_LazyNonceInit(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)
	// GetNonce should only be called once, on the first buildTx call.
	statedb.EXPECT().GetNonce(common.Address{}).Return(uint64(42)).Times(1)

	buildTx := InternalTxBuilder(statedb)
	tx1 := buildTx(nil, common.Address{})
	require.Equal(t, uint64(42), tx1.Nonce())
	tx2 := buildTx(nil, common.Address{})
	require.Equal(t, uint64(43), tx2.Nonce())
	tx3 := buildTx(nil, common.Address{})
	require.Equal(t, uint64(44), tx3.Nonce())
}

// --- InternalTxBuilder: starts with MaxUint64 sentinel ---------------------

func TestInternalTxBuilder_InitialNonceIsSentinel(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)
	statedb.EXPECT().GetNonce(common.Address{}).Return(uint64(math.MaxUint64))

	buildTx := InternalTxBuilder(statedb)
	tx := buildTx(nil, common.Address{})
	// When statedb returns MaxUint64 as nonce, that's what we get.
	require.Equal(t, uint64(math.MaxUint64), tx.Nonce())
}
