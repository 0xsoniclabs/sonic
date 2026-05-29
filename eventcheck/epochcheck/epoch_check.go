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

package epochcheck

import (
	"errors"

	base "github.com/Fantom-foundation/lachesis-base/eventcheck/epochcheck"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	ethmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
)

//go:generate mockgen -source=epoch_check.go -destination=epoch_check_mock.go -package=epochcheck

var (
	ErrTooManyParents    = errors.New("event has too many parents")
	ErrTooBigGasUsed     = errors.New("event uses too much gas power")
	ErrWrongGasUsed      = errors.New("event has incorrect gas power")
	ErrUnderpriced       = errors.New("event transaction underpriced")
	ErrTooBigExtra       = errors.New("event extra data is too large")
	ErrWrongVersion      = errors.New("event has wrong version")
	ErrUnsupportedTxType = errors.New("unsupported tx type")
	ErrNotRelevant       = base.ErrNotRelevant
	ErrAuth              = base.ErrAuth
)

// Reader returns currents epoch and its validators group.
type Reader interface {
	base.Reader
	GetEpochRules() (opera.Rules, idx.Epoch)
}

// Checker which require only current epoch info
type Checker struct {
	Base   *base.Checker
	reader Reader
}

func New(reader Reader) *Checker {
	return &Checker{
		Base:   base.New(reader),
		reader: reader,
	}
}

func CalcGasPowerUsed(e inter.EventPayloadI, rules opera.Rules) (uint64, error) {
	gasCfg := rules.Economy.Gas

	if rules.Upgrades.Brio {
		var overflow bool
		txsGas := uint64(0)
		// In the single-proposer protocol, the gas usage of individual transactions
		// is not attributed to the individual proposer, since each proposer needs
		// to be able to create proposals with the full gas limit. Thus, only the
		// transactions being part of the distributed proposal protocol are counted.
		for _, tx := range e.TransactionsToMeter() {
			if txsGas, overflow = ethmath.SafeAdd(txsGas, tx.Gas()); overflow {
				return 0, ErrTooBigGasUsed
			}
		}

		parentsGas := uint64(0)
		if idx.Event(len(e.Parents())) > rules.Dag.MaxFreeParents {
			if parentsGas, overflow = ethmath.SafeMul(uint64(idx.Event(len(e.Parents()))-rules.Dag.MaxFreeParents), gasCfg.ParentGas); overflow {
				return 0, ErrTooBigGasUsed
			}
		}

		var extraGas uint64
		if extraGas, overflow = ethmath.SafeMul(uint64(len(e.Extra())), gasCfg.ExtraDataGas); overflow {
			return 0, ErrTooBigGasUsed
		}

		var mpsGas uint64
		if mpsGas, overflow = ethmath.SafeMul(uint64(len(e.MisbehaviourProofs())), gasCfg.MisbehaviourProofGas); overflow {
			return 0, ErrTooBigGasUsed
		}

		bvsGas := uint64(0)
		if e.BlockVotes().Start != 0 {
			var blockVotesGas uint64
			if blockVotesGas, overflow = ethmath.SafeMul(uint64(len(e.BlockVotes().Votes)), gasCfg.BlockVoteGas); overflow {
				return 0, ErrTooBigGasUsed
			}
			if bvsGas, overflow = ethmath.SafeAdd(gasCfg.BlockVotesBaseGas, blockVotesGas); overflow {
				return 0, ErrTooBigGasUsed
			}
		}

		ersGas := uint64(0)
		if e.EpochVote().Epoch != 0 {
			ersGas = gasCfg.EpochVoteGas
		}

		total := uint64(0)
		for _, v := range []uint64{txsGas, parentsGas, extraGas, gasCfg.EventGas, mpsGas, bvsGas, ersGas} {
			if total, overflow = ethmath.SafeAdd(total, v); overflow {
				return 0, ErrTooBigGasUsed
			}
		}
		return total, nil
	}

	// preBrio: ignore overflows
	txsGas := uint64(0)
	// In the single-proposer protocol, the gas usage of individual transactions
	// is not attributed to the individual proposer, since each proposer needs
	// to be able to create proposals with the full gas limit. Thus, only the
	// transactions being part of the distributed proposal protocol are counted.
	for _, tx := range e.TransactionsToMeter() {
		txsGas += tx.Gas()
	}

	parentsGas := uint64(0)
	if idx.Event(len(e.Parents())) > rules.Dag.MaxFreeParents {
		parentsGas = uint64(idx.Event(len(e.Parents()))-rules.Dag.MaxFreeParents) * gasCfg.ParentGas
	}
	extraGas := uint64(len(e.Extra())) * gasCfg.ExtraDataGas

	mpsGas := uint64(len(e.MisbehaviourProofs())) * gasCfg.MisbehaviourProofGas

	bvsGas := uint64(0)
	if e.BlockVotes().Start != 0 {
		bvsGas = gasCfg.BlockVotesBaseGas + uint64(len(e.BlockVotes().Votes))*gasCfg.BlockVoteGas
	}

	ersGas := uint64(0)
	if e.EpochVote().Epoch != 0 {
		ersGas = gasCfg.EpochVoteGas
	}

	return txsGas + parentsGas + extraGas + gasCfg.EventGas + mpsGas + bvsGas + ersGas, nil
}

func (v *Checker) checkGas(e inter.EventPayloadI, rules opera.Rules) error {
	if e.GasPowerUsed() > rules.Economy.Gas.MaxEventGas {
		return ErrTooBigGasUsed
	}
	calculatedGasPowerUsed, err := CalcGasPowerUsed(e, rules)
	if err != nil {
		return ErrTooBigGasUsed
	}
	if e.GasPowerUsed() != calculatedGasPowerUsed {
		return ErrWrongGasUsed
	}
	return nil
}

func CheckTxs(txs types.Transactions, rules opera.Rules) error {
	maxType := uint8(types.LegacyTxType)
	if rules.Upgrades.Berlin {
		maxType = types.AccessListTxType
	}
	if rules.Upgrades.London {
		maxType = types.DynamicFeeTxType
	}
	if rules.Upgrades.Sonic {
		maxType = types.BlobTxType
	}
	if rules.Upgrades.Allegro {
		maxType = types.SetCodeTxType
	}
	for _, tx := range txs {
		if tx.Type() > maxType {
			return ErrUnsupportedTxType
		}
	}
	return nil
}

// Validate event
func (v *Checker) Validate(e inter.EventPayloadI) error {
	if err := v.Base.Validate(e); err != nil {
		return err
	}
	rules, epoch := v.reader.GetEpochRules()
	// Check epoch of the rules to prevent a race condition
	if e.Epoch() != epoch {
		return base.ErrNotRelevant
	}
	if idx.Event(len(e.Parents())) > rules.Dag.MaxParents {
		return ErrTooManyParents
	}
	if uint32(len(e.Extra())) > rules.Dag.MaxExtraData {
		return ErrTooBigExtra
	}
	if err := v.checkGas(e, rules); err != nil {
		return err
	}
	if err := CheckTxs(e.Transactions(), rules); err != nil {
		return err
	}

	version := uint8(0)
	if rules.Upgrades.SingleProposerBlockFormation {
		version = 3
	} else if rules.Upgrades.Sonic {
		version = 2
	} else if rules.Upgrades.Llr {
		version = 1
	}
	if e.Version() != version {
		return ErrWrongVersion
	}
	return nil
}
