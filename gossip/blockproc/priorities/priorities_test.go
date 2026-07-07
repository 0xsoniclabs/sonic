// Copyright 2026 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package priorities

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// fakeVM is a VirtualMachine returning a canned result/error.
type fakeVM struct {
	result []byte
	err    error
	gotIn  []byte
}

func (f *fakeVM) Call(_, _ common.Address, input []byte, _ uint64, _ *uint256.Int) ([]byte, uint64, error) {
	f.gotIn = input
	return f.result, 0, f.err
}

func enabledUpgrades() opera.Upgrades {
	u := opera.GetBrioUpgrades()
	u.TransactionPriorities = true
	return u
}

func makeTx(t *testing.T) (*types.Transaction, types.Signer) {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	chainConfig := opera.CreateTransientEvmChainConfig(
		opera.FakeNetRules(enabledUpgrades()).NetworkID, nil, 1,
	)
	signer := types.LatestSigner(chainConfig)
	to := common.Address{0xaa}
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		To:    &to,
		Gas:   21000,
		Nonce: 7,
		Value: big.NewInt(5),
		Data:  []byte{0x01, 0x02, 0x03},
	})
	return tx, signer
}

func TestGetPriority_FeatureDisabled_ReturnsZeroWithoutCall(t *testing.T) {
	tx, signer := makeTx(t)
	vm := &fakeVM{err: fmt.Errorf("must not be called")}
	p, err := GetPriority(opera.GetBrioUpgrades(), vm, signer, tx)
	require.NoError(t, err)
	require.False(t, p.IsPrioritized())
	require.Nil(t, vm.gotIn, "EVM must not be called when feature is off")
}

func TestGetPriority_NilTx_ReportsError(t *testing.T) {
	_, err := GetPriority(enabledUpgrades(), &fakeVM{}, types.LatestSigner(opera.CreateTransientEvmChainConfig(1, nil, 1)), nil)
	require.ErrorContains(t, err, "nil transaction")
}

func TestGetPriority_SenderError_ReportsError(t *testing.T) {
	tx := types.NewTx(&types.LegacyTx{})
	signer := types.LatestSigner(opera.CreateTransientEvmChainConfig(1, nil, 1))
	_, err := GetPriority(enabledUpgrades(), &fakeVM{}, signer, tx)
	require.ErrorContains(t, err, "failed to derive sender")
}

func TestGetPriority_CallError_ReportsError(t *testing.T) {
	tx, signer := makeTx(t)
	vm := &fakeVM{err: fmt.Errorf("call failed")}
	_, err := GetPriority(enabledUpgrades(), vm, signer, tx)
	require.ErrorContains(t, err, "EVM call failed")
}

func TestGetPriority_EmptyResult_ReportsMissingContract(t *testing.T) {
	tx, signer := makeTx(t)
	vm := &fakeVM{result: nil}
	_, err := GetPriority(enabledUpgrades(), vm, signer, tx)
	require.ErrorContains(t, err, "registry contract not found")
}

func TestGetPriority_ValidatesResult(t *testing.T) {
	for _, n := range []int{0, 32, 64, 95, 97, 128} {
		tx, signer := makeTx(t)
		vm := &fakeVM{result: make([]byte, n)}
		_, err := GetPriority(enabledUpgrades(), vm, signer, tx)
		require.Error(t, err, "length %d must be rejected", n)
	}
}

func TestGetPriority_DecodesResult(t *testing.T) {
	tx, signer := makeTx(t)
	id := [32]byte{0xde, 0xad}
	result := make([]byte, 96)
	big.NewInt(2).FillBytes(result[0:32])
	big.NewInt(100).FillBytes(result[32:64])
	copy(result[64:96], id[:])

	vm := &fakeVM{result: result}
	p, err := GetPriority(enabledUpgrades(), vm, signer, tx)
	require.NoError(t, err)
	require.True(t, p.IsPrioritized())
	require.Equal(t, int64(2), p.Level.Int64())
	require.Equal(t, int64(100), p.Weight.Int64())
	require.Equal(t, id, p.Id)
}

