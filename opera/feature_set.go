package opera

import "fmt"

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

func (fs FeatureSet) ToUpgrades() (Upgrades, error) {
	var res Upgrades
	switch fs {
	case SonicFeatures:
		res = Upgrades{
			Berlin:  true,
			London:  true,
			Llr:     false,
			Sonic:   true,
			Allegro: false,
		}
	case AllegroFeatures:
		res = Upgrades{
			Berlin:  true,
			London:  true,
			Llr:     false,
			Sonic:   true,
			Allegro: true,
		}
	default:
		return res, fmt.Errorf("unknown feature set: %v", fs)
	}
	return res, nil
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
