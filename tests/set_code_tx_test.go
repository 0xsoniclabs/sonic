package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/batch"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/0xsoniclabs/sonic/tests/contracts/privilege_deescalation"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetCodeTransaction tests the SetCode transaction type use cases
// described in the EIP-7702 specification: https://eips.ethereum.org/EIPS/eip-7702
// Notice that the test contracts used in this test model the expected behavior
// and do not implement ERC-20 as described in the EIP use case examples.
func TestSetCodeTransaction(t *testing.T) {

	net, err := StartIntegrationTestNet(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to start the fake network: %v", err)
	}
	defer net.Stop()

	t.Run("Operation", func(t *testing.T) {
		// operation tests check basic operation of the SetCode transaction

		t.Run("Delegate can be set and unset", func(t *testing.T) {
			testDelegateCanBeSetAndUnset(t, net)
		})

		t.Run("Invalid authorizations are ignored", func(t *testing.T) {
			testInvalidAuthorizationsAreIgnored(t, net)
		})

		t.Run("Authorizations are executed in order", func(t *testing.T) {
			testAuthorizationsAreExecutedInOrder(t, net)
		})
	})

	t.Run("UseCase", func(t *testing.T) {
		// UseCase tests check the use cases described in the EIP-7702 specification

		t.Run("Transaction Sponsoring", func(t *testing.T) {
			testSponsoring(t, net)
		})

		t.Run("Transaction Batching", func(t *testing.T) {
			testBatching(t, net)
		})

		t.Run("Privilege Deescalation", func(t *testing.T) {
			testPrivilegeDeescalation(t, net)
		})

	})
}

func testSponsoring(t *testing.T, net *IntegrationTestNet) {

	// This test executes a transaction in behalf of another account:
	// - The sponsor account pays for the gas for the transaction
	// - The sponsored account is the context of the transaction, and its state is modified
	// - The delegate account is the contract that will be executed

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// sponsor issues the SetCode transaction and pays for it
	sponsor := makeAccountWithBalance(t, net, 1e18)
	// sponsored is used as context for the call, its state will be modified
	// without paying for the transaction
	sponsored := makeAccountWithBalance(t, net, 0) // < no funds

	// Deploy the a contract to use as delegate
	counter, receipt, err := DeployContract(net, counter.DeployCounter)
	require.NoError(t, err)
	delegate := receipt.ContractAddress

	// Extract the call data of a normal call to the delegate contract
	// to know the ABI encoding of the callData
	callData := getCallData(t, net, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return counter.IncrementCounter(opts)
	})

	// Create a setCode transaction calling the incrementCounter function
	// in the context of the sponsored account.
	setCodeTx := makeEip7702Transaction(t, client, sponsor, sponsored, delegate, callData)
	receipt, err = net.Run(setCodeTx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Check that the sender has paid for the transaction
	effectiveCost := new(big.Int)
	effectiveCost = effectiveCost.Mul(
		receipt.EffectiveGasPrice,
		big.NewInt(int64(receipt.GasUsed)))

	balance, err := client.BalanceAt(context.Background(), sponsor.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t,
		new(big.Int).Sub(
			big.NewInt(1e18), effectiveCost), balance)

	// Read code at sponsored address, must contain the delegate address
	code, err := client.CodeAt(context.Background(), sponsored.Address(), nil)
	require.NoError(t, err)
	expectedCode := append([]byte{0xef, 0x01, 0x00}, delegate[:]...)
	require.Equal(t, expectedCode, code, "code in account is expected to be delegation designation")

	// Read storage at sponsored address (instead of contract address as in a normal tx)
	// counter must exist and be 1
	data, err := client.StorageAt(context.Background(), sponsored.Address(), common.Hash{}, nil)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(1), new(big.Int).SetBytes(data), "unexpected storage value")
}

