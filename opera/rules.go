package opera

import (
	"cmp"
	"encoding/json"
	"math"
	"math/big"
	"slices"
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	ethparams "github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera/contracts/evmwriter"
	"github.com/0xsoniclabs/tosca/go/geth_adapter"
	"github.com/0xsoniclabs/tosca/go/interpreter/lfvm"
)

const (
	MainNetworkID   uint64 = 0xfa
	TestNetworkID   uint64 = 0xfa2
	FakeNetworkID   uint64 = 0xfa3
	DefaultEventGas uint64 = 28000
	berlinBit              = 1 << 0
	londonBit              = 1 << 1
	llrBit                 = 1 << 2

	// hard-forks
	sonicBit   = 1 << 3
	allegroBit = 1 << 4
	brioBit    = 1 << 5

	// optional features
	singleProposerBlockFormationBit = 1 << 63

	MinimumMaxBlockGas          = 5_000_000_000 // < must be large enough to allow internal transactions to seal blocks
	MaximumMaxBlockGas          = math.MaxInt64 // < should fit into 64-bit signed integers to avoid parsing errors in third-party libraries
	defaultTargetGasRate        = 15_000_000    // 15 MGas/s
	defaultEventEmitterInterval = 600 * time.Millisecond
)

var DefaultVMConfig = func() vm.Config {

	// For transaction processing, Tosca's LFVM is used.
	interpreter, err := lfvm.NewInterpreter(lfvm.Config{})
	if err != nil {
		panic(err)
	}
	lfvmFactory := geth_adapter.NewGethInterpreterFactory(interpreter)

	// For tracing, Geth's EVM is used.
	gethFactory := func(evm *vm.EVM) vm.Interpreter {
		return vm.NewEVMInterpreter(evm)
	}

	return vm.Config{
		StatePrecompiles: map[common.Address]vm.PrecompiledStateContract{
			evmwriter.ContractAddress: &evmwriter.PreCompiledContract{},
		},
		Interpreter:           lfvmFactory,
		InterpreterForTracing: gethFactory,

		// Fantom/Sonic modifications
		ChargeExcessGas:                 true,
		IgnoreGasFeeCap:                 true,
		InsufficientBalanceIsNotAnError: true,
		SkipTipPaymentToCoinbase:        true,
	}
}()

type RulesRLP struct {
	Name      string
	NetworkID uint64

	// Graph options
	Dag DagRules

	// Emitter options
	Emitter EmitterRules

	// Epochs options
	Epochs EpochsRules

	// Blockchain options
	Blocks BlocksRules

	// Economy options
	Economy EconomyRules

	Upgrades Upgrades `rlp:"-"`
}

// Rules describes opera net.
// Note keep track of all the non-copiable variables in Copy()
type Rules RulesRLP

// GasPowerRules defines gas power rules in the consensus.
type GasPowerRules struct {
	AllocPerSec        uint64
	MaxAllocPeriod     inter.Timestamp
	StartupAllocPeriod inter.Timestamp
	MinStartupGas      uint64
}

type GasRulesRLPV1 struct {
	MaxEventGas  uint64
	EventGas     uint64
	ParentGas    uint64
	ExtraDataGas uint64
	// Post-LLR fields
	BlockVotesBaseGas    uint64
	BlockVoteGas         uint64
	EpochVoteGas         uint64
	MisbehaviourProofGas uint64
}

type GasRules GasRulesRLPV1

type EpochsRules struct {
	MaxEpochGas      uint64
	MaxEpochDuration inter.Timestamp
}

// DagRules of Lachesis DAG (directed acyclic graph).
type DagRules struct {
	MaxParents     idx.Event
	MaxFreeParents idx.Event // maximum number of parents with no gas cost
	MaxExtraData   uint32
}

