package tests

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestEip3607(t *testing.T) {

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	account := &Account{key}

	net := StartIntegrationTestNetWithJsonGenesis(t,
		IntegrationTestNetOptions{
			Accounts: []makefakegenesis.Account{
				{
					Name:    "Contract",
					Address: account.Address(),
					Balance: big.NewInt(1e18),
					Code:    makefakegenesis.VariableLenCode([]byte("0xabababaab")),
				},
			},
		})

	// create a transaction from the contract address
	tx :=
		signTransaction(t, net.GetChainId(),
			setTransactionDefaults(t, net, &types.LegacyTx{}, account), account)

	receipt, err := net.Run(tx)
	require.NoError(t, err)
	require.NotNil(t, receipt)

}