func testBatching(t *testing.T, net *IntegrationTestNet) {

	// This test executes multiple funds transfers within a single transaction:
	// - The sponsor and sponsored accounts are the same, this is a self-sponsored transaction.
	// - The delegate account is the contract that will be executed, which implements the batch of calls
	// - Multiple receiver accounts will receive the funds

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// sender account batches multiple transfers of funds in a single transaction
	// receivers will receive the funds
	sender := makeAccountWithBalance(t, net, 1e18)
	receiver1 := makeAccountWithBalance(t, net, 0)
	receiver2 := makeAccountWithBalance(t, net, 0)

	batchContract, deployReceipt, err := DeployContract(net, batch.DeployBatch)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, deployReceipt.Status)
	batchContractAddress := deployReceipt.ContractAddress

	// Extract the call data of a normal call to the delegate contract
	// to know the ABI encoding of the callData.
	// This code creates the Batch of calls, which the batch contract will execute
	callData := getCallData(t, net, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return batchContract.Execute(opts, []batch.BatchCallDelegationCall{
			{
				To:    receiver1.Address(),
				Value: big.NewInt(1234),
			},
			{
				To:    receiver2.Address(),
				Value: big.NewInt(4321),
			},
		})
	})

	// Send a SetCode transaction to the batch contract
	tx := makeEip7702Transaction(t, client, sender, sender, batchContractAddress, callData)
	batchReceipt, err := net.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, batchReceipt.Status)

	// Check that the sender has paid for the transaction
	effectiveCost := new(big.Int)
	effectiveCost = effectiveCost.Mul(
		batchReceipt.EffectiveGasPrice,
		big.NewInt(int64(batchReceipt.GasUsed)))
	effectiveCost = effectiveCost.Add(effectiveCost, big.NewInt(1234+4321))

	balance, err := client.BalanceAt(context.Background(), sender.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t,
		new(big.Int).Sub(
			big.NewInt(1e18), effectiveCost), balance)

	// Check that the receivers have received the funds
	balance1, err := client.BalanceAt(context.Background(), receiver1.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(1234), balance1)

	balance2, err := client.BalanceAt(context.Background(), receiver2.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(4321), balance2)
}

func testPrivilegeDeescalation(t *testing.T, net *IntegrationTestNet) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// This test executes a transaction in behalf of another account, using
	// the privilege deescalation pattern:
	// - Account A allows account B to execute certain operations on its behalf
	// - Account B (userAccount) pays for the gas for the transaction
	// - Account A (account) is the context of the transaction, and its state is modified
	// - Some part of the contract interface (DoPayment) is executable from account B
	account := makeAccountWithBalance(t, net, 1e18)     // < will transfer funds
	userAccount := makeAccountWithBalance(t, net, 1e18) // < will pay for gas
	receiver := makeAccountWithBalance(t, net, 0)

	// Deploy the a contract to use as delegate
	contract, receipt, err := DeployContract(net, privilege_deescalation.DeployPrivilegeDeescalation)
	require.NoError(t, err)
	delegate := receipt.ContractAddress

	// Install delegation in account and allow access by userAccount
	callData := getCallData(t, net, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return contract.AllowPayment(opts, userAccount.Address())
	})
	setCodeTx := makeEip7702Transaction(t, client, account, account, delegate, callData)
	receipt, err = net.Run(setCodeTx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Check that authorization has been set
	data, err := client.StorageAt(context.Background(), account.Address(), common.Hash{}, nil)
	require.NoError(t, err)
	addr := userAccount.Address()
	require.Equal(t, addr[:], data[12:32], "contract has not been initialized correctly")

	// "mount" the contract in the address of the delegating account
	delegatedContract, err := privilege_deescalation.NewPrivilegeDeescalation(account.Address(), client)
	require.NoError(t, err)

	accountBalanceBefore, err := client.BalanceAt(context.Background(), account.Address(), nil)
	require.NoError(t, err)

	// issue a normal transaction from userAccount to transfer funds to receiver
	txOpts, err := net.GetTransactOptions(userAccount)
	require.NoError(t, err)
	txOpts.NoSend = true
	tx, err := delegatedContract.DoPayment(txOpts, receiver.Address(), big.NewInt(42))
	require.NoError(t, err)
	receipt, err = net.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// check balances
	accountBalanceAfter, err := client.BalanceAt(context.Background(), account.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, new(big.Int).Sub(accountBalanceBefore, big.NewInt(42)), accountBalanceAfter)

	receivedBalance, err := client.BalanceAt(context.Background(), receiver.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(42), receivedBalance)

	// issue a transaction from and unauthorized account
	unauthorizedAccount := makeAccountWithBalance(t, net, 1e18)
	txOpts, err = net.GetTransactOptions(unauthorizedAccount)
	require.NoError(t, err)
	txOpts.NoSend = true
	tx, err = delegatedContract.AllowPayment(txOpts, receiver.Address())
	require.NoError(t, err)
	receipt, err = net.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status, "tx shall be executed and rejected")

	txOpts, err = net.GetTransactOptions(unauthorizedAccount)
	require.NoError(t, err)
	txOpts.NoSend = true
	tx, err = delegatedContract.DoPayment(txOpts, receiver.Address(), big.NewInt(42))
	require.NoError(t, err)
	receipt, err = net.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status, "tx shall be executed and rejected")
}

