package makegenesis

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/drivertype"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/ier"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/Fantom-foundation/lachesis-base/lachesis"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestGenesisBuilder_ExecuteGenesisTxs_ExecutesTransactionsAccordingToUpgrades(t *testing.T) {
	rules := opera.FakeNetRules(opera.GetAllegroUpgrades())
	builder := NewGenesisBuilder()

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	address := crypto.PubkeyToAddress(key.PublicKey)
	builder.AddBalance(address, big.NewInt(1e18))

	finalizeBlockZero(t, builder, rules)

	setCodeTx := makeSetCodeTransaction(t, new(big.Int).SetUint64(rules.NetworkID), key)
	blockProc := DefaultBlockProc()

	// With sonic features and attempting to execute setcode tx: log.Crit is called and program exits.
	// https://github.com/0xsoniclabs/sonic/blob/03bd8b828db3ac51cb9b06254f9d33c75c12c8bd/gossip/blockproc/evmmodule/evm.go#L130
	// TODO: investigate the suitability of containing log.Crit inside of block processing
	err = builder.ExecuteGenesisTxs(blockProc, []*types.Transaction{setCodeTx})
	require.NoError(t, err)
}

func finalizeBlockZero(t *testing.T, builder *GenesisBuilder, rules opera.Rules) {
	t.Helper()

	genesisTime := inter.Timestamp(1234)

	builder.SetCurrentEpoch(ier.LlrIdxFullEpochRecord{
		LlrFullEpochRecord: ier.LlrFullEpochRecord{
			BlockState: iblockproc.BlockState{
				LastBlock: iblockproc.BlockCtx{
					Idx:     0,
					Time:    genesisTime,
					Atropos: hash.Event{},
				},
				FinalizedStateRoot:    hash.Hash{0x42},
				EpochGas:              0,
				EpochCheaters:         lachesis.Cheaters{},
				CheatersWritten:       0,
				ValidatorStates:       make([]iblockproc.ValidatorBlockState, 0),
				NextValidatorProfiles: make(map[idx.ValidatorID]drivertype.Validator),
				DirtyRules:            nil,
				AdvanceEpochs:         0,
			},
			EpochState: iblockproc.EpochState{
				Epoch:             1,
				EpochStart:        genesisTime + 1,
				PrevEpochStart:    genesisTime,
				EpochStateRoot:    hash.Hash{0x43},
				Validators:        pos.NewBuilder().Build(),
				ValidatorStates:   make([]iblockproc.ValidatorEpochState, 0),
				ValidatorProfiles: make(map[idx.ValidatorID]drivertype.Validator),
				Rules:             rules,
			},
		},
		Idx: 1,
	})

	_, _, err := builder.FinalizeBlockZero(rules, genesisTime)
	require.NoError(t, err)
}

func makeSetCodeTransaction(t *testing.T, chainID *big.Int, key *ecdsa.PrivateKey) *types.Transaction {
	t.Helper()

	address := crypto.PubkeyToAddress(key.PublicKey)

	auth := types.SetCodeAuthorization{
		Address: common.Address{},
		ChainID: *uint256.MustFromBig(chainID),
		Nonce:   0,
	}

	txData := types.SetCodeTx{
		To:        address,
		Gas:       550_000,
		GasFeeCap: uint256.NewInt(10_000_000_000),
		AuthList:  []types.SetCodeAuthorization{auth},
	}

	signer := types.LatestSignerForChainID(chainID)
	tx, err := types.SignTx(types.NewTx(&txData), signer, key)
	require.NoError(t, err)
	return tx
}
