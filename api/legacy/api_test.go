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

package legacy

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
)

const testPassword = "password"

// wallet is a node with one keystore account and a mocked backend around it.
type wallet struct {
	backend *MockBackend
	manager *accounts.Manager
	ks      *keystore.KeyStore
	account accounts.Account
	chainID *big.Int
	sent    []*types.Transaction
}

func newWallet(t *testing.T) *wallet {
	t.Helper()
	ks := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
	// The manager learns the accounts of a keystore when it is created, and the
	// accounts that come later only after a while.
	account, err := ks.NewAccount(testPassword)
	require.NoError(t, err)
	manager := accounts.NewManager(&accounts.Config{InsecureUnlockAllowed: true}, ks)
	t.Cleanup(func() { _ = manager.Close() })

	w := &wallet{manager: manager, ks: ks, account: account, chainID: big.NewInt(146)}
	ctrl := gomock.NewController(t)
	w.backend = NewMockBackend(ctrl)
	block := &evmcore.EvmBlock{}
	block.Number = big.NewInt(1)
	b := w.backend
	b.EXPECT().AccountManager().Return(manager).AnyTimes()
	b.EXPECT().CurrentBlock().Return(block).AnyTimes()
	chainConfig := *params.AllEthashProtocolChanges
	chainConfig.ChainID = w.chainID
	b.EXPECT().ChainConfig(gomock.Any()).Return(&chainConfig).AnyTimes()
	b.EXPECT().ChainID().Return(w.chainID).AnyTimes()
	b.EXPECT().Progress().Return(PeerProgress{CurrentBlock: 1}).AnyTimes()
	b.EXPECT().MinGasPrice().Return(big.NewInt(100)).AnyTimes()
	b.EXPECT().SuggestGasTipCap(gomock.Any(), gomock.Any()).DoAndReturn(
		func(context.Context, uint64) *big.Int { return big.NewInt(7) }).AnyTimes()
	b.EXPECT().GetPoolNonce(gomock.Any(), gomock.Any()).Return(uint64(3), nil).AnyTimes()
	b.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(hexutil.Uint64(21000), nil).AnyTimes()
	b.EXPECT().RPCTxFeeCap().Return(float64(1)).AnyTimes()
	b.EXPECT().UnprotectedAllowed().Return(false).AnyTimes()
	b.EXPECT().ExtRPCEnabled().Return(false).AnyTimes()
	b.EXPECT().SendTx(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, tx *types.Transaction) error {
		w.sent = append(w.sent, tx)
		return nil
	}).AnyTimes()
	return w
}

func (w *wallet) unlock(t *testing.T) {
	t.Helper()
	require.NoError(t, w.ks.Unlock(w.account, testPassword))
}

func (w *wallet) args() TransactionArgs {
	to := common.Address{0x42}
	return TransactionArgs{
		From:  &w.account.Address,
		To:    &to,
		Value: (*hexutil.Big)(big.NewInt(5)),
	}
}

func TestPublicAccountAPI_ListsTheAccountsOfTheNode(t *testing.T) {
	w := newWallet(t)
	require.Equal(t, []common.Address{w.account.Address}, NewPublicAccountAPI(w.manager).Accounts())
	require.Equal(t, []common.Address{w.account.Address}, NewPrivateAccountAPI(w.backend, new(AddrLocker)).ListAccounts())
}