func testDelegateCanBeSetAndUnset(t *testing.T, net *IntegrationTestNet) {
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	account := makeAccountWithBalance(t, net, 1e18)

	// Deploy the a contract to use as delegate
	counter, receipt, err := DeployContract(net, counter.DeployCounter)
	require.NoError(t, err)
	delegateAddress := receipt.ContractAddress

	// set delegation
	callData := getCallData(t, net, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return counter.IncrementCounter(opts)
	})
	setCodeTx := makeEip7702Transaction(t, client, account, account, delegateAddress, callData)
	receipt, err = net.Run(setCodeTx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// check that the code has been set
	codeSet, err := client.CodeAt(context.Background(), account.Address(), nil)
	require.NoError(t, err)
	expectedCode := append([]byte{0xef, 0x01, 0x00}, delegateAddress[:]...)
	require.Equal(t, expectedCode, codeSet, "code in account is expected to be delegation designation")

	// unset by delegating to an empty address
	unsetCodeTx := makeEip7702Transaction(t, client, account, account, common.Address{}, []byte{})
	receipt, err = net.Run(unsetCodeTx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// check that the code has been unset
	codeUnset, err := client.CodeAt(context.Background(), account.Address(), nil)
	require.NoError(t, err)
	require.Equal(t, []byte{}, codeUnset, "code in account is expected to be empty")
}

func testInvalidAuthorizationsAreIgnored(t *testing.T, net *IntegrationTestNet) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId, err := client.ChainID(context.Background())
	require.NoError(t, err, "failed to get chain ID")

	tests := map[string]struct {
		makeAuthorization func(nonce uint64) types.SetCodeAuthorization
	}{
		"authorization nonce too low": {
			makeAuthorization: func(nonce uint64) types.SetCodeAuthorization {
				return types.SetCodeAuthorization{
					ChainID: *uint256.MustFromBig(chainId),
					Address: common.Address{42},
					// for self-sponsored transactions,
					// the correct nonce would be current nonce + 1
					Nonce: nonce,
				}
			},
		},
		"authorization nonce too high": {
			makeAuthorization: func(nonce uint64) types.SetCodeAuthorization {
				return types.SetCodeAuthorization{
					ChainID: *uint256.MustFromBig(chainId),
					Address: common.Address{42},
					// for self-sponsored transactions,
					// the correct nonce would be current nonce + 1
					Nonce: nonce + 2,
				}
			},
		},
		"wrong chain id": {
			makeAuthorization: func(nonce uint64) types.SetCodeAuthorization {
				return types.SetCodeAuthorization{
					ChainID: *uint256.NewInt(0xDeffec8),
					Address: common.Address{42},
					Nonce:   nonce + 1,
				}
			},
		},
	}

	// for each of the invalid authorizations, we test the following scenarios:
	scenarios := map[string]struct {
		makeAuthorizations func(wrong types.SetCodeAuthorization, nonce uint64) []types.SetCodeAuthorization
		check              func(t *testing.T, codes []byte)
	}{
		"single wrong authorization": {
			makeAuthorizations: func(wrong types.SetCodeAuthorization, _ uint64) []types.SetCodeAuthorization {
				return []types.SetCodeAuthorization{wrong}
			},
			check: func(t *testing.T, code []byte) {
				require.Equal(t, []byte{}, code, "code in account is expected to be empty")
			},
		},
		"before correct authorization": {
			makeAuthorizations: func(wrong types.SetCodeAuthorization, nonce uint64) []types.SetCodeAuthorization {
				return []types.SetCodeAuthorization{
					wrong,
					types.SetCodeAuthorization{
						ChainID: *uint256.MustFromBig(chainId),
						Address: common.Address{42},
						Nonce:   nonce + 1,
					},
				}
			},
			check: func(t *testing.T, code []byte) {
				expectedCode := append([]byte{0xef, 0x01, 0x00}, common.Address{42}.Bytes()...)
				require.Equal(t, expectedCode, code, "code in account is expected to be delegation designation")
			},
		},
		"after correct authorization": {
			makeAuthorizations: func(wrong types.SetCodeAuthorization, nonce uint64) []types.SetCodeAuthorization {
				return []types.SetCodeAuthorization{
					types.SetCodeAuthorization{
						ChainID: *uint256.MustFromBig(chainId),
						Address: common.Address{42},
						Nonce:   nonce + 1,
					},
					wrong,
				}
			},
			check: func(t *testing.T, code []byte) {
				expectedCode := append([]byte{0xef, 0x01, 0x00}, common.Address{42}.Bytes()...)
				require.Equal(t, expectedCode, code, "code in account is expected to be delegation designation")
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for name, scenario := range scenarios {
				t.Run(name, func(t *testing.T) {

					account := makeAccountWithBalance(t, net, 1e18) // < will transfer funds

					nonce, err := client.NonceAt(context.Background(), account.Address(), nil)
					require.NoError(t, err, "failed to get nonce for account", account.Address())

					wrongAuthorization := test.makeAuthorization(nonce)
					require.NoError(t, err, "failed to sign SetCode authorization")
					authorizations := scenario.makeAuthorizations(wrongAuthorization, nonce)

					signedAuthorizations := make([]types.SetCodeAuthorization, 0, len(authorizations))
					for _, auth := range authorizations {
						signed, err := types.SignSetCode(account.PrivateKey, auth)
						require.NoError(t, err, "failed to sign SetCode authorization")
						signedAuthorizations = append(signedAuthorizations, signed)
					}

					tx, err := types.SignTx(
						types.NewTx(&types.SetCodeTx{
							ChainID:   uint256.MustFromBig(chainId),
							Nonce:     nonce,
							To:        account.Address(),
							Gas:       150_000,
							GasFeeCap: uint256.NewInt(10e10),
							AuthList:  signedAuthorizations,
						}),
						types.NewPragueSigner(chainId),
						account.PrivateKey,
					)
					require.NoError(t, err, "failed to create transaction")

					// execute transaction
					receipt, err := net.Run(tx)
					require.NoError(t, err)
					// because no delegation is set, transaction call to self will succeed
					require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

					// delegation is not set
					code, err := client.CodeAt(context.Background(), account.Address(), nil)
					require.NoError(t, err)
					scenario.check(t, code)
				})
			}
		})
	}
}

