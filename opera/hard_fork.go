package opera

// HardFork is an enumeration of Sonic hard forks. Each hard fork is a
// fundamental upgrade to the blockchain's rules, involving significant changes
// to the protocol. These changes are not backward-compatible, meaning that
// nodes running the old version of the software will not be able to
// participate in the network after the upgrade. Hard forks can only be enabled,
// but never be disabled. Every network is always in a single hard fork state.
type HardFork int

const (
	Sonic   HardFork = iota // < enables the initial sonic release features
	Allegro                 // < enables the allegro release features
)

// ToUpgrades returns the Upgrades that are enabled by the feature set.
// If called from an unknown feature set, it will return the pre-sonic
// upgrades.
func (hf HardFork) ToUpgrades() Upgrades {
	sonic := hf == Sonic
	allegro := hf == Allegro
	res := Upgrades{
		Berlin:  true,
		London:  true,
		Llr:     false,
		Sonic:   sonic || allegro,
		Allegro: allegro,
	}
	return res
}

func (hf HardFork) String() string {
	switch hf {
	case Sonic:
		return "sonic"
	case Allegro:
		return "allegro"
	default:
		return "unknown"
	}
}
