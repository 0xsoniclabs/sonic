package emitter

import (
	"errors"
	"sync"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/valkeystore"
	"github.com/0xsoniclabs/sonic/vecmt"
)

//go:generate mockgen -source=world.go -destination=world_mock.go -package=emitter External,TxPool,TxSigner,Signer

var (
	ErrNotEnoughGasPower = errors.New("not enough gas power")
)

type (
	// External world
	External interface {
		sync.Locker
		Reader

		Check(e *inter.EventPayload, parents inter.Events) error
		Process(*inter.EventPayload) error
		Broadcast(*inter.EventPayload)
		Build(*inter.MutableEventPayload, func()) error
		DagIndex() *vecmt.Index

		IsBusy() bool
		IsSynced() bool
		PeersNum() int

		StateDB() state.StateDB
		GetUpgradeHeights() []opera.UpgradeHeight
		GetHeader(common.Hash, uint64) *evmcore.EvmHeader
	}

	// TxSigner is a re-export of the types.Signer interface to allow
	// mocking it in tests.
	TxSigner interface {
		types.Signer
	}

	// World is an emitter's environment
	World struct {
		External
		TxPool            TxPool
		EventsSigner      valkeystore.SignerAuthority
		TransactionSigner TxSigner
	}
)

// Reader is a callback for getting events from an external storage.
type Reader interface {
	GetLatestBlockIndex() idx.Block
	GetLatestBlock() *inter.Block
	GetEpochValidators() (*pos.Validators, idx.Epoch)
	GetEvent(hash.Event) *inter.Event
	GetEventPayload(hash.Event) *inter.EventPayload
	GetLastEvent(epoch idx.Epoch, from idx.ValidatorID) *hash.Event
	GetHeads(idx.Epoch) hash.Events
	GetGenesisTime() inter.Timestamp
	GetRules() opera.Rules
}

type TxPool interface {
	// Has returns an indicator whether txpool has a transaction cached with the
	// given hash.
	Has(hash common.Hash) bool
	// Pending should return pending transactions.
	// The slice should be modifiable by the caller.
	Pending(enforceTips bool) (map[common.Address]types.Transactions, error)

	// Count returns the total number of transactions
	Count() int
}
