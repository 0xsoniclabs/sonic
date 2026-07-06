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

package p2p

import (
	"errors"

	"github.com/0xsoniclabs/sonic/p2p/guard"
	"github.com/libp2p/go-libp2p/core/network"
	"google.golang.org/protobuf/proto"
)

// ErrRateLimited is returned by Stream.ReadMessage when the remote peer has
// exceeded its per-peer traffic budget. The protocol should stop serving the
// peer; sustained abuse is handled by the node (disconnect/ban).
var ErrRateLimited = errors.New("p2p: peer exceeded rate limit")

// streamWrapper adapts a libp2p network.Stream to the Stream interface, adding
// protobuf framing, per-peer rate limiting on reads, and metrics.
type streamWrapper struct {
	stream  network.Stream
	limiter *guard.RateLimiter
	metrics *Metrics
	// onAbuse, if set, is invoked when the remote peer commits sustained
	// rate-limit abuse, so the node can disconnect and ban it. The scope is the
	// protocol ID, for logging.
	onAbuse func(peer PeerID, scope string)
}

func newStream(stream network.Stream, limiter *guard.RateLimiter, metrics *Metrics, onAbuse func(PeerID, string)) *streamWrapper {
	return &streamWrapper{stream: stream, limiter: limiter, metrics: metrics, onAbuse: onAbuse}
}

func (s *streamWrapper) Peer() PeerID {
	return s.stream.Conn().RemotePeer()
}

func (s *streamWrapper) ReadMessage(message proto.Message, maxSize int) error {
	protocolID := string(s.stream.Protocol())
	read, err := ReadMessage(s.stream, message, maxSize)
	if err != nil {
		s.metrics.messages.WithLabelValues("in", protocolID, "error").Inc()
		return err
	}
	s.metrics.streamBytes.WithLabelValues("in", protocolID).Add(float64(read))
	if decision := s.limiter.Check(s.Peer().String(), read); !decision.Allowed {
		s.metrics.rateDropped.WithLabelValues(protocolID, "traffic").Inc()
		s.metrics.messages.WithLabelValues("in", protocolID, "rate_limited").Inc()
		if decision.Abusive && s.onAbuse != nil {
			s.onAbuse(s.Peer(), protocolID)
		}
		return ErrRateLimited
	}
	s.metrics.messages.WithLabelValues("in", protocolID, "ok").Inc()
	return nil
}

func (s *streamWrapper) WriteMessage(message proto.Message, maxSize int) error {
	protocolID := string(s.stream.Protocol())
	written, err := WriteMessage(s.stream, message, maxSize)
	if err != nil {
		s.metrics.messages.WithLabelValues("out", protocolID, "error").Inc()
		return err
	}
	s.metrics.streamBytes.WithLabelValues("out", protocolID).Add(float64(written))
	s.metrics.messages.WithLabelValues("out", protocolID, "ok").Inc()
	return nil
}

func (s *streamWrapper) Close() error {
	return s.stream.Close()
}

func (s *streamWrapper) Reset() error {
	return s.stream.Reset()
}