func TestPrivateAccountAPI_ManagesTheKeystore(t *testing.T) {
	w := newWallet(t)
	api := NewPrivateAccountAPI(w.backend, new(AddrLocker))

	created, err := api.NewAccount(testPassword)
	require.NoError(t, err)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	imported, err := api.ImportRawKey(common.Bytes2Hex(crypto.FromECDSA(key)), testPassword)
	require.NoError(t, err)
	require.Equal(t, crypto.PubkeyToAddress(key.PublicKey), imported)

	_, err = api.ImportRawKey("not a key", testPassword)
	require.Error(t, err)

	require.Eventually(t, func() bool {
		return len(api.ListAccounts()) == 3
	}, 10*time.Second, 10*time.Millisecond)
	require.ElementsMatch(t, []common.Address{w.account.Address, created, imported}, api.ListAccounts())

	ok, err := api.UnlockAccount(t.Context(), created, "wrong", nil)
	require.Error(t, err)
	require.False(t, ok)

	ok, err = api.UnlockAccount(t.Context(), created, testPassword, nil)
	require.NoError(t, err)
	require.True(t, ok)
	require.True(t, api.LockAccount(created))

	tooLong := uint64(1) << 62
	_, err = api.UnlockAccount(t.Context(), created, testPassword, &tooLong)
	require.ErrorContains(t, err, "unlock duration too large")
}

func TestPrivateAccountAPI_UnlockIsRefusedWhenRPCIsExposed(t *testing.T) {
	w := newWallet(t)
	ctrl := gomock.NewController(t)
	exposed := NewMockBackend(ctrl)
	exposed.EXPECT().AccountManager().Return(w.manager).AnyTimes()
	exposed.EXPECT().ExtRPCEnabled().Return(true).AnyTimes()

	ok, err := NewPrivateAccountAPI(exposed, new(AddrLocker)).UnlockAccount(t.Context(), w.account.Address, testPassword, nil)
	require.ErrorContains(t, err, "account unlock with HTTP access is forbidden")
	require.False(t, ok)
}

func TestPrivateAccountAPI_SignsAndRecovers(t *testing.T) {
	w := newWallet(t)
	api := NewPrivateAccountAPI(w.backend, new(AddrLocker))
	message := hexutil.Bytes("hello")

	signature, err := api.Sign(t.Context(), message, w.account.Address, testPassword)
	require.NoError(t, err)
	require.Len(t, signature, crypto.SignatureLength)
	require.Contains(t, []byte{27, 28}, signature[crypto.RecoveryIDOffset])

	recovered, err := api.EcRecover(t.Context(), message, signature)
	require.NoError(t, err)
	require.Equal(t, w.account.Address, recovered)

	_, err = api.Sign(t.Context(), message, w.account.Address, "wrong")
	require.Error(t, err)
	_, err = api.Sign(t.Context(), message, common.Address{0xff}, testPassword)
	require.Error(t, err)

	_, err = api.EcRecover(t.Context(), message, signature[:10])
	require.ErrorContains(t, err, "signature must be 65 bytes long")
	bad := append(hexutil.Bytes(nil), signature...)
	bad[crypto.RecoveryIDOffset] = 5
	_, err = api.EcRecover(t.Context(), message, bad)
	require.ErrorContains(t, err, "V is not 27 or 28")
}

func TestPrivateAccountAPI_WalletsThatDoNotExistAreRefused(t *testing.T) {
	w := newWallet(t)
	api := NewPrivateAccountAPI(w.backend, new(AddrLocker))

	wallets := api.ListWallets()
	require.Len(t, wallets, 1)
	require.NotEmpty(t, wallets[0].URL)

	require.Error(t, api.OpenWallet("keystore://nowhere", nil))
	_, err := api.DeriveAccount("keystore://nowhere", "m/44'/60'/0'/0", nil)
	require.Error(t, err)
	_, err = api.InitializeWallet(t.Context(), "keystore://nowhere")
	require.Error(t, err)
	require.Error(t, api.Unpair(t.Context(), "keystore://nowhere", "1234"))

	// A keystore wallet opens, but derives nothing and cannot be paired.
	url := wallets[0].URL
	require.NoError(t, api.OpenWallet(url, nil))
	_, err = api.DeriveAccount(url, "not a path", nil)
	require.Error(t, err)
	_, err = api.InitializeWallet(t.Context(), url)
	require.ErrorContains(t, err, "does not support initialization")
	require.ErrorContains(t, api.Unpair(t.Context(), url, "1234"), "does not support pairing")
}

