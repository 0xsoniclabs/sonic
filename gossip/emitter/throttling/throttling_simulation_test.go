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
	"slices"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/stretchr/testify/require"
)

func Test_SkipEvents_FrameProgressionWhenAllNodesAreOnline(t *testing.T) {
	t.Parallel()
	stakes := map[string][]int64{
		"single":           {1},
		"uniform_5":        slices.Repeat([]int64{100}, 5),
		"uniform_10":       slices.Repeat([]int64{42}, 10),
		"uniform_100":      slices.Repeat([]int64{21}, 100),
		"two dominating":   {50, 20, 10, 10, 10},
		"three dominating": {40, 30, 20, 5, 5},
	}
	threshold := []float64{
		0.70, 0.75, 0.80, 0.90, 0.95, 1.00,
	}

	for name, stakeDist := range stakes {
		for _, th := range threshold {
			t.Run(fmt.Sprintf("%s/threshold=%.2f", name, th),
				func(t *testing.T) {
					t.Parallel()
					testAllNodesOnline(t, th, stakeDist)
				},
			)
		}
	}
}

// testAllNodesOnline runs a simulation where all nodes are online and checks
// that they all make progress. Furthermore, it checks that nodes in the
// dominant set produce events at every round, while others produce less
// frequently.
func testAllNodesOnline(
	t *testing.T,
	threshold float64,
	stakes []int64,
) {
	const numRounds = 100
	require := require.New(t)

	world := &fakeWorld{
		rules: opera.Rules{
			Economy: opera.EconomyRules{
				BlockMissedSlack: 100,
			},
		},
		validators: makeValidators(stakes...),
	}

	// Run the network for a few rounds, checking that all nodes make progress.
	network := newNetwork(world,
		threshold,
		10,   // repeatedFramesMaxCount: use a large value to disable repeated-frame-based emissions
		1000, // heartbeatFrames: use a larger than repetitions value to disable heartbeat-based emissions
	)
	for cur := range numRounds {
		network.runRound(nil)

		// Each node should progress one frame per round.
		for _, node := range network.nodes {
			require.EqualValues(cur+1, node.lastSeenFrameNumber())
		}
	}

	// Count the number of events produced by each node.
	totalEventsPerNode := make(map[idx.ValidatorID]int)
	for _, event := range network.allEvents {
		totalEventsPerNode[event.Creator()]++
	}

	// Validators of the dominating set must have produced one event per round,
	// while others should have produced less.
	dominantSet := ComputeDominantSet(world.validators, world.validators.TotalWeight(), threshold)
	for i, count := range totalEventsPerNode {
		if _, included := dominantSet[i]; included {
			require.Equal(numRounds, count,
				"dominant node %d did not emit in every round", i)
		} else {
			require.Less(count, numRounds, "suppressed node %d emitted in more rounds than needed", i)
		}
	}
}

func Test_SkipEvents_NodesBeingOffline(t *testing.T) {
	t.Parallel()
	const threshold = 0.75
	cases := map[string]struct {
		stakes      []int64
		offlineMask offlineValidators
	}{
		"single dominating node is offline": {
			// 5 nodes, each 20% stake; threshold 75% => the last node could throttle
			stakes:      []int64{20, 20, 20, 20, 20},
			offlineMask: offlineValidators{1}, // < first node is offline
		},

		"two dominating nodes are offline": {
			// 10 nodes, each 10% stake; threshold 75%; last 2 nodes could throttle
			stakes:      slices.Repeat([]int64{10}, 10),
			offlineMask: offlineValidators{1, 2}, // < first two nodes are offline
		},

		"second-most dominating nodes is offline": {
			// 10 nodes, each 10% stake; threshold 75%; last 2 nodes could throttle
			stakes:      slices.Repeat([]int64{10}, 10),
			offlineMask: offlineValidators{2}, // < second node is offline
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			testPartiallyOnlineNodes(
				t,
				threshold,
				test.stakes,
				test.offlineMask,
			)
		})
	}
}

