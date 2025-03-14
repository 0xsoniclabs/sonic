package emitter

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0xsoniclabs/consensus/emitter/ancestor"
	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/inter/pos"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/0xsoniclabs/sonic/utils/txtime"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/0xsoniclabs/sonic/gossip/emitter/originatedtxs"
	"github.com/0xsoniclabs/sonic/gossip/gasprice/gaspricelimits"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/utils/errlock"
)

const (
	SenderCountBufferSize = 20000
	PayloadIndexerSize    = 5000
)

var (
	emittedEventsCounter        = metrics.GetOrRegisterCounter("emitter/events", nil)                    // amount of emitted events
	emittedEventsTxsCounter     = metrics.GetOrRegisterCounter("emitter/txs", nil)                       // amount of txs in emitted events
	emittedGasCounter           = metrics.GetOrRegisterCounter("emitter/gas", nil)                       // consumed validator gas
	txsSkippedNoValidatorGas    = metrics.GetOrRegisterCounter("emitter/skipped/novalidatorgas", nil)    // validator does not have enough gas
	txsSkippedEpochRules        = metrics.GetOrRegisterCounter("emitter/skipped/epochrules", nil)        // tx skipped because of epoch rules (like insufficient gasPrice)
	txsSkippedConflictingSender = metrics.GetOrRegisterCounter("emitter/skipped/conflictingsender", nil) // tx by given sender in some unconfirmed event
	txsSkippedNotMyTurn         = metrics.GetOrRegisterCounter("emitter/skipped/notmyturn", nil)         // tx should be handled by other validator
	txsSkippedOutdated          = metrics.GetOrRegisterCounter("emitter/skipped/outdated", nil)          // tx skipped because it is outdated

	skippedOfflineValidatorsCounter = metrics.GetOrRegisterCounter("emitter/skipped_offline", nil)

	eventTimeToConfirmTimer = metrics.GetOrRegisterTimer("emitter/timetoconfirm", nil)
	txTimeToEmitTimer       = metrics.GetOrRegisterTimer("emitter/timetoemit", nil)
	txEndToEndTimer         = metrics.GetOrRegisterTimer("emitter/endtoendtime", nil)
)

type Emitter struct {
	config Config

	world World

	syncStatus syncStatus

	prevIdleTime       time.Time
	prevEmittedAtTime  time.Time
	prevEmittedAtBlock idx.Block
	originatedTxs      *originatedtxs.Buffer
	pendingGas         uint64

	// note: track validators and epoch internally to avoid referring to
	// validators of a future epoch inside OnEventConnected of last epoch event
	validators *pos.Validators
	epoch      idx.Epoch

	// challenges is deadlines when each validator should emit an event
	challenges map[idx.ValidatorID]time.Time
	// offlineValidators is a map of validators which are likely to be offline
	// This map may be different on different instances
	offlineValidators     map[idx.ValidatorID]bool
	expectedEmitIntervals map[idx.ValidatorID]time.Duration
	stakeRatio            map[idx.ValidatorID]uint64

	prevRecheckedChallenges time.Time

	quorumIndexer  *ancestor.QuorumIndexer
	fcIndexer      *ancestor.FCIndexer
	payloadIndexer *ancestor.PayloadIndexer

	intervals                EmitIntervals
	globalConfirmingInterval time.Duration

	done chan struct{}
	wg   sync.WaitGroup

	maxParents idx.Event

	cache struct {
		sortedTxs *transactionsByPriceAndNonce
		poolTime  time.Time
		poolBlock idx.Block
		poolCount int
	}

	emittedEventFile *os.File
	emittedBvsFile   *os.File
	emittedEvFile    *os.File

	logger.Periodic

	baseFeeSource BaseFeeSource

	lastTimeAnEventWasConfirmed atomic.Pointer[time.Time]

	// TODO: improve this to support re-sending signatures if those get lost.
	signatureMutex             sync.Mutex
	pendingCommitteeSignatures []inter.CommitteeSignature
	pendingBlockSignatures     []inter.BlockSignature
}

type BaseFeeSource interface {
	GetCurrentBaseFee() *big.Int
}