func TestPrivateAccountAPI_SendTransactionSignsWithThePassword(t *testing.T) {
	w := newWallet(t)
	api := NewPrivateAccountAPI(w.backend, new(AddrLocker))

	hash, err := api.SendTransaction(t.Context(), w.args(), testPassword)
	require.NoError(t, err)
	require.Len(t, w.sent, 1)
	require.Equal(t, hash, w.sent[0].Hash())
	sender, err := types.Sender(types.LatestSignerForChainID(w.chainID), w.sent[0])
	require.NoError(t, err)
	require.Equal(t, w.account.Address, sender)
	require.EqualValues(t, 3, w.sent[0].Nonce(), "the nonce is the one of the pool")
	require.EqualValues(t, 21000, w.sent[0].Gas(), "the gas is estimated")

	// The deprecated name does the same.
	_, err = api.SignAndSendTransaction(t.Context(), w.args(), testPassword)
	require.NoError(t, err)
	require.Len(t, w.sent, 2)

	_, err = api.SendTransaction(t.Context(), w.args(), "wrong")
	require.Error(t, err)
	unknown := w.args()
	unknown.From = &common.Address{0xff}
	_, err = api.SendTransaction(t.Context(), unknown, testPassword)
	require.Error(t, err)
	require.Len(t, w.sent, 2)
}

func TestPrivateAccountAPI_SignTransactionNeedsAFullyDefinedTransaction(t *testing.T) {
	w := newWallet(t)
	api := NewPrivateAccountAPI(w.backend, new(AddrLocker))

	full := w.args()
	gas := hexutil.Uint64(21000)
	nonce := hexutil.Uint64(9)
	full.Gas, full.Nonce = &gas, &nonce
	full.GasPrice = (*hexutil.Big)(big.NewInt(100))

	for name, mutate := range map[string]func(*TransactionArgs){
		"sender":   func(a *TransactionArgs) { a.From = nil },
		"gas":      func(a *TransactionArgs) { a.Gas = nil },
		"gasPrice": func(a *TransactionArgs) { a.GasPrice = nil },
		"nonce":    func(a *TransactionArgs) { a.Nonce = nil },
	} {
		args := full
		mutate(&args)
		_, err := api.SignTransaction(t.Context(), args, testPassword)
		require.Error(t, err, name)
	}

	signed, err := api.SignTransaction(t.Context(), full, testPassword)
	require.NoError(t, err)
	require.NotNil(t, signed.Tx)
	require.EqualValues(t, 9, signed.Tx.Nonce())
	decoded := new(types.Transaction)
	require.NoError(t, decoded.UnmarshalBinary(signed.Raw))
	require.Equal(t, signed.Tx.Hash(), decoded.Hash())
	require.Empty(t, w.sent, "a signed transaction is not sent")

	expensive := full
	expensive.GasPrice = (*hexutil.Big)(new(big.Int).Mul(big.NewInt(params.Ether), big.NewInt(1000)))
	_, err = api.SignTransaction(t.Context(), expensive, testPassword)
	require.ErrorContains(t, err, "exceeds the configured cap")
}

func TestPublicTransactionPoolAPI_SendTransactionNeedsAnUnlockedAccount(t *testing.T) {
	w := newWallet(t)
	api := NewPublicTransactionPoolAPI(w.backend, new(AddrLocker))

	_, err := api.SendTransaction(t.Context(), w.args())
	require.ErrorIs(t, err, keystore.ErrLocked)

	w.unlock(t)
	hash, err := api.SendTransaction(t.Context(), w.args())
	require.NoError(t, err)
	require.Len(t, w.sent, 1)
	require.Equal(t, hash, w.sent[0].Hash())

	unknown := w.args()
	unknown.From = &common.Address{0xff}
	_, err = api.SendTransaction(t.Context(), unknown)
	require.Error(t, err)

	nonce := hexutil.Uint64(5)
	withNonce := w.args()
	withNonce.Nonce = &nonce
	_, err = api.SendTransaction(t.Context(), withNonce)
	require.NoError(t, err)
	require.EqualValues(t, 5, w.sent[1].Nonce())
}

