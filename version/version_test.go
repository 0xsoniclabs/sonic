package version

import "testing"

func TestMakeVersion_AcceptValidVersionNumber(t *testing.T) {
	tests := map[string]struct {
		major, minor, patch int
		rc                  int
		released            bool
	}{
		"1.2.3":     {major: 1, minor: 2, patch: 3, released: true},
		"1.2.0-dev": {major: 1, minor: 2, released: false},
		"1.2.3-rc4": {major: 1, minor: 2, patch: 3, rc: 4},
	}

	for want, test := range tests {
		version, err := makeVersion(test.major, test.minor, test.patch, test.rc, test.released)
		if err != nil {
			t.Errorf("makeVersion(%d, %d, %d, %d, %v) returned an error: %v", test.major, test.minor, test.patch, test.rc, test.released, err)
		}
		if got := version.String(); got != want {
			t.Errorf("makeVersion(%d, %d, %d, %d, %v) = %q, want %q", test.major, test.minor, test.patch, test.rc, test.released, got, want)
		}
	}
}

func TestMakeVersion_DetectsInvalidVersionNumber(t *testing.T) {
	tests := map[string]struct {
		major, minor, patch int
		rc                  int
		released            bool
	}{
		"release of candidate":         {major: 1, minor: 2, patch: 3, rc: 1, released: true},
		"patch version in development": {major: 1, minor: 2, patch: 3, released: false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := makeVersion(test.major, test.minor, test.patch, test.rc, test.released)
			if err == nil {
				t.Errorf("expected an error, got nil")
			}
		})
	}
}
