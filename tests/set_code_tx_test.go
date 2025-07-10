// Copyright 2025 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package tests

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/contracts/batch"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/0xsoniclabs/sonic/tests/contracts/privilege_deescalation"
	"github.com/0xsoniclabs/sonic/tests/contracts/sponsoring"
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

	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		Upgrades: AsPointer(opera.GetAllegroUpgrades()),
	})

	t.Run("Operation", func(t *testing.T) {
		t.Parallel()
		// operation tests check basic operation of the SetCode transaction

		t.Run("Delegate can be set and unset", func(t *testing.T) {
			t.Parallel()
			session := net.SpawnSession(t)
			testDelegateCanBeSetAndUnset(t, session)
		})

		t.Run("Invalid authorizations are ignored", func(t *testing.T) {
			t.Parallel()
			testInvalidAuthorizationsAreIgnored(t, net)
		})

		t.Run("Authorizations are executed in order", func(t *testing.T) {
			t.Parallel()
			session := net.SpawnSession(t)
			testAuthorizationsAreExecutedInOrder(t, session)
		})

		t.Run("Multiple accounts can submit authorizations", func(t *testing.T) {
			t.Parallel()
			session := net.SpawnSession(t)
			testMultipleAccountsCanSubmitAuthorizations(t, session)
		})

		t.Run("Authorization succeeds with failing tx", func(t *testing.T) {
			t.Parallel()
			session := net.SpawnSession(t)
			testAuthorizationSucceedsWithFailingTx(t, session)
		})

		t.Run("Authorization can be issued from a non existing account", func(t *testing.T) {
			t.Parallel()
			session := net.SpawnSession(t)
			testAuthorizationFromNonExistingAccount(t, session)
		})

		t.Run("Delegations cannot be transitive", func(t *testing.T) {
			t.Parallel()
			session := net.SpawnSession(t)
			testNoDelegateToDelegated(t, session)
		})

		t.Run("Delegations can trigger chains of calls", func(t *testing.T) {
			t.Parallel()
			session := net.SpawnSession(t)
			testChainOfCalls(t, session)
		})

	})

	t.Run("UseCase", func(t *testing.T) {
		t.Parallel()
		// UseCase tests check the use cases described in the EIP-7702 specification

		t.Run("Transaction Sponsoring", func(t *testing.T) {
			t.Parallel()
			session := net.SpawnSession(t)
			testSponsoring(t, session)
		})

		t.Run("Transaction Batching", func(t *testing.T) {
			t.Parallel()
			session := net.SpawnSession(t)
			testBatching(t, session)
		})

		t.Run("Privilege Deescalation", func(t *testing.T) {
			t.Parallel()
			session := net.SpawnSession(t)
			testPrivilegeDeescalation(t, session)
		})
	})
}

// testSponsoring executes a transaction in behalf of another account:
// - The sponsor account pays for the gas for the transaction
// - The sponsored account is the context of the transaction, and its state is modified
// - The delegate account is the contract that will be executed
func testSponsoring(t *testing.T, net IntegrationTestNetSession) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// sponsor issues the SetCode transaction and pays for it
	sponsor := makeAccountWithBalance(t, net, big.NewInt(1e18))
	// sponsored is used as context for the call, its state will be modified
	// without paying for the transaction
	sponsored := makeAccountWithBalance(t, net, big.NewInt(10))
	receiver := makeAccountWithBalance(t, net, new(big.Int))

	// Deploy the contract to forward the call
	sponsoringDelegate, receipt, err := DeployContract(net, sponsoring.DeploySponsoring)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	delegateAddress := receipt.ContractAddress

	// Prepare calldata for the sponsoring transaction
	// - sponsor pays the gas fees
	// - sponsored pays the value transfer
	// - receiver receives the funds (called contract/address)
	callData := getCallData(t, net, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		// Transfer 10 wei to receiver account
		return sponsoringDelegate.Execute(opts, receiver.Address(), big.NewInt(10), nil)
	})

	// Create a setCode transaction calling the incrementCounter function
	// in the context of the sponsored account.
	setCodeTx := makeEip7702Transaction(t, client, sponsor, sponsored, delegateAddress, callData)
	receipt, err = net.Run(setCodeTx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Check that the three account states are correctly modified:
	effectiveCost := new(big.Int)
	effectiveCost = effectiveCost.Mul(
		receipt.EffectiveGasPrice,
		big.NewInt(int64(receipt.GasUsed)))
	balance, err := client.BalanceAt(t.Context(), sponsor.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t,
		new(big.Int).Sub(
			big.NewInt(1e18), effectiveCost),
		balance, "sponsor balance must be reduced by the transaction cost")

	balance, err = client.BalanceAt(t.Context(), sponsored.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t,
		int64(0),
		balance.Int64(), "sponsored balance must be reduced by the value transferred")

	balance, err = client.BalanceAt(t.Context(), receiver.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t,
		big.NewInt(10),
		balance, "receiver balance must increase by the value transferred")
}