func TestPublicTransactionPoolAPI_FillTransactionSetsTheDefaults(t *testing.T) {
	w := newWallet(t)
	api := NewPublicTransactionPoolAPI(w.backend, new(AddrLocker))

	filled, err := api.FillTransaction(t.Context(), w.args())
	require.NoError(t, err)
	tx := filled.Tx
	require.EqualValues(t, 3, tx.Nonce())
	require.EqualValues(t, 21000, tx.Gas())
	require.EqualValues(t, 7, tx.GasTipCap().Int64(), "the tip is the suggested one")
	require.EqualValues(t, 107, tx.GasFeeCap().Int64(), "the fee cap is the tip and the minimal price")
	require.Equal(t, w.chainID, tx.ChainId())
	decoded := new(types.Transaction)
	require.NoError(t, decoded.UnmarshalBinary(filled.Raw))
	require.Equal(t, tx.Hash(), decoded.Hash())

	legacy := w.args()
	legacy.GasPrice = (*hexutil.Big)(big.NewInt(500))
	filled, err = api.FillTransaction(t.Context(), legacy)
	require.NoError(t, err)
	require.Equal(t, uint8(types.LegacyTxType), filled.Tx.Type())

	both := w.args()
	both.GasPrice = (*hexutil.Big)(big.NewInt(500))
	both.MaxFeePerGas = (*hexutil.Big)(big.NewInt(500))
	_, err = api.FillTransaction(t.Context(), both)
	require.ErrorContains(t, err, "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")

	inverted := w.args()
	inverted.MaxPriorityFeePerGas = (*hexutil.Big)(big.NewInt(50))
	inverted.MaxFeePerGas = (*hexutil.Big)(big.NewInt(10))
	_, err = api.FillTransaction(t.Context(), inverted)
	require.ErrorContains(t, err, "maxFeePerGas (0xa) < maxPriorityFeePerGas (0x32)")

	create := w.args()
	create.To = nil
	_, err = api.FillTransaction(t.Context(), create)
	require.ErrorContains(t, err, "contract creation without any data provided")

	data := hexutil.Bytes{1}
	conflict := w.args()
	other := hexutil.Bytes{2}
	conflict.Data, conflict.Input = &data, &other
	_, err = api.FillTransaction(t.Context(), conflict)
	require.ErrorContains(t, err, `both "data" and "input" are set and not equal`)
}

func TestPublicTransactionPoolAPI_FillTransactionReportsTheFailureOfTheEstimate(t *testing.T) {
	w := newWallet(t)
	ctrl := gomock.NewController(t)
	b := NewMockBackend(ctrl)
	block := &evmcore.EvmBlock{}
	block.Number = big.NewInt(1)
	b.EXPECT().CurrentBlock().Return(block).AnyTimes()
	b.EXPECT().ChainConfig(gomock.Any()).Return(params.AllEthashProtocolChanges).AnyTimes()
	b.EXPECT().ChainID().Return(w.chainID).AnyTimes()
	b.EXPECT().MinGasPrice().Return(big.NewInt(100)).AnyTimes()
	b.EXPECT().SuggestGasTipCap(gomock.Any(), gomock.Any()).Return(big.NewInt(7)).AnyTimes()
	b.EXPECT().GetPoolNonce(gomock.Any(), gomock.Any()).Return(uint64(0), nil).AnyTimes()
	b.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(hexutil.Uint64(0), errors.New("out of gas"))

	_, err := NewPublicTransactionPoolAPI(b, new(AddrLocker)).FillTransaction(t.Context(), w.args())
	require.ErrorContains(t, err, "out of gas")
}