// EmitterRules contains options for the emitter of Lachesis events.
type EmitterRules struct {
	// Interval defines the length of the period
	// between events produced by the emitter in nanoseconds.
	//
	// The Interval is used to control the rate of event
	// production by the emitter. It thus indirectly controls
	// the rate of blocks production on the network, by providing
	// a lower bound. The actual block production rate is also
	// influenced by the number of validators, their weighting,
	// and the inter-connection of events. However, the Interval
	// should provide an effective mean to control the block
	// production rate.
	Interval inter.Timestamp

	// StallThreshold defines a maximum time the confirmation of
	// new events may be delayed before the emitter considers the
	// network stalled in nanoseconds.
	//
	// The emitter has two modes: normal and stalled. In normal
	// mode, the emitter produces events at a regular interval, as
	// defined by the Interval option. In stalled mode, the emitter
	// produces events at a much lower rate, to avoid building up
	// a backlog of events. The StallThreshold defines the upper
	// limit of delay seen for new confirmed events before the emitter
	// switches to stalled mode.
	//
	// This option is disabled if Interval is set to 0.
	StallThreshold inter.Timestamp

	// StallInterval defines the length of the period between
	// events produced by the emitter in nanoseconds when the
	// network is stalled.
	StalledInterval inter.Timestamp
}

// BlocksMissed is information about missed blocks from a staker
type BlocksMissed struct {
	BlocksNum idx.Block
	Period    inter.Timestamp
}

// EconomyRules contains economy constants
type EconomyRules struct {
	BlockMissedSlack idx.Block

	Gas GasRules

	// MinGasPrice defines a lower boundary for the gas price
	// on the network. However, its interpretation is different
	// in the context of the Fantom and Sonic networks.
	//
	// On the Fantom network: MinGasPrice is the minimum gas price
	// defining the base fee of a block. The MinGasPrice is set by
	// the node driver and SFC on the Fantom network and adjusted
	// based on load observed during an epoch. Base fees charged
	// on the network correspond exactly to the MinGasPrice.
	//
	// On the Sonic network: this parameter is ignored. Base fees
	// are controlled by the MinBaseFee parameter.
	MinGasPrice *big.Int

	// MinBaseFee is a lower bound for the base fee on the network.
	// This option is only supported by the Sonic network. On the
	// Fantom network it is ignored.
	//
	// On the Sonic network, base fees are automatically adjusted
	// after each block based on the observed gas consumption rate.
	// The value set by this parameter is a lower bound for these
	// adjustments. Base fees may never fall below this value.
	// Adjustments are made dynamically analogous to EIP-1559.
	// See https://eips.ethereum.org/EIPS/eip-1559 and https://t.ly/BKrcr
	// for additional information.
	MinBaseFee *big.Int

	ShortGasPower GasPowerRules
	LongGasPower  GasPowerRules
}

// BlocksRules contains blocks constants
type BlocksRules struct {
	MaxBlockGas             uint64 // technical hard limit, gas is mostly governed by gas power allocation
	MaxEmptyBlockSkipPeriod inter.Timestamp
}

type Upgrades struct {
	// -- Fantom Chain --
	Berlin bool
	London bool
	Llr    bool

	// -- Sonic Chain Hard Forks --
	Sonic   bool // < launch version of the Sonic chain, introducing Cancun features
	Allegro bool // < first hard fork of the Sonic chain, introducing Prague features
	Brio    bool // < second hard fork of the Sonic chain, introducing Osaka features

	// -- Optional Features --

	// SingleProposerBlockFormation enables the creation of full block proposals
	// by a single proposer, rather than a distributed event-based protocol.
	// This feature is introduced by V2.1 of the Sonic client. It thus
	//
	//    MUST ONLY BE ENABLED WHEN ALL NODES ARE RUNNING V2.1 OR LATER
	//
	// Any node not running V2.1 or later will ignore this flag, will not be
	// able to process the new payload format used by this protocol, and
	// eventually drop of the network due to the inability to stay synced.
	//
	// Given the conditions stated above, the feature is considered optional.
	// It can be enabled or disabled at any time. Changes in the feature state
	// become effective at the start of the next epoch.
	SingleProposerBlockFormation bool
}

// UpgradeHeight contains the information about the block height at which
// the upgrades become effective. The upgrades are defined by the Upgrades
// struct, which contains the feature flags for the Sonic chain.
// The Height field is the block height at which the upgrades become effective.
// The Time field is the timestamp, in the current implementation, it is ignored
// (See [CreateTransientEvmChainConfig] for details).
type UpgradeHeight struct {
	Upgrades Upgrades
	Height   idx.Block
	Time     inter.Timestamp
}

