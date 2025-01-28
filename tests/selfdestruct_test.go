package tests

import (
	"context"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/selfdestruct"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func TestSelfDestruct(t *testing.T) {
	require := require.New(t)

	net, err := StartIntegrationTestNet(t.TempDir())
	require.NoError(err, "failed to start test network")
	defer net.Stop()

	t.Run("constructor", func(t *testing.T) {
		testSelfDestruct_Constructor(t, net)
	})

	t.Run("nested call", func(t *testing.T) {
		testSelfDestruct_NestedCall(t, net)
	})
}

func testSelfDestruct_Constructor(t *testing.T, net *IntegrationTestNet) {
	contractInitialBalance := int64(1234)

	tests := map[string]struct {
		deployTx  deployTxFunction[selfdestruct.SelfDestruct]
		executeTx executeTxFunction[selfdestruct.SelfDestruct]
		effects   map[string]effectFunction
	}{
		// This test checks the testing infrastructure itself
		"sanity check/no selfdestruct": {
			deployTx: func(opts *bind.TransactOpts, backend bind.ContractBackend, _ common.Address) (common.Address, *types.Transaction, *selfdestruct.SelfDestruct, error) {
				opts.Value = big.NewInt(contractInitialBalance)
				return selfdestruct.DeploySelfDestruct(opts, backend,
					false,            // do not selfdestruct in constructor
					false,            // beneficiary is not self
					common.Address{}) // ignored, no selfdestruct
			},
			effects: map[string]effectFunction{
				"contract keeps balance": contractBalanceIs(contractInitialBalance),
				"storage is not deleted": contractStorageIs(123),
				"code is not deleted":    contractCodeSizeIsNot(0),
			},
		},
		"different tx/beneficiary is other account": {
			deployTx: func(opts *bind.TransactOpts, backend bind.ContractBackend, _ common.Address) (common.Address, *types.Transaction, *selfdestruct.SelfDestruct, error) {
				opts.Value = big.NewInt(contractInitialBalance)
				return selfdestruct.DeploySelfDestruct(opts, backend,
					false,            // do not selfdestruct in constructor
					false,            // beneficiary is not self
					common.Address{}) // ignored, no selfdestruct
			},
			executeTx: func(contract *selfdestruct.SelfDestruct, opts *bind.TransactOpts, beneficiaryAddress common.Address) (*types.Transaction, error) {
				return contract.DestroyContract(opts,
					false, // beneficiary is not self
					beneficiaryAddress)
			},
			effects: map[string]effectFunction{
				"the current execution frame halts": executionHalted(),
				"contract looses balance":           contractBalanceIs(0),
				"beneficiary gains balance":         beneficiaryBalanceIs(contractInitialBalance),
				"storage is not deleted":            contractStorageIs(123),
				"code is not deleted":               contractCodeSizeIsNot(0),
			},
		},
		"different tx/beneficiary is same account": {
			deployTx: func(opts *bind.TransactOpts, backend bind.ContractBackend, _ common.Address) (common.Address, *types.Transaction, *selfdestruct.SelfDestruct, error) {
				opts.Value = big.NewInt(contractInitialBalance)
				return selfdestruct.DeploySelfDestruct(opts, backend,
					false,            // do not selfdestruct in constructor
					true,             // beneficiary is self
					common.Address{}) // ignored, no selfdestruct
			},
			executeTx: func(contract *selfdestruct.SelfDestruct, opts *bind.TransactOpts, beneficiaryAddress common.Address) (*types.Transaction, error) {
				return contract.DestroyContract(opts, true, beneficiaryAddress)
			},
			effects: map[string]effectFunction{
				"the current execution frame halts": executionHalted(),
				"storage is not deleted":            contractStorageIs(123),
				"code is not deleted":               contractCodeSizeIsNot(0),
				"balance is not burned":             contractBalanceIs(contractInitialBalance),
			},
		},
		"same tx/beneficiary is other account": {
			deployTx: func(opts *bind.TransactOpts, backend bind.ContractBackend, beneficiaryAddress common.Address) (common.Address, *types.Transaction, *selfdestruct.SelfDestruct, error) {
				opts.Value = big.NewInt(contractInitialBalance)
				return selfdestruct.DeploySelfDestruct(opts, backend,
					true,  // selfdestruct in constructor
					false, // beneficiary is not self
					beneficiaryAddress)
			},
			effects: map[string]effectFunction{
				"the current execution frame halts": executionHalted(),
				"code is deleted":                   contractCodeSizeIs(0),
				"storage is deleted":                contractStorageIs(0),
				"contract looses balance":           contractBalanceIs(0),
				"beneficiary gains balance":         beneficiaryBalanceIs(contractInitialBalance),
			},
		},
		"same tx/beneficiary is same account": {
			deployTx: func(opts *bind.TransactOpts, backend bind.ContractBackend, beneficiaryAddress common.Address) (common.Address, *types.Transaction, *selfdestruct.SelfDestruct, error) {
				opts.Value = big.NewInt(contractInitialBalance)
				return selfdestruct.DeploySelfDestruct(opts, backend,
					true,             // selfdestruct in constructor
					true,             // beneficiary is self
					common.Address{}) // ignored
			},
			effects: map[string]effectFunction{
				"the current execution frame halts": executionHalted(),
				"code is deleted":                   contractCodeSizeIs(0),
				"storage is deleted":                contractStorageIs(0),
				"balance is burned":                 contractBalanceIs(0),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			// New beneficiary address for each test
			beneficiaryAddress := common.Address{}
			rand.Read(beneficiaryAddress[:])

			// First transaction deploys contract
			contract, deployReceipt, err := DeployContract(net,
				func(to *bind.TransactOpts, cb bind.ContractBackend) (common.Address, *types.Transaction, *selfdestruct.SelfDestruct, error) {
					return test.deployTx(to, cb, beneficiaryAddress)
				})
			require.NoError(err)
			require.Equal(
				types.ReceiptStatusSuccessful,
				deployReceipt.Status,
				"failed to deploy contract",
			)
			allLogs := deployReceipt.Logs

			var executionReceipt *types.Receipt
			// Second transaction executes some contract function (if any)
			if test.executeTx != nil {
				executionReceipt, err = net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
					return test.executeTx(contract, opts, beneficiaryAddress)
				})
				require.NoError(err)
				require.Equal(
					types.ReceiptStatusSuccessful,
					executionReceipt.Status,
					"failed to execute contract",
				)
				allLogs = append(allLogs, deployReceipt.Logs...)
			}

			// create client to query the network about address properties
			client, err := net.GetClient()
			require.NoError(err)
			defer client.Close()

			// check effects
			effectContext := effectContext{
				client:             client,
				contract:           contract,
				executionReceipt:   executionReceipt,
				allLogs:            allLogs,
				contractAddress:    deployReceipt.ContractAddress,
				beneficiaryAddress: beneficiaryAddress,
			}
			for name, effect := range test.effects {
				t.Run(name, func(t *testing.T) {
					effect(require, &effectContext)
				})
			}
		})
	}
}