func TestPublicTransactionPoolAPI_SignsTextAndTransactions(t *testing.T) {
	w := newWallet(t)
	api := NewPublicTransactionPoolAPI(w.backend, new(AddrLocker))
	data := hexutil.Bytes("hello")

	_, err := api.Sign(w.account.Address, data)
	require.ErrorIs(t, err, keystore.ErrLocked)
	w.unlock(t)
	signature, err := api.Sign(w.account.Address, data)
	require.NoError(t, err)
	recovered, err := NewPrivateAccountAPI(w.backend, new(AddrLocker)).EcRecover(t.Context(), data, signature)
	require.NoError(t, err)
	require.Equal(t, w.account.Address, recovered)
	_, err = api.Sign(common.Address{0xff}, data)
	require.Error(t, err)

	full := w.args()
	gas, nonce := hexutil.Uint64(21000), hexutil.Uint64(9)
	full.Gas, full.Nonce = &gas, &nonce
	full.GasPrice = (*hexutil.Big)(big.NewInt(100))
	signed, err := api.SignTransaction(t.Context(), full)
	require.NoError(t, err)
	sender, err := types.Sender(types.LatestSignerForChainID(w.chainID), signed.Tx)
	require.NoError(t, err)
	require.Equal(t, w.account.Address, sender)

	for name, mutate := range map[string]func(*TransactionArgs){
		"gas":      func(a *TransactionArgs) { a.Gas = nil },
		"gasPrice": func(a *TransactionArgs) { a.GasPrice = nil },
		"nonce":    func(a *TransactionArgs) { a.Nonce = nil },
	} {
		args := full
		mutate(&args)
		_, err := api.SignTransaction(t.Context(), args)
		require.Error(t, err, name)
	}

	expensive := full
	expensive.GasPrice = (*hexutil.Big)(new(big.Int).Mul(big.NewInt(params.Ether), big.NewInt(1000)))
	_, err = api.SignTransaction(t.Context(), expensive)
	require.ErrorContains(t, err, "exceeds the configured cap")

	unknown := full
	unknown.From = &common.Address{0xff}
	_, err = api.SignTransaction(t.Context(), unknown)
	require.Error(t, err)
}

func TestPublicTransactionPoolAPI_ResendReplacesAPendingTransaction(t *testing.T) {
	w := newWallet(t)
	w.unlock(t)
	api := NewPublicTransactionPoolAPI(w.backend, new(AddrLocker))

	nonce, gas := hexutil.Uint64(4), hexutil.Uint64(21000)
	args := w.args()
	args.Nonce, args.Gas = &nonce, &gas
	args.GasPrice = (*hexutil.Big)(big.NewInt(100))
	pending, err := api.SignTransaction(t.Context(), args)
	require.NoError(t, err)
	w.backend.EXPECT().GetPoolTransactions().Return(types.Transactions{pending.Tx}, nil).AnyTimes()

	newPrice := (*hexutil.Big)(big.NewInt(300))
	hash, err := api.Resend(t.Context(), args, newPrice, nil)
	require.NoError(t, err)
	require.Len(t, w.sent, 1)
	require.Equal(t, hash, w.sent[0].Hash())
	require.EqualValues(t, 300, w.sent[0].GasPrice().Int64())
	require.EqualValues(t, 4, w.sent[0].Nonce())

	newGas := hexutil.Uint64(30000)
	_, err = api.Resend(t.Context(), args, nil, &newGas)
	require.NoError(t, err)
	require.EqualValues(t, 30000, w.sent[1].Gas())

	missing := args
	missing.Nonce = nil
	_, err = api.Resend(t.Context(), missing, nil, nil)
	require.ErrorContains(t, err, "missing transaction nonce")

	other := args
	otherNonce := hexutil.Uint64(5)
	other.Nonce = &otherNonce
	_, err = api.Resend(t.Context(), other, nil, nil)
	require.ErrorContains(t, err, "not found")

	expensive := (*hexutil.Big)(new(big.Int).Mul(big.NewInt(params.Ether), big.NewInt(1000)))
	_, err = api.Resend(t.Context(), args, expensive, nil)
	require.ErrorContains(t, err, "exceeds the configured cap")
}

