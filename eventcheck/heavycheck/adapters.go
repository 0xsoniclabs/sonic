package heavycheck

import (
	"github.com/Fantom-foundation/lachesis-base/inter/dag"

	"github.com/0xsoniclabs/sonic/inter"
)

type EventsOnly struct {
	*Checker
}

func (c *EventsOnly) Enqueue(e dag.Event, onValidated func(error)) error {
	return c.EnqueueEvent(e.(inter.EventPayloadI), onValidated)
}
