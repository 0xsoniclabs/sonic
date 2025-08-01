// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ethapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"slices"
	"time"

	cc "github.com/0xsoniclabs/carmen/go/common"
	"github.com/0xsoniclabs/carmen/go/common/immutable"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/gossip/gasprice/gaspricelimits"
	bip39 "github.com/tyler-smith/go-bip39"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/gasprice"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/0xsoniclabs/sonic/utils/signers/gsignercache"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/accounts/scwallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	geth_math "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
)

const (
	// defaultTraceTimeout is the amount of time a single transaction can execute
	// by default before being forcefully aborted.
	defaultTraceTimeout = 5 * time.Second
)

var (
	noUncles = []evmcore.EvmHeader{}
)

// PublicEthereumAPI provides an API to access Ethereum related information.
// It offers only methods that operate on public data that is freely available to anyone.
type PublicEthereumAPI struct {
	b Backend
}

// NewPublicEthereumAPI creates a new Ethereum protocol API.
func NewPublicEthereumAPI(b Backend) *PublicEthereumAPI {
	return &PublicEthereumAPI{b}
}

// GasPrice returns a suggestion for a gas price for legacy transactions.
func (s *PublicEthereumAPI) GasPrice(ctx context.Context) (*hexutil.Big, error) {
	// Right now, we are not suggesting any tips since those have no real
	// effect on the Sonic network. So the suggested gas price is a slightly
	// increased base fee to provide a buffer for short-term price fluctuations.
	price := s.b.CurrentBlock().Header().BaseFee
	price = gaspricelimits.GetSuggestedGasPriceForNewTransactions(price)
	return (*hexutil.Big)(price), nil
}

// MaxPriorityFeePerGas returns a suggestion for a gas tip cap for dynamic fee transactions.
func (s *PublicEthereumAPI) MaxPriorityFeePerGas(ctx context.Context) (*hexutil.Big, error) {
	tipcap := s.b.SuggestGasTipCap(ctx, gasprice.AsDefaultCertainty)
	return (*hexutil.Big)(tipcap), nil
}

type feeHistoryResult struct {
	OldestBlock  *hexutil.Big     `json:"oldestBlock"`
	Reward       [][]*hexutil.Big `json:"reward,omitempty"`
	BaseFee      []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio []float64        `json:"gasUsedRatio"`
}

var errInvalidPercentile = errors.New("invalid reward percentile")

func (s *PublicEthereumAPI) FeeHistory(ctx context.Context, blockCount geth_math.HexOrDecimal64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*feeHistoryResult, error) {
	res := &feeHistoryResult{}
	res.Reward = make([][]*hexutil.Big, 0, blockCount)
	res.BaseFee = make([]*hexutil.Big, 0, blockCount)
	res.GasUsedRatio = make([]float64, 0, blockCount)
	res.OldestBlock = (*hexutil.Big)(new(big.Int))

	// validate input parameters
	if blockCount == 0 {
		return res, nil
	}
	if blockCount > 1024 {
		blockCount = 1024
	}
	for i, p := range rewardPercentiles {
		if p < 0 || p > 100 {
			return nil, fmt.Errorf("%w: %f", errInvalidPercentile, p)
		}
		if i > 0 && p < rewardPercentiles[i-1] {
			return nil, fmt.Errorf("%w: #%d:%f > #%d:%f", errInvalidPercentile, i-1, rewardPercentiles[i-1], i, p)
		}
	}
	last, err := s.b.ResolveRpcBlockNumberOrHash(ctx, rpc.BlockNumberOrHash{BlockNumber: &lastBlock})
	if err != nil {
		return nil, err
	}
	oldest := last
	if oldest > idx.Block(blockCount) {
		oldest -= idx.Block(blockCount - 1)
	} else {
		oldest = 0
	}

	baseFee := s.b.MinGasPrice()

	tips := make([]*hexutil.Big, 0, len(rewardPercentiles))
	for _, p := range rewardPercentiles {
		tip := s.b.SuggestGasTipCap(ctx, uint64(gasprice.DecimalUnit*p/100.0))
		tips = append(tips, (*hexutil.Big)(tip))
	}
	res.OldestBlock.ToInt().SetUint64(uint64(oldest))
	for i := uint64(0); i < uint64(last-oldest+1); i++ {
		res.Reward = append(res.Reward, tips)
		res.BaseFee = append(res.BaseFee, (*hexutil.Big)(baseFee))
		res.GasUsedRatio = append(res.GasUsedRatio, 0.99)
	}
	return res, nil
}

func (s *PublicEthereumAPI) BlobBaseFee(ctx context.Context) *hexutil.Big {
	// As blobs are not supported yet, blob base fee is equal to min blob gas price
	// because calculation of blob base fee is based on the blob gas price and
	// excess blob gas and that is always 0 for now
	return (*hexutil.Big)(big.NewInt(params.BlobTxMinBlobGasprice))
}

// Syncing returns true if node is syncing
func (s *PublicEthereumAPI) Syncing() (interface{}, error) {
	progress := s.b.Progress()
	// Return not syncing if the synchronisation already completed
	if time.Since(progress.CurrentBlockTime.Time()) <= 90*time.Minute { // should be >> MaxEmitInterval
		return false, nil
	}
	// Otherwise gather the block sync stats
	return map[string]interface{}{
		"startingBlock":    hexutil.Uint64(0), // back-compatibility
		"currentEpoch":     hexutil.Uint64(progress.CurrentEpoch),
		"currentBlock":     hexutil.Uint64(progress.CurrentBlock),
		"currentBlockHash": progress.CurrentBlockHash.Hex(),
		"currentBlockTime": hexutil.Uint64(progress.CurrentBlockTime),
		"highestBlock":     hexutil.Uint64(progress.HighestBlock),
		"highestEpoch":     hexutil.Uint64(progress.HighestEpoch),
		"pulledStates":     hexutil.Uint64(0), // back-compatibility
		"knownStates":      hexutil.Uint64(0), // back-compatibility
	}, nil
}

// PublicTxPoolAPI offers and API for the transaction pool. It only operates on data that is non confidential.
type PublicTxPoolAPI struct {
	b Backend
}

// NewPublicTxPoolAPI creates a new tx pool service that gives information about the transaction pool.
func NewPublicTxPoolAPI(b Backend) *PublicTxPoolAPI {
	return &PublicTxPoolAPI{b}
}

// Content returns the transactions contained within the transaction pool.
func (s *PublicTxPoolAPI) Content() map[string]map[string]map[string]*RPCTransaction {
	content := map[string]map[string]map[string]*RPCTransaction{
		"pending": make(map[string]map[string]*RPCTransaction),
		"queued":  make(map[string]map[string]*RPCTransaction),
	}
	pending, queue := s.b.TxPoolContent()

	curHeader := s.b.CurrentBlock().Header()
	// Flatten the pending transactions
	for account, txs := range pending {
		dump := make(map[string]*RPCTransaction)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = newRPCPendingTransaction(tx, curHeader.BaseFee)
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, txs := range queue {
		dump := make(map[string]*RPCTransaction)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = newRPCPendingTransaction(tx, curHeader.BaseFee)
		}
		content["queued"][account.Hex()] = dump
	}
	return content
}

// ContentFrom returns the transactions contained within the transaction pool.
func (s *PublicTxPoolAPI) ContentFrom(addr common.Address) map[string]map[string]*RPCTransaction {
	content := make(map[string]map[string]*RPCTransaction, 2)
	pending, queue := s.b.TxPoolContentFrom(addr)
	curHeader := s.b.CurrentBlock().Header()

	// Build the pending transactions
	dump := make(map[string]*RPCTransaction, len(pending))
	for _, tx := range pending {
		dump[fmt.Sprintf("%d", tx.Nonce())] = newRPCPendingTransaction(tx, curHeader.BaseFee)
	}
	content["pending"] = dump

	// Build the queued transactions
	dump = make(map[string]*RPCTransaction, len(queue))
	for _, tx := range queue {
		dump[fmt.Sprintf("%d", tx.Nonce())] = newRPCPendingTransaction(tx, curHeader.BaseFee)
	}
	content["queued"] = dump

	return content
}

// Status returns the number of pending and queued transaction in the pool.
func (s *PublicTxPoolAPI) Status() map[string]hexutil.Uint {
	pending, queue := s.b.Stats()
	return map[string]hexutil.Uint{
		"pending": hexutil.Uint(pending),
		"queued":  hexutil.Uint(queue),
	}
}

// Inspect retrieves the content of the transaction pool and flattens it into an
// easily inspectable list.
func (s *PublicTxPoolAPI) Inspect() map[string]map[string]map[string]string {
	content := map[string]map[string]map[string]string{
		"pending": make(map[string]map[string]string),
		"queued":  make(map[string]map[string]string),
	}
	pending, queue := s.b.TxPoolContent()

	// Define a formatter to flatten a transaction into a string
	var format = func(tx *types.Transaction) string {
		if to := tx.To(); to != nil {
			return fmt.Sprintf("%s: %v wei + %v gas × %v wei", tx.To().Hex(), tx.Value(), tx.Gas(), tx.GasPrice())
		}
		return fmt.Sprintf("contract creation: %v wei + %v gas × %v wei", tx.Value(), tx.Gas(), tx.GasPrice())
	}
	// Flatten the pending transactions
	for account, txs := range pending {
		dump := make(map[string]string)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = format(tx)
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, txs := range queue {
		dump := make(map[string]string)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = format(tx)
		}
		content["queued"][account.Hex()] = dump
	}
	return content
}

// PublicAccountAPI provides an API to access accounts managed by this node.
// It offers only methods that can retrieve accounts.
type PublicAccountAPI struct {
	am *accounts.Manager
}

// NewPublicAccountAPI creates a new PublicAccountAPI.
func NewPublicAccountAPI(am *accounts.Manager) *PublicAccountAPI {
	return &PublicAccountAPI{am: am}
}

// Accounts returns the collection of accounts this node manages
func (s *PublicAccountAPI) Accounts() []common.Address {
	return s.am.Accounts()
}

// PrivateAccountAPI provides an API to access accounts managed by this node.
// It offers methods to create, (un)lock en list accounts. Some methods accept
// passwords and are therefore considered private by default.
type PrivateAccountAPI struct {
	am        *accounts.Manager
	nonceLock *AddrLocker
	b         Backend
}

// NewPrivateAccountAPI create a new PrivateAccountAPI.
func NewPrivateAccountAPI(b Backend, nonceLock *AddrLocker) *PrivateAccountAPI {
	return &PrivateAccountAPI{
		am:        b.AccountManager(),
		nonceLock: nonceLock,
		b:         b,
	}
}

// ListAccounts will return a list of addresses for accounts this node manages.
func (s *PrivateAccountAPI) ListAccounts() []common.Address {
	return s.am.Accounts()
}

// RawWallet is a JSON representation of an accounts.Wallet interface, with its
// data contents extracted into plain fields.
type RawWallet struct {
	URL      string             `json:"url"`
	Status   string             `json:"status"`
	Failure  string             `json:"failure,omitempty"`
	Accounts []accounts.Account `json:"accounts,omitempty"`
}

// ListWallets will return a list of wallets this node manages.
func (s *PrivateAccountAPI) ListWallets() []RawWallet {
	wallets := make([]RawWallet, 0) // return [] instead of nil if empty
	for _, wallet := range s.am.Wallets() {
		status, failure := wallet.Status()

		raw := RawWallet{
			URL:      wallet.URL().String(),
			Status:   status,
			Accounts: wallet.Accounts(),
		}
		if failure != nil {
			raw.Failure = failure.Error()
		}
		wallets = append(wallets, raw)
	}
	return wallets
}