func TestPublicTransactionPoolAPI_ResendReportsAFailingPool(t *testing.T) {
	w := newWallet(t)
	w.backend.EXPECT().GetPoolTransactions().Return(nil, errors.New("pool is down"))
	nonce := hexutil.Uint64(4)
	args := w.args()
	args.Nonce = &nonce
	_, err := NewPublicTransactionPoolAPI(w.backend, new(AddrLocker)).Resend(t.Context(), args, nil, nil)
	require.ErrorContains(t, err, "pool is down")
}

func TestSubmitTransaction(t *testing.T) {
	w := newWallet(t)
	w.unlock(t)
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	protected := types.MustSignNewTx(key, types.LatestSignerForChainID(w.chainID),
		&types.LegacyTx{Nonce: 1, GasPrice: big.NewInt(100), Gas: 21000, To: &common.Address{1}})
	hash, err := SubmitTransaction(t.Context(), w.backend, protected)
	require.NoError(t, err)
	require.Equal(t, protected.Hash(), hash)

	unprotected := types.MustSignNewTx(key, types.HomesteadSigner{},
		&types.LegacyTx{Nonce: 1, GasPrice: big.NewInt(100), Gas: 21000, To: &common.Address{1}})
	_, err = SubmitTransaction(t.Context(), w.backend, unprotected)
	require.ErrorContains(t, err, "only replay-protected (EIP-155) transactions allowed over RPC")

	contract := types.MustSignNewTx(key, types.LatestSignerForChainID(w.chainID),
		&types.LegacyTx{Nonce: 2, GasPrice: big.NewInt(100), Gas: 100000, Data: []byte{1}})
	_, err = SubmitTransaction(t.Context(), w.backend, contract)
	require.NoError(t, err, "a contract creation is submitted like any transaction")

	ctrl := gomock.NewController(t)
	failing := NewMockBackend(ctrl)
	failing.EXPECT().RPCTxFeeCap().Return(float64(0)).AnyTimes()
	failing.EXPECT().UnprotectedAllowed().Return(true).AnyTimes()
	failing.EXPECT().SendTx(gomock.Any(), gomock.Any()).Return(errors.New("rejected"))
	_, err = SubmitTransaction(t.Context(), failing, protected)
	require.ErrorContains(t, err, "rejected")
}

func TestCheckTxFee(t *testing.T) {
	ether := big.NewInt(params.Ether)
	require.NoError(t, checkTxFee(new(big.Int).Mul(ether, big.NewInt(1000)), 1_000_000, 0), "no cap")
	require.NoError(t, checkTxFee(big.NewInt(1), 21000, 1))
	require.NoError(t, checkTxFee(ether, 1, 1), "a fee at the cap is fine")
	require.ErrorContains(t, checkTxFee(ether, 2, 1), "tx fee (2.00 FTM) exceeds the configured cap (1.00 FTM)")
}

func TestAddrLocker_LocksPerAddress(t *testing.T) {
	var l AddrLocker
	a, b := common.Address{1}, common.Address{2}
	l.LockAddr(a)
	l.LockAddr(b) // another address is not held up

	acquired := make(chan struct{})
	go func() {
		l.LockAddr(a)
		close(acquired)
		l.UnlockAddr(a)
	}()
	select {
	case <-acquired:
		t.Fatal("the lock of an address was taken twice")
	case <-time.After(50 * time.Millisecond):
	}
	l.UnlockAddr(a)
	select {
	case <-acquired:
	case <-time.After(5 * time.Second):
		t.Fatal("the lock of an address was not released")
	}
	l.UnlockAddr(b)
}

func TestPublicDAGChainAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	b := NewMockBackend(ctrl)
	api := NewPublicDAGChainAPI(b)
	ctx := t.Context()

	event := &inter.MutableEventPayload{}
	event.SetEpoch(3)
	event.SetLamport(9)
	payload := event.Build()

	b.EXPECT().GetEvent(gomock.Any(), "found").Return(&payload.Event, nil)
	b.EXPECT().GetEvent(gomock.Any(), "missing").Return(nil, nil)
	b.EXPECT().GetEvent(gomock.Any(), "broken").Return(nil, errors.New("broken"))
	got, err := api.GetEvent(ctx, "found")
	require.NoError(t, err)
	require.EqualValues(t, 3, got["epoch"].(hexutil.Uint64))
	_, err = api.GetEvent(ctx, "missing")
	require.ErrorContains(t, err, "event missing not found")
	_, err = api.GetEvent(ctx, "broken")
	require.ErrorContains(t, err, "broken")

	b.EXPECT().GetEventPayload(gomock.Any(), "found").Return(payload, nil)
	b.EXPECT().GetEventPayload(gomock.Any(), "missing").Return(nil, nil)
	b.EXPECT().GetEventPayload(gomock.Any(), "broken").Return(nil, errors.New("broken"))
	got, err = api.GetEventPayload(ctx, "found", true)
	require.NoError(t, err)
	require.Contains(t, got, "transactions")
	_, err = api.GetEventPayload(ctx, "missing", false)
	require.ErrorContains(t, err, "event missing not found")
	_, err = api.GetEventPayload(ctx, "broken", false)
	require.ErrorContains(t, err, "broken")

	b.EXPECT().GetHeads(gomock.Any(), rpc.PendingBlockNumber).Return(nil, nil)
	b.EXPECT().GetHeads(gomock.Any(), rpc.BlockNumber(2)).Return(nil, errors.New("epoch is not in range"))
	heads, err := api.GetHeads(ctx, rpc.PendingBlockNumber)
	require.NoError(t, err)
	require.Empty(t, heads)
	_, err = api.GetHeads(ctx, rpc.BlockNumber(2))
	require.ErrorContains(t, err, "epoch is not in range")
}

func TestPublicAbftAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	b := NewMockBackend(ctrl)
	api := NewPublicAbftAPI(b)
	ctx := t.Context()

	b.EXPECT().GetEpochBlockState(gomock.Any(), rpc.BlockNumber(1)).Return(nil, nil, nil)
	validators, err := api.GetValidators(ctx, rpc.BlockNumber(1))
	require.NoError(t, err)
	require.Nil(t, validators)
	b.EXPECT().GetEpochBlockState(gomock.Any(), rpc.BlockNumber(2)).Return(nil, nil, errors.New("no epoch"))
	_, err = api.GetValidators(ctx, rpc.BlockNumber(2))
	require.ErrorContains(t, err, "no epoch")

	b.EXPECT().GetDowntime(gomock.Any(), idx.ValidatorID(1)).Return(idx.Block(5), inter.Timestamp(7), nil)
	b.EXPECT().GetDowntime(gomock.Any(), idx.ValidatorID(2)).Return(idx.Block(0), inter.Timestamp(0), errors.New("broken"))
	downtime, err := api.GetDowntime(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{"offlineBlocks": hexutil.Uint64(5), "offlineTime": hexutil.Uint64(7)}, downtime)
	_, err = api.GetDowntime(ctx, 2)
	require.ErrorContains(t, err, "broken")

	b.EXPECT().GetUptime(gomock.Any(), idx.ValidatorID(1)).Return(big.NewInt(11), nil)
	b.EXPECT().GetUptime(gomock.Any(), idx.ValidatorID(2)).Return(nil, nil)
	b.EXPECT().GetUptime(gomock.Any(), idx.ValidatorID(3)).Return(nil, errors.New("broken"))
	uptime, err := api.GetEpochUptime(ctx, 1)
	require.NoError(t, err)
	require.EqualValues(t, 11, uptime)
	uptime, err = api.GetEpochUptime(ctx, 2)
	require.NoError(t, err)
	require.Zero(t, uptime)
	_, err = api.GetEpochUptime(ctx, 3)
	require.ErrorContains(t, err, "broken")

	b.EXPECT().GetOriginatedFee(gomock.Any(), idx.ValidatorID(1)).Return(big.NewInt(13), nil)
	b.EXPECT().GetOriginatedFee(gomock.Any(), idx.ValidatorID(2)).Return(nil, errors.New("broken"))
	fee, err := api.GetOriginatedEpochFee(ctx, 1)
	require.NoError(t, err)
	require.EqualValues(t, 13, fee.ToInt().Int64())
	_, err = api.GetOriginatedEpochFee(ctx, 2)
	require.ErrorContains(t, err, "broken")
}