// testPartiallyOnlineNodes runs a simulation where all nodes are online and checks
// that they all make progress. Furthermore, it checks that nodes in the
// dominant set produce events at every round, while others produce less
// frequently.
func testPartiallyOnlineNodes(
	t *testing.T,
	threshold float64,
	stakes []int64,
	offline offlineValidators,
) {
	const numRounds = 100
	const repeatedFramesMaxCount = 10
	const heartbeatFrames = 1000
	require := require.New(t)

	world := &fakeWorld{
		rules: opera.Rules{
			Economy: opera.EconomyRules{
				BlockMissedSlack: 100,
			},
		},
		validators: makeValidators(stakes...),
	}

	// Run the network for a few rounds, checking that all nodes make progress.
	network := newNetwork(world, threshold, repeatedFramesMaxCount, heartbeatFrames)
	for range numRounds {
		network.runRound(offline)
	}

	// Check whether progress was made by nodes. Since some nodes were offline,
	// others non-dominant nodes should have started emitting frames as well,
	// preserving progress. However, progress may be slower by the number of
	// allowed repeated frames.
	wantedFrames := idx.Frame(numRounds / repeatedFramesMaxCount)
	for i, node := range network.nodes {
		require.LessOrEqual(
			wantedFrames, node.lastSeenFrameNumber(),
			"node %d did not make expected progress", i+1,
		)
	}

	// Count the number of events produced by each node.
	totalEventsPerNode := make(map[idx.ValidatorID]int)
	for _, event := range network.allEvents {
		totalEventsPerNode[event.Creator()]++
	}

	// Offline nodes must not have produced any events.
	for id, count := range totalEventsPerNode {
		if offline.isOffline(id) {
			require.Zero(count, "offline node %d emitted events", id)
		}
	}
}
func TestThrottler_CanSkipEvent_NetworkStallsWhenOneThirdOfStakesIsOffline(t *testing.T) {
	const threshold = 0.75
	const numNodes = 10
	const repeatedFramesMaxCount = 4
	const heartbeatFrames = 1000
	require := require.New(t)

	stakes := slices.Repeat([]int64{10}, numNodes)

	world := &fakeWorld{
		rules: opera.Rules{
			Economy: opera.EconomyRules{
				BlockMissedSlack: 4,
			},
		},
		validators: makeValidators(stakes...),
	}

	// Run the network for a few rounds, checking that all nodes make progress.
	network := newNetwork(world, threshold, repeatedFramesMaxCount, heartbeatFrames)

	// -- All Online --

	// In the first round, everyone is online, and all nodes should make progress.
	network.runRound(nil)
	for _, node := range network.nodes {
		require.EqualValues(1, node.lastSeenFrameNumber())
	}

	// -- Drop 40% Stake --

	// In the second round, 4 nodes go offline (40% of stake).
	offline := offlineValidators{1, 2, 3, 4}
	network.runRound(offline)

	// Nodes still see new frames based on the results of round 1.
	for _, node := range network.nodes {
		require.EqualValues(2, node.lastSeenFrameNumber())
	}

	// But after this, the network stalls.
	for range 10 {
		network.runRound(offline)
		for _, node := range network.nodes {
			require.EqualValues(2, node.lastSeenFrameNumber())
		}
	}

	// -- Bring back 8/10 nodes --

	// Bringing back some nodes (80% of stake) should allow progress again.
	offline = offlineValidators{1, 2} // only 2 nodes offline now
	network.runRound(offline)

	// In the first round after recovery, nodes should still be at frame 2,
	// since only after this round enough events for frame 2 enabling the
	// progression to frame 3 have been signed and distributed.
	for _, node := range network.nodes {
		require.EqualValues(2, node.lastSeenFrameNumber())
	}

	network.runRound(offline)

	// In the second round after recovery, nodes should have progressed to frame 3.
	for _, node := range network.nodes {
		require.EqualValues(3, node.lastSeenFrameNumber())
	}
}