// OpenWallet initiates a hardware wallet opening procedure, establishing a USB
// connection and attempting to authenticate via the provided passphrase. Note,
// the method may return an extra challenge requiring a second open (e.g. the
// Trezor PIN matrix challenge).
func (s *PrivateAccountAPI) OpenWallet(url string, passphrase *string) error {
	wallet, err := s.am.Wallet(url)
	if err != nil {
		return err
	}
	pass := ""
	if passphrase != nil {
		pass = *passphrase
	}
	return wallet.Open(pass)
}

// DeriveAccount requests a HD wallet to derive a new account, optionally pinning
// it for later reuse.
func (s *PrivateAccountAPI) DeriveAccount(url string, path string, pin *bool) (accounts.Account, error) {
	wallet, err := s.am.Wallet(url)
	if err != nil {
		return accounts.Account{}, err
	}
	derivPath, err := accounts.ParseDerivationPath(path)
	if err != nil {
		return accounts.Account{}, err
	}
	if pin == nil {
		pin = new(bool)
	}
	return wallet.Derive(derivPath, *pin)
}

// NewAccount will create a new account and returns the address for the new account.
func (s *PrivateAccountAPI) NewAccount(password string) (common.Address, error) {
	ks, err := fetchKeystore(s.am)
	if err != nil {
		return common.Address{}, err
	}
	acc, err := ks.NewAccount(password)
	if err == nil {
		log.Info("Your new key was generated", "address", acc.Address)
		log.Warn("Please backup your key file!", "path", acc.URL.Path)
		log.Warn("Please remember your password!")
		return acc.Address, nil
	}
	return common.Address{}, err
}

// fetchKeystore retrieves the encrypted keystore from the account manager.
func fetchKeystore(am *accounts.Manager) (*keystore.KeyStore, error) {
	if ks := am.Backends(keystore.KeyStoreType); len(ks) > 0 {
		return ks[0].(*keystore.KeyStore), nil
	}
	return nil, errors.New("local keystore not used")
}

// ImportRawKey stores the given hex encoded ECDSA key into the key directory,
// encrypting it with the passphrase.
func (s *PrivateAccountAPI) ImportRawKey(privkey string, password string) (common.Address, error) {
	key, err := crypto.HexToECDSA(privkey)
	if err != nil {
		return common.Address{}, err
	}
	ks, err := fetchKeystore(s.am)
	if err != nil {
		return common.Address{}, err
	}
	acc, err := ks.ImportECDSA(key, password)
	return acc.Address, err
}

// UnlockAccount will unlock the account associated with the given address with
// the given password for duration seconds. If duration is nil it will use a
// default of 300 seconds. It returns an indication if the account was unlocked.
func (s *PrivateAccountAPI) UnlockAccount(ctx context.Context, addr common.Address, password string, duration *uint64) (bool, error) {
	// When the API is exposed by external RPC(http, ws etc), unless the user
	// explicitly specifies to allow the insecure account unlocking, otherwise
	// it is disabled.
	if s.b.ExtRPCEnabled() {
		return false, errors.New("account unlock with HTTP access is forbidden")
	}

	const max = uint64(time.Duration(math.MaxInt64) / time.Second)
	var d time.Duration
	if duration == nil {
		d = 300 * time.Second
	} else if *duration > max {
		return false, errors.New("unlock duration too large")
	} else {
		d = time.Duration(*duration) * time.Second
	}
	ks, err := fetchKeystore(s.am)
	if err != nil {
		return false, err
	}
	err = ks.TimedUnlock(accounts.Account{Address: addr}, password, d)
	if err != nil {
		log.Warn("Failed account unlock attempt", "address", addr, "err", err)
	}
	return err == nil, err
}

// LockAccount will lock the account associated with the given address when it's unlocked.
func (s *PrivateAccountAPI) LockAccount(addr common.Address) bool {
	if ks, err := fetchKeystore(s.am); err == nil {
		return ks.Lock(addr) == nil
	}
	return false
}

// signTransaction sets defaults and signs the given transaction
// NOTE: the caller needs to ensure that the nonceLock is held, if applicable,
// and release it after the transaction has been submitted to the tx pool
func (s *PrivateAccountAPI) signTransaction(ctx context.Context, args *TransactionArgs, passwd string) (*types.Transaction, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: args.from()}
	wallet, err := s.am.Find(account)
	if err != nil {
		return nil, err
	}
	// Set some sanity defaults and terminate on failure
	if err := args.setDefaults(ctx, s.b); err != nil {
		return nil, err
	}
	// Assemble the transaction and sign with the wallet
	tx := args.toTransaction()

	chainID := s.b.ChainConfig(s.b.Progress().CurrentBlock).ChainID
	return wallet.SignTxWithPassphrase(account, passwd, tx, chainID)
}

// SendTransaction will create a transaction from the given arguments and
// tries to sign it with the key associated with args.From. If the given
// passwd isn't able to decrypt the key it fails.
func (s *PrivateAccountAPI) SendTransaction(ctx context.Context, args TransactionArgs, passwd string) (common.Hash, error) {
	if args.Nonce == nil {
		// Hold the addresse's mutex around signing to prevent concurrent assignment of
		// the same nonce to multiple accounts.
		s.nonceLock.LockAddr(args.from())
		defer s.nonceLock.UnlockAddr(args.from())
	}
	signed, err := s.signTransaction(ctx, &args, passwd)
	if err != nil {
		log.Warn("Failed transaction send attempt", "from", args.from(), "to", args.To, "value", args.Value.ToInt(), "err", err)
		return common.Hash{}, err
	}
	return SubmitTransaction(ctx, s.b, signed)
}

// SignTransaction will create a transaction from the given arguments and
// tries to sign it with the key associated with args.From. If the given passwd isn't
// able to decrypt the key it fails. The transaction is returned in RLP-form, not broadcast
// to other nodes
func (s *PrivateAccountAPI) SignTransaction(ctx context.Context, args TransactionArgs, passwd string) (*SignTransactionResult, error) {
	// No need to obtain the noncelock mutex, since we won't be sending this
	// tx into the transaction pool, but right back to the user
	if args.From == nil {
		return nil, fmt.Errorf("sender not specified")
	}
	if args.Gas == nil {
		return nil, fmt.Errorf("gas not specified")
	}
	if args.GasPrice == nil && (args.MaxFeePerGas == nil || args.MaxPriorityFeePerGas == nil) {
		return nil, fmt.Errorf("missing gasPrice or maxFeePerGas/maxPriorityFeePerGas")
	}
	if args.Nonce == nil {
		return nil, fmt.Errorf("nonce not specified")
	}
	// Before actually signing the transaction, ensure the transaction fee is reasonable.
	tx := args.toTransaction()
	if err := checkTxFee(tx.GasPrice(), tx.Gas(), s.b.RPCTxFeeCap()); err != nil {
		return nil, err
	}
	signed, err := s.signTransaction(ctx, &args, passwd)
	if err != nil {
		log.Warn("Failed transaction sign attempt", "from", args.from(), "to", args.To, "value", args.Value.ToInt(), "err", err)
		return nil, err
	}
	data, err := signed.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return &SignTransactionResult{data, signed}, nil
}

// Sign calculates an Ethereum ECDSA signature for:
// keccack256("\x19Ethereum Signed Message:\n" + len(message) + message))
//
// Note, the produced signature conforms to the secp256k1 curve R, S and V values,
// where the V value will be 27 or 28 for legacy reasons.
//
// The key used to calculate the signature is decrypted with the given password.
//
// https://github.com/ethereum/go-ethereum/wiki/Management-APIs#personal_sign
func (s *PrivateAccountAPI) Sign(ctx context.Context, data hexutil.Bytes, addr common.Address, passwd string) (hexutil.Bytes, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := s.b.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Assemble sign the data with the wallet
	signature, err := wallet.SignTextWithPassphrase(account, passwd, data)
	if err != nil {
		log.Warn("Failed data sign attempt", "address", addr, "err", err)
		return nil, err
	}
	signature[crypto.RecoveryIDOffset] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	return signature, nil
}

// EcRecover returns the address for the account that was used to create the signature.
// Note, this function is compatible with eth_sign and personal_sign. As such it recovers
// the address of:
// hash = keccak256("\x19Ethereum Signed Message:\n"${message length}${message})
// addr = ecrecover(hash, signature)
//
// Note, the signature must conform to the secp256k1 curve R, S and V values, where
// the V value must be 27 or 28 for legacy reasons.
//
// https://github.com/ethereum/go-ethereum/wiki/Management-APIs#personal_ecRecover
func (s *PrivateAccountAPI) EcRecover(ctx context.Context, data, sig hexutil.Bytes) (common.Address, error) {
	if len(sig) != crypto.SignatureLength {
		return common.Address{}, fmt.Errorf("signature must be %d bytes long", crypto.SignatureLength)
	}
	if sig[crypto.RecoveryIDOffset] != 27 && sig[crypto.RecoveryIDOffset] != 28 {
		return common.Address{}, fmt.Errorf("invalid Ethereum signature (V is not 27 or 28)")
	}
	sig[crypto.RecoveryIDOffset] -= 27 // Transform yellow paper V from 27/28 to 0/1

	rpk, err := crypto.SigToPub(accounts.TextHash(data), sig)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*rpk), nil
}

// SignAndSendTransaction was renamed to SendTransaction. This method is deprecated
// and will be removed in the future. It primary goal is to give clients time to update.
func (s *PrivateAccountAPI) SignAndSendTransaction(ctx context.Context, args TransactionArgs, passwd string) (common.Hash, error) {
	return s.SendTransaction(ctx, args, passwd)
}

// InitializeWallet initializes a new wallet at the provided URL, by generating and returning a new private key.
func (s *PrivateAccountAPI) InitializeWallet(ctx context.Context, url string) (string, error) {
	wallet, err := s.am.Wallet(url)
	if err != nil {
		return "", err
	}

	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", err
	}

	seed := bip39.NewSeed(mnemonic, "")

	switch wallet := wallet.(type) {
	case *scwallet.Wallet:
		return mnemonic, wallet.Initialize(seed)
	default:
		return "", fmt.Errorf("specified wallet does not support initialization")
	}
}

// Unpair deletes a pairing between wallet and geth.
func (s *PrivateAccountAPI) Unpair(ctx context.Context, url string, pin string) error {
	wallet, err := s.am.Wallet(url)
	if err != nil {
		return err
	}

	switch wallet := wallet.(type) {
	case *scwallet.Wallet:
		return wallet.Unpair([]byte(pin))
	default:
		return fmt.Errorf("specified wallet does not support pairing")
	}
}

// PublicBlockChainAPI provides an API to access the Ethereum blockchain.
// It offers only methods that operate on public data that is freely available to anyone.
type PublicBlockChainAPI struct {
	b Backend
}

// NewPublicBlockChainAPI creates a new Ethereum blockchain API.
func NewPublicBlockChainAPI(b Backend) *PublicBlockChainAPI {
	return &PublicBlockChainAPI{b}
}

// CurrentEpoch returns current epoch number.
func (s *PublicBlockChainAPI) CurrentEpoch(ctx context.Context) hexutil.Uint64 {
	return hexutil.Uint64(s.b.CurrentEpoch(ctx))
}

// GetRules returns network rules for an epoch
func (s *PublicBlockChainAPI) GetRules(ctx context.Context, epoch rpc.BlockNumber) (*opera.Rules, error) {
	_, es, err := s.b.GetEpochBlockState(ctx, epoch)
	if err != nil {
		return nil, err
	}
	if es == nil {
		return nil, nil
	}
	return &es.Rules, nil
}

