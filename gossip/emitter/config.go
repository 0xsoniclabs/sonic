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

package emitter

import (
	"math/rand/v2"
	"time"

	"github.com/0xsoniclabs/sonic/version"

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

// EmitIntervals is the configuration of emit intervals.
type EmitIntervals struct {
	Min                        time.Duration
	Max                        time.Duration
	Confirming                 time.Duration // emit time when there's no txs to originate, but at least 1 tx to confirm
	ParallelInstanceProtection time.Duration
	DoublesignProtection       time.Duration
}

type ValidatorConfig struct {
	ID     idx.ValidatorID
	PubKey validatorpk.PubKey
}

type FileConfig struct {
	Path     string
	SyncMode bool
}

// Config is the configuration of events emitter.
type Config struct {
	VersionToPublish string

	Validator ValidatorConfig

	EmitIntervals EmitIntervals // event emission intervals

	MaxTxsPerAddress int

	MaxParents idx.Event

	// thresholds on GasLeft
	LimitedTpsThreshold uint64
	NoTxsThreshold      uint64
	EmergencyThreshold  uint64

	TxsCacheInvalidation time.Duration

	PrevEmittedEventFile FileConfig
	PrevBlockVotesFile   FileConfig
	PrevEpochVoteFile    FileConfig
}

// DefaultConfig returns the default configurations for the events emitter.
func DefaultConfig() Config {
	return Config{
		VersionToPublish: version.String(),

		EmitIntervals: EmitIntervals{
			Min:                        150 * time.Millisecond,
			Max:                        10 * time.Minute,
			Confirming:                 170 * time.Millisecond,
			DoublesignProtection:       27 * time.Minute, // should be greater than MaxEmitInterval
			ParallelInstanceProtection: 1 * time.Minute,
		},

		MaxTxsPerAddress: TxTurnNonces,

		MaxParents: 0,

		LimitedTpsThreshold: opera.DefaultEventGas * 120,
		NoTxsThreshold:      opera.DefaultEventGas * 30,
		EmergencyThreshold:  opera.DefaultEventGas * 5,

		TxsCacheInvalidation: 200 * time.Millisecond,
	}
}

// RandomizeEmitTime and return new config
func (cfg EmitIntervals) RandomizeEmitTime(rand *rand.Rand) EmitIntervals {
	config := cfg
	// value = value - 0.1 * value + 0.1 * random value
	if config.Max > 10 {
		config.Max = config.Max - config.Max/10 + time.Duration(rand.Int64N(int64(config.Max/10)))
	}
	// value = value + 0.33 * random value
	if config.DoublesignProtection > 3 {
		config.DoublesignProtection = config.DoublesignProtection + time.Duration(rand.Int64N(int64(config.DoublesignProtection/3)))
	}
	return config
}

// FakeConfig returns the testing configurations for the events emitter.
func FakeConfig(num idx.Validator) Config {
	cfg := DefaultConfig()
	cfg.EmitIntervals.Max = 10 * time.Second // don't wait long in fakenet
	cfg.EmitIntervals.DoublesignProtection = cfg.EmitIntervals.Max / 2
	if num <= 1 {
		// disable self-fork protection if fakenet 1/1
		cfg.EmitIntervals.DoublesignProtection = 0
	}
	return cfg
}
