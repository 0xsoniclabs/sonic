package concurrent

import (
	"sync"

	"github.com/0xsoniclabs/consensus/consensus"
)

type EventsSet struct {
	sync.RWMutex
	Val consensus.EventHashSet
}

func WrapEventsSet(v consensus.EventHashSet) *EventsSet {
	return &EventsSet{
		RWMutex: sync.RWMutex{},
		Val:     v,
	}
}
