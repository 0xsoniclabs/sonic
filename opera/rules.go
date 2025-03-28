package opera

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/0xsoniclabs/consensus/consensus"

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
	sonicBit               = 1 << 3
	allegroBit             = 1 << 4

	defaultMaxBlockGas          = 1_000_000_000
	defaultTargetGasRate        = 15_000_000 // 15 MGas/s
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
	MaxParents     consensus.Seq
	MaxFreeParents consensus.Seq // maximum number of parents with no gas cost
	MaxExtraData   uint32
}

// EmitterRules contains options for the emitter of Lachesis events.
type EmitterRules struct {
	// Interval defines the length of the period
	// between events produced by the emitter in milliseconds.
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
	// network stalled.
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
	// events produced by the emitter in milliseconds when the
	// network is stalled.
	StalledInterval inter.Timestamp
}

// BlocksMissed is information about missed blocks from a staker
type BlocksMissed struct {
	BlocksNum consensus.BlockID
	Period    inter.Timestamp
}

// EconomyRules contains economy constants
type EconomyRules struct {
	BlockMissedSlack consensus.BlockID

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

	// -- Sonic Chain --
	Sonic   bool // < launch version of the Sonic chain
	Allegro bool // < first hard fork of the Sonic chain
}

type UpgradeHeight struct {
	Upgrades Upgrades
	Height   consensus.BlockID
	Time     inter.Timestamp
}

var BaseChainConfig = ethparams.ChainConfig{
	ChainID:                 big.NewInt(1337),
	HomesteadBlock:          big.NewInt(0),
	DAOForkBlock:            nil,
	DAOForkSupport:          false,
	EIP150Block:             big.NewInt(0),
	EIP155Block:             big.NewInt(0),
	EIP158Block:             big.NewInt(0),
	ByzantiumBlock:          big.NewInt(0),
	ConstantinopleBlock:     big.NewInt(0),
	PetersburgBlock:         big.NewInt(0),
	IstanbulBlock:           big.NewInt(0),
	MuirGlacierBlock:        big.NewInt(0), // EIP-2384: Muir Glacier Difficulty Bomb Delay - relevant for ethereum only
	BerlinBlock:             nil,           // to be overwritten in EvmChainConfig
	LondonBlock:             nil,           // to be overwritten in EvmChainConfig
	ArrowGlacierBlock:       nil,           // EIP-4345: Difficulty Bomb Delay - relevant for ethereum only
	GrayGlacierBlock:        nil,           // EIP-5133: Delaying Difficulty Bomb - relevant for ethereum only
	MergeNetsplitBlock:      nil,
	ShanghaiTime:            nil, // to be overwritten in EvmChainConfig
	CancunTime:              nil, // to be overwritten in EvmChainConfig
	PragueTime:              nil, // to be overwritten in EvmChainConfig
	VerkleTime:              nil,
	TerminalTotalDifficulty: nil,
	Ethash:                  new(ethparams.EthashConfig),
	Clique:                  nil,
}

// EvmChainConfig returns ChainConfig for transactions signing and execution
func (r Rules) EvmChainConfig(hh []UpgradeHeight) *ethparams.ChainConfig {
	cfg := BaseChainConfig
	cfg.ChainID = new(big.Int).SetUint64(r.NetworkID)
	for i, h := range hh {
		height := new(big.Int)
		timestamp := new(uint64)
		if i > 0 {
			height.SetUint64(uint64(h.Height))
			*timestamp = uint64(h.Time)
		}
		if cfg.BerlinBlock == nil && h.Upgrades.Berlin {
			cfg.BerlinBlock = height
		}
		if !h.Upgrades.Berlin {
			// disabling upgrade breaks the history replay - should be never used
			cfg.BerlinBlock = nil
		}

		if cfg.LondonBlock == nil && h.Upgrades.London {
			cfg.LondonBlock = height
		}
		if !h.Upgrades.London {
			// disabling upgrade breaks the history replay - should be never used
			cfg.LondonBlock = nil
		}

		if cfg.CancunTime == nil && h.Upgrades.Sonic {
			cfg.ShanghaiTime = timestamp
			cfg.CancunTime = timestamp
		}

		if cfg.PragueTime == nil && h.Upgrades.Allegro {
			cfg.PragueTime = timestamp
		}

		if !h.Upgrades.Sonic {
			// disabling upgrade breaks the history replay - should be never used
			cfg.ShanghaiTime = nil
			cfg.CancunTime = nil
		}

		if !h.Upgrades.Allegro {
			cfg.PragueTime = nil
		}
	}
	return &cfg
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
			MaxBlockGas:             defaultMaxBlockGas,
			MaxEmptyBlockSkipPeriod: inter.Timestamp(1 * time.Minute),
		},
	}
}

func FakeNetRules() Rules {
	return Rules{
		Name:      "fake",
		NetworkID: FakeNetworkID,
		Dag:       DefaultDagRules(),
		Emitter:   DefaultEmitterRules(),
		Epochs:    FakeNetEpochsRules(),
		Economy:   FakeEconomyRules(),
		Blocks: BlocksRules{
			MaxBlockGas:             defaultMaxBlockGas,
			MaxEmptyBlockSkipPeriod: inter.Timestamp(3 * time.Second),
		},
		Upgrades: Upgrades{
			Berlin:  true,
			London:  true,
			Llr:     false,
			Sonic:   true,
			Allegro: true,
		},
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
		MaxEpochDuration: inter.Timestamp(4 * time.Hour),
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
	return cp
}

func (r Rules) String() string {
	b, _ := json.Marshal(&r)
	return string(b)
}
