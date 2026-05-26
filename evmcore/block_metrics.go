// Copyright 2026 Sonic Operations Ltd
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

package evmcore

import "github.com/0xsoniclabs/sonic/utils"

//go:generate mockgen -source=block_metrics.go -destination=block_metrics_mock.go -package=evmcore

// BlockExecutionMetrics collects metrics related to the execution of
// bundles and sponsored transactions within a block.
type BlockExecutionMetrics interface {
	IncSponsoredTx()
	IncSkippedSponsoredTx()
	IncExecutedBundle()
	IncRolledBackBundle()
	ObserveBundleEfficiency(usedGas, totalGas uint64)
}

// NoBlockExecutionMetrics ignores metrics collection, used by tools running
// block processor functions outside of the context of a block processor
// (e.g. pre-check, single proposer executor)
var NoBlockExecutionMetrics BlockExecutionMetrics = &noopBlockExecutionMetrics{}

// NewBlockExecutionMetrics creates a new BlockExecutionMetrics instance with
// the given metric counters and histogram.
func NewBlockExecutionMetrics(
	sponsoredTxs utils.MetricsCounter,
	skippedSponsoredTxs utils.MetricsCounter,
	executedBundles utils.MetricsCounter,
	rolledBackBundles utils.MetricsCounter,
	bundleEfficiency utils.MetricsHistogram,
) BlockExecutionMetrics {
	return &defaultBlockExecutionMetrics{
		sponsoredTxs:        sponsoredTxs,
		skippedSponsoredTxs: skippedSponsoredTxs,
		executedBundles:     executedBundles,
		rolledBackBundles:   rolledBackBundles,
		bundleEfficiency:    bundleEfficiency,
	}
}

type defaultBlockExecutionMetrics struct {
	sponsoredTxs        utils.MetricsCounter
	skippedSponsoredTxs utils.MetricsCounter
	executedBundles     utils.MetricsCounter
	rolledBackBundles   utils.MetricsCounter
	bundleEfficiency    utils.MetricsHistogram
}

func (m *defaultBlockExecutionMetrics) IncSponsoredTx() {
	if m.sponsoredTxs != nil {
		m.sponsoredTxs.Inc(int64(1))
	}
}

func (m *defaultBlockExecutionMetrics) IncSkippedSponsoredTx() {
	if m.skippedSponsoredTxs != nil {
		m.skippedSponsoredTxs.Inc(int64(1))
	}
}

func (m *defaultBlockExecutionMetrics) IncExecutedBundle() {
	if m.executedBundles != nil {
		m.executedBundles.Inc(int64(1))
	}
}

func (m *defaultBlockExecutionMetrics) IncRolledBackBundle() {
	if m.rolledBackBundles != nil {
		m.rolledBackBundles.Inc(int64(1))
	}
}

func (m *defaultBlockExecutionMetrics) ObserveBundleEfficiency(usedGas, totalGas uint64) {
	if m.bundleEfficiency != nil {
		m.bundleEfficiency.Observe(float64(usedGas) / float64(totalGas))
	}
}

// noopBlockExecutionMetrics is a no-op implementation of BlockExecutionMetrics
// that discards all metric updates.
type noopBlockExecutionMetrics struct{}

func (n *noopBlockExecutionMetrics) IncSponsoredTx()                    {}
func (n *noopBlockExecutionMetrics) IncSkippedSponsoredTx()             {}
func (n *noopBlockExecutionMetrics) IncExecutedBundle()                  {}
func (n *noopBlockExecutionMetrics) IncRolledBackBundle()                {}
func (n *noopBlockExecutionMetrics) ObserveBundleEfficiency(_, _ uint64) {}