// testBatching executes multiple funds transfers within a single transaction:
// - The sponsor and sponsored accounts are the same, this is a self-sponsored transaction.
// - The delegate account is the contract that will be executed, which implements the batch of calls
// - Multiple receiver accounts will receive the funds
func testBatching(t *testing.T, net IntegrationTestNetSession) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// sender account batches multiple transfers of funds in a single transaction
	// receivers will receive the funds
	sender := makeAccountWithBalance(t, net, big.NewInt(1e18)) // < pays transaction and transfers funds
	receiver1 := NewAccount()
	receiver2 := NewAccount()

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

	balance, err := client.BalanceAt(t.Context(), sender.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t,
		new(big.Int).Sub(
			big.NewInt(1e18), effectiveCost), balance)

	// Check that the receivers have received the funds
	balance1, err := client.BalanceAt(t.Context(), receiver1.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(1234), balance1)

	balance2, err := client.BalanceAt(t.Context(), receiver2.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(4321), balance2)
}

// testPrivilegeDeescalation executes a transaction where an account allows restricted access
// to its internal state to a second account.
func testPrivilegeDeescalation(t *testing.T, session IntegrationTestNetSession) {

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// - Account A allows account B to execute certain operations on its behalf
	// - Account A (account) is the context of the transaction, and its state is modified
	// - Account B (userAccount) pays for the gas for the transaction
	// - Some part of the contract interface (DoPayment) is executable from account B
	account := makeAccountWithBalance(t, session, big.NewInt(1e18))     // < will transfer funds
	userAccount := makeAccountWithBalance(t, session, big.NewInt(1e18)) // < will pay for gas
	receiver := NewAccount()                                            // < will receive funds

	// Deploy the a contract to use as delegate
	contract, receipt, err := DeployContract(session, privilege_deescalation.DeployPrivilegeDeescalation)
	require.NoError(t, err)
	delegate := receipt.ContractAddress

	// Install delegation in account and allow access by userAccount
	callData := getCallData(t, session, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		// The contract will whitelist the userAccount to execute the DoPayment function
		// within the same transaction that sets the delegation.
		return contract.AllowPayment(opts, userAccount.Address())
	})
	setCodeTx := makeEip7702Transaction(t, client, account, account, delegate, callData)
	receipt, err = session.Run(setCodeTx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Check that authorization has been set
	data, err := client.StorageAt(t.Context(), account.Address(), common.Hash{}, nil)
	require.NoError(t, err)
	addr := userAccount.Address()
	require.Equal(t, addr[:], data[12:32], "contract has not been initialized correctly")

	// "mount" the contract in the address of the delegating account
	delegatedContract, err := privilege_deescalation.NewPrivilegeDeescalation(account.Address(), client)
	require.NoError(t, err)

	accountBalanceBefore, err := client.BalanceAt(t.Context(), account.Address(), nil)
	require.NoError(t, err)

	// issue a normal transaction from userAccount to transfer funds to receiver
	txOpts, err := session.GetTransactOptions(userAccount)
	require.NoError(t, err)
	txOpts.NoSend = true
	tx, err := delegatedContract.DoPayment(txOpts, receiver.Address(), big.NewInt(1234))
	require.NoError(t, err)
	receipt, err = session.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// check balances
	accountBalanceAfter, err := client.BalanceAt(t.Context(), account.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, new(big.Int).Sub(accountBalanceBefore, big.NewInt(1234)), accountBalanceAfter)

	receivedBalance, err := client.BalanceAt(t.Context(), receiver.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(1234), receivedBalance)

	// issue a transaction from and unauthorized account
	unauthorizedAccount := makeAccountWithBalance(t, session, big.NewInt(1e18))
	txOpts, err = session.GetTransactOptions(unauthorizedAccount)
	require.NoError(t, err)
	txOpts.NoSend = true
	tx, err = delegatedContract.AllowPayment(txOpts, receiver.Address())
	require.NoError(t, err)
	receipt, err = session.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status, "tx shall be executed and rejected")

	txOpts, err = session.GetTransactOptions(unauthorizedAccount)
	require.NoError(t, err)
	txOpts.NoSend = true
	tx, err = delegatedContract.DoPayment(txOpts, receiver.Address(), big.NewInt(42))
	require.NoError(t, err)
	receipt, err = session.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status, "tx shall be executed and rejected")
}

