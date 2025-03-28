package concurrent

import (
	"sync"

	"github.com/0xsoniclabs/consensus/consensus"
)

type ValidatorEventsSet struct {
	sync.RWMutex
	Val map[consensus.ValidatorID]consensus.EventHash
}

func WrapValidatorEventsSet(v map[consensus.ValidatorID]consensus.EventHash) *ValidatorEventsSet {
	return &ValidatorEventsSet{
		RWMutex: sync.RWMutex{},
		Val:     v,
	}
}