func testAuthorizationsAreExecutedInOrder(t *testing.T, net *IntegrationTestNet) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId, err := client.ChainID(context.Background())
	require.NoError(t, err, "failed to get chain ID")

	account := makeAccountWithBalance(t, net, 1e18) // < will transfer funds

	nonce, err := client.NonceAt(context.Background(), account.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", account.Address())

	authorizationA, err := types.SignSetCode(account.PrivateKey,
		types.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(chainId),
			Address: common.Address{42},
			Nonce:   nonce + 1,
		})
	require.NoError(t, err, "failed to sign SetCode authorization")
	authorizationB, err := types.SignSetCode(account.PrivateKey,
		types.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(chainId),
			Address: common.Address{24},
			Nonce:   nonce + 2,
		})
	require.NoError(t, err, "failed to sign SetCode authorization")

	tx, err := types.SignTx(
		types.NewTx(&types.SetCodeTx{
			ChainID:   uint256.MustFromBig(chainId),
			Nonce:     nonce,
			To:        account.Address(),
			Gas:       150_000,
			GasFeeCap: uint256.NewInt(10e10),
			AuthList: []types.SetCodeAuthorization{
				authorizationA,
				authorizationB,
			},
		}),
		types.NewPragueSigner(chainId),
		account.PrivateKey,
	)
	require.NoError(t, err, "failed to create transaction")

	// execute transaction
	receipt, err := net.Run(tx)
	require.NoError(t, err)
	// because no delegation is set, transaction call to self will succeed
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// delegation is not set
	code, err := client.CodeAt(context.Background(), account.Address(), nil)
	require.NoError(t, err)
	expectedCode := append([]byte{0xef, 0x01, 0x00}, common.Address{24}.Bytes()...)
	require.Equal(t, expectedCode, code, "code in account is expected to be delegation designation")
}

