// This package tracks Sonic's version number. The version number follows
// the Semantic Versioning 2.0.0 specification (https://semver.org/) with
// the following possible pre-release tags:
//   - `dev` for development versions
//   - `rcX` for release candidates
package version

import (
	"fmt"
)

// Version information, to be manually updated for each named version.
const (
	// The major and minor version of this project. These are manually updated
	// for each release. The main branch is always the next minor version
	// compared to the latest release branch.
	Major = 2
	Minor = 1

	// The patch version, which must only be non-zero for release candidates
	// and official releases. All development versions must have a patch
	// version of 0.
	Patch = 0

	// The release-candidate version. This is set to 0 for official releases
	// and on development branches.
	ReleaseCandidate = 0

	// Marks this version as a release version. This must only be set to true
	// for official releases on a release branch. Unreleased versions are either
	// release candidates with the meta `rcX` or development versions with the
	// meta `dev`.
	IsRelease = false
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
	Major      int
	Minor      int
	Patch      int
	PreRelease string
}

// IsRelease returns true if the version is a release version. It returns false
// if the version has a meta string or is dirty.
func (v Version) IsRelease() bool {
	return v.PreRelease == ""
}

func (v Version) String() string {
	res := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		res += "-" + v.PreRelease
	}
	return res
}

// makeVersion checks the version components for validity and returns a new
// Version instance if valid.
func makeVersion(major, minor, patch, rc int, released bool) (Version, error) {
	if released && rc > 0 {
		return Version{}, fmt.Errorf("invalid version: can not be both a release and a release candidate")
	}
	if !released && rc == 0 && patch != 0 {
		return Version{}, fmt.Errorf("invalid version: patch version must be 0 for development versions")
	}
	_version := Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
	if rc > 0 {
		_version.PreRelease = fmt.Sprintf("rc%d", rc)
	} else if !released {
		_version.PreRelease = "dev"
	}
	return _version, nil
}

var _version Version

func init() {
	version, err := makeVersion(Major, Minor, Patch, ReleaseCandidate, IsRelease)
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
