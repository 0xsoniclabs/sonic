package verwatcher

import "github.com/0xsoniclabs/sonic/version"

type versionNumber uint64

func getVersionNumber() versionNumber {
	return toVersionNumber(version.Get())
}

func toVersionNumber(version version.Version) versionNumber {
	// A released version is a higher version number than a development version.
	released := 0
	if version.IsRelease() {
		// By using 256 for a released version we have the option to introduce
		// pre-release version support in the future if needed.
		released = 256
	}

	return versionNumber(
		uint64(version.Major)<<48 |
			uint64(version.Minor)<<32 |
			uint64(version.Patch)<<16 |
			uint64(released),
	)
}

func (v versionNumber) String() string {
	version := version.Version{
		Major: int(v>>48) & 0xffff,
		Minor: int(v>>32) & 0xffff,
		Patch: int(v>>16) & 0xffff,
	}
	if v&0xffff < 256 {
		version.Meta = "pre"
	}
	return version.String()
}