// makeEip7702Transaction creates a legacy transaction from a CallMsg, filling in the nonce
// and gas limit.
func makeEip7702Transaction(t *testing.T,
	client *ethclient.Client,
	sponsor *Account, // signs and pays for the tx
	sponsored *Account, // the account where the delegator will be written in
	delegate common.Address, // the address of the delegate contract
	callData []byte,
) *types.Transaction {
	t.Helper()

	chainId, err := client.ChainID(context.Background())
	require.NoError(t, err, "failed to get chain ID")

	sponsoredNonce, err := client.NonceAt(context.Background(), sponsored.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", sponsored.Address())

	sponsorNonce, err := client.NonceAt(context.Background(), sponsor.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", sponsor.Address())

	// If self sponsored, there are two nonces values to take care of, the transaction
	// nonce and the authorization nonce. The authorization nonce is checked after
	// the transaction has incremented nonce. Therefore, the authorization nonce
	// needs to be 1 higher than the transaction nonce.
	nonceIncrement := uint64(0)
	if sponsor == sponsored {
		nonceIncrement = 1
	}

	authorization, err := types.SignSetCode(sponsored.PrivateKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(chainId),
		Address: delegate,
		Nonce:   sponsoredNonce + nonceIncrement,
	})
	require.NoError(t, err, "failed to sign SetCode authorization")

	tx := types.NewTx(&types.SetCodeTx{
		ChainID:   uint256.MustFromBig(chainId),
		Nonce:     sponsorNonce,
		To:        sponsored.Address(),
		Gas:       150_000,
		GasFeeCap: uint256.NewInt(10e10),
		AuthList: []types.SetCodeAuthorization{
			authorization,
		},
		Data: callData,
	})

	signer := types.NewPragueSigner(chainId)
	tx, err = types.SignTx(tx, signer, sponsor.PrivateKey)
	require.NoError(t, err, "failed to sign transaction")
	return tx
}

// getCallData creates a transaction and returns the data field of the transaction.
// This function can be used to retrieve the ABI encoding of a the call data, and
// use such encoding to create a SetCode transaction.
func getCallData(t *testing.T, net *IntegrationTestNet,
	transactionConstructor func(*bind.TransactOpts) (*types.Transaction, error)) []byte {
	txOpts, err := net.GetTransactOptions(&net.validator)
	require.NoError(t, err)
	txOpts.NoSend = true // <- create the transaction to read callData, but do not send it.
	tx, err := transactionConstructor(txOpts)
	require.NoError(t, err)
	return tx.Data()
}
