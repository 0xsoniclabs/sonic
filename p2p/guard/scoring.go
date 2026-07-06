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

package guard

import (
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

// GossipScoreParams returns the base gossipsub peer-scoring parameters and
// thresholds used to punish spamming or misbehaving peers. Per-topic scoring is
// contributed by the individual gossip topics; this provides the global
// envelope (decay, caps, and the gray/publish/gossip thresholds).
func GossipScoreParams() (*pubsub.PeerScoreParams, *pubsub.PeerScoreThresholds) {
	params := &pubsub.PeerScoreParams{
		Topics:        make(map[string]*pubsub.TopicScoreParams),
		DecayInterval: time.Second,
		DecayToZero:   0.01,
		// Behaviour penalty punishes protocol violations (e.g. excessive
		// duplicate or out-of-order messages).
		BehaviourPenaltyWeight: -10,
		BehaviourPenaltyDecay:  0.9,
		// No application-specific score contribution by default; higher layers
		// may replace this to reward known-good peers (e.g. validators).
		AppSpecificScore:            func(peer.ID) float64 { return 0 },
		AppSpecificWeight:           1,
		IPColocationFactorWeight:    -5,
		IPColocationFactorThreshold: 5,
		RetainScore:                 10 * time.Minute,
	}
	thresholds := &pubsub.PeerScoreThresholds{
		GossipThreshold:             -100,
		PublishThreshold:            -200,
		GraylistThreshold:           -400,
		AcceptPXThreshold:           10,
		OpportunisticGraftThreshold: 1,
	}
	return params, thresholds
}
