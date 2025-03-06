package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/batch"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/0xsoniclabs/sonic/tests/contracts/privilege_deescalation"
	"github.com/0xsoniclabs/sonic/tests/contracts/transitive_call"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetCodeTransaction tests the SetCode transaction introduced with Prague
// see: https://eips.ethereum.org/EIPS/eip-7702
//
// the test has the following structure:
// - Operation tests check basic operation of the SetCode transaction
// - UseCase tests check the use cases described in the EIP-7702 specification
//   - Transaction Sponsoring
//   - Transaction Batching
//   - Privilege Deescalation
//
// Notice that the test contracts used in this test model the expected behavior
// and do not implement ERC-20 as described in the EIP use case examples.
func TestSetCodeTransaction(t *testing.T) {

	net := StartIntegrationTestNet(t)

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

		t.Run("Multiple accounts can submit authorizations", func(t *testing.T) {
			testMultipleAccountsCanSubmitAuthorizations(t, net)
		})

		t.Run("Authorization succeeds with failing tx", func(t *testing.T) {
			testAuthorizationSucceedsWithFailingTx(t, net)
		})

		t.Run("Authorization can be issued from a non existing account", func(t *testing.T) {
			testAuthorizationFromNonExistingAccount(t, net)
		})

		t.Run("Delegations cannot be transitive", func(t *testing.T) {
			testNoDelegateToDelegated(t, net)
		})

		t.Run("Delegations can trigger chains of calls", func(t *testing.T) {
			testChainOfCalls(t, net)
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

// testSponsoring executes a transaction in behalf of another account:
// - The sponsor account pays for the gas for the transaction
// - The sponsored account is the context of the transaction, and its state is modified
// - The delegate account is the contract that will be executed
func testSponsoring(t *testing.T, net *IntegrationTestNet) {

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

// testBatching executes multiple funds transfers within a single transaction:
// - The sponsor and sponsored accounts are the same, this is a self-sponsored transaction.
// - The delegate account is the contract that will be executed, which implements the batch of calls
// - Multiple receiver accounts will receive the funds
func testBatching(t *testing.T, net *IntegrationTestNet) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// sender account batches multiple transfers of funds in a single transaction
	// receivers will receive the funds
	sender := makeAccountWithBalance(t, net, 1e18) // < pays transaction and transfers funds
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

	// Check that the sender has paid for the transaction and transfers
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

// testPrivilegeDeescalation executes a transaction where an account allows restricted access
// to its internal state to a second account.
func testPrivilegeDeescalation(t *testing.T, net *IntegrationTestNet) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// - Account A allows account B to execute certain operations on its behalf
	// - Account A (account) is the context of the transaction, and its state is modified
	// - Account B (userAccount) pays for the gas for the transaction
	// - Some part of the contract interface (DoPayment) is executable from account B
	account := makeAccountWithBalance(t, net, 1e18)     // < will transfer funds
	userAccount := makeAccountWithBalance(t, net, 1e18) // < will pay for gas
	receiver := makeAccountWithBalance(t, net, 0)       // < will receive funds

	// Deploy the a contract to use as delegate
	contract, receipt, err := DeployContract(net, privilege_deescalation.DeployPrivilegeDeescalation)
	require.NoError(t, err)
	delegate := receipt.ContractAddress

	// Install delegation in account and allow access by userAccount
	callData := getCallData(t, net, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		// The contract will whitelist the userAccount to execute the DoPayment function
		// within the same transaction that sets the delegation.
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
	tx, err := delegatedContract.DoPayment(txOpts, receiver.Address(), big.NewInt(1234))
	require.NoError(t, err)
	receipt, err = net.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// check balances
	accountBalanceAfter, err := client.BalanceAt(context.Background(), account.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, new(big.Int).Sub(accountBalanceBefore, big.NewInt(1234)), accountBalanceAfter)

	receivedBalance, err := client.BalanceAt(context.Background(), receiver.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(1234), receivedBalance)

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

// testDelegateCanBeSetAndUnset checks that a delegate can be set and unset
// The EIP-7702 specification describes the method to restore an EOA code to
// its original state by setting the delegate to the zero address.
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

	// check that code has been set
	codeSet, err := client.CodeAt(context.Background(), account.Address(), nil)
	require.NoError(t, err)
	expectedCode := append([]byte{0xef, 0x01, 0x00}, delegateAddress[:]...)
	require.Equal(t, expectedCode, codeSet, "code in account is expected to be delegation designation")

	// unset by delegating to an empty address
	unsetCodeTx := makeEip7702Transaction(t, client, account, account, common.Address{}, []byte{})
	receipt, err = net.Run(unsetCodeTx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// check that code has been unset
	codeUnset, err := client.CodeAt(context.Background(), account.Address(), nil)
	require.NoError(t, err)
	require.Equal(t, []byte{}, codeUnset, "code in account is expected to be empty")
}

// testInvalidAuthorizationsAreIgnored checks that invalid authorizations are ignored
// whilst the transaction is still executed.
func testInvalidAuthorizationsAreIgnored(t *testing.T, net *IntegrationTestNet) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId, err := client.ChainID(context.Background())
	require.NoError(t, err, "failed to get chain ID")

	// list invalid authorizations
	wrongAuthorizations := map[string]struct {
		makeAuthorization func(authority *Account, nonce uint64) (types.SetCodeAuthorization, error)
	}{
		"authorization nonce too low": {
			makeAuthorization: func(authority *Account, nonce uint64) (types.SetCodeAuthorization, error) {
				return types.SignSetCode(NewAccount().PrivateKey, types.SetCodeAuthorization{
					ChainID: *uint256.MustFromBig(chainId),
					Address: common.Address{42},
					// for self-sponsored transactions,
					// the correct nonce would be current nonce + 1
					Nonce: nonce,
				})
			},
		},
		"authorization nonce too high": {
			makeAuthorization: func(authority *Account, nonce uint64) (types.SetCodeAuthorization, error) {
				return types.SignSetCode(NewAccount().PrivateKey, types.SetCodeAuthorization{
					ChainID: *uint256.MustFromBig(chainId),
					Address: common.Address{42},
					// for self-sponsored transactions,
					// the correct nonce would be current nonce + 1
					Nonce: nonce + 2,
				})
			},
		},
		"wrong chain id": {
			makeAuthorization: func(authority *Account, nonce uint64) (types.SetCodeAuthorization, error) {
				return types.SignSetCode(NewAccount().PrivateKey, types.SetCodeAuthorization{
					ChainID: *uint256.NewInt(0xDeffec8),
					Address: common.Address{42},
					Nonce:   nonce + 1,
				})
			},
		},
		"invalid signature": {
			makeAuthorization: func(authority *Account, nonce uint64) (types.SetCodeAuthorization, error) {
				return types.SetCodeAuthorization{
					ChainID: *uint256.MustFromBig(chainId),
					Address: common.Address{42},
					Nonce:   nonce + 1,
					// signature defaulted
				}, nil
			},
		},
	}

	// for each of the invalid authorization, the following scenarios are tested:
	scenarios := map[string]struct {
		makeAuthorizations func(t *testing.T, wrong types.SetCodeAuthorization, rightAccount *Account) []types.SetCodeAuthorization
		check              func(t *testing.T, wrongAccount, rightAccount *Account)
	}{
		"single wrong authorization": {
			makeAuthorizations: func(t *testing.T, wrong types.SetCodeAuthorization, rightAccount *Account) []types.SetCodeAuthorization {
				return []types.SetCodeAuthorization{wrong}
			},
			check: func(t *testing.T, wrongAccount, _ *Account) {
				code, err := client.CodeAt(context.Background(), wrongAccount.Address(), nil)
				require.NoError(t, err)
				require.Equal(t, []byte{}, code, "code in account is expected to be unmodified")
			},
		},
		"before correct authorization": {
			makeAuthorizations: func(t *testing.T, wrong types.SetCodeAuthorization, rightAccount *Account) []types.SetCodeAuthorization {
				nonce, err := client.NonceAt(context.Background(), rightAccount.Address(), nil)
				require.NoError(t, err, "failed to get nonce for account", rightAccount.Address())

				valid, err := types.SignSetCode(rightAccount.PrivateKey,
					types.SetCodeAuthorization{
						ChainID: *uint256.MustFromBig(chainId),
						Address: common.Address{43},
						Nonce:   nonce,
					})
				require.NoError(t, err, "failed to sign SetCode authorization")
				return []types.SetCodeAuthorization{wrong, valid}
			},
			check: func(t *testing.T, wrongAccount, rightAccount *Account) {
				code, err := client.CodeAt(context.Background(), wrongAccount.Address(), nil)
				require.NoError(t, err)
				require.Equal(t, []byte{}, code, "code in account is expected to be unmodified")

				code, err = client.CodeAt(context.Background(), rightAccount.Address(), nil)
				require.NoError(t, err)
				expectedCode := append([]byte{0xef, 0x01, 0x00}, common.Address{43}.Bytes()...)
				require.Equal(t, expectedCode, code, "code in account is expected to be delegation designation")
			},
		},
		"after correct authorization": {
			makeAuthorizations: func(t *testing.T, wrong types.SetCodeAuthorization, rightAccount *Account) []types.SetCodeAuthorization {
				nonce, err := client.NonceAt(context.Background(), rightAccount.Address(), nil)
				require.NoError(t, err, "failed to get nonce for account", rightAccount.Address())

				valid, err := types.SignSetCode(rightAccount.PrivateKey,
					types.SetCodeAuthorization{
						ChainID: *uint256.MustFromBig(chainId),
						Address: common.Address{43},
						Nonce:   nonce,
					})
				require.NoError(t, err, "failed to sign SetCode authorization")
				return []types.SetCodeAuthorization{valid, wrong}
			},
			check: func(t *testing.T, wrongAccount, rightAccount *Account) {
				code, err := client.CodeAt(context.Background(), wrongAccount.Address(), nil)
				require.NoError(t, err)
				require.Equal(t, []byte{}, code, "code in account is expected to be unmodified")

				code, err = client.CodeAt(context.Background(), rightAccount.Address(), nil)
				require.NoError(t, err)
				expectedCode := append([]byte{0xef, 0x01, 0x00}, common.Address{43}.Bytes()...)
				require.Equal(t, expectedCode, code, "code in account is expected to be delegation designation")
			},
		},
	}

	for name, test := range wrongAuthorizations {
		t.Run(name, func(t *testing.T) {
			for name, scenario := range scenarios {
				t.Run(name, func(t *testing.T) {

					wrongAuthAccount := makeAccountWithBalance(t, net, 1e18) // < will transfer funds
					nonce, err := client.NonceAt(context.Background(), wrongAuthAccount.Address(), nil)
					require.NoError(t, err, "failed to get nonce for account", wrongAuthAccount.Address())

					wrongAuthorization, err := test.makeAuthorization(wrongAuthAccount, nonce)
					require.NoError(t, err, "failed to sign SetCode authorization")

					rightAuthAccount := makeAccountWithBalance(t, net, 1e18) // < will transfer funds
					authorizations := scenario.makeAuthorizations(t, wrongAuthorization, rightAuthAccount)

					tx, err := types.SignTx(
						types.NewTx(&types.SetCodeTx{
							ChainID:   uint256.MustFromBig(chainId),
							Nonce:     nonce,
							To:        wrongAuthAccount.Address(),
							Gas:       150_000,
							GasFeeCap: uint256.NewInt(10e10),
							AuthList:  authorizations,
						}),
						types.NewPragueSigner(chainId),
						wrongAuthAccount.PrivateKey,
					)
					require.NoError(t, err, "failed to create transaction")

					// execute transaction
					receipt, err := net.Run(tx)
					require.NoError(t, err)
					// because no delegation is set, transaction call to self (no code) will succeed
					require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

					// check delegations
					scenario.check(t, wrongAuthAccount, rightAuthAccount)
				})
			}
		})
	}
}

// testAuthorizationsAreExecutedInOrder checks that authorizations are executed in order
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

	// last delegation is set
	code, err := client.CodeAt(context.Background(), account.Address(), nil)
	require.NoError(t, err)
	expectedCode := append([]byte{0xef, 0x01, 0x00}, common.Address{24}.Bytes()...)
	require.Equal(t, expectedCode, code, "code in account is expected to be delegation designation")
}

// testMultipleAccountsCanSubmitAuthorizations checks that multiple accounts can submit authorizations
// and those accounts may be unrelated to the account receiver of the transaction.
func testMultipleAccountsCanSubmitAuthorizations(t *testing.T, net *IntegrationTestNet) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId, err := client.ChainID(context.Background())
	require.NoError(t, err, "failed to get chain ID")

	account := makeAccountWithBalance(t, net, 1e18) // < Pays for the transaction
	nonce, err := client.NonceAt(context.Background(), account.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", account.Address())

	authorizerA := makeAccountWithBalance(t, net, 0) // < no cost
	authorizerANonce, err := client.NonceAt(context.Background(), authorizerA.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", authorizerA.Address())

	authorizerB := makeAccountWithBalance(t, net, 0) // < no cost
	authorizerBNonce, err := client.NonceAt(context.Background(), authorizerB.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", authorizerB.Address())

	authorizationA, err := types.SignSetCode(authorizerA.PrivateKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(chainId),
		Address: common.Address{42},
		// because authorizerA is not the receiver of the transaction, the nonce is the current nonce
		Nonce: authorizerANonce,
	})
	require.NoError(t, err, "failed to sign SetCode authorization")

	authorizationB, err := types.SignSetCode(authorizerB.PrivateKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(chainId),
		Address: common.Address{42},
		// because authorizerB is not the receiver of the transaction, the nonce is the current nonce
		Nonce: authorizerBNonce,
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
	// transaction call to self will succeed
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// check that delegations are set to the correct address
	expectedCode := append([]byte{0xef, 0x01, 0x00}, common.Address{42}.Bytes()...)

	codeA, err := client.CodeAt(context.Background(), authorizerA.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, expectedCode, codeA, "code in account is expected to be delegation designation")

	codeB, err := client.CodeAt(context.Background(), authorizerB.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, expectedCode, codeB, "code in account is expected to be delegation designation")
}

// testAuthorizationSucceedsWithFailingTx checks that an authorization is executed
// even if the transaction fails
func testAuthorizationSucceedsWithFailingTx(t *testing.T, net *IntegrationTestNet) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	account := makeAccountWithBalance(t, net, 1e18) // < Pays for the transaction

	_, receipt, err := DeployContract(net, counter.DeployCounter)
	require.NoError(t, err)
	delegateAddress := receipt.ContractAddress

	tx := makeEip7702Transaction(t, client, account, account, delegateAddress, nil)
	receipt, err = net.Run(tx)
	require.NoError(t, err)
	// Submitting a call to the counter contract without callData must fail
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status)

	// delegation is set nevertheless
	code, err := client.CodeAt(context.Background(), account.Address(), nil)
	require.NoError(t, err)
	expectedCode := append([]byte{0xef, 0x01, 0x00}, delegateAddress.Bytes()...)
	require.Equal(t, expectedCode, code, "code in account is expected to be delegation designation")
}

// testAuthorizationFromNonExistingAccount checks that an authorization can signed by
// a non exiting account, creating the account on the fly.
func testAuthorizationFromNonExistingAccount(t *testing.T, net *IntegrationTestNet) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	sponsor := makeAccountWithBalance(t, net, 1e18) // < Pays for the transaction

	// create an account from a key without endowing it
	key, _ := crypto.GenerateKey()
	nonExistingAccount := Account{
		PrivateKey: key,
	}

	tx := makeEip7702Transaction(t, client, sponsor, &nonExistingAccount, common.Address{42}, nil)
	_, err = net.Run(tx)
	require.NoError(t, err)
	// whenever the transaction succeeds, is not relevant for this test

	// check that account exists and has a delegation designation
	code, err := client.CodeAt(context.Background(), nonExistingAccount.Address(), nil)
	require.NoError(t, err)
	expectedCode := append([]byte{0xef, 0x01, 0x00}, common.Address{42}.Bytes()...)
	require.Equal(t, expectedCode, code, "code in account is expected to be delegation designation")
}

// testNoDelegateToDelegated checks that delegations cannot be transitive
// The EIP-7702 specification does not allow for transitive delegations:
// > In case a delegation designator points to another designator, creating a
// > potential  chain or loop of designators, clients must retrieve only the
// > first code and then stop following the designator chain.
func testNoDelegateToDelegated(t *testing.T, net *IntegrationTestNet) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId, err := client.ChainID(context.Background())
	require.NoError(t, err, "failed to get chain ID")

	sponsor := makeAccountWithBalance(t, net, 1e18)
	account1 := makeAccountWithBalance(t, net, 0)
	account2 := makeAccountWithBalance(t, net, 0)

	// deploy the batch counterContract
	_, deployReceipt, err := DeployContract(net, counter.DeployCounter)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, deployReceipt.Status)
	counterContractAddress := deployReceipt.ContractAddress

	sponsorNonce, err := client.NonceAt(context.Background(), sponsor.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", sponsor.Address())
	account1Nonce, err := client.NonceAt(context.Background(), account1.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", account1.Address())
	account2Nonce, err := client.NonceAt(context.Background(), account2.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", account2.Address())

	// auth1 delegates to account2
	auth1, err := types.SignSetCode(account1.PrivateKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(chainId),
		Address: account2.Address(),
		Nonce:   account1Nonce,
	})
	require.NoError(t, err, "failed to sign SetCode authorization")

	// auth2 delegates to the counterContract
	auth2, err := types.SignSetCode(account2.PrivateKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(chainId),
		Address: counterContractAddress,
		Nonce:   account2Nonce,
	})
	require.NoError(t, err, "failed to sign SetCode authorization")
	authorizations := []types.SetCodeAuthorization{auth1, auth2}

	// Set delegator designator to both accounts
	tx, err := types.SignTx(
		types.NewTx(&types.SetCodeTx{
			ChainID:   uint256.MustFromBig(chainId),
			Nonce:     sponsorNonce,
			To:        sponsor.Address(),
			Gas:       500_000,
			GasFeeCap: uint256.NewInt(10e10),
			AuthList:  authorizations,
		}),
		types.NewPragueSigner(chainId),
		sponsor.PrivateKey,
	)
	require.NoError(t, err, "failed to sign transaction")
	receipt, err := net.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// use "mounted" contract in account1 to call the method.
	// this reduces confusion about correctness of the test, when it
	// fails because of ABI issues
	counterInAccount1, err := counter.NewCounter(account1.Address(), client)
	require.NoError(t, err)
	receipt, err = net.Apply(counterInAccount1.IncrementCounter)
	require.NoError(t, err)
	assert.Equal(t, types.ReceiptStatusFailed, receipt.Status)

	// account2 has direct delegation, must succeed
	counterInAccount2, err := counter.NewCounter(account2.Address(), client)
	require.NoError(t, err)
	receipt, err = net.Apply(counterInAccount2.IncrementCounter)
	require.NoError(t, err)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
}

// testChainOfCalls checks that delegations can used over transitive calls between contracts.
func testChainOfCalls(t *testing.T, net *IntegrationTestNet) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId, err := client.ChainID(context.Background())
	require.NoError(t, err, "failed to get chain ID")

	sponsor := makeAccountWithBalance(t, net, 1e18) // < Pays for the transaction gas, originates the call-chain
	account1 := makeAccountWithBalance(t, net, 0)
	account2 := makeAccountWithBalance(t, net, 0)

	// deploy the batch contract
	contract, deployReceipt, err := DeployContract(net, transitive_call.DeployTransitiveCall)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, deployReceipt.Status)
	transitiveContractAddress := deployReceipt.ContractAddress

	sponsorNonce, err := client.NonceAt(context.Background(), sponsor.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", sponsor.Address())
	account1Nonce, err := client.NonceAt(context.Background(), account1.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", account1.Address())
	account2Nonce, err := client.NonceAt(context.Background(), account2.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", account2.Address())

	// both accounts delegate to the transitive contract
	auth1, err := types.SignSetCode(account1.PrivateKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(chainId),
		Address: transitiveContractAddress,
		Nonce:   account1Nonce,
	})
	require.NoError(t, err, "failed to sign SetCode authorization")

	auth2, err := types.SignSetCode(account2.PrivateKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(chainId),
		Address: transitiveContractAddress,
		Nonce:   account2Nonce,
	})
	require.NoError(t, err, "failed to sign SetCode authorization")
	authorizations := []types.SetCodeAuthorization{auth1, auth2}

	// calldata is used to fill the arguments passed to the contract call.
	// id defines a list of addresses to call, one after the other
	// value from the original transaction is carried with the call
	callData := getCallData(t, net, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return contract.TransitiveCall(opts, []common.Address{
			account1.Address(), // does one call to itself
			account2.Address(), // calls account2
		})
	})

	// The following transaction initiates a call Chain: account1 -> account1 -> account2
	// all paid by sponsor
	tx, err := types.SignTx(
		types.NewTx(&types.SetCodeTx{
			ChainID:   uint256.MustFromBig(chainId),
			Nonce:     sponsorNonce,
			To:        account1.Address(),
			Gas:       500_000,
			GasFeeCap: uint256.NewInt(10e10),
			AuthList:  authorizations,
			Data:      callData,
			Value:     uint256.NewInt(1234), // < will be sent through the call chain
		}),
		types.NewPragueSigner(chainId),
		sponsor.PrivateKey,
	)
	require.NoError(t, err, "failed to sign transaction")

	receipt, err := net.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Check balance has flown correctly
	balance, err := client.BalanceAt(context.Background(), account1.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), balance.Uint64())

	balance, err = client.BalanceAt(context.Background(), account2.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(1234), balance)

	// Check the number of contract invocations on each account (via local storage)
	contractInAccount1, err := transitive_call.NewTransitiveCall(account1.Address(), client)
	require.NoError(t, err)
	count, err := contractInAccount1.GetCount(nil)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(2), count)

	contractInAccount2, err := transitive_call.NewTransitiveCall(account2.Address(), client)
	require.NoError(t, err)
	count, err = contractInAccount2.GetCount(nil)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(1), count)
}

// makeEip7702Transaction creates a SetCode transaction
// - client is used to talk to the node
// - sponsor account pays for the gas for the transaction
// - sponsored account is the context of the transaction, and its state is modified
// - delegate account is the contract address installed in the delegator code
// - callData is used to pass the encoded use of the delegate's called method ABI
func makeEip7702Transaction(t *testing.T,
	client *ethclient.Client,
	sponsor *Account,
	sponsored *Account,
	delegate common.Address,
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
