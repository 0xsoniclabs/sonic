package opera

// FeatureSet is an enumeration of different releases, each one enabling a
// different set of features. These are an abstraction that allows to reason
// about the different releases instead of isolated upgrades.
// Feature sets are exclusive, and a (fake-)net cannot be configured with
// more than one feature set at the time.
type FeatureSet int

const (
	SonicFeatures   FeatureSet = iota // < enables the initial sonic release features
	AllegroFeatures                   // < enables the allegro release features
)

// ToUpgrades returns the Upgrades that are enabled by the feature set.
// If called from an unknown feature set, it will return the pre-sonic
// upgrades.
func (fs FeatureSet) ToUpgrades() Upgrades {
	sonic := fs == SonicFeatures
	allegro := fs == AllegroFeatures
	res := Upgrades{
		Berlin:  true,
		London:  true,
		Llr:     false,
		Sonic:   sonic || allegro,
		Allegro: allegro,
	}
	return res
}

func (fs FeatureSet) String() string {
	switch fs {
	case SonicFeatures:
		return "sonic"
	case AllegroFeatures:
		return "allegro"
	default:
		return "unknown"
	}
}