// testDelegateCanBeSetAndUnset checks that a delegate can be set and unset
// The EIP-7702 specification describes the method to restore an EOA code to
// its original state by setting the delegate to the zero address.
func testDelegateCanBeSetAndUnset(t *testing.T, session IntegrationTestNetSession) {

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	account := makeAccountWithBalance(t, session, big.NewInt(1e18))

	// Deploy the a contract to use as delegate
	counter, receipt, err := DeployContract(session, counter.DeployCounter)
	require.NoError(t, err)
	delegateAddress := receipt.ContractAddress

	// set delegation
	callData := getCallData(t, session, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return counter.IncrementCounter(opts)
	})
	setCodeTx := makeEip7702Transaction(t, client, account, account, delegateAddress, callData)
	receipt, err = session.Run(setCodeTx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// check that code has been set
	codeSet, err := client.CodeAt(t.Context(), account.Address(), nil)
	require.NoError(t, err)
	expectedCode := append([]byte{0xef, 0x01, 0x00}, delegateAddress[:]...)
	require.Equal(t, expectedCode, codeSet, "code in account is expected to be delegation designation")

	// wait until previous transaction has been
	err = waitUntilTransactionIsRetiredFromPool(t, client, setCodeTx)
	require.NoError(t, err, "transaction should be retired from the pool")

	// unset by delegating to an empty address
	unsetCodeTx := makeEip7702Transaction(t, client, account, account, common.Address{}, []byte{})
	receipt, err = session.Run(unsetCodeTx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// check that code has been unset
	codeUnset, err := client.CodeAt(t.Context(), account.Address(), nil)
	require.NoError(t, err)
	require.Equal(t, []byte{}, codeUnset, "code in account is expected to be empty")
}

// testInvalidAuthorizationsAreIgnored checks that invalid authorizations are ignored
// whilst the transaction is still executed.
func testInvalidAuthorizationsAreIgnored(t *testing.T, net *IntegrationTestNet) {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId, err := client.ChainID(t.Context())
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
				code, err := client.CodeAt(t.Context(), wrongAccount.Address(), nil)
				require.NoError(t, err)
				require.Equal(t, []byte{}, code, "code in account is expected to be unmodified")
			},
		},
		"before correct authorization": {
			makeAuthorizations: func(t *testing.T, wrong types.SetCodeAuthorization, rightAccount *Account) []types.SetCodeAuthorization {
				nonce, err := client.NonceAt(t.Context(), rightAccount.Address(), nil)
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
				code, err := client.CodeAt(t.Context(), wrongAccount.Address(), nil)
				require.NoError(t, err)
				require.Equal(t, []byte{}, code, "code in account is expected to be unmodified")

				code, err = client.CodeAt(t.Context(), rightAccount.Address(), nil)
				require.NoError(t, err)
				expectedCode := append([]byte{0xef, 0x01, 0x00}, common.Address{43}.Bytes()...)
				require.Equal(t, expectedCode, code, "code in account is expected to be delegation designation")
			},
		},
		"after correct authorization": {
			makeAuthorizations: func(t *testing.T, wrong types.SetCodeAuthorization, rightAccount *Account) []types.SetCodeAuthorization {
				nonce, err := client.NonceAt(t.Context(), rightAccount.Address(), nil)
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
				code, err := client.CodeAt(t.Context(), wrongAccount.Address(), nil)
				require.NoError(t, err)
				require.Equal(t, []byte{}, code, "code in account is expected to be unmodified")

				code, err = client.CodeAt(t.Context(), rightAccount.Address(), nil)
				require.NoError(t, err)
				expectedCode := append([]byte{0xef, 0x01, 0x00}, common.Address{43}.Bytes()...)
				require.Equal(t, expectedCode, code, "code in account is expected to be delegation designation")
			},
		},
	}

	for caseName, test := range wrongAuthorizations {
		for scenarioName, scenario := range scenarios {
			t.Run(fmt.Sprintf("%s/%s", caseName, scenarioName), func(t *testing.T) {
				t.Parallel()
				session := net.SpawnSession(t)

				wrongAuthAccount := makeAccountWithBalance(t, session, big.NewInt(1e18)) // < will transfer funds
				nonce, err := client.NonceAt(t.Context(), wrongAuthAccount.Address(), nil)
				require.NoError(t, err, "failed to get nonce for account", wrongAuthAccount.Address())

				wrongAuthorization, err := test.makeAuthorization(wrongAuthAccount, nonce)
				require.NoError(t, err, "failed to sign SetCode authorization")

				rightAuthAccount := makeAccountWithBalance(t, session, big.NewInt(1e18)) // < will transfer funds
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
				receipt, err := session.Run(tx)
				require.NoError(t, err)
				// because no delegation is set, transaction call to self (no code) will succeed
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

				// check delegations
				scenario.check(t, wrongAuthAccount, rightAuthAccount)
			})
		}
	}
}

