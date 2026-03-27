package rpctest

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

var _ state.StateDB = testState{}

var (
	addr1 = common.HexToAddress("0x01")
	addr2 = common.HexToAddress("0x02")
	key1  = common.HexToHash("0xaa")
	key2  = common.HexToHash("0xbb")
	val1  = common.HexToHash("0xcc")
	val2  = common.HexToHash("0xdd")
)

func TestGetBalance_ZeroForMissingAccount(t *testing.T) {
	t.Parallel()
	s := newTestState()
	got := s.GetBalance(addr1)
	require.Zero(t, got.Sign(), "expected zero balance")
}

func TestSetAndGetBalance(t *testing.T) {
	t.Parallel()
	s := newTestState()
	amount := uint256.NewInt(42)
	s.SetBalance(addr1, amount)
	got := s.GetBalance(addr1)
	require.Zero(t, got.Cmp(amount), "expected %v, got %v", amount, got)
}

func TestAddBalance(t *testing.T) {
	t.Parallel()
	s := newTestState()
	s.SetBalance(addr1, uint256.NewInt(10))

	prev := s.AddBalance(addr1, uint256.NewInt(5), tracing.BalanceChangeUnspecified)
	require.Zero(t, prev.Cmp(uint256.NewInt(10)), "expected previous balance 10, got %v", &prev)
	got := s.GetBalance(addr1)
	require.Zero(t, got.Cmp(uint256.NewInt(15)), "expected 15, got %v", got)
}

func TestAddBalance_NewAccount(t *testing.T) {
	t.Parallel()
	s := newTestState()
	prev := s.AddBalance(addr1, uint256.NewInt(7), tracing.BalanceChangeUnspecified)
	require.Zero(t, prev.Sign(), "expected zero previous balance, got %v", &prev)
	got := s.GetBalance(addr1)
	require.Zero(t, got.Cmp(uint256.NewInt(7)), "expected 7, got %v", got)
}

func TestSubBalance(t *testing.T) {
	t.Parallel()
	s := newTestState()
	s.SetBalance(addr1, uint256.NewInt(10))

	prev := s.SubBalance(addr1, uint256.NewInt(3), tracing.BalanceChangeUnspecified)
	require.Zero(t, prev.Cmp(uint256.NewInt(10)), "expected previous balance 10, got %v", &prev)
	got := s.GetBalance(addr1)
	require.Zero(t, got.Cmp(uint256.NewInt(7)), "expected 7, got %v", got)
}

func TestGetNonce_ZeroForMissingAccount(t *testing.T) {
	t.Parallel()
	s := newTestState()
	require.Zero(t, s.GetNonce(addr1), "expected 0 nonce")
}

func TestSetAndGetNonce(t *testing.T) {
	t.Parallel()
	s := newTestState()
	s.SetNonce(addr1, 42, tracing.NonceChangeUnspecified)
	require.Equal(t, uint64(42), s.GetNonce(addr1), "expected 42 nonce")
}

func TestGetCode_NilForMissingAccount(t *testing.T) {
	t.Parallel()
	s := newTestState()
	require.Nil(t, s.GetCode(addr1), "expected nil code")
}

func TestSetAndGetCode(t *testing.T) {
	t.Parallel()
	s := newTestState()
	code := []byte{0x60, 0x00}
	s.SetCode(addr1, code, tracing.CodeChangeUnspecified)
	got := s.GetCode(addr1)
	require.Equal(t, code, got, "expected code to match")
}

func TestSetCode_ReturnsPrevious(t *testing.T) {
	t.Parallel()
	s := newTestState()
	old := []byte{0x01}
	s.SetCode(addr1, old, tracing.CodeChangeUnspecified)
	prev := s.SetCode(addr1, []byte{0x02}, tracing.CodeChangeUnspecified)
	require.Equal(t, []byte{0x01}, prev, "expected previous code [0x01]")
}

func TestGetCodeHash_EmptyForMissingAccount(t *testing.T) {
	t.Parallel()
	s := newTestState()
	require.Equal(t, common.Hash{}, s.GetCodeHash(addr1), "expected empty hash")
}

func TestGetCodeHash_ExistingAccount(t *testing.T) {
	t.Parallel()
	s := newTestState()
	code := []byte{0x60, 0x00}
	s.SetCode(addr1, code, tracing.CodeChangeUnspecified)
	want := crypto.Keccak256Hash(code)
	require.Equal(t, want, s.GetCodeHash(addr1), "expected code hash to match")
}

func TestGetCodeSize(t *testing.T) {
	t.Parallel()
	s := newTestState()
	require.Zero(t, s.GetCodeSize(addr1), "expected 0 code size")
	s.SetCode(addr1, []byte{0x01, 0x02, 0x03}, tracing.CodeChangeUnspecified)
	require.Equal(t, 3, s.GetCodeSize(addr1), "expected code size 3")
}

func TestSetAndGetState(t *testing.T) {
	t.Parallel()
	s := newTestState()
	s.SetState(addr1, key1, val1)
	require.Equal(t, val1, s.GetState(addr1, key1), "expected value to match")
}