func TestThrottler_CanSkipEvent_SuppressedValidatorsEmit_WhenDominatingValidatorsAreAbsent(t *testing.T) {
	t.Parallel()

	for _, dominantTimeout := range []uint{1, 2, 5, 10} {
		t.Run(fmt.Sprintf("dominantTimeout=%d", dominantTimeout), func(t *testing.T) {
			t.Parallel()

			world := &fakeWorld{
				rules: opera.Rules{
					Economy: opera.EconomyRules{
						BlockMissedSlack: 1000,
					},
				},
				validators: makeValidators(80, 20),
			}
			net := newNetwork(world,
				0.75,
				uint64(dominantTimeout),
				1000, // heartbeatFrames: use a larger than repetitions value to disable heartbeat-based emissions
			)

			events := net.runRound(nil)
			require.ElementsMatch(t,
				world.validators.IDs(),
				slices.Collect(maps.Keys(events)),
				"all nodes emit genesis")

			events = net.runRound(nil)
			require.ElementsMatch(t,
				[]idx.ValidatorID{1},
				slices.Collect(maps.Keys(events)),
				"only dominant node emits")

			for range dominantTimeout {
				events = net.runRound(offlineValidators{1})
				require.ElementsMatch(t,
					[]idx.ValidatorID{},
					slices.Collect(maps.Keys(events)),
					"dominant node is offline but timeout is not reached yet, nobody emits")
			}

			events = net.runRound(offlineValidators{1})
			require.ElementsMatch(t,
				[]idx.ValidatorID{2},
				slices.Collect(maps.Keys(events)),
				"after dominant timeout attempts, non-dominant node emits")

			events = net.runRound(nil)
			require.ElementsMatch(t,
				[]idx.ValidatorID{1, 2},
				slices.Collect(maps.Keys(events)),
				"node comes online again, both emit because 2 does not known yet")

			events = net.runRound(nil)
			require.ElementsMatch(t,
				[]idx.ValidatorID{1},
				slices.Collect(maps.Keys(events)),
				"one round after, only dominant emits again")
		})
	}
}

func TestThrottler_CanSkipEvent_SuppressedValidatorsEmitHeartbeat(t *testing.T) {
	t.Parallel()

	for _, dominantTimeout := range []uint64{1, 2, 3, 7, 11} {
		for _, heartbeatAttempts := range []uint64{7, 11, 15, 25, 100} {
			t.Run(fmt.Sprintf("dominantTimeout=%d/heartbeatAttempts=%d",
				dominantTimeout, heartbeatAttempts),
				func(t *testing.T) {
					t.Parallel()

					world := &fakeWorld{
						rules: opera.Rules{
							Economy: opera.EconomyRules{
								BlockMissedSlack: 100,
							},
						},
						validators: makeValidators(80, 20),
					}

					net := newNetwork(world,
						0.75,
						dominantTimeout,
						heartbeatAttempts,
					)

					events := net.runRound(nil)
					require.ElementsMatch(t,
						world.validators.IDs(),
						slices.Collect(maps.Keys(events)),
						"all nodes emit genesis")

					for i := range heartbeatAttempts/2 - 1 {
						events = net.runRound(nil)
						require.ElementsMatch(t,
							[]idx.ValidatorID{1},
							slices.Collect(maps.Keys(events)),
							"only dominant node emits, try %d", i)
					}

					events = net.runRound(offlineValidators{1})
					require.ElementsMatch(t,
						[]idx.ValidatorID{2},
						slices.Collect(maps.Keys(events)),
						"this is a heartbeat emission, and 1 is offline, so only 2 emits")

					// validator 2 will take some time to figure out that
					// validator 1 is offline and start emitting on its own,
					// this is the first happening of two possible conditions:
					// - heartbeat-based emission
					// - dominant-timeout-based emission
					for i := range min(dominantTimeout-1, heartbeatAttempts/2-1) {
						events = net.runRound(offlineValidators{1})
						require.ElementsMatch(t,
							[]idx.ValidatorID{},
							slices.Collect(maps.Keys(events)),
							"offline dominant node, but not yet heartbeat emission, try %d", i)
					}

					events = net.runRound(offlineValidators{1})
					require.ElementsMatch(t,
						[]idx.ValidatorID{2},
						slices.Collect(maps.Keys(events)),
						"validator 2 must emit due to heartbeat or dominant timeout being reached")
				})
		}
	}
}

