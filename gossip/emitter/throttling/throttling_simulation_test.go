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
	"slices"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/stretchr/testify/require"
)

func Test_SkipEvents_FrameProgressionWhenAllNodesAreOnline(t *testing.T) {
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
			t.Run(
				fmt.Sprintf("%s/threshold=%.2f", name, th),
				func(t *testing.T) {
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
	numNodes := len(stakes)

	world := &fakeWorld{
		rules: opera.Rules{
			Economy: opera.EconomyRules{
				BlockMissedSlack: 4,
			},
		},
		validators: makeValidators(stakes...),
	}

	// Run the network for a few rounds, checking that all nodes make progress.
	network := newNetwork(numNodes, world, threshold, 10)
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
	dominantSet, _ := ComputeDominantSet(world.validators, threshold)
	for i, count := range totalEventsPerNode {
		if _, included := dominantSet[i]; included {
			require.Equal(numRounds, count)
		} else {
			require.Less(count, numRounds)
		}
	}
}

func Test_SkipEvents_NodesBeingOffline(t *testing.T) {
	const threshold = 0.75
	cases := map[string]struct {
		stakes      []int64
		offlineMask offlineMask
	}{
		"single dominating node is offline": {
			// 5 nodes, each 20% stake; threshold 75% => the last node could throttle
			stakes:      []int64{20, 20, 20, 20, 20},
			offlineMask: offlineMask{true}, // < first node is offline
		},

		"two dominating nodes are offline": {
			// 10 nodes, each 10% stake; threshold 75%; last 2 nodes could throttle
			stakes:      slices.Repeat([]int64{10}, 10),
			offlineMask: offlineMask{true, true}, // < first two nodes are offline
		},

		"second-most dominating nodes is offline": {
			// 10 nodes, each 10% stake; threshold 75%; last 2 nodes could throttle
			stakes:      slices.Repeat([]int64{10}, 10),
			offlineMask: offlineMask{1: true}, // < second node is offline
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			testPartiallyOnlineNodes(
				t,
				threshold,
				test.stakes,
				test.offlineMask,
			)
		})
	}
}

// testAllNodesOnline runs a simulation where all nodes are online and checks
// that they all make progress. Furthermore, it checks that nodes in the
// dominant set produce events at every round, while others produce less
// frequently.
func testPartiallyOnlineNodes(
	t *testing.T,
	threshold float64,
	stakes []int64,
	offlineMask offlineMask,
) {
	const numRounds = 100
	const repeatedFramesMaxCount = 10
	require := require.New(t)
	numNodes := len(stakes)

	world := &fakeWorld{
		rules: opera.Rules{
			Economy: opera.EconomyRules{
				BlockMissedSlack: 4,
			},
		},
		validators: makeValidators(stakes...),
	}

	// Run the network for a few rounds, checking that all nodes make progress.
	network := newNetwork(numNodes, world, threshold, repeatedFramesMaxCount)
	for range numRounds {
		network.runRound(offlineMask)
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
	for i, count := range totalEventsPerNode {
		if offlineMask.isOffline(int(i - 1)) {
			require.Zero(count, "offline node %d emitted events", i)
		}
	}
}

func Test_SkipEvents_NetworkStallsWhenOneThirdOfStakesIsOffline(t *testing.T) {
	const threshold = 0.75
	const numNodes = 10
	const repeatedFramesMaxCount = 4
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
	network := newNetwork(numNodes, world, threshold, repeatedFramesMaxCount)

	// -- All Online --

	// In the first round, everyone is online, and all nodes should make progress.
	network.runRound(nil)
	for _, node := range network.nodes {
		require.EqualValues(1, node.lastSeenFrameNumber())
	}

	// -- Drop 40% Stake --

	// In the second round, 4 nodes go offline (40% of stake).
	offline := offlineMask{true, true, true, true}
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
	offline = offlineMask{true, true} // only 2 nodes offline now
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

func Test_SkipEvents_OfflineNodes_GradualIncreaseInEmittedEvents(t *testing.T) {
	const threshold = 0.75
	const numNodes = 10
	const repeatedFramesMaxCount = 4
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
	network := newNetwork(numNodes, world, threshold, repeatedFramesMaxCount)

	// -- All Online --

	// In the first round, everyone is online, and all nodes should make progress.
	events := network.runRound(nil)
	require.Len(events, 8) // 2 least dominant nodes throttle

	// If one node goes offline (10% of stake), throttling nodes are kicking in.
	offline := offlineMask{true}
	events = network.runRound(offline)
	require.Len(events, 7) // 1 offline + 2 throttling nodes

	// This is a steady state, since progress is made.
	for range 5 {
		events = network.runRound(offline)
		require.Len(events, 7) // 1 offline + 2 throttling nodes
	}

	// If another node goes offline (20% of stake), extra nodes remain throttled.
	offline = offlineMask{true, true}
	events = network.runRound(offline)
	require.Len(events, 6) // 2 offline + 2 throttling node

	// 6/10 is to low for progress, so network stalls until we reach the max
	// repeated frames.
	for range repeatedFramesMaxCount - 1 {
		events = network.runRound(offline)
		require.Len(events, 6) // 2 offline + 2 throttling node
	}

	// After reaching the max repeated frames, throttling nodes emit again,
	// allowing progress.
	events = network.runRound(offline)
	require.Len(events, 8) // 2 offline, nobody throttled

	// This was a one-time thing. Now that there is progress, nodes are throttling again.
	for range repeatedFramesMaxCount {
		events = network.runRound(offline)
		require.Len(events, 6) // 2 offline + 2 throttling node
	}

	// And this repeats indefinitely.
	// TODO: decide whether this is fine, or whether we want this to be smoother.
	for range 100 {
		events = network.runRound(offline)
		require.Len(events, 8) // 2 offline, nobody throttled

		for range repeatedFramesMaxCount {
			events = network.runRound(offline)
			require.Len(events, 6) // 2 offline + 2 throttling node
		}
	}

	// -- Bring back all nodes --

	// Bringing back all nodes should restore full emission.
	offline = offlineMask{}
	events = network.runRound(offline)
	require.Len(events, 10) // all nodes emit

	// After this, nodes throttle again.
	for range 100 {
		events = network.runRound(offline)
		require.Len(events, 8) // 2 throttling nodes
	}
}

// --- Simulation Infrastructure ---

// network simulates a set of nodes communicating with each other.
type network struct {
	nodes     []*node
	allEvents []*inter.EventPayload
}

func newNetwork(
	numNodes int,
	world WorldReader,
	dominantSetThreshold float64,
	repeatedFramesMaxCount uint,
) *network {
	nodes := make([]*node, 0, numNodes)
	for i := range numNodes {
		id := idx.ValidatorID(i + 1)
		nodes = append(nodes, newNode(id, world,
			dominantSetThreshold,
			repeatedFramesMaxCount,
		))
	}
	return &network{
		nodes: nodes,
	}
}

func (n *network) runRound(
	offlineMask offlineMask,
) []*inter.EventPayload {
	// Collect events from all nodes.
	events := make([]*inter.EventPayload, 0)
	for i, node := range n.nodes {
		if offlineMask.isOffline(i) {
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
	return events
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
	repeatedFramesMaxCount uint,
) *node {
	return &node{
		throttler:        *NewThrottlingState(selfId, dominantSetThreshold, repeatedFramesMaxCount, world),
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
	for id, parent := range node.lastEventPerPeer {
		parents = append(parents, parent)
		parentIds = append(parentIds, parent.ID())
		maxLamport = idx.MaxLamport(maxLamport, parent.Lamport())
		if id == builder.Creator() {
			selfParent = parent
		}
	}
	builder.SetParents(parentIds)

	// set extra data: latest block index, to check that nodes with stake
	// do emit and are not flagged as inactive
	latestBlock := bigendian.Uint64ToBytes(uint64(node.world.GetLatestBlockIndex()))
	builder.SetExtra(latestBlock)

	builder.SetLamport(maxLamport + 1)
	if selfParent != nil {
		builder.SetCreationTime(inter.MaxTimestamp(inter.Timestamp(time.Now().UnixNano()), selfParent.CreationTime()+1))
	}

	validators, _ := node.world.GetEpochValidators()
	builder.SetFrame(getFrameNumber(validators, parents))
	event := builder.Build()

	if node.throttler.SkipEventEmission(event) {
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

func (f *fakeWorld) GetLastEvent(epoch idx.Epoch, from idx.ValidatorID) *hash.Event {
	return nil
}
func (f *fakeWorld) GetEvent(hash.Event) *inter.Event {
	return nil
}

type offlineMask []bool

func (m offlineMask) isOffline(i int) bool {
	return i < len(m) && m[i]
}