func TestSetState_ReturnsPrevious(t *testing.T) {
	t.Parallel()
	s := newTestState()
	s.SetState(addr1, key1, val1)
	prev := s.SetState(addr1, key1, val2)
	require.Equal(t, val1, prev, "expected previous value to match")
}

func TestGetState_ZeroForMissing(t *testing.T) {
	t.Parallel()
	s := newTestState()
	require.Equal(t, common.Hash{}, s.GetState(addr1, key1), "expected zero hash")
}

func TestGetStateAndCommittedState(t *testing.T) {
	t.Parallel()
	s := newTestState()
	s.SetState(addr1, key1, val1)
	current, committed := s.GetStateAndCommittedState(addr1, key1)
	require.Equal(t, val1, current, "expected current value to match")
	require.Equal(t, val1, committed, "expected committed value to match")
}

func TestSetStorage(t *testing.T) {
	t.Parallel()
	s := newTestState()
	storage := map[common.Hash]common.Hash{key1: val1, key2: val2}
	s.SetStorage(addr1, storage)
	require.Equal(t, val1, s.GetState(addr1, key1), "expected value for key1")
	require.Equal(t, val2, s.GetState(addr1, key2), "expected value for key2")
}

func TestExist(t *testing.T) {
	t.Parallel()
	s := newTestState()
	require.False(t, s.Exist(addr1), "expected non-existent")
	s.CreateAccount(addr1)
	require.True(t, s.Exist(addr1), "expected existent")
}

func TestEmpty(t *testing.T) {
	t.Parallel()
	s := newTestState()
	require.True(t, s.Empty(addr1), "expected empty for missing account")
	s.CreateAccount(addr1)
	require.True(t, s.Empty(addr1), "expected empty for fresh account")
	s.SetNonce(addr1, 1, tracing.NonceChangeUnspecified)
	require.False(t, s.Empty(addr1), "expected non-empty with nonce > 0")
}

func TestEmpty_WithBalance(t *testing.T) {
	t.Parallel()
	s := newTestState()
	s.SetBalance(addr1, uint256.NewInt(1))
	require.False(t, s.Empty(addr1), "expected non-empty with positive balance")
}

func TestEmpty_WithCode(t *testing.T) {
	t.Parallel()
	s := newTestState()
	s.SetCode(addr1, []byte{0x01}, tracing.CodeChangeUnspecified)
	require.False(t, s.Empty(addr1), "expected non-empty with code")
}

func TestCreateAccount(t *testing.T) {
	t.Parallel()
	s := newTestState()
	s.CreateAccount(addr1)
	require.True(t, s.Exist(addr1), "expected account to exist")
	require.True(t, s.Empty(addr1), "expected fresh account to be empty")
}

func TestCreateContract(t *testing.T) {
	t.Parallel()
	s := newTestState()
	s.CreateContract(addr1)
	require.True(t, s.Exist(addr1), "expected contract to exist")
}

func TestError_ReturnsNil(t *testing.T) {
	t.Parallel()
	s := newTestState()
	require.NoError(t, s.Error(), "expected nil error")
}

func TestCopy(t *testing.T) {
	t.Parallel()
	s := newTestState()
	s.state[addr1] = TestAccount{
		Nonce:   5,
		Balance: big.NewInt(100),
		Code:    []byte{0x01},
		Store:   map[common.Hash]common.Hash{key1: val1},
	}

	cp := s.Copy().(testState)

	require.Equal(t, uint64(5), cp.GetNonce(addr1), "expected nonce 5")
	require.Zero(t, cp.GetBalance(addr1).Cmp(uint256.NewInt(100)), "expected balance 100")
	require.Equal(t, val1, cp.GetState(addr1, key1), "expected state value to match")

	cp.SetNonce(addr1, 99, tracing.NonceChangeUnspecified)
	require.Equal(t, uint64(5), s.GetNonce(addr1), "copy mutation affected original")
}

func TestSelfDestruct(t *testing.T) {
	t.Parallel()
	s := newTestState()
	s.SetBalance(addr1, uint256.NewInt(50))

	prev := s.SelfDestruct(addr1)
	require.Zero(t, prev.Cmp(uint256.NewInt(50)), "expected previous balance 50")
	require.False(t, s.Exist(addr1), "expected account to be removed")
}

func TestSelfDestruct6780(t *testing.T) {
	t.Parallel()
	s := newTestState()
	s.SetBalance(addr1, uint256.NewInt(25))

	prev, ok := s.SelfDestruct6780(addr1)
	require.Zero(t, prev.Cmp(uint256.NewInt(25)), "expected previous balance 25")
	require.True(t, ok, "expected ok to be true")
	require.False(t, s.Exist(addr1), "expected account to be removed")
}

func TestEndBlock_ChannelCloses(t *testing.T) {
	t.Parallel()
	s := newTestState()
	ch := s.EndBlock(0)
	_, open := <-ch
	require.False(t, open, "expected closed channel")
}
