package many

import (
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/stretchr/testify/require"
)

func TestManyNodes(t *testing.T) {

	// const numAccounts = 1000
	// senders := make([]*tests.Account, 0, numAccounts)
	// senderAccounts := make([]makefakegenesis.Account, 0, numAccounts)
	// for range numAccounts {
	// 	account := tests.NewAccount()
	// 	senders = append(senders, account)
	// 	senderAccounts = append(senderAccounts, makefakegenesis.Account{
	// 		Address: account.Address(),
	// 		Balance: big.NewInt(9e18),
	// 	})
	// }
	const numValidators = 35
	distribution, err := utils.NewFromMedianAndPercentile(5_000, 0.95, 100_000, nil)
	require.NoError(t, err)
	validatorsStake := make([]uint64, 0, numValidators)
	for range numValidators {
		validatorsStake = append(validatorsStake, uint64(distribution.Sample()))
	}
	_ = validatorsStake

	tests.StartIntegrationTestNetWithJsonGenesis(t, tests.IntegrationTestNetOptions{
		// ValidatorsStake: validatorsStake,
		ValidatorsStake: makefakegenesis.CreateEqualValidatorStake(35),
	})

	// stop := make(chan struct{})
	// done := make(chan struct{})

	// go func() {
	// 	defer close(done)

	// 	for {
	// 		select {
	// 		case <-stop:
	// 			return
	// 		default:
	// 		}

	// 		batch := make([]*types.Transaction, 0, 100)
	// 		for range len(batch) {
	// 			sender := senders[rand.IntN(numAccounts)]
	// 			tx := tests.CreateTransaction(t, net, &types.LegacyTx{
	// 				Value: big.NewInt(1),
	// 				To:    &common.Address{0},
	// 			}, sender)
	// 			batch = append(batch, tx)
	// 		}
	// 		_, err := net.RunAll(batch)
	// 		require.NoError(t, err)
	// 	}
	// }()

	time.Sleep(120 * time.Second)

	// close(stop)
	// <-done
}