func TestThrottler_CanSkipEvent_SuppressedValidatorsFillOfflineProgressively(t *testing.T) {
	t.Parallel()

	world := &fakeWorld{
		rules: opera.Rules{
			Economy: opera.EconomyRules{
				BlockMissedSlack: 100,
			},
		},
		validators: makeValidators(slices.Repeat([]int64{10}, 10)...),
	}
	net := newNetwork(world,
		0.70,
		1,    // suppressed validators will emit one round after dominant validators go offline
		1000, // heartbeatFrames: use a larger than repetitions value to disable heartbeat-based emissions
	)

	events := net.runRound(nil)
	require.ElementsMatch(t,
		world.validators.IDs(),
		slices.Collect(maps.Keys(events)),
		"all nodes emit genesis")

	events = net.runRound(offlineValidators{1})
	require.ElementsMatch(t,
		[]idx.ValidatorID{2, 3, 4, 5, 6, 7},
		slices.Collect(maps.Keys(events)))

	events = net.runRound(offlineValidators{1})
	require.ElementsMatch(t,
		[]idx.ValidatorID{2, 3, 4, 5, 6, 7, 8},
		slices.Collect(maps.Keys(events)))

	events = net.runRound(offlineValidators{1, 2})
	require.ElementsMatch(t,
		[]idx.ValidatorID{3, 4, 5, 6, 7, 8},
		slices.Collect(maps.Keys(events)))

	events = net.runRound(offlineValidators{1, 2})
	require.ElementsMatch(t,
		[]idx.ValidatorID{3, 4, 5, 6, 7, 8, 9},
		slices.Collect(maps.Keys(events)))

	events = net.runRound(offlineValidators{1, 2, 3})
	require.ElementsMatch(t,
		[]idx.ValidatorID{4, 5, 6, 7, 8, 9},
		slices.Collect(maps.Keys(events)))

	events = net.runRound(offlineValidators{1, 2, 3})
	require.ElementsMatch(t,
		[]idx.ValidatorID{4, 5, 6, 7, 8, 9, 10},
		slices.Collect(maps.Keys(events)))

	events = net.runRound(nil)
	require.ElementsMatch(t,
		world.validators.IDs(),
		slices.Collect(maps.Keys(events)))

	events = net.runRound(nil)
	require.ElementsMatch(t,
		[]idx.ValidatorID{1, 2, 3, 4, 5, 6, 7},
		slices.Collect(maps.Keys(events)))
}

// --- Simulation Infrastructure ---

// network simulates a set of nodes communicating with each other.
type network struct {
	nodes     []*node
	allEvents []*inter.EventPayload
}

func newNetwork(
	world WorldReader,
	dominantSetThreshold float64,
	repeatedFramesMaxCount uint64,
	heartbeatFrames uint64,
) *network {
	validators, _ := world.GetEpochValidators()
	numNodes := validators.Len()
	nodes := make([]*node, 0, numNodes)
	for i := range numNodes {
		id := idx.ValidatorID(i + 1)
		nodes = append(nodes, newNode(id, world,
			dominantSetThreshold,
			repeatedFramesMaxCount,
			heartbeatFrames,
		))
	}
	net := &network{
		nodes: nodes,
	}
	// register network in the world for global state access
	world.(*fakeWorld).network = net
	return net
}

func (n *network) runRound(
	offlineMask offlineValidators,
) map[idx.ValidatorID]*inter.EventPayload {
	// Collect events from all nodes.
	events := make([]*inter.EventPayload, 0)
	for i, node := range n.nodes {
		_ = i
		if offlineMask.isOffline(node.selfId) {
			continue
		}
		if event := node.createEvent(); event != nil {
			events = append(events, event)
		}
	}

	// Collect all events in the network history.
	n.allEvents = append(n.allEvents, events...)

	// Distribute events to all nodes.
	for _, event := range events {
		for _, node := range n.nodes {
			node.receiveEvent(event)
		}
	}

	res := make(map[idx.ValidatorID]*inter.EventPayload)
	for _, event := range events {
		res[event.Creator()] = event
	}
	return res
}

// node simulates a node in the network.
type node struct {
	throttler ThrottlingState
	world     WorldReader

	// mini Lachesis implementation:
	// does not find closures in dag, just tracks frames and parents
	selfId           idx.ValidatorID
	lastEventPerPeer map[idx.ValidatorID]inter.EventPayloadI

	currentEpoch idx.Epoch
}

