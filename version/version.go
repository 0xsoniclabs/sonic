package version

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

var (
	GitTag       = "" // Git tag of this release
	versionMajor = 0  // Major version component of the current release
	versionMinor = 0  // Minor version component of the current release
	versionPatch = 0  // Patch version component of the current release
	versionMeta  = "" // Version metadata to append to the version string
)

// Version holds the textual version string.
var Version = func() string {
	// in case of no tag or small/irregular tag, return it as is
	if len(GitTag) < 2 {
		return GitTag
	}
	parseVersion()
	// remove the leading "v" and possible leading white spaces
	return strings.Split(GitTag, "v")[1]
}()

// parseVersion parses the GitTag into major, minor, patch, and meta components.
func parseVersion() {
	parts := strings.SplitN(GitTag, "-", 2)
	versionParts := strings.Split(parts[0], ".")

	// Parse major, minor, and patch
	versionMajor = parseVersionComponent(versionParts, 0, true)
	versionMinor = parseVersionComponent(versionParts, 1, false)
	versionPatch = parseVersionComponent(strings.Split(versionParts[2], "-"), 0, false)

	// Parse meta if available
	if len(parts) > 1 {
		versionMeta = parts[1]
	}
}

// parseVersionComponent parses and returns a specific version component.
// If `stripPrefix` is true, it strips the leading "v" from the major version.
func parseVersionComponent(parts []string, index int, stripPrefix bool) int {
	if len(parts) <= index {
		return 0
	}

	component := parts[index]
	if stripPrefix {
		component = strings.TrimPrefix(component, "v")
	}

	value, err := strconv.Atoi(component)
	if err != nil {
		log.Printf("Failed to parse version component %q: %v", component, err)
		return 0
	}

	return value
}

func VersionWithCommit(gitCommit, gitDate string) string {
	vsn := GitTag
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	if (strings.Split(GitTag, "-")[0] != "") && (gitDate != "") {
		vsn += "-" + gitDate
	}
	return vsn
}

func AsString() string {
	return ToString(uint16(versionMajor), uint16(versionMinor), uint16(versionPatch))
}

func AsU64() uint64 {
	return ToU64(uint16(versionMajor), uint16(versionMinor), uint16(versionPatch))
}

func ToU64(vMajor, vMinor, vPatch uint16) uint64 {
	return uint64(vMajor)*1e12 + uint64(vMinor)*1e6 + uint64(vPatch)
}

func ToString(major, minor, patch uint16) string {
	return fmt.Sprintf("%d.%d.%d", major, minor, patch)
}

func U64ToString(v uint64) string {
	return ToString(uint16((v/1e12)%1e6), uint16((v/1e6)%1e6), uint16(v%1e6))
}
