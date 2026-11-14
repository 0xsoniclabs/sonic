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

package dispatch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Subscriber installs a kind of subscription. It takes the parameters after
// the kind's name as one struct, as a Handler does, installs what feeds the
// subscription before it returns, so that nothing that comes after the
// subscription's id is missed, and starts the producer of its notifications
// with Subscription.Go.
type Subscriber[Params any] func(ctx context.Context, subscription *Subscription, params Params) error

// subscriptionKind is a registered kind of subscription.
type subscriptionKind struct {
	subscribe func(ctx context.Context, notifier *Notifier, params json.RawMessage) (*Subscription, error)
}

// AddSubscription registers a kind of subscription of a namespace, which
// <namespace>_subscribe installs. It panics on a kind registered twice and
// on a parameter struct it cannot serve.
func AddSubscription[Params any](dispatcher *Dispatcher, namespace, kind string, subscriber Subscriber[Params]) {
	key := namespace + "_" + kind
	if _, exists := dispatcher.subscriptionKinds[key]; exists {
		panic(fmt.Sprintf("subscription %s is registered twice", key))
	}
	// The kind's name is the first parameter, as go-ethereum counts them.
	layout := newParamsLayout(reflect.TypeFor[Params]())
	layout.optional = append([]bool{false}, layout.optional...)
	dispatcher.subscriptionKinds[key] = &subscriptionKind{
		subscribe: func(ctx context.Context, notifier *Notifier, raw json.RawMessage) (_ *Subscription, err error) {
			var name string
			var params Params
			fields := fieldsOf(&params)
			if err := layout.decode(raw, func(position int) any {
				if position == 0 {
					return &name
				}
				return fields(position - 1)
			}); err != nil {
				return nil, InvalidParams(err.Error())
			}
			subscription, err := notifier.add(namespace, kind)
			if err != nil {
				return nil, err
			}
			installed := false
			defer func() {
				if !installed {
					notifier.remove(subscription)
				}
			}()
			if err := subscriber(ctx, subscription, params); err != nil {
				return nil, err
			}
			installed = true
			return subscription, nil
		},
	}
}

// Subscribe serves a call of <namespace>_subscribe on a stream connection,
// as go-ethereum's server does: it reads the kind's name, finds the kind,
// decodes the parameters and installs the subscription, whose id it answers.
// The transport activates the subscription once it has written the answer.
func (d *Dispatcher) Subscribe(ctx context.Context, notifier *Notifier, method string, params json.RawMessage) (Response, *Subscription) {
	name, err := SubscriptionName(params)
	if err != nil {
		return Response{Error: InvalidParams(err.Error())}, nil
	}
	namespace, _, _ := strings.Cut(method, "_")
	kind, found := d.subscriptionKinds[namespace+"_"+name]
	if !found {
		return Response{Error: SubscriptionNotFound(namespace, name)}, nil
	}
	var subscription *Subscription
	response := d.run(ctx, method, func(ctx context.Context) (any, error) {
		var err error
		subscription, err = kind.subscribe(ctx, notifier, params)
		if err != nil {
			return nil, err
		}
		return subscription.ID, nil
	})
	if response.Error != nil {
		if subscription != nil {
			notifier.remove(subscription)
		}
		return response, nil
	}
	return response, subscription
}

// SubscriptionNotFound is the error of a kind of subscription no one serves.
func SubscriptionNotFound(namespace, name string) *Error {
	return &Error{Code: CodeMethodNotFound, Message: fmt.Sprintf("no %q subscription in %s namespace", name, namespace)}
}

// Unsubscribe serves a call of <namespace>_unsubscribe, of any namespace and
// on any transport, as go-ethereum's server does: it ends the subscription
// of the connection's notifier, none without one, that the id names.
func Unsubscribe(notifier *Notifier, params json.RawMessage) Response {
	var id ID
	if err := unsubscribeLayout.decode(params, func(int) any { return &id }); err != nil {
		return Response{Error: InvalidParams(err.Error())}
	}
	if notifier == nil || !notifier.Unsubscribe(id) {
		return Response{Error: ToError(ErrSubscriptionNotFound)}
	}
	return encode(true)
}

var unsubscribeLayout = paramsLayout{optional: []bool{false}}

// SubscriptionName reads the name of a kind of subscription, the first
// parameter, with go-ethereum's wording.
func SubscriptionName(params json.RawMessage) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(params))
	if token, _ := decoder.Token(); token != json.Delim('[') {
		return "", errors.New("non-array args")
	}
	token, _ := decoder.Token()
	name, ok := token.(string)
	if !ok {
		return "", errors.New("expected subscription name as first argument")
	}
	return name, nil
}
