package ancestor

import (
	"github.com/0xsoniclabs/consensus/hash"
)

type Metric uint64

type MetricStrategy struct {
	metricFn func(hash.Events) Metric
}

func NewMetricStrategy(metricFn func(hash.Events) Metric) *MetricStrategy {
	return &MetricStrategy{metricFn}
}

// Choose chooses the hash from the specified options
func (st *MetricStrategy) Choose(existing hash.Events, options hash.Events) int {
	var maxI int
	var maxWeight Metric
	// find option with a maximum weight
	for i, opt := range options {
		weight := st.metricFn(append(existing.Copy(), opt))
		if maxWeight == 0 || weight > maxWeight {
			maxI = i
			maxWeight = weight
		}
	}
	return maxI
}
