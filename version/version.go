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

// This package tracks Sonic's version number. The version number follows
// the Semantic Versioning 2.0.0 specification (https://semver.org/) with
// the following possible pre-release tags:
//   - `dev` for development versions
//   - `rc.X` for release candidates, where X is a numeric value
package version

import (
	"fmt"
	"regexp"
	"strconv"
)

// Version information, to be manually updated for each named version.
const (
	// The major and minor version of this project. These are manually updated
	// for each release. The main branch is always the next minor version
	// compared to the latest release branch.
	Major = 2
	Minor = 2

	// The patch version, which must only be non-zero for release candidates
	// and official releases. All development versions must have a patch
	// version of 0.
	Patch = 0

	// The pre-release version. This is set to "dev" for development versions
	// on the main branch and should be updated to "rc.X" for release candidates
	// on release branches only. For a final release, this must be set to an
	// empty string. All other values are invalid.
	PreRelease = "dev"
)

// Get returns the complete version information.
func Get() Version {
	return _version
}

// String returns the version string.
func String() string {
	return Get().String()
}

// StringWithCommit returns the version string with the commit hash and date.
func StringWithCommit() string {
	vsn := Get().String()
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	if gitDate != "" {
		vsn += "-" + gitDate
	}
	return vsn
}

// GitCommit returns the commit hash if available. If not, the empty string
// is returned.
func GitCommit() string {
	return gitCommit
}

// GitDate returns the commit date if available. If not, the empty string
// is returned.
func GitDate() string {
	return gitDate
}

// Version represents a version of the code.
type Version struct {
	Major            int
	Minor            int
	Patch            int
	ReleaseCandidate uint8 // 0 for release versions, >0 for release candidates
	IsDevelopment    bool  // true for development versions
}

// IsRelease returns true if the version is a release version. It returns false
// if the version has a meta string or is dirty.
func (v Version) IsRelease() bool {
	return !v.IsDevelopment && v.ReleaseCandidate == 0
}

func (v Version) String() string {
	res := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.ReleaseCandidate > 0 {
		res += fmt.Sprintf("-rc.%d", v.ReleaseCandidate)
	}
	if v.IsDevelopment {
		res += "-dev"
	}
	return res
}

var _preReleaseRE = regexp.MustCompile(`^(|dev|rc\.(\d+))$`)

// makeVersion checks the version components for validity and returns a new
// Version instance if valid.
func makeVersion(major, minor, patch int, preRelease string) (Version, error) {
	if !_preReleaseRE.MatchString(preRelease) {
		return Version{}, fmt.Errorf("invalid version: invalid pre-release tag %q", preRelease)
	}
	isDevelopment := preRelease == "dev"
	if isDevelopment && patch != 0 {
		return Version{}, fmt.Errorf("invalid version: development versions must have a patch version of 0")
	}
	rcNumber := uint8(0)
	if !isDevelopment && len(preRelease) > 0 {
		number, err := strconv.ParseUint(preRelease[3:], 10, 8)
		if err != nil {
			return Version{}, fmt.Errorf("invalid version: invalid release candidate tag %q", preRelease)
		}
		if number <= 0 {
			return Version{}, fmt.Errorf("invalid version: release candidate number must be >0")
		}
		rcNumber = uint8(number)
	}
	return Version{
		Major:            major,
		Minor:            minor,
		Patch:            patch,
		ReleaseCandidate: rcNumber,
		IsDevelopment:    isDevelopment,
	}, nil
}

var _version Version

func init() {
	// Check that the version is valid at startup.
	version, err := makeVersion(Major, Minor, Patch, PreRelease)
	if err != nil {
		panic(err)
	}
	_version = version
}

// -- set by linker flags --------------------------------------------

var (
	// gitCommit is the commit hash, set by the Makefile.
	gitCommit = ""

	// gitDate is the commit date, set by the Makefile.
	gitDate = ""
)
