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

// Package sonic implements the sonic_ namespace: the methods that are specific
// to the Sonic network, such as the bundle methods.
package sonic

import (
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// API holds the state shared by the handlers of the namespace.
type API struct {
	env   *exec.Env
	chain *dispatch.Chain
	core  core.Core
}

// Register registers the sonic_ methods.
func Register(d *dispatch.Dispatcher, env *exec.Env) {
	a := &API{env: env, chain: env.Chain, core: env.Core}
	dispatch.Add4(d, "sonic_estimateGasForTransactions", a.EstimateGasForTransactions,
		"args", "block?", "stateOverrides?", "blockOverrides?")
	dispatch.Add1(d, "sonic_prepareBundle", a.prepareBundle, "proposal")
	dispatch.Add1(d, "sonic_submitBundle", a.submitBundle, "bundle")
	dispatch.Add1(d, "sonic_getBundleInfo", a.getBundleInfo, "executionPlanHash")
}
