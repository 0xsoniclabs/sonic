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

	FrameBases map[consensus.Frame]consensus.EventHashes

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
		FrameBases:    make(map[consensus.Frame]consensus.EventHashes),
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
		delete(fc.FrameBases, fc.TopFrame-MaxFramesToIndex)
	}
	if selfParent.frame != 0 || e.SelfParent() == nil {
		// indexing only MaxFramesToIndex last frames
		for f := selfParent.frame + 1; f <= e.Frame(); f++ {
			if f+MaxFramesToIndex <= fc.TopFrame {
				continue
			}
			frameBases := fc.FrameBases[f]
			if frameBases == nil {
				frameBases = make(consensus.EventHashes, fc.validators.Len())
			}
			frameBases[fc.validators.GetIdx(e.Creator())] = e.ID()
			fc.FrameBases[f] = frameBases
		}
	}
}

func (fc *FCIndexer) baseProgress(frame consensus.Frame, event consensus.EventHash, chosenHeads consensus.EventHashes) int {
	// This function computes the knowledge of bases amongst validators by counting which validators known which bases.
	// Base knowledge is a binary matrix indexed by bases and validators.
	// The ijth entry of the matrix is 1 if base i is known by validator j in the subgraph of event, and zero otherwise.
	// The function returns a metric counting the number of non-zero entries of the base knowledge matrix.
	bases, ok := fc.FrameBases[frame]
	if !ok {
		return 0
	}
	numNonZero := 0 // number of non-zero entries in the base knowledge matrix
	for _, base := range bases {
		if base == consensus.ZeroEventHash {
			continue
		}
		FCProgress, _ := fc.dagi.ForklessCauseProgress(event, base, nil, chosenHeads)
		numNonZero += FCProgress.NumCounted() // add the number of validators that have observed base
	}
	return numNonZero
}

func (fc *FCIndexer) greater(aID consensus.EventHash, aFrame consensus.Frame, bK int, bFrame consensus.Frame) bool {
	if aFrame != bFrame {
		return aFrame > bFrame
	}
	return fc.baseProgress(bFrame, aID, nil) >= bK
}

// ValidatorsPastMe returns total weight of validators which exceeded knowledge of "my" previous event
// Typically node shouldn't emit an event until the value >= quorum, which happens to lead to an almost optimal events timing
func (fc *FCIndexer) ValidatorsPastMe() consensus.Weight {
	selfFrame := fc.prevSelfFrame

	kGreaterWeight := fc.validators.NewCounter()
	kPrev := fc.baseProgress(selfFrame, fc.prevSelfEvent, nil) // calculate metric of base knowledge for previous self event

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
	return Metric(fc.baseProgress(fc.TopFrame, ids[0], ids[1:]))
}

func (fc *FCIndexer) SearchStrategy() SearchStrategy {
	return fc.searchStrategy
}