// NewEmitter creation.
func NewEmitter(
	config Config,
	world World,
	baseFeeSource BaseFeeSource,
) *Emitter {
	// Randomize event time to decrease chance of 2 parallel instances emitting event at the same time
	// It increases the chance of detecting parallel instances
	rand := rand.New(rand.NewPCG(uint64(os.Getpid()), uint64(time.Now().UnixNano())))
	config.EmitIntervals = config.EmitIntervals.RandomizeEmitTime(rand)

	return &Emitter{
		config:                   config,
		world:                    world,
		originatedTxs:            originatedtxs.New(SenderCountBufferSize),
		intervals:                config.EmitIntervals,
		globalConfirmingInterval: config.EmitIntervals.Confirming,
		Periodic:                 logger.Periodic{Instance: logger.New()},
		baseFeeSource:            baseFeeSource,
	}
}

// init emitter without starting events emission
func (em *Emitter) init() {
	em.syncStatus.startup = time.Now()
	em.syncStatus.lastConnected = time.Now()
	em.syncStatus.p2pSynced = time.Now()
	validators, epoch := em.world.GetEpochValidators()
	em.OnNewEpoch(validators, epoch)

	if len(em.config.PrevEmittedEventFile.Path) != 0 {
		em.emittedEventFile = openPrevActionFile(em.config.PrevEmittedEventFile.Path, em.config.PrevEmittedEventFile.SyncMode)
	}
	if len(em.config.PrevBlockVotesFile.Path) != 0 {
		em.emittedBvsFile = openPrevActionFile(em.config.PrevBlockVotesFile.Path, em.config.PrevBlockVotesFile.SyncMode)
	}
	if len(em.config.PrevEpochVoteFile.Path) != 0 {
		em.emittedEvFile = openPrevActionFile(em.config.PrevEpochVoteFile.Path, em.config.PrevEpochVoteFile.SyncMode)
	}
}

// Start starts event emission.
func (em *Emitter) Start() {
	if em.config.Validator.ID == 0 {
		// short circuit if not a validator
		return
	}
	if em.done != nil {
		return
	}
	em.init()
	em.done = make(chan struct{})

	done := em.done
	if em.config.EmitIntervals.Min == 0 {
		return
	}
	em.wg.Add(1)
	go func() {
		defer em.wg.Done()
		ticker := time.NewTicker(11 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				em.tick()
			case <-done:
				return
			}
		}
	}()
}

// Stop stops event emission.
func (em *Emitter) Stop() {
	if em.done == nil {
		return
	}

	close(em.done)
	em.done = nil
	em.wg.Wait()
}

func (em *Emitter) tick() {
	// track synced time
	if em.world.PeersNum() == 0 {
		// connected time ~= last time when it's true that "not connected yet"
		em.syncStatus.lastConnected = time.Now()
	}
	if !em.world.IsSynced() {
		// synced time ~= last time when it's true that "not synced yet"
		em.syncStatus.p2pSynced = time.Now()
	}
	if em.world.IsBusy() {
		return
	}

	em.recheckChallenges()
	em.recheckIdleTime()
	if time.Since(em.prevEmittedAtTime) >= em.intervals.Min {
		_, err := em.EmitEvent()
		if err != nil {
			em.Log.Error("Event emitting error", "err", err)
		}
	}
}

func (em *Emitter) getSortedTxs(baseFee *big.Int) *transactionsByPriceAndNonce {
	// Short circuit if pool wasn't updated since the cache was built
	poolCount := em.world.TxPool.Count()
	if em.cache.sortedTxs != nil &&
		em.cache.poolBlock == em.world.GetLatestBlockIndex() &&
		em.cache.poolCount == poolCount &&
		time.Since(em.cache.poolTime) < em.config.TxsCacheInvalidation {
		return em.cache.sortedTxs.Copy()
	}
	// Build the cache
	pendingTxs, err := em.world.TxPool.Pending(true)
	if err != nil {
		em.Log.Error("Tx pool transactions fetching error", "err", err)
		return nil
	}
	for from, txs := range pendingTxs {
		// Filter the excessive transactions from each sender
		if len(txs) > em.config.MaxTxsPerAddress {
			pendingTxs[from] = txs[:em.config.MaxTxsPerAddress]
		}
	}
	// Convert to lists of LazyTransactions
	txs := make(map[common.Address][]*txpool.LazyTransaction, len(pendingTxs))
	for from, list := range pendingTxs {
		lazyTxs := make([]*txpool.LazyTransaction, 0, len(list))
		for _, tx := range list {
			lazyTxs = append(lazyTxs, &txpool.LazyTransaction{
				Hash:      tx.Hash(),
				Tx:        tx,
				Time:      tx.Time(),
				GasFeeCap: utils.BigIntToUint256(tx.GasFeeCap()),
				GasTipCap: utils.BigIntToUint256(tx.GasTipCap()),
				Gas:       tx.Gas(),
				BlobGas:   tx.BlobGas(),
			})
		}
		txs[from] = lazyTxs
	}

	sortedTxs := newTransactionsByPriceAndNonce(em.world.TxSigner, txs, baseFee)
	em.cache.sortedTxs = sortedTxs
	em.cache.poolCount = poolCount
	em.cache.poolBlock = em.world.GetLatestBlockIndex()
	em.cache.poolTime = time.Now()
	return sortedTxs.Copy()
}