func testSelfDestruct_NestedCall(t *testing.T, net *IntegrationTestNet) {
	contractInitialBalance := int64(1234)

	tests := map[string]struct {
		transactions []executeTxFunction[selfdestruct.SelfDestructFactory]
		effects      map[string]effectFunction
	}{
		// This test checks the testing infrastructure itself
		"sanity check/no selfdestruct": {
			transactions: []executeTxFunction[selfdestruct.SelfDestructFactory]{
				func(factory *selfdestruct.SelfDestructFactory, opts *bind.TransactOpts, _ common.Address) (*types.Transaction, error) {
					opts.Value = big.NewInt(contractInitialBalance)
					return factory.Create(opts)
				},
			},
			effects: map[string]effectFunction{
				"contract has storage":         contractStorageIs(123),
				"contract has code":            contractCodeSizeIsNot(0),
				"contract keeps balance":       contractBalanceIs(contractInitialBalance),
				"beneficiary gains no balance": beneficiaryBalanceIs(0),
			},
		},
		"different tx/beneficiary is other account": {
			transactions: []executeTxFunction[selfdestruct.SelfDestructFactory]{
				func(factory *selfdestruct.SelfDestructFactory, opts *bind.TransactOpts, _ common.Address) (*types.Transaction, error) {
					opts.Value = big.NewInt(contractInitialBalance)
					return factory.Create(opts)
				},
				func(factory *selfdestruct.SelfDestructFactory, opts *bind.TransactOpts, beneficiaryAddress common.Address) (*types.Transaction, error) {
					return factory.Destroy(opts, beneficiaryAddress)
				},
			},
			effects: map[string]effectFunction{
				"storage is not deleted":       contractStorageIs(123),
				"code is not deleted":          contractCodeSizeIsNot(0),
				"contract looses balance":      contractBalanceIs(0),
				"beneficiary gains balance":    beneficiaryBalanceIs(contractInitialBalance),
				"nested execution frame halts": executionHalted(),
			},
		},
		"different tx/beneficiary is same account": {
			transactions: []executeTxFunction[selfdestruct.SelfDestructFactory]{
				func(factory *selfdestruct.SelfDestructFactory, opts *bind.TransactOpts, _ common.Address) (*types.Transaction, error) {
					opts.Value = big.NewInt(contractInitialBalance)
					return factory.Create(opts)
				},
				func(factory *selfdestruct.SelfDestructFactory, opts *bind.TransactOpts, _ common.Address) (*types.Transaction, error) {
					return factory.DestroyWithoutBeneficiary(opts)
				},
			},
			effects: map[string]effectFunction{
				"storage is not deleted":       contractStorageIs(123),
				"code is not deleted":          contractCodeSizeIsNot(0),
				"balance is not burned":        contractBalanceIs(contractInitialBalance),
				"nested execution frame halts": executionHalted(),
			},
		},
		"same tx/beneficiary is other account": {
			transactions: []executeTxFunction[selfdestruct.SelfDestructFactory]{
				func(factory *selfdestruct.SelfDestructFactory, opts *bind.TransactOpts, beneficiaryAddress common.Address) (*types.Transaction, error) {
					opts.Value = big.NewInt(contractInitialBalance)
					return factory.CreateAndDestroy(opts, beneficiaryAddress)
				},
			},
			effects: map[string]effectFunction{
				"nested execution frame halts":   executionHalted(),
				"code is deleted":                contractCodeSizeIs(0),
				"storage is deleted":             contractStorageIs(0),
				"contract looses balance":        contractBalanceIs(0),
				"beneficiary gains balance":      beneficiaryBalanceIs(contractInitialBalance),
				"storage exists until end of tx": nestedContractValueAfterSelfDestructIs(123),
			},
		},
		"same tx/beneficiary is same account": {
			transactions: []executeTxFunction[selfdestruct.SelfDestructFactory]{
				func(factory *selfdestruct.SelfDestructFactory, opts *bind.TransactOpts, _ common.Address) (*types.Transaction, error) {
					opts.Value = big.NewInt(contractInitialBalance)
					return factory.CreateAndDestroyWithoutBeneficiary(opts)
				},
			},
			effects: map[string]effectFunction{
				"nested execution frame halts":   executionHalted(),
				"code is deleted":                contractCodeSizeIs(0),
				"storage is deleted":             contractStorageIs(0),
				"contract looses balance":        contractBalanceIs(0),
				"storage exists until end of tx": nestedContractValueAfterSelfDestructIs(123),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			// generate a new beneficiary address for each test
			beneficiaryAddress := common.Address{}
			rand.Read(beneficiaryAddress[:])

			// deploy factory contract
			factory, receipt, err := DeployContract(net, selfdestruct.DeploySelfDestructFactory)
			require.NoError(err)
			require.Equal(
				types.ReceiptStatusSuccessful,
				receipt.Status,
				"failed to deploy contract",
			)

			allLogs := []*types.Log{}

			// execute all described transactions
			for _, tx := range test.transactions {
				receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
					return tx(factory, opts, beneficiaryAddress)
				})
				require.NoError(err)
				allLogs = append(allLogs, receipt.Logs...)
			}
			var contractAddress common.Address
			for _, log := range allLogs {
				parse, err := factory.ParseLogDeployed(*log)
				if err == nil {
					contractAddress = parse.Addr
					break
				}
			}
			require.NotEqual(contractAddress,
				common.Address{},
				"factory failed to construct a contract",
			)

			// create client to query the network about address properties
			client, err := net.GetClient()
			require.NoError(err)
			defer client.Close()
			newContract, err := selfdestruct.NewSelfDestruct(contractAddress, client)
			require.NoError(err, "failed to instantiate contract")

			// check effects
			effectContext := effectContext{
				client:             client,
				factory:            factory,
				contract:           newContract,
				allLogs:            allLogs,
				contractAddress:    contractAddress,
				beneficiaryAddress: beneficiaryAddress,
			}
			for name, effect := range test.effects {
				t.Run(name, func(t *testing.T) {
					effect(require, &effectContext)
				})
			}
		})
	}
}

