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

// Exports for the benchmarks of package evmcore_test, which can build a real
// Carmen state through evmstore, an importer of evmcore.

func (pool *TxPool) DemoteUnexecutables_forTesting() { pool.demoteUnexecutables() }
func (pool *TxPool) DropFrozenSenders_forTesting()   { pool.dropFrozenSenders() }
func (pool *TxPool) EvictBundleOnlyTransactionsOfProcessedBundles_forTesting() {
	pool.evictBundleOnlyTransactionsOfProcessedBundles()
}

// ClearCheckCaches_forTesting drops the cached sponsorship and bundle checks.
func (pool *TxPool) ClearCheckCaches_forTesting() {
	pool.subsidiesCheckerCache = utils.NewCheckerCache(-1)
	pool.bundleEvaluationCache = NewBundleEvaluationCache()
}
