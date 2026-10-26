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

package tests

import (
	"sync"

	"github.com/0xsoniclabs/sonic/rpc/core"
)

// feed behaves like the node's event feed: send waits until every
// subscriber took the event or left.
type feed[Event any] struct {
	mutex       sync.Mutex
	subscribers []*feedSubscription[Event]
}

type feedSubscription[Event any] struct {
	owner  *feed[Event]
	events chan<- Event
	errors chan error
	done   chan struct{}
	once   sync.Once
}

func (target *feed[Event]) subscribe(events chan<- Event) core.Subscription {
	subscription := &feedSubscription[Event]{
		owner:  target,
		events: events,
		errors: make(chan error, 1),
		done:   make(chan struct{}),
	}
	target.mutex.Lock()
	defer target.mutex.Unlock()
	target.subscribers = append(target.subscribers, subscription)
	return subscription
}

func (target *feed[Event]) send(event Event) {
	target.mutex.Lock()
	subscribers := append([]*feedSubscription[Event](nil), target.subscribers...)
	target.mutex.Unlock()
	for _, subscription := range subscribers {
		select {
		case subscription.events <- event:
		case <-subscription.done:
		}
	}
}

func (target *feed[Event]) count() int {
	target.mutex.Lock()
	defer target.mutex.Unlock()
	return len(target.subscribers)
}

func (target *feed[Event]) end() {
	target.mutex.Lock()
	subscribers := target.subscribers
	target.subscribers = nil
	target.mutex.Unlock()
	for _, subscription := range subscribers {
		subscription.close()
	}
}

func (target *feed[Event]) remove(subscription *feedSubscription[Event]) {
	target.mutex.Lock()
	defer target.mutex.Unlock()
	for index, candidate := range target.subscribers {
		if candidate == subscription {
			target.subscribers = append(target.subscribers[:index], target.subscribers[index+1:]...)
			return
		}
	}
}

func (subscription *feedSubscription[Event]) Err() <-chan error {
	return subscription.errors
}

func (subscription *feedSubscription[Event]) Unsubscribe() {
	subscription.owner.remove(subscription)
	subscription.close()
}

func (subscription *feedSubscription[Event]) close() {
	subscription.once.Do(func() {
		close(subscription.done)
		close(subscription.errors)
	})
}

// endedSubscription is what a stopped feed answers: a subscription that has
// already ended.
func endedSubscription() core.Subscription {
	ended := &feedSubscription[struct{}]{
		owner:  &feed[struct{}]{},
		errors: make(chan error),
		done:   make(chan struct{}),
	}
	ended.close()
	return ended
}
