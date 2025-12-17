// Copyright 2025 Sonic Operations Ltd
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

package tests

import (
	"os"
	"testing"
)

// TestMain is a functionality offered by the testing package that allows
// us to run some code before and after all tests in the package.
func TestMain(m *testing.M) {

	m.Run()

	// Stop all active networks after tests are done
	for _, net := range activeTestNetInstances {
		net.Stop()
		for i := range net.nodes {
			// it is safe to ignore this error since the tests have ended and
			// the directories are not needed anymore.
			_ = os.RemoveAll(net.nodes[i].directory)
		}
	}
	activeTestNetInstances = nil
}