// effectContext stores all the information needed to check the effects of a test
// Each test will construct a contract that may selfdestruct.
// Each test will deploy a new contract and generate a new beneficiary account to
// avoid clashes between tests. This struct will be filled with the results of
// the each test setup.
type effectContext struct {
	client             *ethclient.Client                 //< client to interact with the network
	contract           *selfdestruct.SelfDestruct        //< contract to test (may have selfdestructed)
	factory            *selfdestruct.SelfDestructFactory //< factory contract to deploy new contracts
	executionReceipt   *types.Receipt                    //< receipt of the execution transaction for constructor tests
	allLogs            []*types.Log                      //< all logs generated by all transactions
	contractAddress    common.Address                    //< address of the contract that may selfdestruct
	beneficiaryAddress common.Address                    //< address of the beneficiary account
}

type effectFunction func(require *require.Assertions, ctx *effectContext)
type deployTxFunction[T any] func(opts *bind.TransactOpts, backend bind.ContractBackend, beneficiaryAddress common.Address) (common.Address, *types.Transaction, *T, error)
type executeTxFunction[T any] func(contract *T, opts *bind.TransactOpts, beneficiaryAddress common.Address) (*types.Transaction, error)

// executionHalted checks that the execution stopped after selfdestruct
// This is done by looking for logs
func executionHalted() effectFunction {
	return func(require *require.Assertions, ctx *effectContext) {
		for _, log := range ctx.allLogs {
			_, err := ctx.contract.ParseLogAfterDestruct(*log)
			require.Error(
				err,
				"execution should have halted, log after selfdestruct should not be present",
			)
		}
	}
}