func (em *Emitter) EmitEvent() (*inter.EventPayload, error) {
	// Check whether the node is a validator
	if !em.isValidator() {
		return nil, nil
	}

	// Check if it's time to emit.
	if !em.isAllowedToEmit() {
		return nil, nil
	}

	minimFeeCap := gaspricelimits.GetMinimumFeeCapForEventEmitter(
		em.baseFeeSource.GetCurrentBaseFee(),
	)
	sortedTxs := em.getSortedTxs(minimFeeCap)

	if em.world.IsBusy() {
		return nil, nil
	}

	em.world.Lock()
	defer em.world.Unlock()

	e, err := em.createEvent(sortedTxs)
	if e == nil || err != nil {
		return nil, err
	}
	em.syncStatus.prevLocalEmittedID = e.ID()

	err = em.world.Process(e)
	if err != nil {
		em.Log.Error("Self-event connection failed", "err", err.Error())
		return nil, err
	}
	// write event ID to avoid doublesigning in future after a crash
	em.writeLastEmittedEventID(e.ID())
	// broadcast the event
	em.world.Broadcast(e)

	// metrics
	emittedEventsTxsCounter.Inc(int64(e.Txs().Len()))
	emittedGasCounter.Inc(int64(e.GasPowerUsed()))
	emittedEventsCounter.Inc(1)

	em.prevEmittedAtTime = time.Now() // record time after connecting, to add the event processing time
	em.prevEmittedAtBlock = em.world.GetLatestBlockIndex()

	return e, nil
}

func (em *Emitter) loadPrevEmitTime() time.Time {
	prevEventID := em.world.GetLastEvent(em.epoch, em.config.Validator.ID)
	if prevEventID == nil {
		return em.prevEmittedAtTime
	}
	prevEvent := em.world.GetEvent(*prevEventID)
	if prevEvent == nil {
		return em.prevEmittedAtTime
	}
	return prevEvent.CreationTime().Time()
}

