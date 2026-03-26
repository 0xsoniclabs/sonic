package sonic

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/ethapi"
	testbackend "github.com/0xsoniclabs/sonic/rpc/test_backend"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func Test_TestBackend(t *testing.T) {
	address1 := common.HexToAddress("0xadd01")
	address2 := common.HexToAddress("0xadd02")

	chain := testbackend.NewBlockchain()
	chain.SetAccount(address1, testbackend.AccountState{
		Nonce:   1,
		Balance: big.NewInt(1000),
	})

	api := ethapi.NewPublicBundleAPI(chain)

	result, err := api.PrepareBundle(t.Context(),
		ethapi.PrepareBundleArgs{
			Transactions: []ethapi.TransactionArgs{
				{
					From: &address1,
					To:   &address2,
				},
			},
		})
	require.NoError(t, err)

	require.Len(t, result.Transactions, 1)
	require.Equal(t, address1, *result.Transactions[0].From)
	require.Equal(t, address2, *result.Transactions[0].To)
	require.Len(t, *result.Transactions[0].AccessList, 1)
}