// testAuthorizationsAreExecutedInOrder checks that authorizations are executed in order
func testAuthorizationsAreExecutedInOrder(t *testing.T, session IntegrationTestNetSession) {

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId, err := client.ChainID(t.Context())
	require.NoError(t, err, "failed to get chain ID")

	account := makeAccountWithBalance(t, session, big.NewInt(1e18)) // < will transfer funds

	nonce, err := client.NonceAt(t.Context(), account.Address(), nil)
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
	receipt, err := session.Run(tx)
	require.NoError(t, err)
	// because no delegation is set, transaction call to self will succeed
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// last delegation is set
	code, err := client.CodeAt(t.Context(), account.Address(), nil)
	require.NoError(t, err)
	expectedCode := append([]byte{0xef, 0x01, 0x00}, common.Address{24}.Bytes()...)
	require.Equal(t, expectedCode, code, "code in account is expected to be delegation designation")
}

// testMultipleAccountsCanSubmitAuthorizations checks that multiple accounts can submit authorizations
// and those accounts may be unrelated to the account receiver of the transaction.
func testMultipleAccountsCanSubmitAuthorizations(t *testing.T, session IntegrationTestNetSession) {

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId, err := client.ChainID(t.Context())
	require.NoError(t, err, "failed to get chain ID")

	account := makeAccountWithBalance(t, session, big.NewInt(1e18)) // < Pays for the transaction
	nonce, err := client.NonceAt(t.Context(), account.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", account.Address())

	authorizerA := NewAccount() // < no cost
	authorizerANonce, err := client.NonceAt(t.Context(), authorizerA.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", authorizerA.Address())

	authorizerB := NewAccount() // < no cost
	authorizerBNonce, err := client.NonceAt(t.Context(), authorizerB.Address(), nil)
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
	receipt, err := session.Run(tx)
	require.NoError(t, err)
	// transaction call to self will succeed
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// check that delegations are set to the correct address
	expectedCode := append([]byte{0xef, 0x01, 0x00}, common.Address{42}.Bytes()...)

	codeA, err := client.CodeAt(t.Context(), authorizerA.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, expectedCode, codeA, "code in account is expected to be delegation designation")

	codeB, err := client.CodeAt(t.Context(), authorizerB.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, expectedCode, codeB, "code in account is expected to be delegation designation")
}

// testAuthorizationSucceedsWithFailingTx checks that an authorization is executed
// even if the transaction fails
func testAuthorizationSucceedsWithFailingTx(t *testing.T, session IntegrationTestNetSession) {

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	account := makeAccountWithBalance(t, session, big.NewInt(1e18)) // < Pays for the transaction

	_, receipt, err := DeployContract(session, counter.DeployCounter)
	require.NoError(t, err)
	delegateAddress := receipt.ContractAddress

	tx := makeEip7702Transaction(t, client, account, account, delegateAddress, nil)
	receipt, err = session.Run(tx)
	require.NoError(t, err)
	// Submitting a call to the counter contract without callData must fail
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status)

	// delegation is set nevertheless
	code, err := client.CodeAt(t.Context(), account.Address(), nil)
	require.NoError(t, err)
	expectedCode := append([]byte{0xef, 0x01, 0x00}, delegateAddress.Bytes()...)
	require.Equal(t, expectedCode, code, "code in account is expected to be delegation designation")
}

// testAuthorizationFromNonExistingAccount checks that an authorization can signed by
// a non exiting account, creating the account on the fly.
func testAuthorizationFromNonExistingAccount(t *testing.T, session IntegrationTestNetSession) {

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	sponsor := makeAccountWithBalance(t, session, big.NewInt(1e18)) // < Pays for the transaction

	// create an account from a key without endowing it
	key, _ := crypto.GenerateKey()
	nonExistingAccount := Account{
		PrivateKey: key,
	}

	tx := makeEip7702Transaction(t, client, sponsor, &nonExistingAccount, common.Address{42}, nil)
	_, err = session.Run(tx)
	require.NoError(t, err)
	// whenever the transaction succeeds, is not relevant for this test

	// check that account exists and has a delegation designation
	code, err := client.CodeAt(t.Context(), nonExistingAccount.Address(), nil)
	require.NoError(t, err)
	expectedCode := append([]byte{0xef, 0x01, 0x00}, common.Address{42}.Bytes()...)
	require.Equal(t, expectedCode, code, "code in account is expected to be delegation designation")
}

// testNoDelegateToDelegated checks that delegations cannot be transitive
// The EIP-7702 specification does not allow for transitive delegations:
// > In case a delegation designator points to another designator, creating a
// > potential  chain or loop of designators, clients must retrieve only the
// > first code and then stop following the designator chain.
func testNoDelegateToDelegated(t *testing.T, session IntegrationTestNetSession) {

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId, err := client.ChainID(t.Context())
	require.NoError(t, err, "failed to get chain ID")

	sponsor := makeAccountWithBalance(t, session, big.NewInt(1e18))
	account1 := NewAccount()
	account2 := NewAccount()

	// deploy the batch counterContract
	_, deployReceipt, err := DeployContract(session, counter.DeployCounter)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, deployReceipt.Status)
	counterContractAddress := deployReceipt.ContractAddress

	sponsorNonce, err := client.NonceAt(t.Context(), sponsor.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", sponsor.Address())
	account1Nonce, err := client.NonceAt(t.Context(), account1.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", account1.Address())
	account2Nonce, err := client.NonceAt(t.Context(), account2.Address(), nil)
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
	receipt, err := session.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// use "mounted" contract in account1 to call the method.
	// this reduces confusion about correctness of the test, when it
	// fails because of ABI issues
	counterInAccount1, err := counter.NewCounter(account1.Address(), client)
	require.NoError(t, err)
	receipt, err = session.Apply(counterInAccount1.IncrementCounter)
	require.NoError(t, err)
	assert.Equal(t, types.ReceiptStatusFailed, receipt.Status)

	// account2 has direct delegation, must succeed
	counterInAccount2, err := counter.NewCounter(account2.Address(), client)
	require.NoError(t, err)
	receipt, err = session.Apply(counterInAccount2.IncrementCounter)
	require.NoError(t, err)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
}

// testChainOfCalls checks that delegations can used over transitive calls between contracts.
func testChainOfCalls(t *testing.T, session IntegrationTestNetSession) {

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId, err := client.ChainID(t.Context())
	require.NoError(t, err, "failed to get chain ID")

	sponsor := makeAccountWithBalance(t, session, big.NewInt(1e18)) // < Pays for the transaction gas, originates the call-chain
	account1 := NewAccount()
	account2 := NewAccount()

	// deploy the batch contract
	contract, deployReceipt, err := DeployContract(session, transitive_call.DeployTransitiveCall)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, deployReceipt.Status)
	transitiveContractAddress := deployReceipt.ContractAddress

	sponsorNonce, err := client.NonceAt(t.Context(), sponsor.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", sponsor.Address())
	account1Nonce, err := client.NonceAt(t.Context(), account1.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", account1.Address())
	account2Nonce, err := client.NonceAt(t.Context(), account2.Address(), nil)
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
	callData := getCallData(t, session, func(opts *bind.TransactOpts) (*types.Transaction, error) {
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

	receipt, err := session.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Check balance has flown correctly
	balance, err := client.BalanceAt(t.Context(), account1.Address(), nil)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), balance.Uint64())

	balance, err = client.BalanceAt(t.Context(), account2.Address(), nil)
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

	chainId, err := client.ChainID(t.Context())
	require.NoError(t, err, "failed to get chain ID")

	sponsoredNonce, err := client.NonceAt(t.Context(), sponsored.Address(), nil)
	require.NoError(t, err, "failed to get nonce for account", sponsored.Address())

	sponsorNonce, err := client.NonceAt(t.Context(), sponsor.Address(), nil)
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
func getCallData(t *testing.T, session IntegrationTestNetSession,
	transactionConstructor func(*bind.TransactOpts) (*types.Transaction, error)) []byte {
	txOpts, err := session.GetTransactOptions(session.GetSessionSponsor())
	require.NoError(t, err)
	txOpts.NoSend = true // <- create the transaction to read callData, but do not send it.
	tx, err := transactionConstructor(txOpts)
	require.NoError(t, err)
	return tx.Data()
}

func TestSetCodeTransaction_IsRejectBeforeAllegro(t *testing.T) {
	net := StartIntegrationTestNet(t)
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId, err := client.ChainID(t.Context())
	require.NoError(t, err, "failed to get chain ID")

	tx := signTransaction(t, chainId, &types.SetCodeTx{}, net.GetSessionSponsor())

	err = client.SendTransaction(t.Context(), tx)
	require.ErrorContains(t, err, "transaction type not supported")
}