func TestGetPriority_EncodesExpectedCalldata(t *testing.T) {
	tx, signer := makeTx(t)
	vm := &fakeVM{result: make([]byte, 96)}
	_, err := GetPriority(enabledUpgrades(), vm, signer, tx)
	require.NoError(t, err)

	zero12 := make([]byte, 12)
	zero24 := make([]byte, 24)

	in := vm.gotIn
	// selector + 6 head words + (len word + padded data)
	require.Equal(t, 4+6*32+32+32, len(in))
	require.Equal(t, uint32(registry.GetPriorityFunctionSelector), binary.BigEndian.Uint32(in[0:4]))
	// from address: 12-byte high padding (address itself is the low 20 bytes)
	require.Equal(t, zero12, in[4:4+12])
	from, err := signer.Sender(tx)
	require.NoError(t, err)
	require.Equal(t, from.Bytes(), in[4+12:4+32])
	// to address: 12-byte high padding
	require.Equal(t, zero12, in[4+32:4+32+12])
	require.Equal(t, tx.To().Bytes(), in[4+32+12:4+64])
	// nonce is the 4th head word: 24-byte high padding + 8-byte value
	require.Equal(t, zero24, in[4+3*32:4+3*32+24])
	require.Equal(t, uint64(7), binary.BigEndian.Uint64(in[4+3*32+24:4+4*32]))
	// data offset (5th head word) = 6*32
	require.Equal(t, zero24, in[4+4*32:4+4*32+24])
	require.Equal(t, uint64(6*32), binary.BigEndian.Uint64(in[4+4*32+24:4+5*32]))
	// gas (6th head word)
	require.Equal(t, zero24, in[4+5*32:4+5*32+24])
	require.Equal(t, uint64(21000), binary.BigEndian.Uint64(in[4+5*32+24:4+6*32]))
	// dynamic data length
	require.Equal(t, zero24, in[4+6*32:4+6*32+24])
	require.Equal(t, uint64(3), binary.BigEndian.Uint64(in[4+6*32+24:4+7*32]))
	// dynamic data (3 bytes) padded to 32 bytes: 29-byte trailing padding
	require.Equal(t, make([]byte, 29), in[4+7*32+3:4+8*32])
}

func TestGetConfig_FeatureDisabled_ReturnsZero(t *testing.T) {
	vm := &fakeVM{err: fmt.Errorf("must not be called")}
	cfg, err := GetConfig(opera.GetBrioUpgrades(), vm)
	require.NoError(t, err)
	require.Equal(t, Config{}, cfg)
	require.Nil(t, vm.gotIn)
}

func TestGetConfig_CallError_ReportsError(t *testing.T) {
	vm := &fakeVM{err: fmt.Errorf("call failed")}
	_, err := GetConfig(enabledUpgrades(), vm)
	require.ErrorContains(t, err, "EVM call failed")
}

func TestGetConfig_EmptyResult_ReportsMissingContract(t *testing.T) {
	vm := &fakeVM{result: nil}
	_, err := GetConfig(enabledUpgrades(), vm)
	require.ErrorContains(t, err, "registry contract not found")
}

