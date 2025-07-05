// Copyright 2025 Sonic Operations Ltd
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

package heavycheck

import (
	"errors"
	"runtime"
	"sync"

	"github.com/0xsoniclabs/consensus/consensus"
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
	GetEpochPubKeys() (map[consensus.ValidatorID]validatorpk.PubKey, consensus.Epoch)
	GetEpochPubKeysOf(consensus.Epoch) map[consensus.ValidatorID]validatorpk.PubKey
	GetEpochBlockStart(consensus.Epoch) consensus.BlockID
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