// CreateTransientEvmChainConfig creates an instance of ethparams.ChainConfig
// for the given block height. The instance of ethparams.ChainConfig shall not be
// stored for later use, and it should not be considered as a canonical source of
// information about the chain configuration. It is only valid for the given block
// height and should be used only for the purpose of configuring Geth tooling.
//
// - chainID is semantically equivalent to the Rules.NetworkID
// - upgradeHeights is a list of UpgradeHeight instances that define the
// block heights at which upgrades become effective.
// - currentBlockHeight is the current block height at which the chain config is
// created.
//
// Note about timestamps:
// go-ethereum's ChainConfig uses timestamps to determine the activation of
// upgrades from Shanghai onwards. However, Sonic timestamps are measure in
// nanoseconds, go-ethereum routines use seconds, and the Sonic network has a
// sub second cadence.  Therefore it is not possible to use timestamps to
// determine the activation of upgrades. Instead, Sonic relies solely on block
// heights.
// Timestamps contained in the returned instance are always set to 0, which
// considered to be always a past timestamp. This is the reason why the returned
// ChainConfig is not valid processing any other block height than the one
// specified by CurrentBlockHeight.
func CreateTransientEvmChainConfig(
	chainID uint64,
	upgradeHeights []UpgradeHeight,
	currentBlockHeight idx.Block,
) *ethparams.ChainConfig {

	timestampInThePast := uint64(0)

	cfg := ethparams.ChainConfig{
		ChainID: new(big.Int).SetUint64(chainID),
		// Following upgrades are always enabled in Sonic (from block height 0):
		HomesteadBlock:      big.NewInt(0),
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		// Following upgrades are always enabled in Sonic (past timestamp):
		ShanghaiTime: &timestampInThePast,
		CancunTime:   &timestampInThePast,
	}

	sortedUpgradeHeights := make([]UpgradeHeight, len(upgradeHeights))
	copy(sortedUpgradeHeights, upgradeHeights)

	slices.SortFunc(sortedUpgradeHeights, func(a, b UpgradeHeight) int {
		return cmp.Compare(a.Height, b.Height)
	})

	// reverse iterate through the upgrade heights
	for i := len(sortedUpgradeHeights) - 1; i >= 0; i-- {
		if sortedUpgradeHeights[i].Height <= currentBlockHeight {
			upgrade := sortedUpgradeHeights[i].Upgrades

			if upgrade.Allegro {
				cfg.PragueTime = &timestampInThePast
			}

			if upgrade.Brio {
				cfg.OsakaTime = &timestampInThePast
			}

			break
		}
	}
	return &cfg
}

// GetSonicUpgrades contains the feature flags for the Sonic upgrade.
func GetSonicUpgrades() Upgrades {
	return Upgrades{
		Berlin:  true,
		London:  true,
		Llr:     false,
		Sonic:   true,
		Allegro: false,
	}
}

// GetAllegroUpgrades contains the feature flags for the Allegro upgrade.
func GetAllegroUpgrades() Upgrades {
	return Upgrades{
		Berlin:  true,
		London:  true,
		Llr:     false,
		Sonic:   true,
		Allegro: true,
	}
}

func MainNetRules() Rules {
	return Rules{
		Name:      "main",
		NetworkID: MainNetworkID,
		Dag:       DefaultDagRules(),
		Emitter:   DefaultEmitterRules(),
		Epochs:    DefaultEpochsRules(),
		Economy:   DefaultEconomyRules(),
		Blocks: BlocksRules{
			MaxBlockGas:             MinimumMaxBlockGas,
			MaxEmptyBlockSkipPeriod: inter.Timestamp(1 * time.Minute),
		},
		Upgrades: GetAllegroUpgrades(),
	}
}

func FakeNetRules(upgrades Upgrades) Rules {
	return Rules{
		Name:      "fake",
		NetworkID: FakeNetworkID,
		Dag:       DefaultDagRules(),
		Emitter:   DefaultEmitterRules(),
		Epochs:    FakeNetEpochsRules(),
		Economy:   FakeEconomyRules(),
		Blocks: BlocksRules{
			MaxBlockGas:             MinimumMaxBlockGas,
			MaxEmptyBlockSkipPeriod: inter.Timestamp(4 * time.Second),
		},
		Upgrades: upgrades,
	}
}

// DefaultEconomyRules returns mainnet economy
func DefaultEconomyRules() EconomyRules {
	rules := EconomyRules{
		BlockMissedSlack: 50,
		Gas:              DefaultGasRules(),
		MinGasPrice:      big.NewInt(1e9),
		MinBaseFee:       big.NewInt(1e9), // 1 Gwei
		ShortGasPower:    DefaultGasPowerRules(),
		LongGasPower:     DefaultGasPowerRules(),
	}
	return rules
}

