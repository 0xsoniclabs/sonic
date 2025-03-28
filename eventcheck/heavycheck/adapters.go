package heavycheck

import (
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/sonic/inter"
)

type EventsOnly struct {
	*Checker
}

func (c *EventsOnly) Enqueue(e consensus.Event, onValidated func(error)) error {
	return c.Checker.EnqueueEvent(e.(inter.EventPayloadI), onValidated)
}
