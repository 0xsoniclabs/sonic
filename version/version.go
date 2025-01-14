package version

import (
	"fmt"
	"strconv"
	"strings"
)

// Version holds the textual version string.
var (
	Version      = ""
	versionMajor = 0
	versionMinor = 0
	versionPatch = 0
	// versionMeta  = "" // not used for now
)

func init() {
	// in case of no tag or small/irregular tag, return it as is
	if len(Version) >= 2 {
		versionMajor, versionMinor, versionPatch, _ = parseVersion(Version)
	}
}

// parseVersion parses the GitTag into major, minor, patch, and meta components.
func parseVersion(gitTag string) (vMajor, vMinor, vPatch int, vMeta string) {
	parts := strings.SplitN(gitTag, "-", 2)
	versionParts := strings.Split(parts[0], ".")

	// Parse major, minor, and patch
	vMajor = parseVersionComponent(versionParts, 0, true)
	vMinor = parseVersionComponent(versionParts, 1, false)
	if len(versionParts) > 2 {
		dashSplits := strings.Split(versionParts[2], "-")
		vPatch = parseVersionComponent(dashSplits, 0, false)
	}
	// Parse meta if available
	if (vMajor != 0 || vMinor != 0 || vPatch != 0) && len(parts) > 1 {
		vMeta = parts[1]
	}
	return
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
		return 0
	}

	return value
}

func VersionWithCommit(gitCommit, gitDate string) string {
	vsn := Version
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	if (strings.Split(Version, "-")[0] != "") && (gitDate != "") {
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
