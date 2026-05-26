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

import (
	"testing"
)

func TestBlockExecutionMetrics_NoUpdateIfUnderlyingMetricIsNil(t *testing.T) {
	tests := map[string]struct {
		call func(m *defaultBlockExecutionMetrics)
	}{
		"IncSponsoredTx": {
			call: func(m *defaultBlockExecutionMetrics) { m.IncSponsoredTx() },
		},
		"IncSkippedSponsoredTx": {
			call: func(m *defaultBlockExecutionMetrics) { m.IncSkippedSponsoredTx() },
		},
		"IncExecutedBundle": {
			call: func(m *defaultBlockExecutionMetrics) { m.IncExecutedBundle() },
		},
		"IncRolledBackBundle": {
			call: func(m *defaultBlockExecutionMetrics) { m.IncRolledBackBundle() },
		},
		"ObserveBundleEfficiency": {
			call: func(m *defaultBlockExecutionMetrics) { m.ObserveBundleEfficiency(100, 200) },
		},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			m := &defaultBlockExecutionMetrics{} // all underlying metrics are nil
			testCase.call(m)
		})
	}
}
