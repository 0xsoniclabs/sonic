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
