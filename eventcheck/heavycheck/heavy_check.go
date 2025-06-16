package heavycheck

import (
	"errors"
	"runtime"
	"sync"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/eventcheck/epochcheck"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/valkeystore"
)

//go:generate mockgen -source=heavy_check.go -destination=heavy_check_mock.go -package=heavycheck

var (
	ErrWrongEventSig    = errors.New("event has wrong signature")
	ErrMalformedTxSig   = errors.New("tx has wrong signature")
	ErrWrongPayloadHash = errors.New("event has wrong payload hash")
	ErrPubkeyChanged    = errors.New("validator pubkey has changed, cannot create BVs/EV for older epochs")

	errTerminated = errors.New("terminated") // internal err
)

// Reader is accessed by the validator to get the current state.
type Reader interface {
	GetEpochPubKeys() (map[idx.ValidatorID]validatorpk.PubKey, idx.Epoch)
	GetEpochPubKeysOf(idx.Epoch) map[idx.ValidatorID]validatorpk.PubKey
	GetEpochBlockStart(idx.Epoch) idx.Block
}

// Checker which requires only parents list + current epoch info
type Checker struct {
	config   Config
	txSigner types.Signer
	reader   Reader

	tasksQ chan *taskData
	quit   chan struct{}
	wg     sync.WaitGroup
}

type taskData struct {
	event inter.EventPayloadI

	onValidated func(error)
}

// New validator which performs heavy checks, related to signatures validation and Merkle tree validation
func New(config Config, reader Reader, txSigner types.Signer) *Checker {
	if config.Threads == 0 {
		config.Threads = runtime.NumCPU()
		if config.Threads > 1 {
			config.Threads--
		}
		if config.Threads < 1 {
			config.Threads = 1
		}
	}
	return &Checker{
		config:   config,
		txSigner: txSigner,
		reader:   reader,
		tasksQ:   make(chan *taskData, config.MaxQueuedTasks),
		quit:     make(chan struct{}),
	}
}

func (v *Checker) Start() {
	for i := 0; i < v.config.Threads; i++ {
		v.wg.Add(1)
		go v.loop()
	}
}

func (v *Checker) Stop() {
	close(v.quit)
	v.wg.Wait()
}

func (v *Checker) Overloaded() bool {
	return len(v.tasksQ) > v.config.MaxQueuedTasks/2
}

func (v *Checker) EnqueueEvent(e inter.EventPayloadI, onValidated func(error)) error {
	op := &taskData{
		event:       e,
		onValidated: onValidated,
	}
	select {
	case v.tasksQ <- op:
		return nil
	case <-v.quit:
		return errTerminated
	}
}

// ValidateEvent runs heavy checks for event
func (v *Checker) ValidateEvent(e inter.EventPayloadI) error {
	pubkeys, epoch := v.reader.GetEpochPubKeys()
	if e.Epoch() != epoch {
		return epochcheck.ErrNotRelevant
	}
	// validatorID
	pubkey, ok := pubkeys[e.Creator()]
	if !ok {
		return epochcheck.ErrAuth
	}
	// event sig
	if !valkeystore.VerifySignature(common.Hash(e.HashToSign()), e.Sig().Bytes(), pubkey) {
		return ErrWrongEventSig
	}
	// pre-cache tx sig
	for _, tx := range e.Transactions() {
		_, err := types.Sender(v.txSigner, tx)
		if err != nil {
			return ErrMalformedTxSig
		}
	}
	// Payload hash
	if e.PayloadHash() != inter.CalcPayloadHash(e) {
		return ErrWrongPayloadHash
	}

	return nil
}

func (v *Checker) loop() {
	defer v.wg.Done()
	for {
		select {
		case op := <-v.tasksQ:
			if op.event != nil {
				op.onValidated(v.ValidateEvent(op.event))
			}

		case <-v.quit:
			return
		}
	}
}
