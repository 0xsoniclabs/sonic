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

package eventmodule

import (
	"github.com/0xsoniclabs/sonic/gossip/blockproc"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
)

type ValidatorEventsModule struct{}

func New() *ValidatorEventsModule {
	return &ValidatorEventsModule{}
}

func (m *ValidatorEventsModule) Start(bs iblockproc.BlockState, es iblockproc.EpochState) blockproc.ConfirmedEventsProcessor {
	return &ValidatorEventsProcessor{
		es:                     es,
		bs:                     bs,
		validatorHighestEvents: make(inter.EventIs, es.Validators.Len()),
	}
}

type ValidatorEventsProcessor struct {
	es                     iblockproc.EpochState
	bs                     iblockproc.BlockState
	validatorHighestEvents inter.EventIs
}

func (p *ValidatorEventsProcessor) ProcessConfirmedEvent(e inter.EventI) {
	creatorIdx := p.es.Validators.GetIdx(e.Creator())
	prev := p.validatorHighestEvents[creatorIdx]
	if prev == nil || e.Seq() > prev.Seq() {
		p.validatorHighestEvents[creatorIdx] = e
	}
	p.bs.EpochGas += e.GasPowerUsed()
}

func (p *ValidatorEventsProcessor) Finalize(block iblockproc.BlockCtx, _ bool) iblockproc.BlockState {
	for _, v := range p.bs.EpochCheaters {
		creatorIdx := p.es.Validators.GetIdx(v)
		p.validatorHighestEvents[creatorIdx] = nil
	}
	for creatorIdx, e := range p.validatorHighestEvents {
		if e == nil {
			continue
		}
		info := p.bs.ValidatorStates[creatorIdx]
		if block.Idx <= info.LastBlock+p.es.Rules.Economy.BlockMissedSlack {
			prevOnlineTime := info.LastOnlineTime
			if p.es.Rules.Upgrades.Berlin {
				prevOnlineTime = inter.MaxTimestamp(info.LastOnlineTime, p.es.EpochStart)
			}
			if e.MedianTime() > prevOnlineTime {
				info.Uptime += e.MedianTime() - prevOnlineTime
			}
		}
		info.LastGasPowerLeft = e.GasPowerLeft()
		info.LastOnlineTime = e.MedianTime()
		info.LastBlock = block.Idx
		info.LastEvent = iblockproc.EventInfo{
			ID:           e.ID(),
			GasPowerLeft: e.GasPowerLeft(),
			Time:         e.MedianTime(),
		}
		p.bs.ValidatorStates[creatorIdx] = info
	}
	return p.bs
}
