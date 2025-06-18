package emitter

import (
	"context"
	"fmt"
	"iter"
	"math"
	"math/big"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/eventcheck/proposalcheck"
	"github.com/0xsoniclabs/sonic/gossip/emitter/scheduler"
	"github.com/0xsoniclabs/sonic/gossip/randao"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestSingleProposerProtocol_NonProposingValidators_ProtocolIsLive is an
// integration test for the single proposer protocol running simulations with
// various number of nodes, confirmation delays, and subsets of honest nodes.
// In each simulation, it is checked that progress can be made by the honest
// nodes, even if dishonest nodes do not propose anything.
func TestSingleProposerProtocol_SilentValidators_ProtocolIsLive(t *testing.T) {
	t.Parallel()
	for numNodes := range 6 {
		t.Run(
			fmt.Sprintf("numNodes=%d", numNodes),
			func(t *testing.T) {
				t.Parallel()
				for delay := range inter.TurnTimeoutInFrames + 2 {
					t.Run(
						fmt.Sprintf("confirmationDelay=%d", delay),
						func(t *testing.T) {
							t.Parallel()
							for honestNodes := range enumerateNodeMasks(numNodes) {
								if honestNodes.IsEmpty() {
									continue // at least one honest node is required
								}
								t.Run(
									fmt.Sprintf("honestNodes=%s", honestNodes),
									func(t *testing.T) {
										t.Parallel()
										testNetworkWithSilentNodes(
											t, numNodes,
											idx.Frame(delay), honestNodes,
										)
									},
								)
							}
						},
					)
				}
			},
		)
	}
}

func testNetworkWithSilentNodes(
	t *testing.T,
	numNodes int,
	confirmationDelay idx.Frame,
	honestNodes NodeMask,
) {
	const NumBlocks = 50
	maxRounds := NumBlocks * numNodes * int(confirmationDelay+1) * 10
	require := require.New(t)

	network := NewNetwork(t, numNodes)
	rounds := 0

	pending := []inter.EventPayloadI{}

	proposalCounts := map[idx.ValidatorID]int{}
	for network.GetNode(0).GetBlockHeight() < NumBlocks {
		for i, sender := range network.Nodes() {

			// Honest nodes may propose events with proposals, while dishonest
			// nodes propose valid events without proposals.
			var err error
			var event inter.EventPayloadI
			if honestNodes.Contains(i) {
				event, err = sender.EmitEvent()
				require.NoError(err)
			} else {
				event, err = sender.EmitEventWithoutProposal()
				require.NoError(err)
			}

			// Distribute the event to all nodes synchronously. Thus, all nodes
			// will receive the event at the same time.
			network.BroadCastEvent(event)

			// Keep track of in-flight proposals.
			if proposal := event.Payload().Proposal; proposal != nil {
				pending = append(pending, event)
				proposalCounts[event.Creator()]++
			}

			// Inform nodes about confirmed proposals.
			pending = slices.DeleteFunc(pending, func(p inter.EventPayloadI) bool {
				if event.Frame() > p.Frame()+confirmationDelay {
					for _, node := range network.Nodes() {
						node.ConfirmProposal(*p.Payload().Proposal)
					}
					return true
				}
				return false
			})
		}

		// Check that progress is made.
		rounds++
		require.Less(rounds, maxRounds,
			"Max rounds reached without reaching the target number of %d blocks, current height: %d",
			NumBlocks, network.GetNode(0).GetBlockHeight(),
		)
	}

	// Check the contributions of each node.
	for creator := range idx.ValidatorID(numNodes) {
		if honestNodes.Contains(int(creator)) {
			require.Greater(proposalCounts[creator], 0, "Node %d should have proposed something", creator)
		} else {
			require.Zero(proposalCounts[creator], "Node %d should not have proposed anything", creator)
		}
	}
}

// --- Simulation infrastructure for the single proposer protocol tests ---

// Network is a collection of simulated nodes that keeps track of administrative
// tasks such as broadcasting events and checking events for validity.
type Network struct {
	t          *testing.T
	validators *pos.Validators
	nodes      []*Node

	payloads map[hash.Event]inter.Payload
	checker  *proposalcheck.Checker
}

// NewNetwork creates a new network with the given number of nodes.
func NewNetwork(t *testing.T, numNodes int) *Network {
	builder := pos.NewBuilder()
	for id := range idx.ValidatorID(numNodes) {
		builder.Set(id, 1)
	}
	validators := builder.Build()
	nodes := make([]*Node, numNodes)
	for i := range nodes {
		nodes[i] = &Node{
			validator:  idx.ValidatorID(i),
			validators: validators,
		}
	}

	res := &Network{
		t:          t,
		validators: validators,
		nodes:      nodes,
		payloads:   make(map[hash.Event]inter.Payload),
	}

	ctrl := gomock.NewController(t)
	reader := proposalcheck.NewMockReader(ctrl)
	reader.EXPECT().GetEpochValidators().Return(res.validators).AnyTimes()
	reader.EXPECT().GetEventPayload(gomock.Any()).DoAndReturn(
		func(eventHash hash.Event) inter.Payload {
			return res.payloads[eventHash]
		},
	).AnyTimes()

	res.checker = proposalcheck.New(reader)

	return res
}

func (n *Network) Nodes() []*Node {
	return n.nodes
}

func (n *Network) GetNode(i int) *Node {
	return n.nodes[i]
}

func (n *Network) BroadCastEvent(event inter.EventPayloadI) {
	require.NoError(n.t, n.checker.Validate(event))

	n.payloads[event.ID()] = *event.Payload()
	for _, node := range n.Nodes() {
		node.ReceiveEvent(event)
	}
}

// Node is a simulated node in the network that can emit events on demand. It
// also tracks its own local state, including the latest events seen from other
// nodes and the current block height.
type Node struct {
	validator  idx.ValidatorID
	validators *pos.Validators
	lastBlock  uint64

	tips     map[idx.ValidatorID]inter.EventPayloadI
	payloads map[hash.Event]inter.Payload
	tracker  inter.ProposalTracker
}

func (n *Node) GetBlockHeight() uint64 {
	return n.lastBlock
}

func (n *Node) EmitEvent() (inter.EventPayloadI, error) {
	return n.emitEventInternal(true)
}

func (n *Node) EmitEventWithoutProposal() (inter.EventPayloadI, error) {
	return n.emitEventInternal(false)
}

func (n *Node) emitEventInternal(
	includeProposalIfPossible bool,
) (inter.EventPayloadI, error) {
	// This function builds an event with payload data sufficient to pass the
	// proposal checker.
	event := inter.MutableEventPayload{}
	event.SetVersion(3)
	event.SetCreator(n.validator)

	// Add parents by referencing all latest events from other validators.
	selfParent := n.tips[n.validator]
	if selfParent == nil {
		// Create a genesis event for this epoch.
		event.SetSeq(1)
		event.SetFrame(1)
	} else {
		// Create an event with parents.
		event.SetSeq(selfParent.Seq() + 1)
		parents := []hash.Event{selfParent.ID()}
		for id, tip := range n.tips {
			if id != n.validator {
				parents = append(parents, tip.ID())
			}
		}
		event.SetParents(parents)
		event.SetFrame(n.getNextFrameNumber())

		creationTime := selfParent.CreationTime()
		for _, tip := range n.tips {
			if tip.CreationTime() > creationTime {
				creationTime = tip.CreationTime()
			}
		}
		creationTime += inter.Timestamp(500 * time.Millisecond)
		event.SetCreationTime(creationTime)
		event.SetMedianTime(creationTime) // is not checked, but needs to progress
	}

	// Create the payload for the event.
	creator := n.validator
	if !includeProposalIfPossible {
		creator = idx.ValidatorID(math.MaxUint32) // < invalid creator
	}
	payload, err := createPayload(
		&fakeWorld{node: n},
		creator,
		n.validators,
		&event,
		&n.tracker,
		nil,
		fakeScheduler{},
		fakeRandaoMixer{},
		fakeTimerMetric{},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create payload: %w", err)
	}
	event.SetPayload(payload)

	// Complete the event by sealing it and be done.
	return event.Build(), nil
}

func (n *Node) getNextFrameNumber() idx.Frame {
	res := idx.Frame(1)
	seen := map[idx.Frame]int{}
	for _, tip := range n.tips {
		seen[tip.Frame()]++
		if tip.Frame() > res {
			res = tip.Frame()
		}
	}

	for frame, count := range seen {
		if count > int(n.validators.Len()*2/3) {
			res = frame + 1
		}
	}
	return res
}

func (n *Node) ReceiveEvent(event inter.EventPayloadI) {
	// Keep track of known events of other validators.
	if n.tips == nil {
		n.tips = make(map[idx.ValidatorID]inter.EventPayloadI)
	}
	n.tips[event.Creator()] = event

	// Keep track of payloads.
	if n.payloads == nil {
		n.payloads = make(map[hash.Event]inter.Payload)
	}
	n.payloads[event.ID()] = *event.Payload()

	// Keep track of seen proposals.
	if proposal := event.Payload().Proposal; proposal != nil {
		n.tracker.RegisterSeenProposal(event.Frame(), proposal.Number)
	}
}

func (n *Node) ConfirmProposal(proposal inter.Proposal) {
	// A confirmed proposal means that the local block state can move one step
	// forward. Out-of-order confirmed proposals are ignored.
	newBlock := uint64(proposal.Number)
	if newBlock == n.lastBlock+1 {
		n.lastBlock = newBlock
	}
}

// --- fakes required for the implementation of nodes ---

// fakeWorld adapts the Node's internal state to the interface required by the
// payload creation logic.
type fakeWorld struct {
	node *Node
}

func (w *fakeWorld) GetEventPayload(id hash.Event) inter.Payload {
	return w.node.payloads[id]
}

func (w *fakeWorld) GetLatestBlock() *inter.Block {
	return (&inter.BlockBuilder{}).
		WithNumber(w.node.lastBlock).
		WithBaseFee(big.NewInt(1_000_000)).
		Build()
}

func (w *fakeWorld) GetRules() opera.Rules {
	return opera.Rules{}
}

// fakeRandaoMixer is producing fake RANDAO reveals for the tests.
type fakeRandaoMixer struct{}

func (fakeRandaoMixer) MixRandao(common.Hash) (randao.RandaoReveal, common.Hash, error) {
	return randao.RandaoReveal{}, common.Hash{}, nil
}

// fakeScheduler is a no-op scheduler for the tests. It does not schedule any
// transactions, as the tests do not require transaction scheduling.
type fakeScheduler struct{}

func (fakeScheduler) Schedule(
	context.Context,
	*scheduler.BlockInfo,
	scheduler.PrioritizedTransactions,
	scheduler.Limits,
) []*types.Transaction {
	return nil
}

// fakeTimerMetric is a no-op timer metric for the tests. It ignores any calls.
type fakeTimerMetric struct{}

func (fakeTimerMetric) Update(time.Duration) {}

// NodeMask is a bitmask used by the test infrastructure above to identify
// subsets of nodes. In particular, it is used to identify honest nodes.
type NodeMask uint32

func (mask NodeMask) Contains(id int) bool {
	return (mask & (1 << id)) != 0
}

func (mask NodeMask) IsEmpty() bool {
	return mask == 0
}

func (mask NodeMask) String() string {
	builder := strings.Builder{}
	builder.WriteString("{")
	for i := 0; i < 32; i++ {
		if mask.Contains(i) {
			if builder.Len() > 1 {
				builder.WriteString(",")
			}
			builder.WriteString(fmt.Sprintf("%d", i))
		}
	}
	builder.WriteString("}")
	return builder.String()
}

// enumerateNodeMasks generates all possible node masks for a given number of
// nodes. The maximum number of nodes is 32.
func enumerateNodeMasks(numNodes int) iter.Seq[NodeMask] {
	if numNodes < 0 || numNodes > 32 {
		panic(fmt.Sprintf("numNodes must be in range [0, 32], got %d", numNodes))
	}
	return func(yield func(NodeMask) bool) {
		for mask := NodeMask(0); mask < (1 << numNodes); mask++ {
			if !yield(mask) {
				return
			}
		}
	}
}
