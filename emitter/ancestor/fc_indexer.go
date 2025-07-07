package ancestor

import (
	"github.com/0xsoniclabs/consensus/consensus"
)

const (
	MaxFramesToIndex = 500
)

type highestEvent struct {
	id    consensus.EventHash
	frame consensus.Frame
}

type FCIndexer struct {
	dagi       DagIndex
	validators *consensus.Validators
	me         consensus.ValidatorID

	prevSelfEvent consensus.EventHash
	prevSelfFrame consensus.Frame

	TopFrame consensus.Frame

	FrameRoots map[consensus.Frame]consensus.EventHashes

	highestEvents map[consensus.ValidatorID]highestEvent

	searchStrategy SearchStrategy
}

type DagIndex interface {
	ForklessCauseProgress(aID, bID consensus.EventHash, candidateParents, chosenParents consensus.EventHashes) (*consensus.WeightCounter, []*consensus.WeightCounter)
}

func NewFCIndexer(validators *consensus.Validators, dagi DagIndex, me consensus.ValidatorID) *FCIndexer {
	fc := &FCIndexer{
		dagi:          dagi,
		validators:    validators,
		me:            me,
		FrameRoots:    make(map[consensus.Frame]consensus.EventHashes),
		highestEvents: make(map[consensus.ValidatorID]highestEvent),
	}
	fc.searchStrategy = NewMetricStrategy(fc.GetMetricOf)
	return fc
}

func (fc *FCIndexer) ProcessEvent(e consensus.Event) {
	if e.Creator() == fc.me {
		fc.prevSelfEvent = e.ID()
		fc.prevSelfFrame = e.Frame()
	}
	selfParent := fc.highestEvents[e.Creator()]
	fc.highestEvents[e.Creator()] = highestEvent{
		id:    e.ID(),
		frame: e.Frame(),
	}
	if fc.TopFrame < e.Frame() {
		fc.TopFrame = e.Frame()
		// frames should get incremented by one, so gaps shouldn't be possible
		delete(fc.FrameRoots, fc.TopFrame-MaxFramesToIndex)
	}
	if selfParent.frame != 0 || e.SelfParent() == nil {
		// indexing only MaxFramesToIndex last frames
		for f := selfParent.frame + 1; f <= e.Frame(); f++ {
			if f+MaxFramesToIndex <= fc.TopFrame {
				continue
			}
			frameRoots := fc.FrameRoots[f]
			if frameRoots == nil {
				frameRoots = make(consensus.EventHashes, fc.validators.Len())
			}
			frameRoots[fc.validators.GetIdx(e.Creator())] = e.ID()
			fc.FrameRoots[f] = frameRoots
		}
	}
}

func (fc *FCIndexer) rootProgress(frame consensus.Frame, event consensus.EventHash, chosenHeads consensus.EventHashes) int {
	// This function computes the knowledge of roots amongst validators by counting which validators known which roots.
	// Root knowledge is a binary matrix indexed by roots and validators.
	// The ijth entry of the matrix is 1 if root i is known by validator j in the subgraph of event, and zero otherwise.
	// The function returns a metric counting the number of non-zero entries of the root knowledge matrix.
	roots, ok := fc.FrameRoots[frame]
	if !ok {
		return 0
	}
	numNonZero := 0 // number of non-zero entries in the root knowledge matrix
	for _, root := range roots {
		if root == consensus.ZeroEventHash {
			continue
		}
		FCProgress, _ := fc.dagi.ForklessCauseProgress(event, root, nil, chosenHeads)
		numNonZero += FCProgress.NumCounted() // add the number of validators that have observed root
	}
	return numNonZero
}

func (fc *FCIndexer) greater(aID consensus.EventHash, aFrame consensus.Frame, bK int, bFrame consensus.Frame) bool {
	if aFrame != bFrame {
		return aFrame > bFrame
	}
	return fc.rootProgress(bFrame, aID, nil) >= bK
}

// ValidatorsPastMe returns total weight of validators which exceeded knowledge of "my" previous event
// Typically node shouldn't emit an event until the value >= quorum, which happens to lead to an almost optimal events timing
func (fc *FCIndexer) ValidatorsPastMe() consensus.Weight {
	selfFrame := fc.prevSelfFrame

	kGreaterWeight := fc.validators.NewCounter()
	kPrev := fc.rootProgress(selfFrame, fc.prevSelfEvent, nil) // calculate metric of root knowledge for previous self event

	for creator, e := range fc.highestEvents {
		if fc.greater(e.id, e.frame, kPrev, selfFrame) {
			kGreaterWeight.CountVoteByID(creator)
		}
	}
	return kGreaterWeight.Sum() // self should not create a new event
}

func (fc *FCIndexer) GetMetricOf(ids consensus.EventHashes) Metric {
	if fc.TopFrame == 0 {
		return 0
	}
	return Metric(fc.rootProgress(fc.TopFrame, ids[0], ids[1:]))
}

func (fc *FCIndexer) SearchStrategy() SearchStrategy {
	return fc.searchStrategy
}