// GetEpochBlock returns block height in a beginning of an epoch
func (s *PublicBlockChainAPI) GetEpochBlock(ctx context.Context, epoch rpc.BlockNumber) (hexutil.Uint64, error) {
	bs, _, err := s.b.GetEpochBlockState(ctx, epoch)
	if err != nil {
		return 0, err
	}
	if bs == nil {
		return 0, nil
	}
	return hexutil.Uint64(bs.LastBlock.Idx), nil
}

// ChainId is the EIP-155 replay-protection chain id for the current ethereum chain config.
func (s *PublicBlockChainAPI) ChainId() (*hexutil.Big, error) {
	// Sonic is always EIP-155 compliant, so we can safely return the chain ID
	return (*hexutil.Big)(s.b.ChainID()), nil
}

// BlockNumber returns the block number of the chain head.
func (s *PublicBlockChainAPI) BlockNumber() hexutil.Uint64 {
	header, _ := s.b.HeaderByNumber(context.Background(), rpc.LatestBlockNumber) // latest header should always be available
	return hexutil.Uint64(header.Number.Uint64())
}

// GetBalance returns the amount of wei for the given address in the state of the
// given block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta
// block numbers are also allowed.
func (s *PublicBlockChainAPI) GetBalance(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.U256, error) {
	state, _, err := s.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	defer state.Release()
	return (*hexutil.U256)(state.GetBalance(address)), state.Error()
}

// GetAccountResult is result struct for GetAccount.
// The result contains:
// 1) CodeHash - hash of the code for the given address
// 2) StorageRoot - storage root for the given address
// 3) Balance - the amount of wei for the given address
// 4) Nonce - the number of transactions for given address
type GetAccountResult struct {
	CodeHash    common.Hash    `json:"codeHash"`
	StorageRoot common.Hash    `json:"storageRoot"`
	Balance     *hexutil.U256  `json:"balance"`
	Nonce       hexutil.Uint64 `json:"nonce"`
}

// GetAccount returns the information about account with given address in the state of the given block number.
// The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta block numbers are also allowed.
func (s *PublicBlockChainAPI) GetAccount(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*GetAccountResult, error) {
	state, header, err := s.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if err != nil {
		return nil, err
	}
	defer state.Release()
	proof, err := state.GetProof(address, nil)
	if err != nil {
		return nil, err
	}
	codeHash, _, err := proof.GetCodeHash(cc.Hash(header.Root), cc.Address(address))
	if err != nil {
		return nil, err
	}
	_, storageRoot, _ := proof.GetAccountElements(cc.Hash(header.Root), cc.Address(address))
	balance, _, err := proof.GetBalance(cc.Hash(header.Root), cc.Address(address))
	if err != nil {
		return nil, err
	}
	nonce, _, err := proof.GetNonce(cc.Hash(header.Root), cc.Address(address))
	if err != nil {
		return nil, err
	}
	u256Balance := balance.Uint256()
	return &GetAccountResult{
		CodeHash:    common.Hash(codeHash),
		StorageRoot: common.Hash(storageRoot),
		Balance:     (*hexutil.U256)(&u256Balance),
		Nonce:       hexutil.Uint64(nonce.ToUint64()),
	}, state.Error()
}

