package opera

import (
	"slices"
	"strings"
)

//go:generate stringer -type=Feature

// Feature is an enumeration of features that can be enabled or disabled in
// the client. Feature implementations should use this enumeration to determine
// if a feature is enabled or disabled in the current configuration.
//
// Upgrades (aka hard forks) are enabling or disabling features in the client
// as a group. By separating the upgrades from individual features, development
// teams can enable or disable features independently for tests and when defining
// the feature set of an Upgrade to be rolled out on the network, arbitrary
// subsets of features can be selected.
type Feature int

// A list of feature flags. Users of the feature flags should not rely on the numeric
// values of the flags. They may be re-ordered or re-numbered in the future.
const (
	// Sonic specific features
	SonicCertificateChain Feature = iota
	NetworkRuleChecks

	// EIPs
	EIP7702_SetEoaCode
)

// Features is a utility type for sets of features.
type Features struct {
	flags uint64 // < bit-map of enabled features
}

// NewFeatures creates a new feature set with the given features enabled.
func NewFeatures(features ...Feature) Features {
	res := Features{}
	for _, f := range features {
		res = res.Enable(f)
	}
	return res
}

// Has returns true if the feature is enabled in the set.
func (f Features) Has(feature Feature) bool {
	return f.flags&(1<<feature) != 0
}

func (f Features) Enable(feature Feature) Features {
	f.flags |= 1 << feature
	return f
}

func (f Features) Disable(feature Feature) Features {
	f.flags &^= 1 << feature
	return f
}

// Features enumerates all features that are enabled in the set.
func (f Features) Features() []Feature {
	var res []Feature
	for feature := Feature(0); feature < Feature(64); feature++ {
		if f.Has(feature) {
			res = append(res, feature)
		}
	}
	return res
}

// String lists enabled features in a human-readable format.
func (f Features) String() string {
	var res []string
	for _, feature := range f.Features() {
		res = append(res, feature.String())
	}
	slices.Sort(res)
	return "{" + strings.Join(res, ",") + "}"
}
