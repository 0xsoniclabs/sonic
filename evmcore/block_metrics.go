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
	IncInvalidBundle()
	ObserveBundleEfficiency(usedGas, totalExecGas uint64)
}

type SonicBlockExecutionMetrics struct {
	SponsoredTxs        utils.MetricsCounter
	SkippedSponsoredTxs utils.MetricsCounter
	ExecutedBundles     utils.MetricsCounter
	RolledBackBundles   utils.MetricsCounter
	InvalidBundles      utils.MetricsCounter
	BundleEfficiency    utils.MetricsHistogram
}

func (m *SonicBlockExecutionMetrics) IncSponsoredTx() {
	if m.SponsoredTxs != nil {
		m.SponsoredTxs.Inc(int64(1))
	}
}

func (m *SonicBlockExecutionMetrics) IncSkippedSponsoredTx() {
	if m.SkippedSponsoredTxs != nil {
		m.SkippedSponsoredTxs.Inc(int64(1))
	}
}

func (m *SonicBlockExecutionMetrics) IncExecutedBundle() {
	if m.ExecutedBundles != nil {
		m.ExecutedBundles.Inc(int64(1))
	}
}

func (m *SonicBlockExecutionMetrics) IncRolledBackBundle() {
	if m.RolledBackBundles != nil {
		m.RolledBackBundles.Inc(int64(1))
	}
}

func (m *SonicBlockExecutionMetrics) IncInvalidBundle() {
	if m.InvalidBundles != nil {
		m.InvalidBundles.Inc(int64(1))
	}
}

func (m *SonicBlockExecutionMetrics) ObserveBundleEfficiency(usedGas, totalExecGas uint64) {
	if m.BundleEfficiency != nil && totalExecGas > 0 {
		m.BundleEfficiency.Observe(float64(usedGas) / float64(totalExecGas))
	}
}
