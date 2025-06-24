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

package opera

import "encoding/json"

func UpdateRules(src Rules, diff []byte) (Rules, error) {
	changed := src.Copy()
	if err := json.Unmarshal(diff, &changed); err != nil {
		return Rules{}, err
	}

	// protect readonly fields
	changed.NetworkID = src.NetworkID
	changed.Name = src.Name

	// check validity of the new rules
	if changed.Upgrades.Allegro {
		if err := changed.Validate(src); err != nil {
			return Rules{}, err
		}
	}
	return changed, nil
}