// createEvent is not safe for concurrent use.
func (em *Emitter) createEvent(sortedTxs *transactionsByPriceAndNonce) (*inter.EventPayload, error) {
	if synced := em.logSyncStatus(em.isSyncedToEmit()); !synced {
		// I'm reindexing my old events, so don't create events until connect all the existing self-events
		return nil, nil
	}

	var (
		selfParentSeq  idx.Event
		selfParentTime inter.Timestamp
		parents        hash.Events
		maxLamport     idx.Lamport
	)

	// Find parents
	selfParent, parents, ok := em.chooseParents(em.epoch, em.config.Validator.ID)
	if !ok {
		return nil, nil
	}
	prevEmitted := em.readLastEmittedEventID()
	if prevEmitted != nil && prevEmitted.Epoch() >= em.epoch {
		if selfParent == nil || *selfParent != *prevEmitted {
			errlock.Permanent(errors.New("Local database does not contain last emitted event - sync the node before enabling validation to avoid doublesign"))
		}
	}

	// Set parent-dependent fields
	parentHeaders := make(inter.Events, len(parents))
	for i, p := range parents {
		parent := em.world.GetEvent(p)
		if parent == nil {
			em.Log.Crit("Emitter: head not found", "mutEvent", p.String())
		}
		parentHeaders[i] = parent
		if parentHeaders[i].Creator() == em.config.Validator.ID && i != 0 {
			// there are 2 heads from me, i.e. due to a fork, chooseParents could have found multiple self-parents
			em.Periodic.Error(5*time.Second, "I've created a fork, events emitting isn't allowed", "creator", em.config.Validator.ID)
			return nil, nil
		}
		maxLamport = idx.MaxLamport(maxLamport, parent.Lamport())
	}

	selfParentSeq = 0
	selfParentTime = 0
	var selfParentHeader *inter.Event
	if selfParent != nil {
		selfParentHeader = parentHeaders[0]
		selfParentSeq = selfParentHeader.Seq()
		selfParentTime = selfParentHeader.CreationTime()
	}

	version := uint8(0)
	if em.world.GetRules().Upgrades.Allegro {
		version = 3
	} else if em.world.GetRules().Upgrades.Sonic {
		version = 2
	} else if em.world.GetRules().Upgrades.Llr {
		version = 1
	}

	mutEvent := &inter.MutableEventPayload{}
	mutEvent.SetVersion(version)
	mutEvent.SetEpoch(em.epoch)
	mutEvent.SetSeq(selfParentSeq + 1)
	mutEvent.SetCreator(em.config.Validator.ID)

	mutEvent.SetParents(parents)
	mutEvent.SetLamport(maxLamport + 1)
	mutEvent.SetCreationTime(inter.MaxTimestamp(inter.Timestamp(time.Now().UnixNano()), selfParentTime+1))

	// node version
	if mutEvent.Seq() <= 1 && len(em.config.VersionToPublish) > 0 {
		version := []byte("v-" + em.config.VersionToPublish)
		if uint32(len(version)) <= em.world.GetRules().Dag.MaxExtraData {
			mutEvent.SetExtra(version)
		}
	}

	// set consensus fields
	err := em.world.Build(mutEvent, nil)
	if err != nil {
		if err == ErrNotEnoughGasPower {
			em.Periodic.Warn(time.Second, "Not enough gas power to emit event. Too small stake?",
				"stake%", 100*float64(em.validators.Get(em.config.Validator.ID))/float64(em.validators.TotalWeight()))
		} else {
			em.Log.Warn("Dropped event while emitting", "err", err)
		}
		return nil, nil
	}

	// Add txs
	em.addTxs(mutEvent, sortedTxs)

	// Add signatures
	if version == 3 {
		em.addSignatures(mutEvent)
	}

	// calc Payload hash
	mutEvent.SetPayloadHash(inter.CalcPayloadHash(mutEvent))

	// sign
	bSig, err := em.world.Signer.Sign(em.config.Validator.PubKey, mutEvent.HashToSign().Bytes())
	if err != nil {
		em.Periodic.Error(time.Second, "Failed to sign event", "err", err)
		return nil, err
	}
	var sig inter.Signature
	copy(sig[:], bSig)
	mutEvent.SetSig(sig)

	// build clean event
	event := mutEvent.Build()

	// check
	if err := em.world.Check(event, parentHeaders); err != nil {
		em.Periodic.Error(time.Second, "Emitted incorrect event", "err", err)
		return nil, err
	}

	// set mutEvent name for debug
	em.nameEventForDebug(event)

	for _, tx := range event.Txs() {
		txTime := txtime.Get(tx.Hash()) // time when was the tx seen first time
		if !txTime.Equal(time.Time{}) {
			txTimeToEmitTimer.Update(time.Since(txTime))
		}
	}

	return event, nil
}

func (em *Emitter) idle() bool {
	return em.originatedTxs.Empty()
}

func (em *Emitter) isValidator() bool {
	return em.config.Validator.ID != 0 && em.validators.Exists(em.config.Validator.ID)
}

func (em *Emitter) nameEventForDebug(e *inter.EventPayload) {
	name := []rune(hash.GetNodeName(e.Creator()))
	if len(name) < 1 {
		return
	}

	name = name[len(name)-1:]
	hash.SetEventName(e.ID(), fmt.Sprintf("%s%03d",
		strings.ToLower(string(name)),
		e.Seq()))
}

func (em *Emitter) EnqueueCommitteeSignatureForBroadcast(sig inter.CommitteeSignature) {
	em.signatureMutex.Lock()
	defer em.signatureMutex.Unlock()
	if !em.world.GetRules().Upgrades.Allegro {
		return
	}
	em.pendingCommitteeSignatures = append(em.pendingCommitteeSignatures, sig)
}

func (em *Emitter) EnqueueBlockSignatureForBroadcast(sig inter.BlockSignature) {
	em.signatureMutex.Lock()
	defer em.signatureMutex.Unlock()
	if !em.world.GetRules().Upgrades.Allegro {
		return
	}
	em.pendingBlockSignatures = append(em.pendingBlockSignatures, sig)
}

func (em *Emitter) addSignatures(e *inter.MutableEventPayload) {
	em.signatureMutex.Lock()
	defer em.signatureMutex.Unlock()
	e.SetCommitteeSignatures(em.pendingCommitteeSignatures)
	em.pendingCommitteeSignatures = nil
	e.SetBlockSignatures(em.pendingBlockSignatures)
	em.pendingBlockSignatures = nil
}
