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

package throttling

import (
	"fmt"
	"maps"
	"math/rand"
	"slices"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SkipEvents_SimulateFrameCounting(t *testing.T) {

	// This test simulates a network of nodes using the throttling mechanism.
	// The test consist of several epochs, each with a different stake distribution
	// among validators. In each epoch, nodes create and exchange events, applying
	// the throttling logic to decide whether to emit events based on their stake
	// and the dominant set of validators.
	//
	// To verify the resiliency of the throttling mechanism, the test drops events
	// randomly to simulate blackouts or misbehaving nodes.

	const epochEmissions = 50
	const maxNumNodes = 10

	testRules := opera.Rules{
		Economy: opera.EconomyRules{
			BlockMissedSlack: 4,
		},
	}
	fakeWorld := &fakeWorld{
		rules: testRules,
	}

	for dominantThreshold := 0.67; dominantThreshold <= 0.95; dominantThreshold += 0.05 {

		net := newNetwork(t, maxNumNodes, fakeWorld, dominantThreshold, 3)
		for testName, stakes := range map[string][]int64{
			"single validator":            {1000},
			"single dominant validator":   makeTestStakeDistribution(t, 10, 1, dominantThreshold),
			"two dominant validators":     makeTestStakeDistribution(t, 10, 2, dominantThreshold),
			"three dominant validators":   makeTestStakeDistribution(t, 10, 3, dominantThreshold),
			"uniform stake":               makeTestStakeDistribution(t, 10, 0, dominantThreshold),
			"non dominated uniform stake": makeTestStakeDistribution(t, 3, 0, dominantThreshold),
		} {

			t.Run(fmt.Sprintf("%v/threshold=%f", testName, dominantThreshold),
				func(t *testing.T) {

					require.GreaterOrEqual(t, maxNumNodes, len(stakes),
						"not enough initial nodes for this distribution")
					fakeWorld.validators = makeValidators(stakes...)

					// =========================================================
					// Run n emissions
					// =========================================================

					for i := range epochEmissions {

						events := make([]*inter.EventPayload, 0)
						for id, node := range net.nodes {

							// only validators with stake do emit
							if !fakeWorld.validators.Exists(id) {
								continue
							}

							event := node.createEvent()
							skip := node.throttler.SkipEventEmission(event)
							if !skip {
								events = append(events, event)
							}
						}

						// shuffle events
						rand.Shuffle(len(events), func(i, j int) {
							events[i], events[j] = events[j], events[i]
						})
						// drop first element with a probability. This simulates validators
						// being offline or misbehaving.
						if rand.Intn(100) < 40 {
							events = events[1:]
						}

						for _, event := range events {
							for _, node := range net.nodes {
								node.receiveEvent(event)
							}
						}

						// Simulate block progression, this tests a mechanism which forces
						// suppressed nodes to emit eventually. So that they are not
						// considered inactive.
						fakeWorld.lastBlock = idx.Block(i / 3)
					}

					dominantSet, _ := ComputeDominantSet(fakeWorld.validators, dominantThreshold)

					// =========================================================
					// Check expectations
					// =========================================================

					t.Log("epoch stakes:", stakes)
					t.Log("dominant set:", slices.Collect(maps.Keys(dominantSet)))
					for _, node := range net.nodes {
						seenPeers := slices.Collect(maps.Keys(node.lastEventPerPeer))
						t.Log("  - node", node.selfId,
							"has seen peers", seenPeers,
							"reached frame", node.lastSeenFrameNumber())
					}

					for _, node := range net.nodes {

						eventsInEpoch := slices.Collect(maps.Values(node.confirmedEvents))
						lastFrame := idx.Frame(0)
						stakeInFrames := make(map[idx.Frame]pos.Weight)
						for _, event := range eventsInEpoch {
							validatorStake := fakeWorld.validators.Get(event.Creator())
							stakeInFrames[event.Frame()] += validatorStake
							lastFrame = max(lastFrame, event.Frame())
						}

						// All events collected must belong to the current epoch
						for _, event := range eventsInEpoch {
							require.Equal(t, node.currentEpoch, event.Epoch(),
								"all events must belong to the current epoch")
						}

						// Each node must have seen events in every frame
						for frame := idx.Frame(1); frame <= lastFrame; frame++ {
							_, ok := stakeInFrames[frame]
							require.True(t, ok,
								"node %d has not seen any events for frame %d",
								node.selfId, frame)
						}

						// Each node must have seen events on each frame emitted
						// from validators with a super-majority of stake
						totalStake := fakeWorld.validators.TotalWeight()
						superMajorityStake := (totalStake*2)/3 + 1
						for frame, stakeForFrame := range stakeInFrames {
							if frame == lastFrame {
								// last frame may be incomplete
								continue
							}
							require.GreaterOrEqual(t, stakeForFrame, superMajorityStake,
								"node %d: frame %d does not have super-majority stake %d < (%d / %d)",
								node.selfId, frame, stakeForFrame, superMajorityStake, totalStake)
						}

						// Verify that each node with stake in this epoch takes part
						// in the formation of blocks, within the slack defined.
						// This is verified by checking that the node has seen events
						// carrying latest block indexes at least every BlockMissedSlack/2
						// emissions.
						if fakeWorld.validators.Exists(node.selfId) {
							blocksSeen := make([]idx.Block, 0)
							for _, event := range eventsInEpoch {
								blockInEvent := bigendian.BytesToUint64(event.Extra())
								blocksSeen = append(blocksSeen, idx.Block(blockInEvent))
							}
							maximumMissedBlockInterval := 0
							slices.Sort(blocksSeen)
							for i := 1; i < len(blocksSeen)-1; i++ {
								maximumMissedBlockInterval = max(maximumMissedBlockInterval,
									int(blocksSeen[i]-blocksSeen[i-1]))
							}

							assert.LessOrEqual(t, maximumMissedBlockInterval,
								int(testRules.Economy.BlockMissedSlack),
								"node %d has missed too many blocks (%d) between emissions",
								node.selfId, maximumMissedBlockInterval)
						}

						// Each node must reach a frame number of at least 1/3 of
						// the total emissions in the epoch. If no events were dropped,
						// this could be set to 100%.
						require.GreaterOrEqual(t, node.lastSeenFrameNumber(),
							idx.Frame(epochEmissions/3),
							"node %d did not reach expected frame", node.selfId)
					}

					// prepare for next epoch
					for _, node := range net.nodes {
						node.reset()
						node.currentEpoch++
					}
				})
		}
	}
}

// network simulates a set of nodes communicating with each other.
type network struct {
	t     testing.TB
	nodes map[idx.ValidatorID]*node
}

func newNetwork(t testing.TB,
	numNodes int,
	world WorldReader,
	dominantSetThreshold float64,
	repeatedFramesMaxCount uint,
) *network {
	nodes := make(map[idx.ValidatorID]*node)
	for i := range numNodes {
		id := idx.ValidatorID(i + 1)
		nodes[id] = newNode(t, id, world,
			dominantSetThreshold,
			repeatedFramesMaxCount,
		)
	}
	return &network{
		t:     t,
		nodes: nodes,
	}
}

// node simulates a node in the network.
type node struct {
	t         testing.TB
	throttler ThrottlingState
	world     WorldReader

	// mini Lachesis implementation:
	// does not find closures in dag, just tracks frames and parents
	selfId           idx.ValidatorID
	parentlessEvents []inter.EventPayloadI
	confirmedEvents  map[hash.Event]inter.EventPayloadI
	ownEvents        map[hash.Event]inter.EventPayloadI
	lastEventPerPeer map[idx.ValidatorID]inter.EventPayloadI

	lastSequenceNumber idx.Event
	lastFrame          idx.Frame
	currentEpoch       idx.Epoch
}

// newNode creates a new node in the network.
func newNode(
	t testing.TB,
	selfId idx.ValidatorID,
	world WorldReader,
	dominantSetThreshold float64,
	repeatedFramesMaxCount uint,
) *node {
	node := &node{
		t:         t,
		throttler: *NewThrottlingState(selfId, dominantSetThreshold, repeatedFramesMaxCount, world),
		world:     world,

		selfId: selfId,
	}
	node.reset()
	return node
}

// reset clears the node state for a new epoch.
func (node *node) reset() {
	node.parentlessEvents = make([]inter.EventPayloadI, 0)
	node.confirmedEvents = make(map[hash.Event]inter.EventPayloadI)
	node.ownEvents = make(map[hash.Event]inter.EventPayloadI)
	node.lastEventPerPeer = make(map[idx.ValidatorID]inter.EventPayloadI)
	node.lastSequenceNumber = 0
	node.lastFrame = 1
}

// createEvent creates a new event for the node. It uses the last known
// events from other nodes as parents.
func (node *node) createEvent() *inter.EventPayload {

	builder := &inter.MutableEventPayload{}
	builder.SetVersion(2)
	builder.SetCreator(node.selfId)
	builder.SetSeq(node.lastSequenceNumber + 1)
	builder.SetEpoch(node.currentEpoch)

	maxLamport := idx.Lamport(0)
	parents := hash.Events{}
	var selfParent inter.EventPayloadI
	for id, parent := range node.lastEventPerPeer {
		parents = append(parents, parent.ID())
		maxLamport = idx.MaxLamport(maxLamport, parent.Lamport())
		if id == builder.Creator() {
			selfParent = parent
		}
	}
	builder.SetParents(parents)

	// set extra data: latest block index, to check that nodes with stake
	// do emit and are not flagged as inactive
	latestBlock := bigendian.Uint64ToBytes(uint64(node.world.GetLatestBlockIndex()))
	builder.SetExtra(latestBlock)

	builder.SetLamport(maxLamport + 1)
	if selfParent != nil {
		builder.SetCreationTime(inter.MaxTimestamp(inter.Timestamp(time.Now().UnixNano()), selfParent.CreationTime()+1))
	}

	builder.SetFrame(node.getNextFrameNumber())
	event := builder.Build()
	node.ownEvents[event.ID()] = event
	return event
}

// getNextFrameNumber computes the next frame number for the node's own event.
// The node checks the last known events from other
// nodes and computes the accumulated stake of the validators that have emitted
// events in each frame. To advance to the next frame, the node requires that
// the accumulated stake in a frame exceeds 2/3+1 of the total stake.
func (node *node) getNextFrameNumber() idx.Frame {
	validators, _ := node.world.GetEpochValidators()

	stakeSeen := make(map[idx.Frame]pos.Weight)
	for creator, event := range node.lastEventPerPeer {
		require.Equal(node.t, creator, event.Creator())
		validatorStake := validators.Get(creator)
		stakeSeen[event.Frame()] += validatorStake
	}

	totalStake := validators.TotalWeight()
	nextFrame := node.lastFrame
	for frame, stake := range stakeSeen {
		if stake > (totalStake*2)/3+1 && frame >= nextFrame {
			nextFrame = max(nextFrame, frame+1)
		}
	}
	node.lastFrame = nextFrame

	require.GreaterOrEqual(node.t, nextFrame, node.lastFrame,
		"frame counter must be monotonic for own events")

	return nextFrame
}

// receiveEvent simulates receiving an event from the network.
func (node *node) receiveEvent(event *inter.EventPayload) {

	if event.Creator() == node.selfId {
		node.confirmedEvents[event.ID()] = event
		node.lastEventPerPeer[event.Creator()] = event
		node.lastSequenceNumber = event.Seq()
		return
	}

	require := require.New(node.t)
	require.NotNil(event, "event must not be nil")
	require.NotZero(event.ID(), "event ID must be set")
	require.NotZero(event.Creator(), "event creator must be set")
	require.NotZero(event.Frame(), "frame must be at least 1")
	lastPeerEvent, ok := node.lastEventPerPeer[event.Creator()]
	if ok {
		require.Less(lastPeerEvent.Seq(), event.Seq(), "sequence number must be monotonic per creator")
		require.LessOrEqual(lastPeerEvent.Frame(), event.Frame(), "frame counter must be monotonic per creator")
	}

	allParentsKnown := node.resolveEvent(event)
	if allParentsKnown {

		// can we resolve more parentless events now?
		for _, pe := range node.parentlessEvents {
			node.resolveEvent(pe.(*inter.EventPayload))
		}

	} else {
		node.parentlessEvents = append(node.parentlessEvents, event)
	}
}

// resolveEvent tries to resolve the given event by checking if all its parents
// are known to the node. If all parents are known, the event is marked as
// confirmed and added to the node's state.
func (node *node) resolveEvent(event *inter.EventPayload) bool {
	allParentsKnown := true
	for _, parentID := range event.Parents() {
		if _, ok := node.confirmedEvents[parentID]; !ok {
			allParentsKnown = false
			break
		}
	}

	if allParentsKnown {
		// mark event as confirmed
		node.confirmedEvents[event.ID()] = event
		node.lastEventPerPeer[event.Creator()] = event
		if event.Creator() == node.selfId {
			node.lastSequenceNumber = event.Seq()
		}
	}

	return allParentsKnown
}

// lastSeenFrameNumber returns the highest frame number seen among confirmed events
func (node *node) lastSeenFrameNumber() idx.Frame {
	res := idx.Frame(0)
	for _, event := range node.lastEventPerPeer {
		res = max(res, event.Frame())
	}
	return res
}

type fakeWorld struct {
	validators *pos.Validators
	rules      opera.Rules
	lastBlock  idx.Block
}

func (f *fakeWorld) GetEpochValidators() (*pos.Validators, idx.Epoch) {
	return f.validators, 0
}

func (f *fakeWorld) GetLatestBlockIndex() idx.Block {
	return f.lastBlock
}

func (f *fakeWorld) GetRules() opera.Rules {
	return f.rules
}

// makeTestStakeDistribution creates a stake distribution for testing purposes.
// It creates a list of stakes for 'length' validators, where the first
// 'dominators' validators hold enough stake to exceed the 'threshold' fraction
// of the total stake. The remaining validators share the rest of the stake
// equally. The stakes are then shuffled randomly.
func makeTestStakeDistribution(t *testing.T, length int, dominators int, threshold float64) []int64 {
	t.Helper()

	const totalStake = 1_000

	require.LessOrEqual(t, dominators, length)
	require.Greater(t, threshold, 0.667)
	require.LessOrEqual(t, threshold, 1.0)

	stakes := make([]int64, length)

	dominatorStake := int64(float64(totalStake) * threshold)
	remainingStake := totalStake - dominatorStake

	for i := range dominators {
		stakes[i] = dominatorStake / int64(dominators)
	}
	for i := dominators; i < length; i++ {
		stakes[i] = remainingStake / int64(length-dominators)
	}

	rand.Shuffle(len(stakes), func(i, j int) {
		stakes[i], stakes[j] = stakes[j], stakes[i]
	})

	return stakes
}

func TestMakeTestStakeDistribution(t *testing.T) {

	for dominantCount := range 10 {
		for length := dominantCount; length <= 10; length++ {
			stakes := makeTestStakeDistribution(t, length, dominantCount, 0.75)

			for _, stake := range stakes {
				require.GreaterOrEqual(t, stake, int64(0))
			}
		}
	}
}
