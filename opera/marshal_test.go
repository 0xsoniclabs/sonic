package opera

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func TestUpdateRules(t *testing.T) {
	require := require.New(t)

	var exp Rules
	exp.Epochs.MaxEpochGas = 99

	exp.Dag.MaxParents = 5
	exp.Economy.MinGasPrice = big.NewInt(7)
	exp.Economy.MinBaseFee = big.NewInt(1e9)
	exp.Blocks.MaxBlockGas = 1000
	got, err := UpdateRules(exp, []byte(`{"Dag":{"MaxParents":5},"Economy":{"MinGasPrice":7},"Blocks":{"MaxBlockGas":1000}}`))
	require.NoError(err)
	require.Equal(exp.String(), got.String(), "mutate fields")

	exp.Dag.MaxParents = 0
	got, err = UpdateRules(exp, []byte(`{"Name":"xxx","NetworkID":1,"Dag":{"MaxParents":0}}`))
	require.NoError(err)
	require.Equal(exp.String(), got.String(), "readonly fields")

	got, err = UpdateRules(exp, []byte(`{}`))
	require.NoError(err)
	require.Equal(exp.String(), got.String(), "empty diff")

	_, err = UpdateRules(exp, []byte(`}{`))
	require.Error(err)
}

func TestUpdateRules_CanUpdateHardForks(t *testing.T) {
	require := require.New(t)

	rules := Rules{
		Economy: EconomyRules{
			MinGasPrice: big.NewInt(1),
			MinBaseFee:  big.NewInt(2),
		},
		Upgrades: Upgrades{
			Berlin:  true,
			London:  false,
			Sonic:   true,
			Allegro: false,
		},
	}

	got, err := UpdateRules(rules, []byte(`{"Upgrades":{"Berlin":false,"London":true,"Sonic":false,"Allegro":true}}`))
	require.NoError(err)
	require.Equal(Upgrades{
		Berlin:  false,
		London:  true,
		Sonic:   false,
		Allegro: true,
	}, got.Upgrades)
}

func TestMainNetRulesRLP(t *testing.T) {
	rules := MainNetRules()
	require := require.New(t)

	b, err := rlp.EncodeToBytes(rules)
	require.NoError(err)

	decodedRules := Rules{}
	require.NoError(rlp.DecodeBytes(b, &decodedRules))

	require.Equal(rules.String(), decodedRules.String())
}

func TestUpgradesRLP_CanBeEncodedAndDecoded(t *testing.T) {
	require := require.New(t)
	setUpgrade := []func(*Upgrades){
		func(u *Upgrades) { u.Berlin = true },
		func(u *Upgrades) { u.London = true },
		func(u *Upgrades) { u.Llr = true },
		func(u *Upgrades) { u.Sonic = true },
		func(u *Upgrades) { u.Allegro = true },
	}

	for mask := range 1 << len(setUpgrade) {
		upgrades := Upgrades{}
		for i, set := range setUpgrade {
			if mask&(1<<i) != 0 {
				set(&upgrades)
			}
		}

		b, err := rlp.EncodeToBytes(upgrades)
		require.NoError(err)

		decodedUpgrades := Upgrades{}
		require.NoError(rlp.DecodeBytes(b, &decodedUpgrades))

		require.Equal(upgrades, decodedUpgrades)
	}
}

func TestRulesBerlinCompatibilityRLP(t *testing.T) {
	require := require.New(t)

	b1, err := rlp.EncodeToBytes(Upgrades{
		Berlin: true,
	})
	require.NoError(err)

	b2, err := rlp.EncodeToBytes(struct {
		Berlin bool
	}{true})
	require.NoError(err)

	require.Equal(b2, b1)
}

func TestGasRulesLLRCompatibilityRLP(t *testing.T) {
	require := require.New(t)

	b1, err := rlp.EncodeToBytes(GasRules{
		MaxEventGas:          1,
		EventGas:             2,
		ParentGas:            3,
		ExtraDataGas:         4,
		BlockVotesBaseGas:    0,
		BlockVoteGas:         0,
		EpochVoteGas:         0,
		MisbehaviourProofGas: 0,
	})
	require.NoError(err)

	b2, err := rlp.EncodeToBytes(struct {
		MaxEventGas  uint64
		EventGas     uint64
		ParentGas    uint64
		ExtraDataGas uint64
	}{1, 2, 3, 4})
	require.NoError(err)

	require.Equal(b2, b1)
}