func TestPublicBlockChainAPI_EpochMethods(t *testing.T) {
	ctrl := gomock.NewController(t)
	b := NewMockBackend(ctrl)
	api := NewPublicBlockChainAPI(b)
	ctx := t.Context()

	state := &iblockproc.BlockState{}
	state.LastBlock.Idx = 77
	b.EXPECT().GetEpochBlockState(gomock.Any(), rpc.BlockNumber(4)).Return(state, nil, nil)
	b.EXPECT().GetEpochBlockState(gomock.Any(), rpc.BlockNumber(5)).Return(nil, nil, nil)
	b.EXPECT().GetEpochBlockState(gomock.Any(), rpc.BlockNumber(6)).Return(nil, nil, errors.New("broken"))
	block, err := api.GetEpochBlock(ctx, 4)
	require.NoError(t, err)
	require.EqualValues(t, 77, block)
	block, err = api.GetEpochBlock(ctx, 5)
	require.NoError(t, err)
	require.Zero(t, block)
	_, err = api.GetEpochBlock(ctx, 6)
	require.ErrorContains(t, err, "broken")

	b.EXPECT().CurrentEpoch(gomock.Any()).Return(idx.Epoch(10)).AnyTimes()
	b.EXPECT().SealedEpochTiming(gomock.Any()).Return(inter.Timestamp(100), inter.Timestamp(200)).AnyTimes()
	for _, epoch := range []rpc.BlockNumber{rpc.LatestBlockNumber, 9} {
		stats, err := api.GetEpochStats(ctx, epoch)
		require.NoError(t, err)
		require.EqualValues(t, 9, stats["epoch"].(hexutil.Uint64))
		require.EqualValues(t, 100, stats["start"].(hexutil.Uint64))
		require.EqualValues(t, 200, stats["end"].(hexutil.Uint64))
	}
	_, err = api.GetEpochStats(ctx, 3)
	require.ErrorContains(t, err, "doesn't support retrieving previous sealed epochs")
}

func TestPrivateDebugAPI_DatabaseMaintenanceIsNotSupported(t *testing.T) {
	api := NewPrivateDebugAPI(nil)
	_, err := api.ChaindbProperty("stats")
	require.ErrorContains(t, err, "does not provide db properties")
	require.ErrorContains(t, api.ChaindbCompact(), "does not use compaction")
	require.ErrorContains(t, api.SetHead(1), "cannot rewind blocks")
}

func TestPublicNetAPI(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	server := &p2p.Server{Config: p2p.Config{PrivateKey: key, NoDial: true, NoDiscovery: true, MaxPeers: 0}}
	require.NoError(t, server.Start())
	defer server.Stop()

	api := NewPublicNetAPI(server, 146)
	require.True(t, api.Listening())
	require.Zero(t, api.PeerCount())
}