// FakeEconomyRules returns fakenet economy
func FakeEconomyRules() EconomyRules {
	return DefaultEconomyRules()
}

func DefaultDagRules() DagRules {
	return DagRules{
		MaxParents:     10,
		MaxFreeParents: 3,
		MaxExtraData:   128,
	}
}

func DefaultEmitterRules() EmitterRules {
	return EmitterRules{
		Interval:        inter.Timestamp(defaultEventEmitterInterval.Nanoseconds()),
		StallThreshold:  inter.Timestamp(30 * time.Second),
		StalledInterval: inter.Timestamp(60 * time.Second),
	}
}

func DefaultEpochsRules() EpochsRules {
	return EpochsRules{
		MaxEpochGas:      defaultTargetGasRate * 300, // ~5 minute epoch
		MaxEpochDuration: inter.Timestamp(1 * time.Hour),
	}
}

func DefaultGasRules() GasRules {
	return GasRules{
		MaxEventGas:          defaultTargetGasRate*1000/uint64(defaultEventEmitterInterval.Milliseconds()) + DefaultEventGas,
		EventGas:             DefaultEventGas,
		ParentGas:            2400,
		ExtraDataGas:         25,
		BlockVotesBaseGas:    1024,
		BlockVoteGas:         512,
		EpochVoteGas:         1536,
		MisbehaviourProofGas: 71536,
	}
}

func FakeNetEpochsRules() EpochsRules {
	cfg := DefaultEpochsRules()
	cfg.MaxEpochDuration = inter.Timestamp(10 * time.Minute)
	return cfg
}

// DefaultGasPowerRules is long-window config
func DefaultGasPowerRules() GasPowerRules {
	return GasPowerRules{
		// In total, the network can spend 2x the target rate of gas per second.
		// This allocation rate is distributed among validators weighted by their
		// stake. Validators gain gas power to spend on events accordingly.
		//
		// The selected value is twice as high as the targeted gas rate to allow
		// for some head-room in a stable network load situation. If the network
		// load is higher than the target rate, gas prices will increase exponentially
		// and the demand for transactions should decrease.
		AllocPerSec: 2 * defaultTargetGasRate,

		// Validators can at most spend 5s of gas in one event. This accumulation is
		// required to accommodate large transactions with a gas limit larger than
		// the allocation share of a single validator. For instance, if there would
		// be 10 validators with even stake, and the allocation rate would be 10 MGas/s,
		// the maximum gas each validator could spend per second would be 1 MGas/s.
		// With this setting, a single validator could accumulate up to 5 MGas of gas
		// over a period of 5 seconds to spend in a single event.
		MaxAllocPeriod: inter.Timestamp(5 * time.Second),

		StartupAllocPeriod: inter.Timestamp(time.Second),
		MinStartupGas:      DefaultEventGas * 20,
	}
}

func (r Rules) Copy() Rules {
	cp := r
	cp.Economy.MinGasPrice = new(big.Int).Set(r.Economy.MinGasPrice)

	// there is a bug in pre-Allegro versions that MinBaseFee is not deep copied.
	// Since switching to deep-copy is not possible in a network running combination
	// of Allegro and pre-Allegro versions, we need to enable this fix only when Allegro is applied.
	if cp.Upgrades.Allegro {
		cp.Economy.MinBaseFee = new(big.Int).Set(r.Economy.MinBaseFee)
	}

	return cp
}

// Validate checks the rules for consistency and safety. Rules are considered safe if
// they do not risk stalling the network or preventing future rule updates.
//
// Note: the validation is very liberal to allow a maximum flexibility in the rules.
// It merely checks for the most critical configuration errors that may lead to network
// stalls or rule update issues. However, many valid configurations may still result
// in undesirable network behavior. Rule-setters need to be aware of the implications
// of their choices and should always test their rules in a controlled environment.
// This validation is not a substitute for proper testing.
//
// previous Rules is used to check for changes in the rules that may conflict with
// currently running network. It is expected that the rules are validated before
// they are applied to the network, and that the previous rules are the rules currently
// running.
func (r Rules) Validate(previous Rules) error {
	return validate(previous, r)
}

func (r Rules) String() string {
	b, _ := json.Marshal(&r)
	return string(b)
}
