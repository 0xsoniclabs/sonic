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

package opera

import "iter"

// GetSonicUpgrades contains the feature flags for the Sonic upgrade.
func GetSonicUpgrades() Upgrades {
	return Upgrades{
		Berlin:  true,
		London:  true,
		Llr:     false,
		Sonic:   true,
		Allegro: false,
	}
}

// GetAllegroUpgrades contains the feature flags for the Allegro upgrade.
func GetAllegroUpgrades() Upgrades {
	return Upgrades{
		Berlin:  true,
		London:  true,
		Llr:     false,
		Sonic:   true,
		Allegro: true,
	}
}

// GetBrioUpgrades contains the feature flags for the Brio upgrade.
func GetBrioUpgrades() Upgrades {
	return Upgrades{
		Berlin:  true,
		London:  true,
		Llr:     false,
		Sonic:   true,
		Allegro: true,
		Brio:    true,
	}
}

// GetAllHardForksInOrder returns an iterator over all hard forks and their
// corresponding feature flags in order.
// This function returns an iterator and not a map to preserve the hardfork order.
// Some tests can use this property.
func GetAllHardForksInOrder() iter.Seq2[string, Upgrades] {

	hardforks := []struct {
		name     string
		upgrades Upgrades
	}{
		{"Sonic", GetSonicUpgrades()},
		{"Allegro", GetAllegroUpgrades()},
		{"Brio", GetBrioUpgrades()},
	}

	return func(yield func(string, Upgrades) bool) {
		for _, hf := range hardforks {
			if !yield(hf.name, hf.upgrades) {
				return
			}
		}
	}
}
