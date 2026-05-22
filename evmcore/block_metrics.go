package evmcore

import "github.com/0xsoniclabs/sonic/utils"

//go:generate mockgen -source=metrics.go -destination=metrics_mock.go -package=core_types

// BlockExecutionMetrics collects metrics related to the execution of
// bundles and sponsored transactions within a block.
type BlockExecutionMetrics struct {
	SponsoredTxs        utils.MetricsCounter
	SkippedSponsoredTxs utils.MetricsCounter

	ExecutedBundles   utils.MetricsCounter
	RolledBackBundles utils.MetricsCounter

	BundleEfficiency utils.MetricsHistogramWrapper
}

func (m *BlockExecutionMetrics) IncSponsoredTx() {
	if m.SponsoredTxs != nil {
		m.SponsoredTxs.Inc(int64(1))
	}
}
func (m *BlockExecutionMetrics) IncSkippedSponsoredTx() {
	if m.SkippedSponsoredTxs != nil {
		m.SkippedSponsoredTxs.Inc(int64(1))
	}
}

func (m *BlockExecutionMetrics) IncExecutedBundle() {
	if m.ExecutedBundles != nil {
		m.ExecutedBundles.Inc(int64(1))
	}
}
func (m *BlockExecutionMetrics) IncRolledBackBundle() {
	if m.RolledBackBundles != nil {
		m.RolledBackBundles.Inc(int64(1))
	}
}

func (m *BlockExecutionMetrics) UpdateBundleEfficiencyHistogram(usedGas, totalGas uint64) {
	if m.BundleEfficiency != nil && totalGas > 0 {
		efficiency := float64(usedGas) / float64(totalGas)
		m.BundleEfficiency.Update(efficiency)
	}
}

// NoOpBlockExecutionMetrics ignores metrics collection, used by tools running
// block processor functions outside of the context of a block processor
// (e.g. pre-check, single proposer executor)
var NoOpBlockExecutionMetrics BlockExecutionMetrics
