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

package sealmodule

import (
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/Fantom-foundation/lachesis-base/lachesis"

	"github.com/0xsoniclabs/sonic/gossip/blockproc"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
)

type OperaEpochsSealerModule struct{}

func New() *OperaEpochsSealerModule {
	return &OperaEpochsSealerModule{}
}

func (m *OperaEpochsSealerModule) Start(block iblockproc.BlockCtx, bs iblockproc.BlockState, es iblockproc.EpochState) blockproc.SealerProcessor {
	return &OperaEpochsSealer{
		block: block,
		es:    es,
		bs:    bs,
	}
}

type OperaEpochsSealer struct {
	block iblockproc.BlockCtx
	es    iblockproc.EpochState
	bs    iblockproc.BlockState
}

func (s *OperaEpochsSealer) EpochSealing() bool {
	sealEpoch := s.bs.EpochGas >= s.es.Rules.Epochs.MaxEpochGas
	sealEpoch = sealEpoch || (s.block.Time-s.es.EpochStart) >= s.es.Rules.Epochs.MaxEpochDuration
	sealEpoch = sealEpoch || s.bs.AdvanceEpochs > 0
	return sealEpoch || s.bs.EpochCheaters.Len() != 0
}

func (s *OperaEpochsSealer) Update(bs iblockproc.BlockState, es iblockproc.EpochState) {
	s.bs, s.es = bs, es
}

// SealEpoch is called after pre-internal transactions are executed
func (s *OperaEpochsSealer) SealEpoch() (iblockproc.BlockState, iblockproc.EpochState) {
	// Select new validators
	oldValidators := s.es.Validators
	builder := pos.NewBigBuilder()
	for v, profile := range s.bs.NextValidatorProfiles {
		builder.Set(v, profile.Weight)
	}
	newValidators := builder.Build()
	s.es.Validators = newValidators
	s.es.ValidatorProfiles = s.bs.NextValidatorProfiles.Copy()

	// Build new []ValidatorEpochState and []ValidatorBlockState
	newValidatorEpochStates := make([]iblockproc.ValidatorEpochState, newValidators.Len())
	newValidatorBlockStates := make([]iblockproc.ValidatorBlockState, newValidators.Len())
	for newValIdx := idx.Validator(0); newValIdx < newValidators.Len(); newValIdx++ {
		// default values
		newValidatorBlockStates[newValIdx] = iblockproc.ValidatorBlockState{
			Originated: new(big.Int),
		}
		// inherit validator's state from previous epoch, if any
		valID := newValidators.GetID(newValIdx)
		if !oldValidators.Exists(valID) {
			// new validator
			newValidatorBlockStates[newValIdx].LastBlock = s.block.Idx
			newValidatorBlockStates[newValIdx].LastOnlineTime = s.block.Time
			continue
		}
		oldValIdx := oldValidators.GetIdx(valID)
		newValidatorBlockStates[newValIdx] = s.bs.ValidatorStates[oldValIdx]
		newValidatorBlockStates[newValIdx].DirtyGasRefund = 0
		newValidatorBlockStates[newValIdx].Uptime = 0
		newValidatorEpochStates[newValIdx].GasRefund = s.bs.ValidatorStates[oldValIdx].DirtyGasRefund
		newValidatorEpochStates[newValIdx].PrevEpochEvent = s.bs.ValidatorStates[oldValIdx].LastEvent
	}
	s.es.ValidatorStates = newValidatorEpochStates
	s.bs.ValidatorStates = newValidatorBlockStates
	s.es.Validators = newValidators

	// dirty data becomes active
	s.es.PrevEpochStart = s.es.EpochStart
	s.es.EpochStart = s.block.Time
	if s.bs.DirtyRules != nil {
		s.es.Rules = s.bs.DirtyRules.Copy()
		s.bs.DirtyRules = nil
	}
	s.es.EpochStateRoot = s.bs.FinalizedStateRoot

	s.bs.EpochGas = 0
	s.bs.EpochCheaters = lachesis.Cheaters{}
	s.bs.CheatersWritten = 0
	newEpoch := s.es.Epoch + 1
	s.es.Epoch = newEpoch

	if s.bs.AdvanceEpochs > 0 {
		s.bs.AdvanceEpochs--
	}

	return s.bs, s.es
}
