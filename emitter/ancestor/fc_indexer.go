package ancestor

import (
	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/inter/pos"
)

const (
	MaxFramesToIndex = 500
)

type highestEvent struct {
	id    hash.Event
	frame idx.Frame
}

type FCIndexer struct {
	dagi       DagIndex
	validators *pos.Validators
	me         idx.ValidatorID

	prevSelfEvent hash.Event
	prevSelfFrame idx.Frame

	TopFrame idx.Frame

	FrameRoots map[idx.Frame]hash.Events

	highestEvents map[idx.ValidatorID]highestEvent

	searchStrategy SearchStrategy
}

type DagIndex interface {
	ForklessCauseProgress(aID, bID hash.Event, candidateParents, chosenParents hash.Events) (*pos.WeightCounter, []*pos.WeightCounter)
}

func NewFCIndexer(validators *pos.Validators, dagi DagIndex, me idx.ValidatorID) *FCIndexer {
	fc := &FCIndexer{
		dagi:          dagi,
		validators:    validators,
		me:            me,
		FrameRoots:    make(map[idx.Frame]hash.Events),
		highestEvents: make(map[idx.ValidatorID]highestEvent),
	}
	fc.searchStrategy = NewMetricStrategy(fc.GetMetricOf)
	return fc
}

func (fc *FCIndexer) ProcessEvent(e dag.Event) {
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
				frameRoots = make(hash.Events, fc.validators.Len())
			}
			frameRoots[fc.validators.GetIdx(e.Creator())] = e.ID()
			fc.FrameRoots[f] = frameRoots
		}
	}
}

func (fc *FCIndexer) rootProgress(frame idx.Frame, event hash.Event, chosenHeads hash.Events) int {
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
		if root == hash.ZeroEvent {
			continue
		}
		FCProgress, _ := fc.dagi.ForklessCauseProgress(event, root, nil, chosenHeads)
		numNonZero += FCProgress.NumCounted() // add the number of validators that have observed root
	}
	return numNonZero
}

func (fc *FCIndexer) greater(aID hash.Event, aFrame idx.Frame, bK int, bFrame idx.Frame) bool {
	if aFrame != bFrame {
		return aFrame > bFrame
	}
	return fc.rootProgress(bFrame, aID, nil) >= bK
}

// ValidatorsPastMe returns total weight of validators which exceeded knowledge of "my" previous event
// Typically node shouldn't emit an event until the value >= quorum, which happens to lead to an almost optimal events timing
func (fc *FCIndexer) ValidatorsPastMe() pos.Weight {
	selfFrame := fc.prevSelfFrame

	kGreaterWeight := fc.validators.NewCounter()
	kPrev := fc.rootProgress(selfFrame, fc.prevSelfEvent, nil) // calculate metric of root knowledge for previous self event

	for creator, e := range fc.highestEvents {
		if fc.greater(e.id, e.frame, kPrev, selfFrame) {
			kGreaterWeight.Count(creator)
		}
	}
	return kGreaterWeight.Sum() // self should not create a new event
}

func (fc *FCIndexer) GetMetricOf(ids hash.Events) Metric {
	if fc.TopFrame == 0 {
		return 0
	}
	return Metric(fc.rootProgress(fc.TopFrame, ids[0], ids[1:]))
}

func (fc *FCIndexer) SearchStrategy() SearchStrategy {
	return fc.searchStrategy
}