// nestedContractValueAfterSelfDestructIs reads the internal storage of the contract
// and compares its content to the expected value
// use this matcher when the contract is being destroyed by a nested call, but
// the internal value of the contract storage will be emitted before the
// transaction is  completed.
func nestedContractValueAfterSelfDestructIs(value int64) effectFunction {
	return func(require *require.Assertions, ctx *effectContext) {
		for _, log := range ctx.allLogs {
			storage, err := ctx.factory.ParseLogContractStorage(*log)
			if err != nil {
				continue
			}
			require.Equal(
				value,
				storage.Value.Int64(),
				"storage value differs",
			)
			return
		}
		require.Fail("no log with storage value found")
	}
}

// contractBalanceIs reads the contract balance and compare it to the expected value
func contractBalanceIs(expected int64) effectFunction {
	return func(require *require.Assertions, ctx *effectContext) {
		balance, err := ctx.client.BalanceAt(context.Background(), ctx.contractAddress, nil)
		require.NoError(err)
		require.Equal(
			expected,
			balance.Int64(),
			"balance not expected",
		)
	}
}

// contractStorageIs reads the contract storage and compare it to the expected value
func contractStorageIs(expected int64) effectFunction {
	return func(require *require.Assertions, ctx *effectContext) {
		data, err := ctx.client.StorageAt(context.Background(), ctx.contractAddress, common.Hash{}, nil)
		require.NoError(err)
		storage := new(big.Int).SetBytes(data)
		require.Equal(
			expected,
			storage.Int64(),
			"storage value differs",
		)
	}
}

func beneficiaryBalanceIs(expected int64) effectFunction {
	return func(require *require.Assertions, ctx *effectContext) {
		balance, err := ctx.client.BalanceAt(context.Background(), ctx.beneficiaryAddress, nil)
		require.NoError(err)
		require.Equal(
			expected,
			balance.Int64(),
			"balance not expected",
		)
	}
}

func contractCodeSizeIs(expected int) effectFunction {
	return func(require *require.Assertions, ctx *effectContext) {
		code, err := ctx.client.CodeAt(context.Background(), ctx.contractAddress, nil)
		require.NoError(err)
		require.Equal(
			expected,
			len(code),
			"code size not expected",
		)
	}
}

func contractCodeSizeIsNot(notExpected int) effectFunction {
	return func(require *require.Assertions, ctx *effectContext) {
		code, err := ctx.client.CodeAt(context.Background(), ctx.contractAddress, nil)
		require.NoError(err)
		require.NotEqual(
			notExpected,
			len(code),
			"code size not expected",
		)
	}
}