// AccountResult is result struct for GetProof
type AccountResult struct {
	Address      common.Address  `json:"address"`
	AccountProof []string        `json:"accountProof"`
	Balance      *hexutil.U256   `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []StorageResult `json:"storageProof"`
}

// StorageResult is result struct for GetProof
type StorageResult struct {
	Key   string       `json:"key"`
	Value *hexutil.Big `json:"value"`
	Proof []string     `json:"proof"`
}

// GetProof returns the Merkle-proof for a given account and optionally some storage keys.
func (s *PublicBlockChainAPI) GetProof(ctx context.Context, address common.Address, storageKeys []string, blockNrOrHash rpc.BlockNumberOrHash) (*AccountResult, error) {
	state, header, err := s.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	defer state.Release()

	keys := make([]common.Hash, len(storageKeys))
	for i, key := range storageKeys {
		keys[i] = common.HexToHash(key)
	}
	proof, err := state.GetProof(address, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}

	storageProof := make([]StorageResult, len(keys))
	for i, key := range keys {
		value, _, err := proof.GetState(cc.Hash(header.Root), cc.Address(address), cc.Key(key))
		if err != nil {
			return nil, err
		}
		elements, _ := proof.GetStorageElements(cc.Hash(header.Root), cc.Address(address), cc.Key(keys[i]))
		storageProof[i] = StorageResult{
			Key:   key.Hex(),
			Value: (*hexutil.Big)(new(big.Int).SetBytes(value[:])),
			Proof: toHexSlice(elements),
		}
	}

	accountProof, storageHash, _ := proof.GetAccountElements(cc.Hash(header.Root), cc.Address(address))

	codeHash, _, err := proof.GetCodeHash(cc.Hash(header.Root), cc.Address(address))
	if err != nil {
		return nil, err
	}
	balance, _, err := proof.GetBalance(cc.Hash(header.Root), cc.Address(address))
	if err != nil {
		return nil, err
	}
	nonce, _, err := proof.GetNonce(cc.Hash(header.Root), cc.Address(address))
	if err != nil {
		return nil, err
	}
	u256Balance := balance.Uint256()
	return &AccountResult{
		Address:      address,
		AccountProof: toHexSlice(accountProof),
		Balance:      (*hexutil.U256)(&u256Balance),
		CodeHash:     common.Hash(codeHash),
		Nonce:        hexutil.Uint64(nonce.ToUint64()),
		StorageHash:  common.Hash(storageHash),
		StorageProof: storageProof,
	}, state.Error()
}

// GetHeaderByNumber returns the requested canonical block header.
// * When blockNr is -1 the chain head is returned.
// * When blockNr is -2 the pending chain head is returned.
func (s *PublicBlockChainAPI) GetHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*evmcore.EvmHeaderJson, error) {
	header, err := s.b.HeaderByNumber(ctx, number)
	if header == nil || err != nil {
		return nil, err
	}
	return s.getHeaderWithReceipts(ctx, header, rpc.BlockNumber(header.Number.Uint64()))
}

// GetHeaderByHash returns the requested header by hash.
func (s *PublicBlockChainAPI) GetHeaderByHash(ctx context.Context, hash common.Hash) (*evmcore.EvmHeaderJson, error) {
	header, err := s.b.HeaderByHash(ctx, hash)
	if header == nil || err != nil {
		return nil, err
	}
	return s.getHeaderWithReceipts(ctx, header, rpc.BlockNumber(header.Number.Uint64()))
}

func (s *PublicBlockChainAPI) getHeaderWithReceipts(ctx context.Context, header *evmcore.EvmHeader, blkNumber rpc.BlockNumber) (*evmcore.EvmHeaderJson, error) {
	receipts, err := s.getBlockReceipts(ctx, blkNumber)
	if receipts == nil || err != nil {
		return nil, err
	}
	return header.ToJson(receipts), nil
}

func (s *PublicBlockChainAPI) getBlockReceipts(ctx context.Context, blkNumber rpc.BlockNumber) (types.Receipts, error) {
	if blkNumber == rpc.EarliestBlockNumber {
		return types.Receipts{}, nil
	}
	return s.b.GetReceiptsByNumber(ctx, blkNumber)
}

// GetBlockByNumber returns the requested canonical block.
//   - When blockNr is -1 the chain head is returned.
//   - When blockNr is -2 the pending chain head is returned.
//   - When fullTx is true all transactions in the block are returned, otherwise
//     only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (*evmcore.EvmBlockJson, error) {
	block, err := s.b.BlockByNumber(ctx, number)
	if block != nil && err == nil {
		receipts, err := s.getBlockReceipts(ctx, rpc.BlockNumber(block.NumberU64()))
		if err != nil {
			return nil, err
		}
		return RPCMarshalBlock(block, receipts, true, fullTx)
	}
	return nil, err
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (*evmcore.EvmBlockJson, error) {
	block, err := s.b.BlockByHash(ctx, hash)
	if block != nil && err == nil {
		receipts, err := s.getBlockReceipts(ctx, rpc.BlockNumber(block.NumberU64()))
		if err != nil {
			return nil, err
		}
		return RPCMarshalBlock(block, receipts, true, fullTx)
	}
	return nil, err
}

// GetUncleByBlockNumberAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetUncleByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) (map[string]interface{}, error) {
	block, err := s.b.BlockByNumber(ctx, blockNr)
	if block != nil {
		log.Debug("Requested uncle not found", "number", blockNr, "hash", block.Hash, "index", index)
		return nil, nil
	}
	return nil, err
}

// GetUncleByBlockHashAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetUncleByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) (map[string]interface{}, error) {
	block, err := s.b.BlockByHash(ctx, blockHash)
	if block != nil {
		log.Debug("Requested uncle not found", "number", block.Number, "hash", blockHash, "index", index)
		return nil, nil
	}
	return nil, err
}

// GetUncleCountByBlockNumber returns number of uncles in the block for the given block number
func (s *PublicBlockChainAPI) GetUncleCountByBlockNumber(ctx context.Context, blockNr rpc.BlockNumber) *hexutil.Uint {
	if block, _ := s.b.BlockByNumber(ctx, blockNr); block != nil {
		n := hexutil.Uint(len(noUncles))
		return &n
	}
	return nil
}

// GetUncleCountByBlockHash returns number of uncles in the block for the given block hash
func (s *PublicBlockChainAPI) GetUncleCountByBlockHash(ctx context.Context, blockHash common.Hash) *hexutil.Uint {
	if block, _ := s.b.BlockByHash(ctx, blockHash); block != nil {
		n := hexutil.Uint(len(noUncles))
		return &n
	}
	return nil
}

// GetCode returns the code stored at the given address in the state for the given block number.
func (s *PublicBlockChainAPI) GetCode(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	state, _, err := s.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	defer state.Release()
	code := state.GetCode(address)
	return code, state.Error()
}

// GetStorageAt returns the storage from the state at the given address, key and
// block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta block
// numbers are also allowed.
func (s *PublicBlockChainAPI) GetStorageAt(ctx context.Context, address common.Address, key string, blockNr rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	state, _, err := s.b.StateAndHeaderByNumberOrHash(ctx, blockNr)
	if state == nil || err != nil {
		return nil, err
	}
	defer state.Release()
	res := state.GetState(address, common.HexToHash(key))
	return res[:], state.Error()
}

// OverrideAccount indicates the overriding fields of account during the execution
// of a message call.
// Note, state and stateDiff can't be specified at the same time. If state is
// set, message execution will only use the data in the given state. Otherwise
// if statDiff is set, all diff will be applied first and then execute the call
// message.
type OverrideAccount struct {
	Nonce     *hexutil.Uint64              `json:"nonce"`
	Code      *hexutil.Bytes               `json:"code"`
	Balance   **hexutil.U256               `json:"balance"`
	State     *map[common.Hash]common.Hash `json:"state"`
	StateDiff *map[common.Hash]common.Hash `json:"stateDiff"`
}

// StateOverride is the collection of overridden accounts.
type StateOverride map[common.Address]OverrideAccount

// Apply overrides the fields of specified accounts into the given state.
func (diff *StateOverride) Apply(state state.StateDB) error {
	if diff == nil {
		return nil
	}
	for addr, account := range *diff {
		// Override account nonce.
		if account.Nonce != nil {
			state.SetNonce(addr, uint64(*account.Nonce), tracing.NonceChangeUnspecified)
		}
		// Override account(contract) code.
		if account.Code != nil {
			state.SetCode(addr, *account.Code)
		}
		// Override account balance.
		if account.Balance != nil {
			state.SetBalance(addr, (*uint256.Int)(*account.Balance))
		}
		if account.State != nil && account.StateDiff != nil {
			return fmt.Errorf("account %s has both 'state' and 'stateDiff'", addr.Hex())
		}
		// Replace entire state if caller requires.
		if account.State != nil {
			state.SetStorage(addr, *account.State)
		}
		// Apply state diff into specified accounts.
		if account.StateDiff != nil {
			for key, value := range *account.StateDiff {
				state.SetState(addr, key, value)
			}
		}
	}
	return nil
}

func (diff *StateOverride) HasCodesExceedingOnChainLimit() bool {
	if diff == nil {
		return false
	}
	for _, account := range *diff {
		// Check account(contract) code length.
		if account.Code != nil && len(*account.Code) > params.MaxCodeSize {
			return true
		}
	}
	return false
}

func DoCall(ctx context.Context, b Backend, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *StateOverride, blockOverrides *BlockOverrides, timeout time.Duration, globalGasCap uint64) (*core.ExecutionResult, error) {
	defer func(start time.Time) { log.Debug("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())

	state, header, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	defer state.Release()
	if err := overrides.Apply(state); err != nil {
		return nil, err
	}
	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()

	// Get a new instance of the EVM.
	msg, err := args.ToMessage(globalGasCap, header.BaseFee, log.Root())
	if err != nil {
		return nil, err
	}
	vmConfig, err := GetVmConfig(ctx, b, idx.Block(header.Number.Uint64()))
	if err != nil {
		return nil, err
	}
	if overrides.HasCodesExceedingOnChainLimit() {
		// Use geth as VM for computation
		vmConfig.Tracer = &tracing.Hooks{}
	}

	var blockCtx *vm.BlockContext
	if blockOverrides != nil {
		bctx := getBlockContext(ctx, b, header)
		blockOverrides.apply(&bctx)
		blockCtx = &bctx
	}
	evm, vmError, err := b.GetEVM(ctx, state, header, &vmConfig, blockCtx)
	if err != nil {
		return nil, err
	}
	// Skip gas price checks for API runs.
	evm.Config.NoBaseFee = true
	// Wait for the context to be done and cancel the evm. Even if the
	// EVM has finished, cancelling may be done (repeatedly)
	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()

	// execute EIP-2935 HistoryStorage contract.
	if evm.ChainConfig().IsPrague(header.Number, uint64(header.Time.Unix())) {
		evmcore.ProcessParentBlockHash(header.ParentHash, evm, state)
	}

	// Execute the message.
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	result, err := core.ApplyMessage(evm, msg, gp)
	if err := vmError(); err != nil {
		return nil, err
	}

	// If the timer caused an abort, return an appropriate error message
	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted (timeout = %v)", timeout)
	}
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.GasLimit)
	}
	return result, nil
}

func newRevertError(result *core.ExecutionResult) *revertError {
	reason, errUnpack := abi.UnpackRevert(result.Revert())
	err := errors.New("execution reverted")
	if errUnpack == nil {
		err = fmt.Errorf("execution reverted: %v", reason)
	}
	return &revertError{
		error:  err,
		reason: hexutil.Encode(result.Revert()),
	}
}

// revertError is an API error that encompassas an EVM revertal with JSON error
// code and a binary data blob.
type revertError struct {
	error
	reason string // revert reason hex encoded
}

// ErrorCode returns the JSON error code for a revertal.
// See: https://github.com/ethereum/wiki/wiki/JSON-RPC-Error-Codes-Improvement-Proposal
func (e *revertError) ErrorCode() int {
	return 3
}

// ErrorData returns the hex encoded revert reason.
func (e *revertError) ErrorData() interface{} {
	return e.reason
}

// Call executes the given transaction on the state for the given block number.
//
// Additionally, the caller can specify a batch of contract for fields overriding.
//
// Note, this function doesn't make and changes in the state/blockchain and is
// useful to execute and retrieve values.
func (s *PublicBlockChainAPI) Call(ctx context.Context, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, stateOverrides *StateOverride, blockOverrides *BlockOverrides) (hexutil.Bytes, error) {
	result, err := DoCall(ctx, s.b, args, blockNrOrHash, stateOverrides, blockOverrides, s.b.RPCEVMTimeout(), s.b.RPCGasCap())
	if err != nil {
		return nil, err
	}
	// If the result contains a revert reason, try to unpack and return it.
	if len(result.Revert()) > 0 {
		return nil, newRevertError(result)
	}
	return result.Return(), result.Err
}

// DoEstimateGas - binary search the gas requirement, as it may be higher than the amount used
func DoEstimateGas(ctx context.Context, b Backend, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *StateOverride, blockOverrides *BlockOverrides, gasCap uint64) (hexutil.Uint64, error) {
	// Binary search the gas requirement, as it may be higher than the amount used
	var (
		lo  uint64 = params.TxGas - 1
		hi  uint64
		cap uint64
	)
	// Use zero address if sender unspecified.
	if args.From == nil {
		args.From = new(common.Address)
	}
	// Determine the highest gas limit can be used during the estimation.
	if args.Gas != nil && uint64(*args.Gas) >= params.TxGas {
		hi = uint64(*args.Gas)
	} else {
		hi = b.MaxGasLimit()
	}
	// Normalize the max fee per gas the call is willing to spend.
	var feeCap *big.Int
	if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return 0, errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	} else if args.GasPrice != nil {
		feeCap = args.GasPrice.ToInt()
	} else if args.MaxFeePerGas != nil {
		feeCap = args.MaxFeePerGas.ToInt()
	} else {
		feeCap = common.Big0
	}
	// Recap the highest gas limit with account's available balance.
	if feeCap.BitLen() != 0 {
		state, _, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
		if state == nil || err != nil {
			return 0, err
		}
		defer state.Release()
		if err := overrides.Apply(state); err != nil {
			return 0, err
		}
		balance := state.GetBalance(*args.From) // from can't be nil
		available := utils.Uint256ToBigInt(balance)
		if args.Value != nil {
			if args.Value.ToInt().Cmp(available) >= 0 {
				return 0, errors.New("insufficient funds for transfer")
			}
			available.Sub(available, args.Value.ToInt())
		}
		allowance := new(big.Int).Div(available, feeCap)

		// If the allowance is larger than maximum uint64, skip checking
		if allowance.IsUint64() && hi > allowance.Uint64() {
			transfer := args.Value
			if transfer == nil {
				transfer = new(hexutil.Big)
			}
			log.Warn("Gas estimation capped by limited funds", "original", hi, "balance", balance,
				"sent", transfer.ToInt(), "maxFeePerGas", feeCap, "fundable", allowance)
			hi = allowance.Uint64()
		}
	}
	// Recap the highest gas allowance with specified gascap.
	if gasCap != 0 && hi > gasCap {
		log.Warn("Caller gas above allowance, capping", "requested", hi, "cap", gasCap)
		hi = gasCap
	}
	cap = hi

	// Create a helper to check if a gas allowance results in an executable transaction
	executable := func(gas uint64) (bool, *core.ExecutionResult, error) {
		args.Gas = (*hexutil.Uint64)(&gas)

		result, err := DoCall(ctx, b, args, blockNrOrHash, overrides, blockOverrides, 0, gasCap)
		if err != nil {
			if errors.Is(err, core.ErrIntrinsicGas) ||
				errors.Is(err, core.ErrFloorDataGas) {
				return true, nil, nil // Special case, raise gas limit
			}
			return true, nil, err // Bail out
		}
		return result.Failed(), result, nil
	}
	// Execute the binary search and hone in on an executable gas limit
	for lo+1 < hi {
		mid := (hi + lo) / 2
		failed, _, err := executable(mid)

		// If the error is not nil(consensus error), it means the provided message
		// call or transaction will never be accepted no matter how much gas it is
		// assigned. Return the error directly, don't struggle any more.
		if err != nil {
			return 0, err
		}
		if failed {
			lo = mid
		} else {
			hi = mid
		}
	}
	// Reject the transaction as invalid if it still fails at the highest allowance
	if hi == cap {
		failed, result, err := executable(hi)
		if err != nil {
			return 0, err
		}
		if failed {
			if result != nil && result.Err != vm.ErrOutOfGas {
				if len(result.Revert()) > 0 {
					return 0, newRevertError(result)
				}
				return 0, result.Err
			}
			// Otherwise, the specified gas cap is too low
			return 0, fmt.Errorf("gas required exceeds allowance (%d)", cap)
		}
	}
	return hexutil.Uint64(hi), nil
}

// EstimateGas returns an estimate of the amount of gas needed to execute the
// given transaction against the current pending block.
func (s *PublicBlockChainAPI) EstimateGas(ctx context.Context, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash, overrides *StateOverride, blockOverrides *BlockOverrides) (hexutil.Uint64, error) {
	bNrOrHash := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	if blockNrOrHash != nil {
		bNrOrHash = *blockNrOrHash
	}
	return DoEstimateGas(ctx, s.b, args, bNrOrHash, overrides, blockOverrides, s.b.RPCGasCap())
}

// RPCMarshalBlock converts the given block to the RPC output which depends on fullTx. If inclTx is true transactions are
// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
// transaction hashes.
func RPCMarshalBlock(block *evmcore.EvmBlock, receipts types.Receipts, inclTx bool, fullTx bool) (*evmcore.EvmBlockJson, error) {
	size := hexutil.Uint64(block.EthBlock().Size()) // RPC encoded storage size
	json := &evmcore.EvmBlockJson{
		EvmHeaderJson: block.Header().ToJson(receipts),
		Size:          &size,
	}

	if inclTx {
		formatTx := func(tx *types.Transaction) (interface{}, error) {
			return tx.Hash(), nil
		}
		if fullTx {
			formatTx = func(tx *types.Transaction) (interface{}, error) {
				return newRPCTransactionFromBlockHash(block, tx.Hash()), nil
			}
		}
		txs := block.Transactions
		transactions := make([]interface{}, len(txs))
		var err error
		for i, tx := range txs {
			if transactions[i], err = formatTx(tx); err != nil {
				return nil, err
			}
		}
		json.Txs = transactions
	}
	json.Uncles = make([]common.Hash, 0)

	return json, nil
}

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
	BlockHash           *common.Hash                 `json:"blockHash"`
	BlockNumber         *hexutil.Big                 `json:"blockNumber"`
	From                common.Address               `json:"from"`
	Gas                 hexutil.Uint64               `json:"gas"`
	GasPrice            *hexutil.Big                 `json:"gasPrice"`
	GasFeeCap           *hexutil.Big                 `json:"maxFeePerGas,omitempty"`
	GasTipCap           *hexutil.Big                 `json:"maxPriorityFeePerGas,omitempty"`
	Hash                common.Hash                  `json:"hash"`
	Input               hexutil.Bytes                `json:"input"`
	Nonce               hexutil.Uint64               `json:"nonce"`
	To                  *common.Address              `json:"to"`
	TransactionIndex    *hexutil.Uint64              `json:"transactionIndex"`
	Value               *hexutil.Big                 `json:"value"`
	Type                hexutil.Uint64               `json:"type"`
	Accesses            *types.AccessList            `json:"accessList,omitempty"`
	ChainID             *hexutil.Big                 `json:"chainId,omitempty"`
	V                   *hexutil.Big                 `json:"v"`
	R                   *hexutil.Big                 `json:"r"`
	S                   *hexutil.Big                 `json:"s"`
	MaxFeePerBlobGas    *hexutil.Big                 `json:"maxFeePerBlobGas"`
	BlobVersionedHashes []common.Hash                `json:"blobVersionedHashes"`
	AuthorizationList   []types.SetCodeAuthorization `json:"authorizationList,omitempty"`
}

// newRPCTransaction returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func newRPCTransaction(tx *types.Transaction, blockHash common.Hash, blockNumber uint64, index uint64, baseFee *big.Int) *RPCTransaction {
	// Determine the signer. For replay-protected transactions, use the most permissive
	// signer, because we assume that signers are backwards-compatible with old
	// transactions. For non-protected transactions, the homestead signer signer is used
	// because the return value of ChainId is zero for those transactions.
	var signer types.Signer
	if tx.Protected() {
		signer = gsignercache.Wrap(types.LatestSignerForChainID(tx.ChainId()))
	} else {
		signer = gsignercache.Wrap(types.HomesteadSigner{})
	}
	from, _ := internaltx.Sender(signer, tx)
	v, r, s := tx.RawSignatureValues()
	result := &RPCTransaction{
		Type:     hexutil.Uint64(tx.Type()),
		From:     from,
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    hexutil.Bytes(tx.Data()),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		ChainID:  (*hexutil.Big)(tx.ChainId()),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
	}
	if blockHash != (common.Hash{}) {
		result.BlockHash = &blockHash
		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(blockNumber))
		result.TransactionIndex = (*hexutil.Uint64)(&index)
	}

	copyAccessList := func(tx *types.Transaction, result *RPCTransaction) {
		al := tx.AccessList()
		result.Accesses = &al
	}

	copyDynamicPricingFields := func(tx *types.Transaction, result *RPCTransaction) {
		result.GasFeeCap = (*hexutil.Big)(tx.GasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.GasTipCap())
		// if the transaction has been mined, compute the effective gas price
		if baseFee != nil && blockHash != (common.Hash{}) {
			// price = min(tip, gasFeeCap - baseFee) + baseFee
			price := utils.BigMin(new(big.Int).Add(tx.GasTipCap(), baseFee), tx.GasFeeCap())
			result.GasPrice = (*hexutil.Big)(price)
		} else {
			result.GasPrice = (*hexutil.Big)(tx.GasFeeCap())
		}
	}

	copyBlobFields := func(tx *types.Transaction, result *RPCTransaction) {
		result.MaxFeePerBlobGas = (*hexutil.Big)(tx.BlobGasFeeCap())
		result.BlobVersionedHashes = tx.BlobHashes()
		if result.BlobVersionedHashes == nil {
			result.BlobVersionedHashes = make([]common.Hash, 0)
		}
	}

	copyAuthorizationList := func(tx *types.Transaction, result *RPCTransaction) {
		result.AuthorizationList = tx.SetCodeAuthorizations()
	}

	switch tx.Type() {
	case types.AccessListTxType:
		copyAccessList(tx, result)
	case types.DynamicFeeTxType:
		copyAccessList(tx, result)
		copyDynamicPricingFields(tx, result)
	case types.BlobTxType:
		// BLOB NOTE: the current sonic network supports blobTx so long as they don not contain blobs
		// for this reason they are equivalent to the dynamic fee tx type
		copyAccessList(tx, result)
		copyDynamicPricingFields(tx, result)
		copyBlobFields(tx, result)
	case types.SetCodeTxType:
		copyAccessList(tx, result)
		copyDynamicPricingFields(tx, result)
		copyAuthorizationList(tx, result)
	}
	return result
}

// newRPCPendingTransaction returns a pending transaction that will serialize to the RPC representation
func newRPCPendingTransaction(tx *types.Transaction, baseFee *big.Int) *RPCTransaction {
	return newRPCTransaction(tx, common.Hash{}, 0, 0, baseFee)
}

// newRPCTransactionFromBlockIndex returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockIndex(b *evmcore.EvmBlock, index uint64) *RPCTransaction {
	txs := b.Transactions
	if index >= uint64(len(txs)) {
		return nil
	}
	return newRPCTransaction(txs[index], b.Hash, b.NumberU64(), index, b.BaseFee)
}

// newRPCRawTransactionFromBlockIndex returns the bytes of a transaction given a block and a transaction index.
func newRPCRawTransactionFromBlockIndex(b *evmcore.EvmBlock, index uint64) hexutil.Bytes {
	txs := b.Transactions
	if index >= uint64(len(txs)) {
		return nil
	}
	blob, _ := txs[index].MarshalBinary()
	return blob
}

// newRPCTransactionFromBlockHash returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockHash(b *evmcore.EvmBlock, hash common.Hash) *RPCTransaction {
	for idx, tx := range b.Transactions {
		if tx.Hash() == hash {
			return newRPCTransactionFromBlockIndex(b, uint64(idx))
		}
	}
	return nil
}

// accessListResult returns an optional accesslist
// Its the result of the `debug_createAccessList` RPC call.
// It contains an error if the transaction itself failed.
type accessListResult struct {
	Accesslist *types.AccessList `json:"accessList"`
	Error      string            `json:"error,omitempty"`
	GasUsed    hexutil.Uint64    `json:"gasUsed"`
}

// CreateAccessList creates a EIP-2930 type AccessList for the given transaction.
// Reexec and BlockNrOrHash can be specified to create the accessList on top of a certain state.
func (s *PublicBlockChainAPI) CreateAccessList(ctx context.Context, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash) (*accessListResult, error) {
	bNrOrHash := rpc.BlockNumberOrHashWithNumber(rpc.PendingBlockNumber)
	if blockNrOrHash != nil {
		bNrOrHash = *blockNrOrHash
	}
	acl, gasUsed, vmerr, err := AccessList(ctx, s.b, bNrOrHash, args)
	if err != nil {
		return nil, err
	}
	result := &accessListResult{Accesslist: &acl, GasUsed: hexutil.Uint64(gasUsed)}
	if vmerr != nil {
		result.Error = vmerr.Error()
	}
	return result, nil
}

// AccessList creates an access list for the given transaction.
// If the accesslist creation fails an error is returned.
// If the transaction itself fails, an vmErr is returned.
func AccessList(ctx context.Context, b Backend, blockNrOrHash rpc.BlockNumberOrHash, args TransactionArgs) (acl types.AccessList, gasUsed uint64, vmErr error, err error) {
	// Retrieve the execution context
	db, header, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if db == nil || err != nil {
		return nil, 0, nil, err
	}
	defer db.Release()
	// If the gas amount is not set, extract this as it will depend on access
	// lists and we'll need to reestimate every time
	nogas := args.Gas == nil

	// Ensure any missing fields are filled, extract the recipient and input data
	if err := args.setDefaults(ctx, b); err != nil {
		return nil, 0, nil, err
	}
	var to common.Address
	if args.To != nil {
		to = *args.To
	} else {
		to = crypto.CreateAddress(args.from(), uint64(*args.Nonce))
	}
	// Retrieve the precompiles since they don't need to be added to the access list
	chainConfig := b.ChainConfig(idx.Block(header.Number.Uint64()))
	precompiles := vm.ActivePrecompiles(chainConfig.Rules(header.Number, false, uint64(header.Time.Unix())))

	// addressesToExclude contains sender, receiver and precompiles
	addressesToExclude := map[common.Address]struct{}{args.from(): {}, to: {}}
	for _, addr := range precompiles {
		addressesToExclude[addr] = struct{}{}
	}

	// Create an initial tracer
	prevTracer := logger.NewAccessListTracer(nil, addressesToExclude)
	if args.AccessList != nil {
		prevTracer = logger.NewAccessListTracer(*args.AccessList, addressesToExclude)
	}
	for {
		// Retrieve the current access list to expand
		accessList := prevTracer.AccessList()
		log.Trace("Creating access list", "input", accessList)

		// If no gas amount was specified, each unique access list needs it's own
		// gas calculation. This is quite expensive, but we need to be accurate
		// and it's convered by the sender only anyway.
		if nogas {
			args.Gas = nil
			if err := args.setDefaults(ctx, b); err != nil {
				return nil, 0, nil, err // shouldn't happen, just in case
			}
		}
		// Copy the original db so we don't modify it
		statedb := db.Copy()
		// Set the accesslist to the last al
		args.AccessList = &accessList
		msg, err := args.ToMessage(b.RPCGasCap(), header.BaseFee, log.Root())
		if err != nil {
			statedb.Release()
			return nil, 0, nil, err
		}

		// Apply the transaction with the access list tracer
		tracer := logger.NewAccessListTracer(accessList, addressesToExclude)
		config, err := GetVmConfig(ctx, b, idx.Block(header.Number.Uint64()))
		if err != nil {
			return nil, 0, nil, err
		}
		config.Tracer = tracer.Hooks()
		config.NoBaseFee = true
		vmenv, _, err := b.GetEVM(ctx, statedb, header, &config, nil)
		if err != nil {
			statedb.Release()
			return nil, 0, nil, err
		}
		res, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(msg.GasLimit))
		statedb.Release()
		if err != nil {
			return nil, 0, nil, fmt.Errorf("failed to apply transaction: %v err: %v", args.toTransaction().Hash(), err)
		}
		if tracer.Equal(prevTracer) {
			return accessList, res.UsedGas, res.Err, nil
		}
		prevTracer = tracer
	}
}

// PublicTransactionPoolAPI exposes methods for the RPC interface
type PublicTransactionPoolAPI struct {
	b         Backend
	nonceLock *AddrLocker
	signer    types.Signer
}

// NewPublicTransactionPoolAPI creates a new RPC service with methods specific for the transaction pool.
func NewPublicTransactionPoolAPI(b Backend, nonceLock *AddrLocker) *PublicTransactionPoolAPI {
	// The signer used by the API should always be the 'latest' known one because we expect
	// signers to be backwards-compatible with old transactions.
	chainID := b.ChainID()
	signer := gsignercache.Wrap(types.LatestSignerForChainID(chainID))
	return &PublicTransactionPoolAPI{b, nonceLock, signer}
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block with the given block number.
func (s *PublicTransactionPoolAPI) GetBlockTransactionCountByNumber(ctx context.Context, blockNr rpc.BlockNumber) *hexutil.Uint {
	if block, _ := s.b.BlockByNumber(ctx, blockNr); block != nil {
		n := hexutil.Uint(len(block.Transactions))
		return &n
	}
	return nil
}

// GetBlockTransactionCountByHash returns the number of transactions in the block with the given hash.
func (s *PublicTransactionPoolAPI) GetBlockTransactionCountByHash(ctx context.Context, blockHash common.Hash) *hexutil.Uint {
	if block, _ := s.b.BlockByHash(ctx, blockHash); block != nil {
		n := hexutil.Uint(len(block.Transactions))
		return &n
	}
	return nil
}

// GetTransactionByBlockNumberAndIndex returns the transaction for the given block number and index.
func (s *PublicTransactionPoolAPI) GetTransactionByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) *RPCTransaction {
	if block, _ := s.b.BlockByNumber(ctx, blockNr); block != nil {
		return newRPCTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetTransactionByBlockHashAndIndex returns the transaction for the given block hash and index.
func (s *PublicTransactionPoolAPI) GetTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) *RPCTransaction {
	if block, _ := s.b.BlockByHash(ctx, blockHash); block != nil {
		return newRPCTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetRawTransactionByBlockNumberAndIndex returns the bytes of the transaction for the given block number and index.
func (s *PublicTransactionPoolAPI) GetRawTransactionByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) hexutil.Bytes {
	if block, _ := s.b.BlockByNumber(ctx, blockNr); block != nil {
		return newRPCRawTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetRawTransactionByBlockHashAndIndex returns the bytes of the transaction for the given block hash and index.
func (s *PublicTransactionPoolAPI) GetRawTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) hexutil.Bytes {
	if block, _ := s.b.BlockByHash(ctx, blockHash); block != nil {
		return newRPCRawTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetTransactionCount returns the number of transactions the given address has sent for the given block number
func (s *PublicTransactionPoolAPI) GetTransactionCount(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Uint64, error) {
	// Ask transaction pool for the nonce which includes pending transactions
	if blockNr, ok := blockNrOrHash.Number(); ok && blockNr == rpc.PendingBlockNumber {
		nonce, err := s.b.GetPoolNonce(ctx, address)
		if err != nil {
			return nil, err
		}
		return (*hexutil.Uint64)(&nonce), nil
	}
	// Resolve block number and use its state to ask for the nonce
	state, _, err := s.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	defer state.Release()
	nonce := state.GetNonce(address)
	return (*hexutil.Uint64)(&nonce), state.Error()
}

// GetTransactionByHash returns the transaction for the given hash
func (s *PublicTransactionPoolAPI) GetTransactionByHash(ctx context.Context, hash common.Hash) (*RPCTransaction, error) {
	// Try to return an already finalized transaction
	tx, blockNumber, index, err := s.b.GetTransaction(ctx, hash)
	if err != nil {
		return nil, err
	}
	if tx != nil {
		header, err := s.b.HeaderByNumber(ctx, rpc.BlockNumber(blockNumber))
		if header == nil || err != nil {
			return nil, err
		}
		return newRPCTransaction(tx, header.Hash, blockNumber, index, header.BaseFee), nil
	}
	// No finalized transaction, try to retrieve it from the pool
	if tx := s.b.GetPoolTransaction(hash); tx != nil {
		return newRPCPendingTransaction(tx, s.b.MinGasPrice()), nil
	}

	// Transaction unknown, return as such
	return nil, nil
}

// GetRawTransactionByHash returns the bytes of the transaction for the given hash.
func (s *PublicTransactionPoolAPI) GetRawTransactionByHash(ctx context.Context, hash common.Hash) (hexutil.Bytes, error) {
	// Retrieve a finalized transaction, or a pooled otherwise
	tx, _, _, err := s.b.GetTransaction(ctx, hash)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		if tx = s.b.GetPoolTransaction(hash); tx == nil {
			// Transaction not found anywhere, abort
			return nil, nil
		}
	}
	// Serialize to RLP and return
	return tx.MarshalBinary()
}

// formatTxReceipt encodes transaction receipt into the expected API output.
func (s *PublicTransactionPoolAPI) formatTxReceipt(header *evmcore.EvmHeader, tx *types.Transaction, txIndex uint64, receipt *types.Receipt) map[string]interface{} {
	// Clone the logs before adding transaction meta data to avoid data races
	// due to concurrent accesses.
	logs := slices.Clone(receipt.Logs)
	for i := range logs {
		l := new(types.Log)
		*l = *logs[i] // shallow copy
		logs[i] = l
		l.TxHash = tx.Hash()
		l.BlockHash = header.Hash
		l.BlockNumber = header.Number.Uint64()
		l.TxIndex = uint(txIndex) /* logs cache poisoning hot fix */
		if l.Topics == nil {
			l.Topics = []common.Hash{}
		}
	}

	// Derive the sender.
	signer := gsignercache.Wrap(types.MakeSigner(s.b.ChainConfig(idx.Block(header.Number.Uint64())), header.Number, uint64(header.Time.Unix())))
	from, _ := internaltx.Sender(signer, tx)

	fields := map[string]interface{}{
		"blockHash":         header.Hash,
		"blockNumber":       hexutil.Uint64(header.Number.Uint64()),
		"transactionHash":   tx.Hash(),
		"transactionIndex":  hexutil.Uint64(txIndex),
		"from":              from,
		"to":                tx.To(),
		"gasUsed":           hexutil.Uint64(receipt.GasUsed),
		"cumulativeGasUsed": hexutil.Uint64(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              logs,
		"logsBloom":         &receipt.Bloom,
		"type":              hexutil.Uint(tx.Type()),
	}
	// Assign the effective gas price paid
	if header.BaseFee == nil {
		fields["effectiveGasPrice"] = hexutil.Uint64(tx.GasPrice().Uint64())
	} else {
		// EffectiveGasTip returns an error for negative values, this is no problem here
		gasTip, _ := tx.EffectiveGasTip(header.BaseFee)
		gasPrice := new(big.Int).Add(header.BaseFee, gasTip)
		fields["effectiveGasPrice"] = hexutil.Uint64(gasPrice.Uint64())
	}
	// Assign receipt status or post state.
	if len(receipt.PostState) > 0 {
		fields["root"] = hexutil.Bytes(receipt.PostState)
	} else {
		fields["status"] = hexutil.Uint(receipt.Status)
	}
	if receipt.Logs == nil {
		fields["logs"] = [][]*types.Log{}
	}
	// Transactions without a recipient deploy a contract.
	if tx.To() == nil {
		fields["contractAddress"] = receipt.ContractAddress
	}

	return fields
}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (s *PublicTransactionPoolAPI) GetTransactionReceipt(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
	tx, blockNumber, index, err := s.b.GetTransaction(ctx, hash)
	if tx == nil || err != nil {
		return nil, err
	}
	header, err := s.b.HeaderByNumber(ctx, rpc.BlockNumber(blockNumber)) // retrieve header to get block hash
	if header == nil || err != nil {
		return nil, err
	}
	receipts, err := s.b.GetReceiptsByNumber(ctx, rpc.BlockNumber(blockNumber))
	if receipts == nil || err != nil {
		return nil, err
	}
	if receipts.Len() <= int(index) {
		return nil, nil
	}
	return s.formatTxReceipt(header, tx, index, receipts[index]), nil
}

// GetBlockReceipts returns a set of transaction receipts for the given block by the extended block number.
func (s *PublicTransactionPoolAPI) GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]map[string]interface{}, error) {

	var (
		err    error
		number rpc.BlockNumber
		header *evmcore.EvmHeader
	)

	if blockNr, ok := blockNrOrHash.Number(); ok {
		number = blockNr
		header, err = s.b.HeaderByNumber(ctx, number)
		if header == nil || err != nil {
			return nil, err
		}
	} else if blockHash, ok := blockNrOrHash.Hash(); ok {
		header, err = s.b.HeaderByHash(ctx, blockHash)
		if header == nil || err != nil {
			return nil, err
		}
		number = rpc.BlockNumber(header.Number.Uint64())
	}

	receipts, err := s.b.GetReceiptsByNumber(ctx, number)
	if receipts == nil || err != nil {
		return nil, err
	}

	blkReceipts := make([]map[string]interface{}, len(receipts))
	for i, receipt := range receipts {
		tx, _, index, err := s.b.GetTransaction(ctx, receipt.TxHash)
		if err != nil {
			return nil, err
		}
		blkReceipts[i] = s.formatTxReceipt(header, tx, index, receipt)
	}

	return blkReceipts, nil
}

// sign is a helper function that signs a transaction with the private key of the given address.
func (s *PublicTransactionPoolAPI) sign(addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := s.b.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Request the wallet to sign the transaction
	chainID := s.b.ChainID()
	return wallet.SignTx(account, tx, chainID)
}

// SubmitTransaction is a helper function that submits tx to txPool and logs a message.
func SubmitTransaction(ctx context.Context, b Backend, tx *types.Transaction) (common.Hash, error) {
	// If the transaction fee cap is already specified, ensure the
	// fee of the given transaction is _reasonable_.
	if err := checkTxFee(tx.GasPrice(), tx.Gas(), b.RPCTxFeeCap()); err != nil {
		return common.Hash{}, err
	}
	if !b.UnprotectedAllowed() && !tx.Protected() {
		// Ensure only eip155 signed transactions are submitted if EIP155Required is set.
		return common.Hash{}, errors.New("only replay-protected (EIP-155) transactions allowed over RPC")
	}
	if err := b.SendTx(ctx, tx); err != nil {
		return common.Hash{}, err
	} // Print a log with full tx details for manual investigations and interventions
	chainConfig := b.ChainConfig(idx.Block(b.CurrentBlock().Number.Uint64()))
	signer := gsignercache.Wrap(types.MakeSigner(chainConfig, b.CurrentBlock().Number, uint64(b.CurrentBlock().Time.Unix())))
	from, err := types.Sender(signer, tx)
	if err != nil {
		return common.Hash{}, err
	}

	if tx.To() == nil {
		addr := crypto.CreateAddress(from, tx.Nonce())
		log.Debug("Submitted contract creation", "hash", tx.Hash().Hex(), "from", from, "nonce", tx.Nonce(), "contract", addr.Hex(), "value", tx.Value())
	} else {
		log.Debug("Submitted transaction", "hash", tx.Hash().Hex(), "from", from, "nonce", tx.Nonce(), "recipient", tx.To(), "value", tx.Value())
	}
	return tx.Hash(), nil
}

// SendTransaction creates a transaction for the given argument, sign it and submit it to the
// transaction pool.
func (s *PublicTransactionPoolAPI) SendTransaction(ctx context.Context, args TransactionArgs) (common.Hash, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: args.from()}

	wallet, err := s.b.AccountManager().Find(account)
	if err != nil {
		return common.Hash{}, err
	}

	if args.Nonce == nil {
		// Hold the addresse's mutex around signing to prevent concurrent assignment of
		// the same nonce to multiple accounts.
		s.nonceLock.LockAddr(args.from())
		defer s.nonceLock.UnlockAddr(args.from())
	}

	// Set some sanity defaults and terminate on failure
	if err := args.setDefaults(ctx, s.b); err != nil {
		return common.Hash{}, err
	}
	// Assemble the transaction and sign with the wallet
	tx := args.toTransaction()

	chainID := s.b.ChainID()
	signed, err := wallet.SignTx(account, tx, chainID)
	if err != nil {
		return common.Hash{}, err
	}
	return SubmitTransaction(ctx, s.b, signed)
}

// FillTransaction fills the defaults (nonce, gas, gasPrice or 1559 fields)
// on a given unsigned transaction, and returns it to the caller for further
// processing (signing + broadcast).
func (s *PublicTransactionPoolAPI) FillTransaction(ctx context.Context, args TransactionArgs) (*SignTransactionResult, error) {
	// Set some sanity defaults and terminate on failure
	if err := args.setDefaults(ctx, s.b); err != nil {
		return nil, err
	}
	// Assemble the transaction and obtain rlp
	tx := args.toTransaction()
	data, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return &SignTransactionResult{data, tx}, nil
}

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (s *PublicTransactionPoolAPI) SendRawTransaction(ctx context.Context, encodedTx hexutil.Bytes) (common.Hash, error) {
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(encodedTx); err != nil {
		return common.Hash{}, err
	}
	return SubmitTransaction(ctx, s.b, tx)
}

// Sign calculates an ECDSA signature for:
// keccack256("\x19Ethereum Signed Message:\n" + len(message) + message).
//
// Note, the produced signature conforms to the secp256k1 curve R, S and V values,
// where the V value will be 27 or 28 for legacy reasons.
//
// The account associated with addr must be unlocked.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_sign
func (s *PublicTransactionPoolAPI) Sign(addr common.Address, data hexutil.Bytes) (hexutil.Bytes, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := s.b.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Sign the requested hash with the wallet
	signature, err := wallet.SignText(account, data)
	if err == nil {
		signature[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	}
	return signature, err
}

// SignTransactionResult represents a RLP encoded signed transaction.
type SignTransactionResult struct {
	Raw hexutil.Bytes      `json:"raw"`
	Tx  *types.Transaction `json:"tx"`
}

// SignTransaction will sign the given transaction with the from account.
// The node needs to have the private key of the account corresponding with
// the given from address and it needs to be unlocked.
func (s *PublicTransactionPoolAPI) SignTransaction(ctx context.Context, args TransactionArgs) (*SignTransactionResult, error) {
	if args.Gas == nil {
		return nil, fmt.Errorf("gas not specified")
	}
	if args.GasPrice == nil && (args.MaxPriorityFeePerGas == nil || args.MaxFeePerGas == nil) {
		return nil, fmt.Errorf("missing gasPrice or maxFeePerGas/maxPriorityFeePerGas")
	}
	if args.Nonce == nil {
		return nil, fmt.Errorf("nonce not specified")
	}
	if err := args.setDefaults(ctx, s.b); err != nil {
		return nil, err
	}
	// Before actually sign the transaction, ensure the transaction fee is reasonable.
	tx := args.toTransaction()
	if err := checkTxFee(tx.GasPrice(), tx.Gas(), s.b.RPCTxFeeCap()); err != nil {
		return nil, err
	}
	signed, err := s.sign(args.from(), tx)
	if err != nil {
		return nil, err
	}
	data, err := signed.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return &SignTransactionResult{data, signed}, nil
}

// PendingTransactions returns the transactions that are in the transaction pool
// and have a from address that is one of the accounts this node manages.
func (s *PublicTransactionPoolAPI) PendingTransactions() ([]*RPCTransaction, error) {
	pending, err := s.b.GetPoolTransactions()
	if err != nil {
		return nil, err
	}
	accounts := make(map[common.Address]struct{})
	for _, wallet := range s.b.AccountManager().Wallets() {
		for _, account := range wallet.Accounts() {
			accounts[account.Address] = struct{}{}
		}
	}
	transactions := make([]*RPCTransaction, 0, len(pending))
	for _, tx := range pending {
		from, _ := internaltx.Sender(s.signer, tx)
		if _, exists := accounts[from]; exists {
			transactions = append(transactions, newRPCPendingTransaction(tx, s.b.MinGasPrice()))
		}
	}
	return transactions, nil
}

// Resend accepts an existing transaction and a new gas price and limit. It will remove
// the given transaction from the pool and reinsert it with the new gas price and limit.
func (s *PublicTransactionPoolAPI) Resend(ctx context.Context, sendArgs TransactionArgs, gasPrice *hexutil.Big, gasLimit *hexutil.Uint64) (common.Hash, error) {
	if sendArgs.Nonce == nil {
		return common.Hash{}, fmt.Errorf("missing transaction nonce in transaction spec")
	}
	if err := sendArgs.setDefaults(ctx, s.b); err != nil {
		return common.Hash{}, err
	}
	matchTx := sendArgs.toTransaction()

	// Before replacing the old transaction, ensure the _new_ transaction fee is reasonable.
	var price = matchTx.GasPrice()
	if gasPrice != nil {
		price = gasPrice.ToInt()
	}
	var gas = matchTx.Gas()
	if gasLimit != nil {
		gas = uint64(*gasLimit)
	}
	if err := checkTxFee(price, gas, s.b.RPCTxFeeCap()); err != nil {
		return common.Hash{}, err
	}
	// Iterate the pending list for replacement
	pending, err := s.b.GetPoolTransactions()
	if err != nil {
		return common.Hash{}, err
	}

	for _, p := range pending {
		wantSigHash := s.signer.Hash(matchTx)
		pFrom, err := types.Sender(s.signer, p)
		if err == nil && pFrom == sendArgs.from() && s.signer.Hash(p) == wantSigHash {
			// Match. Re-sign and send the transaction.
			if gasPrice != nil && (*big.Int)(gasPrice).Sign() != 0 {
				sendArgs.GasPrice = gasPrice
			}
			if gasLimit != nil && *gasLimit != 0 {
				sendArgs.Gas = gasLimit
			}
			signedTx, err := s.sign(sendArgs.from(), sendArgs.toTransaction())
			if err != nil {
				return common.Hash{}, err
			}
			if err = s.b.SendTx(ctx, signedTx); err != nil {
				return common.Hash{}, err
			}
			return signedTx.Hash(), nil
		}
	}
	return common.Hash{}, fmt.Errorf("transaction %#x not found", matchTx.Hash())
}

// PublicDebugAPI is the collection of Ethereum APIs exposed over the public
// debugging endpoint.
type PublicDebugAPI struct {
	b               Backend
	maxResponseSize int // in bytes
	structLogLimit  int
}

// NewPublicDebugAPI creates a new API definition for the public debug methods
// of the Ethereum service.
func NewPublicDebugAPI(b Backend, maxResponseSize int, structLogLimit int) *PublicDebugAPI {
	return &PublicDebugAPI{
		b:               b,
		maxResponseSize: maxResponseSize,
		structLogLimit:  structLogLimit,
	}
}

// GetBlockRlp retrieves the RLP encoded for of a single block.
func (api *PublicDebugAPI) GetBlockRlp(ctx context.Context, number uint64) (string, error) {
	block, _ := api.b.BlockByNumber(ctx, rpc.BlockNumber(number))
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	encoded, err := rlp.EncodeToBytes(block)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", encoded), nil
}

// TestSignCliqueBlock fetches the given block number, and attempts to sign it as a clique header with the
// given address, returning the address of the recovered signature
//
// This is a temporary method to debug the externalsigner integration,
func (api *PublicDebugAPI) TestSignCliqueBlock(ctx context.Context, address common.Address, number uint64) (common.Address, error) {
	// This is a user-facing error, so we want to provide a clear message.
	//nolint:staticcheck // ST1005: allow capitalized error message and punctuation
	return common.Address{}, errors.New("Clique isn't supported")
}

// PrintBlock retrieves a block and returns its pretty printed form.
func (api *PublicDebugAPI) PrintBlock(ctx context.Context, number uint64) (string, error) {
	block, err := api.b.BlockByNumber(ctx, rpc.BlockNumber(number))
	if err != nil {
		return "", err
	}
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	return spew.Sdump(block), nil
}

// BlocksTransactionTimes returns the map time => number of transactions
// This data may be used to draw a histogram to calculate a peak TPS of a range of blocks
func (api *PublicDebugAPI) BlocksTransactionTimes(ctx context.Context, untilBlock rpc.BlockNumber, maxBlocks hexutil.Uint64) (map[hexutil.Uint64]hexutil.Uint, error) {

	until, err := api.b.HeaderByNumber(ctx, untilBlock)
	if until == nil || err != nil {
		return nil, err
	}
	untilN := until.Number.Uint64()
	times := map[hexutil.Uint64]hexutil.Uint{}
	for i := untilN; i >= 1 && i+uint64(maxBlocks) > untilN; i-- {
		b, err := api.b.BlockByNumber(ctx, rpc.BlockNumber(i))
		if b == nil || err != nil {
			return nil, err
		}
		if b.Transactions.Len() == 0 {
			continue
		}
		times[hexutil.Uint64(b.Time)] += hexutil.Uint(b.Transactions.Len())
	}

	return times, nil
}

// TraceTransaction returns the structured logs created during the execution of EVM
// and returns them as a JSON object.
func (api *PublicDebugAPI) TraceTransaction(ctx context.Context, hash common.Hash, config *tracers.TraceConfig) (interface{}, error) {
	tx, blockNumber, index, err := api.b.GetTransaction(ctx, hash)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		return nil, fmt.Errorf("transaction %s not found", hash.Hex())
	}
	// It shouldn't happen in practice.
	if blockNumber == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	block, err := api.b.BlockByNumber(ctx, rpc.BlockNumber(blockNumber))
	if err != nil {
		return nil, err
	}
	msg, statedb, err := stateAtTransaction(ctx, block, int(index), api.b)
	if err != nil {
		return nil, err
	}
	defer statedb.Release()

	txctx := &tracers.Context{
		BlockHash:   block.Hash,
		BlockNumber: block.Number,
		TxIndex:     int(index),
		TxHash:      hash,
	}

	return api.traceTx(ctx, tx, msg, txctx, block.Header(), statedb, config, nil)
}

// traceTx configures a new tracer according to the provided configuration, and
// executes the given message in the provided environment. The return value will
// be tracer dependent.
func (api *PublicDebugAPI) traceTx(
	ctx context.Context,
	tx *types.Transaction,
	message *core.Message,
	txctx *tracers.Context,
	blockHeader *evmcore.EvmHeader,
	statedb state.StateDB,
	config *tracers.TraceConfig,
	blockCtx *vm.BlockContext,
) (json.RawMessage, error) {
	var (
		tracer  *tracers.Tracer
		err     error
		timeout = defaultTraceTimeout
		usedGas uint64
	)
	if config == nil {
		config = &tracers.TraceConfig{}
	}

	chainConfig := api.b.ChainConfig(idx.Block(blockHeader.Number.Uint64()))

	// Default tracer is the struct logger
	if config.Tracer == nil {
		if config.Config == nil {
			config.Config = &logger.Config{Limit: api.structLogLimit}
		} else {
			if api.structLogLimit > 0 &&
				(config.Limit == 0 || config.Limit > api.structLogLimit) {

				config.Limit = api.structLogLimit
			}
		}
		logger := logger.NewStructLogger(config.Config)
		tracer = &tracers.Tracer{
			Hooks:     logger.Hooks(),
			GetResult: logger.GetResult,
			Stop:      logger.Stop,
		}
	} else {
		tracer, err = tracers.DefaultDirectory.New(*config.Tracer, txctx, config.TracerConfig, chainConfig)
		if err != nil {
			return nil, err
		}
	}

	evmconfig, err := GetVmConfig(ctx, api.b, idx.Block(blockHeader.Number.Uint64()))
	if err != nil {
		return nil, fmt.Errorf("failed to get vm config: %w", err)
	}
	evmconfig.Tracer = tracer.Hooks
	evmconfig.NoBaseFee = true

	loggingStateDB := evmstore.WrapStateDbWithLogger(statedb, tracer.Hooks)

	vmenv, _, err := api.b.GetEVM(ctx, loggingStateDB, blockHeader, &evmconfig, blockCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get EVM for tracing: %w", err)
	}

	// Define a meaningful timeout of a single transaction trace
	if config.Timeout != nil {
		if timeout, err = time.ParseDuration(*config.Timeout); err != nil {
			return nil, err
		}
	}
	deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
	go func() {
		<-deadlineCtx.Done()
		if errors.Is(deadlineCtx.Err(), context.DeadlineExceeded) {
			tracer.Stop(errors.New("execution timeout"))
			// Stop evm execution. Note cancellation is not necessarily immediate.
			vmenv.Cancel()
		}
	}()
	defer cancel()

	// Call SetTxContext to clear out the statedb access list
	loggingStateDB.SetTxContext(txctx.TxHash, txctx.TxIndex)

	// Run the transaction with tracing enabled.
	_, err = evmcore.ApplyTransactionWithEVM(
		message,
		chainConfig,
		new(core.GasPool).AddGas(message.GasLimit),
		loggingStateDB,
		blockHeader.Number,
		txctx.BlockHash,
		tx,
		&usedGas,
		vmenv,
	)
	if err != nil {
		return nil, fmt.Errorf("tracing failed: %w", err)
	}

	result, err := tracer.GetResult()
	if err != nil {
		return nil, err
	}

	if api.maxResponseSize > 0 && len(result) > api.maxResponseSize {
		return nil, ErrMaxResponseSize
	}

	return result, nil
}

// txTraceResult is the result of a single transaction trace.
type txTraceResult struct {
	TxHash common.Hash `json:"txHash"`           // transaction hash
	Result interface{} `json:"result,omitempty"` // Trace results produced by the tracer
	Error  string      `json:"error,omitempty"`  // Trace failure produced by the tracer
}

// TraceBlockByNumber returns the structured logs created during the execution of
// EVM and returns them as a JSON object.
func (api *PublicDebugAPI) TraceBlockByNumber(ctx context.Context, number rpc.BlockNumber, config *tracers.TraceConfig) ([]*txTraceResult, error) {
	block, err := api.b.BlockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", number)
	}
	return api.traceBlock(ctx, block, config)
}

// TraceBlockByHash returns the structured logs created during the execution of
// EVM and returns them as a JSON object.
func (api *PublicDebugAPI) TraceBlockByHash(ctx context.Context, hash common.Hash, config *tracers.TraceConfig) ([]*txTraceResult, error) {
	block, err := api.b.BlockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block %s not found", hash.Hex())
	}
	return api.traceBlock(ctx, block, config)
}

// traceBlock configures a new tracer according to the provided configuration, and
// executes all the transactions contained within. The return value will be one item
// per transaction, dependent on the requested tracer.
func (api *PublicDebugAPI) traceBlock(ctx context.Context, block *evmcore.EvmBlock, config *tracers.TraceConfig) ([]*txTraceResult, error) {
	if block.NumberU64() == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	statedb, _, err := api.b.StateAndHeaderByNumberOrHash(ctx, rpc.BlockNumberOrHashWithHash(block.ParentHash, false))
	if err != nil {
		return nil, err
	}
	defer statedb.Release()

	var (
		chainConfig   = api.b.ChainConfig(idx.Block(block.Header().Number.Uint64()))
		txs           = block.Transactions
		signer        = gsignercache.Wrap(types.MakeSigner(chainConfig, block.Number, uint64(block.Time.Unix())))
		results       = make([]*txTraceResult, len(txs))
		resultsLength int
	)
	for i, tx := range txs {
		msg, _ := evmcore.TxAsMessage(tx, signer, block.BaseFee)
		txctx := &tracers.Context{
			BlockHash:   block.Hash,
			BlockNumber: block.Number,
			TxIndex:     i,
			TxHash:      tx.Hash(),
		}
		res, err := api.traceTx(ctx, tx, msg, txctx, block.Header(), statedb, config, nil)
		if err != nil {
			results[i] = &txTraceResult{TxHash: tx.Hash(), Error: err.Error()}
			resultsLength += len(err.Error())
		} else {
			results[i] = &txTraceResult{TxHash: tx.Hash(), Result: res}
			resultsLength += len(res)
		}
		statedb.EndTransaction()

		// limit the response size.
		if api.maxResponseSize > 0 && resultsLength > api.maxResponseSize {
			return nil, ErrMaxResponseSize
		}
	}
	return results, nil
}

// stateAtTransaction returns the execution environment of a certain transaction.
func stateAtTransaction(ctx context.Context, block *evmcore.EvmBlock, txIndex int, b Backend) (*core.Message, state.StateDB, error) {
	// Short circuit if it's genesis block.
	if block.NumberU64() == 0 {
		return nil, nil, errors.New("no transaction in genesis")
	}

	// Check correct txIndex
	if txIndex > len(block.Transactions) {
		return nil, nil, fmt.Errorf("transaction index %d out of range for block %#x", txIndex, block.Hash)
	}

	// Lookup the statedb of parent block from the live database,
	// otherwise regenerate it on the flight.
	statedb, _, err := b.StateAndHeaderByNumberOrHash(ctx, rpc.BlockNumberOrHashWithHash(block.ParentHash, false))
	if err != nil {
		return nil, nil, err
	}

	if txIndex == 0 && len(block.Transactions) == 0 {
		return nil, statedb, nil
	}

	// Use the block's VM config for replaying transactions with possible no base fee
	cfg, err := GetVmConfig(ctx, b, idx.Block(block.NumberU64()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get vm config: %w", err)
	}
	cfg.NoBaseFee = true
	vmenv, _, err := b.GetEVM(ctx, statedb, block.Header(), &cfg, nil)
	if err != nil {
		statedb.Release()
		return nil, nil, err
	}

	// execute EIP-2935 HistoryStorage contract.
	if vmenv.ChainConfig().IsPrague(block.Number, uint64(block.Time.Unix())) {
		evmcore.ProcessParentBlockHash(block.ParentHash, vmenv, statedb)
	}

	// Recompute transactions up to the target index.
	chainConfig := b.ChainConfig(idx.Block(block.NumberU64()))
	signer := gsignercache.Wrap(types.MakeSigner(chainConfig, block.Number, uint64(block.Time.Unix())))
	for idx, tx := range block.Transactions {
		// Assemble the transaction call message and return if the requested offset
		msg, err := evmcore.TxAsMessage(tx, signer, block.BaseFee)
		if err != nil {
			return nil, nil, err
		}
		if idx == txIndex {
			return msg, statedb, nil
		}

		// For now, Sonic only supports Blob transactions without blob data.
		if msg.BlobHashes != nil {
			if len(msg.BlobHashes) > 0 {
				continue // blob data is not supported - this tx will be skipped
			}
			// PreCheck requires non-nil blobHashes not to be empty
			msg.BlobHashes = nil
		}

		statedb.SetTxContext(tx.Hash(), idx)
		if _, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(tx.Gas())); err != nil {
			statedb.Release()
			return nil, nil, fmt.Errorf("transaction %#x failed: %v", tx.Hash(), err)
		}
		// Ensure any modifications are committed to the state
		statedb.EndTransaction()
	}
	statedb.Release()
	return nil, nil, fmt.Errorf("transaction index %d out of range for block %#x", txIndex, block.Hash)
}

// TraceCallConfig is the config for traceCall API. It holds one more
// field to override the state for tracing.
type TraceCallConfig struct {
	tracers.TraceConfig
	StateOverrides *StateOverride
	BlockOverrides *BlockOverrides
	TxIndex        *hexutil.Uint
}

// TraceCall is generating traces for non historical transactions.
// It is similar to eth_call but with debug capabilities.
func (api *PublicDebugAPI) TraceCall(ctx context.Context, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, config *TraceCallConfig) (interface{}, error) {

	// If pending block, return error
	if num, ok := blockNrOrHash.Number(); ok && num == rpc.PendingBlockNumber {
		return nil, errors.New("tracing on top of pending is not supported")
	}

	// Get block
	block, err := getEvmBlockFromNumberOrHash(ctx, blockNrOrHash, api.b)
	if err != nil {
		return nil, err
	}

	var txIndex uint
	if config != nil && config.TxIndex != nil {
		txIndex = uint(*config.TxIndex)
	}

	// Get state
	_, statedb, err := stateAtTransaction(ctx, block, int(txIndex), api.b)
	if err != nil {
		return nil, err
	}
	defer statedb.Release()

	blockCtx := getBlockContext(ctx, api.b, &block.EvmHeader)
	if config.BlockOverrides != nil {
		config.BlockOverrides.apply(&blockCtx)
	}

	// Apply state overrides
	if config != nil {
		if err := config.StateOverrides.Apply(statedb); err != nil {
			return nil, err
		}
	}

	tx, msg, err := getTxAndMessage(&args, block, api.b)
	if err != nil {
		return nil, err
	}

	var traceConfig *tracers.TraceConfig
	if config != nil {
		traceConfig = &config.TraceConfig
	}

	return api.traceTx(ctx, tx, msg, new(tracers.Context), &block.EvmHeader, statedb, traceConfig, &blockCtx)
}

// getEvmBlockFromNumberOrHash returns EvmBlock from block number or block hash
func getEvmBlockFromNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, b Backend) (*evmcore.EvmBlock, error) {
	var (
		block *evmcore.EvmBlock
		err   error
	)

	if hash, ok := blockNrOrHash.Hash(); ok {
		block, err = b.BlockByHash(ctx, hash)
		if err != nil {
			return nil, err
		}
	} else if number, ok := blockNrOrHash.Number(); ok {
		block, err = b.BlockByNumber(ctx, number)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("invalid arguments; neither block number nor hash specified")
	}
	return block, nil
}

// getTxAndMessage returns transaction and message constructed from transaction arguments
func getTxAndMessage(args *TransactionArgs, block *evmcore.EvmBlock, b Backend) (*types.Transaction, *core.Message, error) {
	msg, err := args.ToMessage(b.RPCGasCap(), block.BaseFee, log.Root())
	if err != nil {
		return nil, nil, err
	}

	tx := types.NewTx(&types.LegacyTx{
		To:       msg.To,
		Nonce:    msg.Nonce,
		Gas:      msg.GasLimit,
		GasPrice: msg.GasPrice,
		Value:    msg.Value,
		Data:     msg.Data,
	})

	return tx, msg, nil
}

// PrivateDebugAPI is the collection of Ethereum APIs exposed over the private
// debugging endpoint.
type PrivateDebugAPI struct {
	b Backend
}

// NewPrivateDebugAPI creates a new API definition for the private debug methods
// of the Ethereum service.
func NewPrivateDebugAPI(b Backend) *PrivateDebugAPI {
	return &PrivateDebugAPI{b: b}
}

// ChaindbProperty returns leveldb properties of the key-value database.
func (api *PrivateDebugAPI) ChaindbProperty(property string) (string, error) {
	return "", errors.New("carmen database does provide db properties")
}

// ChaindbCompact flattens the entire key-value database into a single level,
// removing all unused slots and merging all keys.
func (api *PrivateDebugAPI) ChaindbCompact() error {
	return errors.New("carmen state database does not use compaction")
}

// SetHead rewinds the head of the blockchain to a previous block.
func (api *PrivateDebugAPI) SetHead(number hexutil.Uint64) error {
	return errors.New("lachesis cannot rewind blocks due to the BFT algorithm")
}

// PublicNetAPI offers network related RPC methods
type PublicNetAPI struct {
	net            *p2p.Server
	networkVersion uint64
}

// NewPublicNetAPI creates a new net API instance.
func NewPublicNetAPI(net *p2p.Server, networkVersion uint64) *PublicNetAPI {
	return &PublicNetAPI{net, networkVersion}
}

// Listening returns an indication if the node is listening for network connections.
func (s *PublicNetAPI) Listening() bool {
	return true // always listening
}

// PeerCount returns the number of connected peers
func (s *PublicNetAPI) PeerCount() hexutil.Uint {
	return hexutil.Uint(s.net.PeerCount())
}

// Version returns the current ethereum protocol version.
func (s *PublicNetAPI) Version() string {
	return fmt.Sprintf("%d", s.networkVersion)
}

// checkTxFee is an internal function used to check whether the fee of
// the given transaction is _reasonable_(under the cap).
func checkTxFee(gasPrice *big.Int, gas uint64, cap float64) error {
	// Short circuit if there is no cap for transaction fee at all.
	if cap == 0 {
		return nil
	}
	feeEth := new(big.Float).Quo(new(big.Float).SetInt(new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gas))), new(big.Float).SetInt(big.NewInt(params.Ether)))
	feeFloat, _ := feeEth.Float64()
	if feeFloat > cap {
		return fmt.Errorf("tx fee (%.2f FTM) exceeds the configured cap (%.2f FTM)", feeFloat, cap)
	}
	return nil
}

// toHexSlice creates a slice of hex-strings based on []byte.
func toHexSlice(b []immutable.Bytes) []string {
	r := make([]string, len(b))
	for i := range b {
		r[i] = hexutil.Encode(b[i].ToBytes())
	}
	return r
}