// newNode creates a new node in the network.
func newNode(
	selfId idx.ValidatorID,
	world WorldReader,
	dominantSetThreshold float64,
	repeatedFramesMaxCount uint64,
	heartbeatFrames uint64,
) *node {
	return &node{
		throttler: *NewThrottlingState(
			selfId,
			dominantSetThreshold,
			repeatedFramesMaxCount,
			heartbeatFrames,
			world),
		world:            world,
		selfId:           selfId,
		lastEventPerPeer: map[idx.ValidatorID]inter.EventPayloadI{},
	}
}

// createEvent creates a new event for the node. The result may be nil if this
// node's throttler decides to skip emission.
func (node *node) createEvent() *inter.EventPayload {

	builder := &inter.MutableEventPayload{}
	builder.SetVersion(2)
	builder.SetCreator(node.selfId)
	builder.SetEpoch(node.currentEpoch)

	maxLamport := idx.Lamport(0)
	parents := []inter.EventPayloadI{}
	parentIds := hash.Events{}
	var selfParent inter.EventPayloadI
	var selfParentPos int
	for id, parent := range node.lastEventPerPeer {
		parents = append(parents, parent)
		parentIds = append(parentIds, parent.ID())
		maxLamport = idx.MaxLamport(maxLamport, parent.Lamport())
		if id == builder.Creator() {
			selfParent = parent
		}
	}

	// set self-parent as first parent
	if selfParent != nil {
		parents[0], parents[selfParentPos] = parents[selfParentPos], parents[0]
	}
	builder.SetParents(parentIds)

	builder.SetLamport(maxLamport + 1)
	if selfParent != nil {
		builder.SetCreationTime(inter.MaxTimestamp(inter.Timestamp(time.Now().UnixNano()), selfParent.CreationTime()+1))
		builder.SetSeq(selfParent.Seq() + 1)
	} else {
		builder.SetCreationTime(inter.Timestamp(time.Now().UnixNano()))
		builder.SetSeq(1) // genesis event has seq 1
	}

	validators, _ := node.world.GetEpochValidators()
	builder.SetFrame(getFrameNumber(validators, parents))
	event := builder.Build()

	if node.throttler.CanSkipEventEmission(event) == SkipEventEmission {
		return nil
	}
	return event
}

// getFrameNumber computes the frame number for an event with the given parents.
func getFrameNumber(
	validators *pos.Validators,
	parents []inter.EventPayloadI,
) idx.Frame {
	// The frame of the new event is at least the frame number of the parents.
	frame := idx.Frame(1)
	for _, parent := range parents {
		frame = max(frame, parent.Frame())
	}

	// If the total stake seen in the parents' frames exceeds 2/3 of
	// the total stake, we can advance to the next frame.
	for {
		stakeSeen := pos.Weight(0)
		for _, parent := range parents {
			creator := parent.Creator()
			if frame <= parent.Frame() {
				stakeSeen += validators.Get(creator)
			}
		}
		if stakeSeen > (validators.TotalWeight()*2)/3 {
			frame++
		} else {
			break
		}
	}

	return frame
}

// receiveEvent simulates receiving an event from the network.
func (node *node) receiveEvent(event *inter.EventPayload) {
	node.lastEventPerPeer[event.Creator()] = event
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
	network    *network
	validators *pos.Validators
	rules      opera.Rules
}

func (f *fakeWorld) GetEpochValidators() (*pos.Validators, idx.Epoch) {
	return f.validators, 0
}

func (f *fakeWorld) GetRules() opera.Rules {
	return f.rules
}

func (f *fakeWorld) GetLastEvent(from idx.ValidatorID) *inter.Event {
	if f.network == nil {
		// for this test to function correctly, the world must have access to the network
		panic("ill-formed test: world has no network")
	}

	var lastEvent *inter.Event
	for _, event := range f.network.allEvents {
		if event.Creator() == from {
			if lastEvent == nil || event.CreationTime() > lastEvent.CreationTime() {
				lastEvent = &event.Event
			}
		}
	}
	return lastEvent
}

type offlineValidators []idx.ValidatorID

func (m offlineValidators) isOffline(id idx.ValidatorID) bool {
	return slices.Contains(m, id)
}
