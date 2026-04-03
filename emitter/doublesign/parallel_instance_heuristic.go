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

package doublesign

import "time"

// DetectParallelInstance should be called after downloading a self-event which wasn't created on this instance
// Returns true if a parallel instance is likely be running
func DetectParallelInstance(s SyncStatus, threshold time.Duration) bool {
	if s.ExternalSelfEventCreated.Before(s.Startup) {
		return false
	}
	return s.Since(s.ExternalSelfEventCreated) < threshold
}
