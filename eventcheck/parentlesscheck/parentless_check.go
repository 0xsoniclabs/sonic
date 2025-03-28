package parentlesscheck

import "github.com/0xsoniclabs/consensus/consensus"

type Checker struct {
	HeavyCheck HeavyCheck
	LightCheck LightCheck
}

type LightCheck func(consensus.Event) error

type HeavyCheck interface {
	Enqueue(e consensus.Event, checked func(error)) error
}

// Enqueue tries to fill gaps the fetcher's future import queue.
func (c *Checker) Enqueue(e consensus.Event, checked func(error)) {
	// Run light checks right away
	err := c.LightCheck(e)
	if err != nil {
		checked(err)
		return
	}

	// Run heavy check in parallel
	_ = c.HeavyCheck.Enqueue(e, checked)
}
