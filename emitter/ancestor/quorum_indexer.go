package ancestor

import (
	"math"
	"sort"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/dagindexer"

	"github.com/0xsoniclabs/sonic/utils/wthreshold"
)

type DiffMetricFn func(thresholdValue, current, update consensus.Seq, validatorIdx consensus.ValidatorIndex) Metric

type QuorumIndexer struct {
	dagi       *dagindexer.Index
	validators *consensus.Validators

	globalMatrix             Matrix
	selfParentSeqs           []consensus.Seq
	globalThresholdValueSeqs []consensus.Seq
	dirty                    bool
	searchStrategy           SearchStrategy

	diffMetricFn DiffMetricFn
}

func NewQuorumIndexer(validators *consensus.Validators, dagi *dagindexer.Index, diffMetricFn DiffMetricFn) *QuorumIndexer {
	return &QuorumIndexer{
		globalMatrix:             NewMatrix(validators.Len(), validators.Len()),
		globalThresholdValueSeqs: make([]consensus.Seq, validators.Len()),
		selfParentSeqs:           make([]consensus.Seq, validators.Len()),
		dagi:                     dagi,
		validators:               validators,
		diffMetricFn:             diffMetricFn,
		dirty:                    true,
	}
}

type Matrix struct {
	buffer  []consensus.Seq
	columns consensus.ValidatorIndex
}

func NewMatrix(rows, cols consensus.ValidatorIndex) Matrix {
	return Matrix{
		buffer:  make([]consensus.Seq, rows*cols),
		columns: cols,
	}
}

func (m Matrix) Row(i consensus.ValidatorIndex) []consensus.Seq {
	return m.buffer[i*m.columns : (i+1)*m.columns]
}

func (m Matrix) Clone() Matrix {
	buffer := make([]consensus.Seq, len(m.buffer))
	copy(buffer, m.buffer)
	return Matrix{
		buffer,
		m.columns,
	}
}

func seqOf(seq dagindexer.BranchSeq) consensus.Seq {
	if seq.IsForkDetected() {
		return math.MaxUint32/2 - 1
	}
	return seq.Seq
}

type weightedSeq struct {
	seq    consensus.Seq
	weight consensus.Weight
}

func (ws weightedSeq) Weight() consensus.Weight {
	return ws.weight
}

func (h *QuorumIndexer) ProcessEvent(event consensus.Event, selfEvent bool) {
	vecClock := h.dagi.GetMergedHighestBefore(event.ID()).VSeq
	creatorIdx := h.validators.GetIdx(event.Creator())
	// update global matrix
	for validatorIdx := consensus.ValidatorIndex(0); validatorIdx < h.validators.Len(); validatorIdx++ {
		seq := seqOf(vecClock.Get(validatorIdx))
		h.globalMatrix.Row(validatorIdx)[creatorIdx] = seq
		if selfEvent {
			h.selfParentSeqs[validatorIdx] = seq
		}
	}
	h.dirty = true
}

func (h *QuorumIndexer) recacheState() {
	// update thresholdValue seqs
	for validatorIdx := consensus.ValidatorIndex(0); validatorIdx < h.validators.Len(); validatorIdx++ {
		pairs := make([]wthreshold.WeightedValue, h.validators.Len())
		for i := range pairs {
			pairs[i] = weightedSeq{
				seq:    h.globalMatrix.Row(validatorIdx)[i],
				weight: h.validators.GetWeightByIdx(consensus.ValidatorIndex(i)),
			}
		}
		sort.Slice(pairs, func(i, j int) bool {
			a, b := pairs[i].(weightedSeq), pairs[j].(weightedSeq)
			return a.seq > b.seq
		})
		thresholdValue := wthreshold.FindThresholdValue(pairs, h.validators.Quorum())
		h.globalThresholdValueSeqs[validatorIdx] = thresholdValue.(weightedSeq).seq
	}
	h.searchStrategy = NewMetricStrategy(h.GetMetricOf)
	h.dirty = false
}

func (h *QuorumIndexer) GetMetricOf(parents consensus.EventHashes) Metric {
	if h.dirty {
		h.recacheState()
	}
	vecClock := make([]dagindexer.HighestBeforeSeq, len(parents))
	for i, parent := range parents {
		vecClock[i] = *h.dagi.GetMergedHighestBefore(parent).VSeq
	}
	var metric Metric
	for validatorIdx := consensus.ValidatorIndex(0); validatorIdx < h.validators.Len(); validatorIdx++ {

		//find the Highest of all the parents
		var update consensus.Seq
		for i, _ := range parents {
			if seqOf(vecClock[i].Get(validatorIdx)) > update {
				update = seqOf(vecClock[i].Get(validatorIdx))
			}
		}
		current := h.selfParentSeqs[validatorIdx]
		thresholdValue := h.globalThresholdValueSeqs[validatorIdx]
		metric += h.diffMetricFn(thresholdValue, current, update, validatorIdx)
	}
	return metric
}

func (h *QuorumIndexer) SearchStrategy() SearchStrategy {
	if h.dirty {
		h.recacheState()
	}
	return h.searchStrategy
}

func (h *QuorumIndexer) GetGlobalThresholdValueSeqs() []consensus.Seq {
	if h.dirty {
		h.recacheState()
	}
	return h.globalThresholdValueSeqs
}

func (h *QuorumIndexer) GetGlobalMatrix() Matrix {
	return h.globalMatrix
}

func (h *QuorumIndexer) GetSelfParentSeqs() []consensus.Seq {
	return h.selfParentSeqs
}
