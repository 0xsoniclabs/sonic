package heavycheck

import (
	"errors"
	"runtime"
	"sync"

	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xsoniclabs/sonic/eventcheck/epochcheck"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
)

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

// verifySignature checks the signature against e.Creator.
func verifySignature(signedHash hash.Hash, sig inter.Signature, pubkey validatorpk.PubKey) bool {
	if pubkey.Type != validatorpk.Types.Secp256k1 {
		return false
	}
	return crypto.VerifySignature(pubkey.Raw, signedHash.Bytes(), sig.Bytes())
}

func (v *Checker) ValidateEventLocator(e inter.SignedEventLocator, authEpoch idx.Epoch, authErr error, checkPayload func() bool) error {
	pubkeys := v.reader.GetEpochPubKeysOf(authEpoch)
	if len(pubkeys) == 0 {
		return authErr
	}
	pubkey, ok := pubkeys[e.Locator.Creator]
	if !ok {
		return epochcheck.ErrAuth
	}
	if checkPayload != nil && !checkPayload() {
		return ErrWrongPayloadHash
	}
	if !verifySignature(e.Locator.HashToSign(), e.Sig, pubkey) {
		return ErrWrongEventSig
	}
	return nil
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
	if !verifySignature(e.HashToSign(), e.Sig(), pubkey) {
		return ErrWrongEventSig
	}
	// pre-cache tx sig
	for _, tx := range e.Txs() {
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
