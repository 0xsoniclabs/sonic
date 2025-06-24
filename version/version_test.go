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

package version

import (
	"testing"
)

func TestMakeVersion_AcceptValidVersionNumber(t *testing.T) {
	tests := map[string]struct {
		major, minor, patch int
		preRelease          string
	}{
		"1.2.3":        {major: 1, minor: 2, patch: 3},
		"1.2.0-dev":    {major: 1, minor: 2, preRelease: "dev"},
		"1.2.3-rc.4":   {major: 1, minor: 2, patch: 3, preRelease: "rc.4"},
		"1.2.3-rc.255": {major: 1, minor: 2, patch: 3, preRelease: "rc.255"},
	}

	for want, test := range tests {
		version, err := makeVersion(test.major, test.minor, test.patch, test.preRelease)
		if err != nil {
			t.Errorf("version %s returned an error: %v", want, err)
		}
		if got := version.String(); got != want {
			t.Errorf("version %s produces wrong result, got %q", want, got)
		}
	}
}

func TestMakeVersion_DetectsInvalidVersionNumber(t *testing.T) {
	tests := map[string]struct {
		major, minor, patch int
		preRelease          string
	}{
		"invalid pre-release format":        {major: 1, minor: 2, patch: 3, preRelease: "xy"},
		"invalid release candidate":         {major: 1, minor: 2, patch: 3, preRelease: "rc"},
		"missing dot in release candidate":  {major: 1, minor: 2, patch: 3, preRelease: "rc1"},
		"non-numeric release candidate":     {major: 1, minor: 2, patch: 3, preRelease: "rc.X"},
		"negative release candidate":        {major: 1, minor: 2, patch: 3, preRelease: "rc.-1"},
		"release candidate 0":               {major: 1, minor: 2, patch: 3, preRelease: "rc.0"},
		"release candidate exceeding 8-bit": {major: 1, minor: 2, patch: 3, preRelease: "rc.256"},
		"patch version in development":      {major: 1, minor: 2, patch: 3, preRelease: "dev"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := makeVersion(test.major, test.minor, test.patch, test.preRelease)
			if err == nil {
				t.Errorf("expected an error, got nil")
			}
		})
	}
}