func TestGetConfig_ValidatesResult(t *testing.T) {
	data := make([]byte, 64)
	binary.BigEndian.PutUint64(data[24:32], 16)
	binary.BigEndian.PutUint64(data[56:64], 4)
	vm := &fakeVM{result: data}
	cfg, err := GetConfig(enabledUpgrades(), vm)
	require.NoError(t, err)
	require.Equal(t, uint64(16), cfg.MaxTxsPerEntityPerBlock)
	require.Equal(t, uint64(4), cfg.MaxTxsPerEntityPerEvent)

	// overflow: a non-zero high byte must be rejected.
	data[0] = 1
	vm.result = data
	_, err = GetConfig(enabledUpgrades(), vm)
	require.ErrorContains(t, err, "do not fit into uint64")

	// wrong length
	vm.result = make([]byte, 96)
	_, err = GetConfig(enabledUpgrades(), vm)
	require.Error(t, err)
}
func TestGetConfig_DecodesResult(t *testing.T) {
	vm := &fakeVM{}

	// a non-zero high byte
	data := make([]byte, 64)
	data[0] = 1
	vm.result = data
	_, err := GetConfig(enabledUpgrades(), vm)
	require.ErrorContains(t, err, "do not fit into uint64")

	// wrong length
	for _, n := range []int{0, 32, 63, 65, 96} {
		vm := &fakeVM{result: make([]byte, n)}
		_, err = GetConfig(enabledUpgrades(), vm)
		require.Error(t, err)
	}
}

func TestGetConfig_EncodesExpectedCalldata(t *testing.T) {
	vm := &fakeVM{result: make([]byte, 64)}
	_, err := GetConfig(enabledUpgrades(), vm)
	require.NoError(t, err)

	in := vm.gotIn
	require.Equal(t, 4, len(in))
	require.Equal(t, uint32(registry.GetPriorityConfigFunctionSelector), binary.BigEndian.Uint32(in[0:4]))
}

// TestGetPriority_AgainstRealBytecode validates the hand-rolled ABI encoding and
// the function selectors against the actually-compiled registry bytecode, run on
// a real EVM over a mocked state. With empty storage the contract returns a
// non-prioritized result and the default config (16, 4).
func TestGetPriorityAndGetConfig_AgainstRealBytecode(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	st := state.NewMockStateDB(ctrl)

	registryAddress := registry.GetAddress()
	code := registry.GetCode()
	hash := crypto.Keccak256Hash(code)

	any := gomock.Any()
	st.EXPECT().Snapshot().Return(1).AnyTimes()
	st.EXPECT().Exist(registryAddress).Return(true).AnyTimes()
	st.EXPECT().GetCode(registryAddress).Return(code).AnyTimes()
	st.EXPECT().GetCodeHash(registryAddress).Return(hash).AnyTimes()
	st.EXPECT().AddRefund(any).AnyTimes()
	st.EXPECT().SubRefund(any).AnyTimes()
	st.EXPECT().GetRefund().Return(uint64(0)).AnyTimes()
	st.EXPECT().SlotInAccessList(any, any).AnyTimes()
	st.EXPECT().AddSlotToAccessList(any, any).AnyTimes()
	st.EXPECT().GetState(any, any).Return(common.Hash{}).AnyTimes()

	upgrades := enabledUpgrades()
	rules := opera.FakeNetRules(upgrades)
	chainConfig := opera.CreateTransientEvmChainConfig(rules.NetworkID, nil, 1)

	key, err := crypto.GenerateKey()
	require.NoError(err)
	signer := types.LatestSigner(chainConfig)
	to := common.Address{0xaa}
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{To: &to, Gas: 21000})

	blockContext := vm.BlockContext{
		BlockNumber: big.NewInt(123),
		BaseFee:     big.NewInt(1),
		Transfer: func(_ vm.StateDB, _ common.Address, _ common.Address, amount *uint256.Int, _ *params.Rules) {
			require.Equal(0, amount.Sign())
		},
		Random: &common.Hash{},
	}
	evm := vm.NewEVM(blockContext, st, chainConfig, opera.GetVmConfig(rules))

	p, err := GetPriority(upgrades, evm, signer, tx)
	require.NoError(err)
	require.False(p.IsPrioritized())

	cfg, err := GetConfig(upgrades, evm)
	require.NoError(err)
	require.Equal(uint64(16), cfg.MaxTxsPerEntityPerBlock)
	require.Equal(uint64(4), cfg.MaxTxsPerEntityPerEvent)
}
