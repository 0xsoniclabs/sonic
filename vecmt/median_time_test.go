package vecmt

import (
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/0xsoniclabs/consensus/vecengine"
	"github.com/stretchr/testify/assert"

	"github.com/0xsoniclabs/kvdb/memorydb"

	"github.com/0xsoniclabs/sonic/inter"
)

func TestMedianTimeOnIndex(t *testing.T) {
	nodes := consensus.GenNodes(5)
	weights := []consensus.Weight{5, 4, 3, 2, 1}
	validators := consensus.ArrayToValidators(nodes, weights)

	vi := NewIndex(func(err error) { panic(err) }, LiteConfig())
	vi.Reset(validators, memorydb.New(), nil)

	assertar := assert.New(t)
	{ // seq=0
		e := consensus.ZeroEventHash
		// validator indexes are sorted by weight amount
		before := NewHighestBefore(consensus.ValidatorIndex(validators.Len()))

		before.VSeq.Set(0, vecengine.BranchSeq{Seq: 0})
		before.VTime.Set(0, 100)

		before.VSeq.Set(1, vecengine.BranchSeq{Seq: 0})
		before.VTime.Set(1, 100)

		before.VSeq.Set(2, vecengine.BranchSeq{Seq: 1})
		before.VTime.Set(2, 10)

		before.VSeq.Set(3, vecengine.BranchSeq{Seq: 1})
		before.VTime.Set(3, 10)

		before.VSeq.Set(4, vecengine.BranchSeq{Seq: 1})
		before.VTime.Set(4, 10)

		vi.SetHighestBefore(e, before)
		assertar.Equal(inter.Timestamp(1), vi.MedianTime(e, 1))
	}

	{ // fork seen = true
		e := consensus.ZeroEventHash
		// validator indexes are sorted by weight amount
		before := NewHighestBefore(consensus.ValidatorIndex(validators.Len()))

		before.SetForkDetected(0)
		before.VTime.Set(0, 100)

		before.SetForkDetected(1)
		before.VTime.Set(1, 100)

		before.VSeq.Set(2, vecengine.BranchSeq{Seq: 1})
		before.VTime.Set(2, 10)

		before.VSeq.Set(3, vecengine.BranchSeq{Seq: 1})
		before.VTime.Set(3, 10)

		before.VSeq.Set(4, vecengine.BranchSeq{Seq: 1})
		before.VTime.Set(4, 10)

		vi.SetHighestBefore(e, before)
		assertar.Equal(inter.Timestamp(10), vi.MedianTime(e, 1))
	}

	{ // normal
		e := consensus.ZeroEventHash
		// validator indexes are sorted by weight amount
		before := NewHighestBefore(consensus.ValidatorIndex(validators.Len()))

		before.VSeq.Set(0, vecengine.BranchSeq{Seq: 1})
		before.VTime.Set(0, 11)

		before.VSeq.Set(1, vecengine.BranchSeq{Seq: 2})
		before.VTime.Set(1, 12)

		before.VSeq.Set(2, vecengine.BranchSeq{Seq: 2})
		before.VTime.Set(2, 13)

		before.VSeq.Set(3, vecengine.BranchSeq{Seq: 3})
		before.VTime.Set(3, 14)

		before.VSeq.Set(4, vecengine.BranchSeq{Seq: 4})
		before.VTime.Set(4, 15)

		vi.SetHighestBefore(e, before)
		assertar.Equal(inter.Timestamp(12), vi.MedianTime(e, 1))
	}

}

func TestMedianTimeOnDAG(t *testing.T) {
	dagAscii := `
 ║
 nodeA001
 ║
 nodeA012
 ║            ║
 ║            nodeB001
 ║            ║            ║
 ║            ╠═══════════ nodeC001
 ║║           ║            ║            ║
 ║╚══════════─╫─══════════─╫─══════════ nodeD001
║║            ║            ║            ║
╚ nodeA002════╬════════════╬════════════╣
 ║║           ║            ║            ║
 ║╚══════════─╫─══════════─╫─══════════ nodeD002
 ║            ║            ║            ║
 nodeA003════─╫─══════════─╫─═══════════╣
 ║            ║            ║
 ╠════════════nodeB002     ║
 ║            ║            ║
 ╠════════════╫═══════════ nodeC002
`

	weights := []consensus.Weight{3, 4, 2, 1}
	genesisTime := inter.Timestamp(1)
	creationTimes := map[string]inter.Timestamp{
		"nodeA001": inter.Timestamp(111),
		"nodeB001": inter.Timestamp(112),
		"nodeC001": inter.Timestamp(13),
		"nodeD001": inter.Timestamp(14),
		"nodeA002": inter.Timestamp(120),
		"nodeD002": inter.Timestamp(20),
		"nodeA012": inter.Timestamp(120),
		"nodeA003": inter.Timestamp(20),
		"nodeB002": inter.Timestamp(20),
		"nodeC002": inter.Timestamp(35),
	}
	medianTimes := map[string]inter.Timestamp{
		"nodeA001": genesisTime,
		"nodeB001": genesisTime,
		"nodeC001": inter.Timestamp(13),
		"nodeD001": genesisTime,
		"nodeA002": inter.Timestamp(112),
		"nodeD002": genesisTime,
		"nodeA012": genesisTime,
		"nodeA003": inter.Timestamp(20),
		"nodeB002": inter.Timestamp(20),
		"nodeC002": inter.Timestamp(35),
	}
	t.Run("testMedianTimeOnDAG", func(t *testing.T) {
		testMedianTime(t, dagAscii, weights, creationTimes, medianTimes, genesisTime)
	})
}

func testMedianTime(t *testing.T, dagAscii string, weights []consensus.Weight, creationTimes map[string]inter.Timestamp, medianTimes map[string]inter.Timestamp, genesis inter.Timestamp) {
	assertar := assert.New(t)

	var ordered consensus.Events
	nodes, _, named := consensus.ASCIIschemeForEach(dagAscii, consensus.ForEachEvent{
		Process: func(e consensus.Event, name string) {
			ordered = append(ordered, &eventWithCreationTime{e, creationTimes[name]})
		},
	})

	validators := consensus.ArrayToValidators(nodes, weights)

	events := make(map[consensus.EventHash]consensus.Event)
	getEvent := func(id consensus.EventHash) consensus.Event {
		return events[id]
	}

	vi := NewIndex(func(err error) { panic(err) }, LiteConfig())
	vi.Reset(validators, memorydb.New(), getEvent)

	// push
	for _, e := range ordered {
		events[e.ID()] = e
		assertar.NoError(vi.Add(e))
		vi.Flush()
	}

	// check
	for name, e := range named {
		expected, ok := medianTimes[name]
		if !ok {
			continue
		}
		assertar.Equal(expected, vi.MedianTime(e.ID(), genesis), name)
	}
}
