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

package verwatcher

import "github.com/0xsoniclabs/sonic/version"

// versionNumber is a 64-bit unsigned integer that represents a version number.
// The version number is encoded as follows:
//   - The 16 most significant bits represent the major version.
//   - The next 16 bits represent the minor version.
//   - The next 16 bits represent the patch version.
//   - The 16 least significant bits represent a development version (0), a release
//     candidate number (1-255), or a release version (>=256).
//
// This encoding allows for easy comparison of version numbers and their release order.
type versionNumber uint64

func getVersionNumber() versionNumber {
	return toVersionNumber(version.Get())
}

func toVersionNumber(version version.Version) versionNumber {
	rcNumber := 0
	if version.IsRelease() {
		// A released version is a higher version number than a development version.
		// There are at most 255 release candidates.
		rcNumber = 256
	} else {
		rcNumber = int(version.ReleaseCandidate)
	}

	return versionNumber(
		uint64(version.Major)<<48 |
			uint64(version.Minor)<<32 |
			uint64(version.Patch)<<16 |
			uint64(rcNumber),
	)
}

func (v versionNumber) String() string {
	rcNumber := int(v) & 0xffff
	if rcNumber > 256 {
		rcNumber = 256
	}
	version := version.Version{
		Major:            int(v>>48) & 0xffff,
		Minor:            int(v>>32) & 0xffff,
		Patch:            int(v>>16) & 0xffff,
		ReleaseCandidate: uint8(rcNumber),
		IsDevelopment:    rcNumber == 0,
	}
	return version.String()
}
