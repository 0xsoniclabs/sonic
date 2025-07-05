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

package gossip

import (
	"sync/atomic"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/0xsoniclabs/sonic/eventcheck/gaspowercheck"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/opera"
)

// GasPowerCheckReader is a helper to run gas power check
type GasPowerCheckReader struct {
	Ctx atomic.Value
}

// GetValidationContext returns current validation context for gaspowercheck
func (r *GasPowerCheckReader) GetValidationContext() *gaspowercheck.ValidationContext {
	return r.Ctx.Load().(*gaspowercheck.ValidationContext)
}

// NewGasPowerContext reads current validation context for gaspowercheck
func NewGasPowerContext(s *Store, validators *consensus.Validators, epoch consensus.Epoch, cfg opera.EconomyRules) *gaspowercheck.ValidationContext {
	// engineMu is locked here

	short := cfg.ShortGasPower
	shortTermConfig := gaspowercheck.Config{
		Idx:                inter.ShortTermGas,
		AllocPerSec:        short.AllocPerSec,
		MaxAllocPeriod:     short.MaxAllocPeriod,
		MinEnsuredAlloc:    cfg.Gas.MaxEventGas,
		StartupAllocPeriod: short.StartupAllocPeriod,
		MinStartupGas:      short.MinStartupGas,
	}

	long := cfg.LongGasPower
	longTermConfig := gaspowercheck.Config{
		Idx:                inter.LongTermGas,
		AllocPerSec:        long.AllocPerSec,
		MaxAllocPeriod:     long.MaxAllocPeriod,
		MinEnsuredAlloc:    cfg.Gas.MaxEventGas,
		StartupAllocPeriod: long.StartupAllocPeriod,
		MinStartupGas:      long.MinStartupGas,
	}

	validatorStates := make([]gaspowercheck.ValidatorState, validators.Len())
	es := s.GetEpochState()
	for i, val := range es.ValidatorStates {
		validatorStates[i].GasRefund = val.GasRefund
		validatorStates[i].PrevEpochEvent = val.PrevEpochEvent
	}

	return &gaspowercheck.ValidationContext{
		Epoch:           epoch,
		Validators:      validators,
		EpochStart:      es.EpochStart,
		ValidatorStates: validatorStates,
		Configs: [inter.GasPowerConfigs]gaspowercheck.Config{
			inter.ShortTermGas: shortTermConfig,
			inter.LongTermGas:  longTermConfig,
		},
	}
}

// ValidatorsPubKeys stores info to authenticate validators
type ValidatorsPubKeys struct {
	Epoch   consensus.Epoch
	PubKeys map[consensus.ValidatorID]validatorpk.PubKey
}

// HeavyCheckReader is a helper to run heavy power checks
type HeavyCheckReader struct {
	Pubkeys atomic.Value
	Store   *Store
}

// GetEpochPubKeys is safe for concurrent use
func (r *HeavyCheckReader) GetEpochPubKeys() (map[consensus.ValidatorID]validatorpk.PubKey, consensus.Epoch) {
	auth := r.Pubkeys.Load().(*ValidatorsPubKeys)

	return auth.PubKeys, auth.Epoch
}

// GetEpochPubKeysOf is safe for concurrent use
func (r *HeavyCheckReader) GetEpochPubKeysOf(epoch consensus.Epoch) map[consensus.ValidatorID]validatorpk.PubKey {
	auth := readEpochPubKeys(r.Store, epoch)
	if auth == nil {
		return nil
	}
	return auth.PubKeys
}

// GetEpochBlockStart is safe for concurrent use
func (r *HeavyCheckReader) GetEpochBlockStart(epoch consensus.Epoch) consensus.BlockID {
	bs, _ := r.Store.GetHistoryBlockEpochState(epoch)
	if bs == nil {
		return 0
	}
	return bs.LastBlock.Idx
}

// readEpochPubKeys reads epoch pubkeys
func readEpochPubKeys(s *Store, epoch consensus.Epoch) *ValidatorsPubKeys {
	es := s.GetHistoryEpochState(epoch)
	if es == nil {
		return nil
	}
	var pubkeys = make(map[consensus.ValidatorID]validatorpk.PubKey, len(es.ValidatorProfiles))
	for id, profile := range es.ValidatorProfiles {
		pubkeys[id] = profile.PubKey
	}
	return &ValidatorsPubKeys{
		Epoch:   epoch,
		PubKeys: pubkeys,
	}
}

// proposalCheckReader is an implementation of the proposalcheck.Reader
// interface providing access to event payload data and epoch validators.
type proposalCheckReader struct {
	store *Store
}

func newProposalCheckReader(store *Store) proposalCheckReader {
	return proposalCheckReader{
		store: store,
	}
}

func (r *proposalCheckReader) GetEpochValidators() *consensus.Validators {
	return r.store.GetValidators()
}

func (r *proposalCheckReader) GetEventPayload(eventID consensus.EventHash) inter.Payload {
	return *r.store.GetEventPayload(eventID).Payload()
}
