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

type SRIndexer struct {
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
	StronglyReachProgress(aID, bID consensus.EventHash, candidateParents, chosenParents consensus.EventHashes) (*consensus.WeightCounter, []*consensus.WeightCounter)
}

func NewSRIndexer(validators *consensus.Validators, dagi DagIndex, me consensus.ValidatorID) *SRIndexer {
	sr := &SRIndexer{
		dagi:          dagi,
		validators:    validators,
		me:            me,
		FrameBases:    make(map[consensus.Frame]consensus.EventHashes),
		highestEvents: make(map[consensus.ValidatorID]highestEvent),
	}
	sr.searchStrategy = NewMetricStrategy(sr.GetMetricOf)
	return sr
}

func (sr *SRIndexer) ProcessEvent(e consensus.Event) {
	if e.Creator() == sr.me {
		sr.prevSelfEvent = e.ID()
		sr.prevSelfFrame = e.Frame()
	}
	selfParent := sr.highestEvents[e.Creator()]
	sr.highestEvents[e.Creator()] = highestEvent{
		id:    e.ID(),
		frame: e.Frame(),
	}
	if sr.TopFrame < e.Frame() {
		sr.TopFrame = e.Frame()
		// frames should get incremented by one, so gaps shouldn't be possible
		delete(sr.FrameBases, sr.TopFrame-MaxFramesToIndex)
	}
	if selfParent.frame != 0 || e.SelfParent() == nil {
		// indexing only MaxFramesToIndex last frames
		for f := selfParent.frame + 1; f <= e.Frame(); f++ {
			if f+MaxFramesToIndex <= sr.TopFrame {
				continue
			}
			frameBases := sr.FrameBases[f]
			if frameBases == nil {
				frameBases = make(consensus.EventHashes, sr.validators.Len())
			}
			frameBases[sr.validators.GetIdx(e.Creator())] = e.ID()
			sr.FrameBases[f] = frameBases
		}
	}
}

func (sr *SRIndexer) baseProgress(frame consensus.Frame, event consensus.EventHash, chosenHeads consensus.EventHashes) int {
	// This function computes the knowledge of bases amongst validators by counting which validators known which bases.
	// Base knowledge is a binary matrix indexed by bases and validators.
	// The ijth entry of the matrix is 1 if base i is known by validator j in the subgraph of event, and zero otherwise.
	// The function returns a metric counting the number of non-zero entries of the base knowledge matrix.
	bases, ok := sr.FrameBases[frame]
	if !ok {
		return 0
	}
	numNonZero := 0 // number of non-zero entries in the base knowledge matrix
	for _, base := range bases {
		if base == consensus.ZeroEventHash {
			continue
		}
		srProgress, _ := sr.dagi.StronglyReachProgress(event, base, nil, chosenHeads)
		numNonZero += srProgress.NumCounted() // add the number of validators that have base as reachable
	}
	return numNonZero
}

func (sr *SRIndexer) greater(aID consensus.EventHash, aFrame consensus.Frame, bK int, bFrame consensus.Frame) bool {
	if aFrame != bFrame {
		return aFrame > bFrame
	}
	return sr.baseProgress(bFrame, aID, nil) >= bK
}

// ValidatorsPastMe returns total weight of validators which exceeded knowledge of "my" previous event
// Typically node shouldn't emit an event until the value >= quorum, which happens to lead to an almost optimal events timing
func (sr *SRIndexer) ValidatorsPastMe() consensus.Weight {
	selfFrame := sr.prevSelfFrame

	kGreaterWeight := sr.validators.NewCounter()
	kPrev := sr.baseProgress(selfFrame, sr.prevSelfEvent, nil) // calculate metric of base knowledge for previous self event

	for creator, e := range sr.highestEvents {
		if sr.greater(e.id, e.frame, kPrev, selfFrame) {
			kGreaterWeight.CountVoteByID(creator)
		}
	}
	return kGreaterWeight.Sum() // self should not create a new event
}

func (sr *SRIndexer) GetMetricOf(ids consensus.EventHashes) Metric {
	if sr.TopFrame == 0 {
		return 0
	}
	return Metric(sr.baseProgress(sr.TopFrame, ids[0], ids[1:]))
}

func (sr *SRIndexer) SearchStrategy() SearchStrategy {
	return sr.searchStrategy
}
