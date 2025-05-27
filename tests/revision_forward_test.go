package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/tosca/go/tosca/vm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestTransaction_DelegationDesignationAddressAccessIsConsideredInAllegro(t *testing.T) {
	gas := uint64(21_000) // transaction base
	gas += 7 * 3          // 7 push instructions
	gas += 2_600          // cold access to recipient
	gas += 10             // gas in recursive call (is fully consumed due to failed execution)

	tests := map[string]struct {
		upgrades opera.Upgrades
		gas      uint64
	}{
		"Sonic": {
			upgrades: opera.GetSonicUpgrades(),
			gas:      gas, // delegate designator ignored, no address access.
		},
		"Allegro": {
			upgrades: opera.GetAllegroUpgrades(),
			gas:      gas + 2_600, // cold access to delegate billed in interpreter.
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			net := StartIntegrationTestNetWithJsonGenesis(t, IntegrationTestNetOptions{
				Upgrades: &test.upgrades,
				Accounts: accountsToDeploy(),
			})

			client, err := net.GetClient()
			require.NoError(t, err)
			defer client.Close()

			sender := makeAccountWithBalance(t, net, big.NewInt(1e18))

			gasPrice, err := client.SuggestGasPrice(context.Background())
			require.NoError(t, err)

			chainId, err := client.ChainID(context.Background())
			require.NoError(t, err)

			recipient := common.HexToAddress("0x44")
			txData := &types.AccessListTx{
				ChainID:    chainId,
				Nonce:      0,
				GasPrice:   gasPrice,
				Gas:        test.gas + 1, // +1 to ensure there was no error which consumed the gas
				To:         &recipient,
				Value:      big.NewInt(0),
				Data:       []byte{},
				AccessList: types.AccessList{},
			}
			tx := signTransaction(t, chainId, txData, sender)

			receipt, err := net.Run(tx)
			require.NoError(t, err)

			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
			require.Equal(t, test.gas, receipt.GasUsed)
		})
	}
}

func accountsToDeploy() []makefakegenesis.Account {
	// account 0x42 code: single invalid instruction (0xee)
	// account 0x43 code: delegation designation to 0x42: 0xef0100...042
	// account 0x44 code: code that calls 0x43

	account42 := makefakegenesis.Account{
		Name:    "account42",
		Address: common.HexToAddress("0x42"),
		Code:    []byte{0xee},
		Nonce:   1,
	}

	account43 := makefakegenesis.Account{
		Name:    "account43",
		Address: common.HexToAddress("0x43"),
		Code:    append([]byte{0xef, 0x01, 0x00}, common.HexToAddress("0x42").Bytes()...),
		Nonce:   1,
	}

	code44 := []byte{
		byte(vm.PUSH1), 0x00, // retSize
		byte(vm.PUSH1), 0x00, // retOffset
		byte(vm.PUSH1), 0x00, // argSize
		byte(vm.PUSH1), 0x00, // argOffset
		byte(vm.PUSH1), 0x00, // value
		byte(vm.PUSH20),
	}
	code44 = append(code44, common.HexToAddress("0x43").Bytes()...) // address
	code44 = append(code44,
		byte(vm.PUSH1), 0x0a, // gas
		byte(vm.CALL), // call
		byte(vm.STOP), // return
	)

	account44 := makefakegenesis.Account{
		Name:    "account44",
		Address: common.HexToAddress("0x44"),
		Code:    code44,
		Nonce:   1,
	}

	return []makefakegenesis.Account{account42, account43, account44}
}
