package tests

import (
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestClient_HardForkIsExecutedAndClientAdoptsNewBehaviors(t *testing.T) {

	net := StartIntegrationTestNetWithFakeGenesis(t,
		IntegrationTestNetOptions{
			// Explicitly set the network to use the Sonic Hard Fork
			Upgrades: AsPointer(opera.GetSonicUpgrades()),
			// Use 2 nodes to test the rules update propagation
			NumNodes: 2,
		},
	)
	client0, err := net.GetClientConnectedToNode(0)
	require.NoError(t, err)
	defer client0.Close()

	chainID, err := client0.ChainID(t.Context())
	require.NoError(t, err)

	account := net.GetSessionSponsor()

	// SetCodeTx cannot be accepted before Prague hard fork
	tx := signTransaction(t, chainID, setTransactionDefaults(t, net, &types.SetCodeTx{}, account), account)
	err = client0.SendTransaction(t.Context(), tx)
	require.ErrorContains(t, err, evmcore.ErrTxTypeNotSupported.Error())

	// Update network rules to enable the Allegro Hard Fork
	type rulesType struct {
		Upgrades struct{ Allegro bool }
	}
	rulesDiff := rulesType{
		Upgrades: struct{ Allegro bool }{Allegro: true},
	}
	updateNetworkRules(t, net, rulesDiff)

	// reach epoch ceiling to apply the new rules
	advanceEpochAndWaitForBlocks(t, net)

	// Submit a transaction that requires the new behavior
	nonce, err := client0.PendingNonceAt(t.Context(), account.Address())
	require.NoError(t, err)
	authorization, err := types.SignSetCode(account.PrivateKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(chainID),
		Address: common.Address{42},
		Nonce:   nonce + 1,
	})
	require.NoError(t, err, "failed to sign SetCode authorization")
	txData := &types.SetCodeTx{AuthList: []types.SetCodeAuthorization{authorization}}
	tx = signTransaction(t, chainID, setTransactionDefaults(t, net, txData, account), account)

	receipt, err := net.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	delegationIndicator :=
		hexutil.MustDecode("0xEF01002A00000000000000000000000000000000000000")

	code, err := client0.CodeAt(t.Context(), account.Address(), nil)
	require.NoError(t, err)
	require.Equal(t, code, delegationIndicator)

	// Check that second node executed the transaction
	client1, err := net.GetClientConnectedToNode(1)
	require.NoError(t, err)

	code, err = client1.CodeAt(t.Context(), account.Address(), nil)
	require.NoError(t, err)
	require.Equal(t, code, delegationIndicator)

}
