# Sonic Integration Testing

This document is designed to help newcomers navigate our integration testing framework. As our test infrastructure has grown, we've developed a robust set of tools to make writing reliable tests as straightforward as possible.
This guide provides an introduction to the framework's core concepts, to start a test network, interact with its nodes, and use helper utilities to write clean and efficient tests.

## Table of content
 - [Overview](#overview)
 - [Starting a Test Net](#starting-a-test-network)
    - [Options](#options)
    - [Frequently used Functionalities](#frequently-used-functionalities)
 - [Client](#client)
 - [Time and Memory limitations](#time-and-memory-limitations)
    - [When NOT to use a Session?](#when-not-to-use-a-session)
    - [Session from shared network](#session-from-shared-network)
- [Require](#require)
- [Miscellaneous utilities](#miscellaneous-utilities)
- [Minimal Examples](#minimal-examples)
    - [Init and Restart](#init-and-restart)
    - [Send transactions in parallel](#send-transactions-in-parallel)

## Overview

Our integration tests simulate a running network with one or many validator nodes, aimed to verify end-to-end properties of the network. Before diving into the details, it's helpful to understand the primary building blocks of our testing infrastructure.

Key concepts:
- `Network`: The main object representing a (possibly multi-node) network. It is a self-contained environment that can be started and stopped for a single test. Its life cycle is usually managed by the test where it is created.
- `Session`: An isolated context within a `Network`. It provides a dedicated account and a safe environment for sending transactions in parallel. While a Network can be shared, a Session is designed to be used by a single test or subtest to avoid resource collisions.
- `Sponsor`: Is a special account with a significant balance that funds and signs transactions. Both `Network` and `Session` have a sponsor account that is used to send transactions.
- `Account`: Is a standard Go common.Address with an associated private key. They are created and funded by a `Sponsor` to be used in test transactions. Usually used as origin/destination for a transaction.


## Starting a test network

The core of the testing infrastructure is defined in integration_test_net.go, which provides the tools to run a network with one or more nodes. To instantiate a new network, use one of the following functions: 
- `net := StartIntegrationTestNet(t)`
- `net := StartIntegrationTestNetWithFakeGenesis(t)`
- `net := StartIntegrationTestNetWithJsonGenesis(t)`

Unless your test specifically requires a fake genesis, StartIntegrationTestNet is the recommended option.

### Options

Each of the initialization functions accepts an optional `IntegrationTestNetOptions` parameter. This options allow you to customize the network, mainly with:
- `Upgrades`: Which hard fork should the network be initialized with (e.g., `Sonic`,`Allegro`).
- `NumNodes`:  Sets the number of nodes to be started in the network.
- `ClientExtraArguments`:  Provides additional command-line arguments for the Sonic Client.
- `ModifyConfig`: Allows for modifications to the client's TOML configuration file.
- `Accounts`: Adds specific accounts into the test genesis.

`Upgrades` and `Nodes` being the most commonly used of all, feel free to play around with these two but be CAREFUL about modifying any of the other options. Next is an example
```Go
net := StartIntegrationTestNetWithJsonGenesis(t,
           tests.IntegrationTestNetOptions{
			   Upgrades: AsPointer(opera.GetAllegroUpgrades()),
		   })
```


### Frequently used Functionalities

As we have developed more tests we noticed certain patterns and made functions to shorten or replace such patterns. The following is a list of those:
- `net.EndowAccount(address, value)`: Transfers `value` tokens to the `address` from the network's treasury/faucet account. (There is an alternative `EndowAccounts`, which does the same but to a list of addresses; use this if you want to initialize multiple accounts faster)
- `net.Run(tx)`: Sends a transaction, waits for its execution, and returns the transaction receipt. (There is an alternative `RunAll`, which does the same but with a list of transactions)
- `net.GetReceipt(txHash)`: Queries the network until it gets a receipt for the given tx hash, or returns timeout. (There is an alternative `GetReceipts`, which does the same but for a list of transactions)
- `net.Apply(issue)`: A utility function for contract interactions that sends a transaction and waits for it to be processed.
- `net.SessionSponsor()`: Returns the account sponsoring the network or session, which is the account used for signing all transactions without explicit signatures (more on sessions in the [Time and Memory limitations](#time-and-memory-limitations) section)
- `net.GetClient()`: Returns a client connected to the node 0 of the network. More on this in [Client](#client). Other nodes can be reached using `net.GetClientConnectedToNode(i)` where `i` is the number id (e.g. 1,2,3) of the node.
- `net.GetChainId`: Returns the chain ID of the network.
- `net.SpawnSession`: Returns a new session with a fresh account. More on this in [Session](#time-and-memory-limitations)
- `DeployContract`: Deploys a given contract on the network returning the receipt of the deploy and an error. The functions of the contract can be called using the afore mentioned `net.Apply` function. For examples on how to write/generate contracts look into `tests/contracts` folder.
- If your test case requires it there is `net.Stop` and `net.Restart`.


## Client

Once you have a running test network the main way to interact with it would be to get an RPC client connected to one of the nodes:
```Go
client, err := net.GetClient()
require.NoError(t, err)
defer client.Close()
```
The error must be checked. The clients are pooled per node and thus must be closed to be return to the pool. Note that pooled clients are basically [ethclient](https://github.com/ethereum/go-ethereum/tree/master/ethclient)s and have all of their functionalities. Amongst which the most commonly used are:
- `client.BlockByNumber(ctx, number)` queries for the specified block, returning the block and an error
- `client.BlockReceipt(ctx,BlockNumberOrHash)` queries for the receipts of the transactions included in the block of the number/hash given.
- `client.NonceAt(ctx, addr, blockNum)` returns the nonce of the given address at the given block height.
- A big flexible tool is `client.Client().Call()` which is using the `Call()` of the underlying [`rpcClient`](https://github.com/ethereum/go-ethereum/blob/master/rpc/client.go#L79), which can be used to call all the RPC functions provided by Sonic.

## Time and Memory limitations 

Originally, each integration test started its own network, which became slow as the number of tests grew. To reduce runtime, `t.Parallel()` was introduced when test cases inside an integration test could be executed in parallel, but this created a risk of synchronization issues on nonces and other shared network resources.

A Session solves this by providing a dedicated address for a single test. When you spawn a session, it creates a new, independent account. By using a session, a test can send transactions without worrying about nonce collisions or resource conflicts with other tests running in parallel.

Sessions provide almost all the functionalities of a Network (with a few exceptions) and can be generated by calling `net.SpawnSession()` or `getIntegrationTestNetSession(t, upgrade)`.


### When NOT to use a Session?
While `Networks` and `Sessions` offer almost the same functionality there are cases when a network could be shared amongst many sessions in parallel. 
If your test case has any of the following requirements it should stick to using only ONE network :
- The network needs to be restarted
- The network rules need to be updated
- The networks epoch needs to be forcibly advanced.

### Session from shared network
If an integration test does not do any of the actions mentioned in the previous section, it is probably a good candidate to use a session from a shared network, reusing resources and reducing overall runtime.
Sessions can be produced by the usual `net := StartIntegrationTestNet(t)` by simply calling `session := net.SpawnSession(t)`. 
Alternatively, sessions can be produced by `session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())`. This method requires the `Upgrades` to be provided and spawns a session in a network using that specif Upgrade. This new session is part of a shared network amongst multiple tests, so it is safe to run in parallel, just be mindful that calling `t.Parallel()` should always be done after spawning new sessions (regardless of whether they come from a shared network or not).
The reason for this is that calling `t.Parallel` will make the rest of the test run in a parallel context with other tests marked as parallel as well. If two (or more) tests running in parallel try to spawn a session from the same network, then they would end up executing endowments to the new addresses, and since this process is automated, the nonce is queried from the status of the network and the endowments for the new sessions could be sent with the same nonce.
Instead of the following risky order
```Go
func TestSomething(t *testing.T){
	t.Parallel(t) // first start the parallel section
	// if another test has the same pattern, multiple attempts to create new sessions in parallel
	// could make them fail
	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	someTest(t, session)
}
```
the safer option should be the one to use
```Go
func TestSomething(t *testing.T){
	// the code before t.Parallel is executed sequentially for all tests in the same context
	// hence it is safe to interact with shared resources and 
	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	t.Parallel(t) // start the parallel section
	someTest(t, session)
}
```
In the case of multiple sub-tests executing in parallel the following pattern is recommended
```Go
func TestManySubCases(t *testing.T){
	// first get a session
	session := getIntegrationTestNetSesion(t, opera.GetSonicUpgrades())
	t.Parallel()
	t.Run("subcase1", func(){
		subSession := session.SpawnSession()
		t.Parallel()
		subcase1(subSession)
	})
	t.Run("subcase2", func(){
		subSession := session.SpawnSession()
		t.Parallel()
		subcase2(subSession)
	})
}
```

## Adding a Test

Here are some considerations for adding new integration tests:
1) If there is already an integration test file with the same domain, consider adding your test there before creating a new file.
0) Consider if your test cases need to do any of the actions listed in [when not to use a session](#when-not-to-use-a-session), if so, then start your own network. But if not, this is probably a good candidate to use a session from a shared network.
0) If multiple properties or values need to be verified, analyze if it is possible to split them into sub cases using `t.Run` and even more, if each sub test can be parallelized with `t.Parallel()`. Keep in mind running tests in parallel might require you to use a `Session`.
0) If all the tests in the new file take over 2 minutes consider moving it to its own sub-package so that go can automatically choose to run it in parallel with tests from other packages. 

## Require
To improve readability and simplify assertion logic, we use the [`require`](https://github.com/stretchr/testify/blob/master/require/doc.go) package from testify. This replaces the basic `if got != want` pattern with more expressive and concise methods.


For example, instead of this:
```Go
if got != want {
    t.Error("something went wrong, wanted %v but got %v", want, got)
}
```
You can use the following:
```Go
require.Equal(want, got, "something went wrong")
```
Similarly this package provides a broad variety of methods to simplify checking.


## Miscellaneous utilities
The `test_utils.go` file contains common helper functions used across many integration tests:

- `SignTransaction(t, chainId, tx, from)` signs tx as if it came `from` to be used on the network with the `chainId`
- `SetTransactionDefaults(t, netOrSession, tx, sender)` uses a client to query the network for the next Nonce and an acceptable gas price.
- `UpdateNetworkRules(t, net, changes)` is used to update the network rules
- `GetNetworkRules(t, net)` returns the set of active rules in the given network
- `GetEpochOfBlock(t, client, blockNumber)` uses the given client to query the block info and returns the epoch number of that block
- `MakeAccountWithBalance(t, netOrSession, balance)` creates a new account giving it `balance`.
- `WaitFor(ctx, predicate)` is a function that will execute `predicate` over some incremental delay until a timeout is reached.
- `AdvanceEpochAndWaitForBlocks(t, net)` advances the epoch of the provided network and waits for the first two blocks of the new epoch to be produced.

These functions are designed to prevent the re-implementation of common logic across different test files. 

Note that because some integration tests are in sub-packages, some of these functions need to be public, but to enforce that they are only used in tests, they take a `t *testing.T` parameter. Keep this in mind when implementing new helper functions.

## Minimal Examples
### Init and Restart
```Go
package tests

func TestIntegrationTestNet_CanStartRestartAndStopIntegrationTestNet(t *testing.T) {
	net := StartIntegrationTestNet(t)
	err := net.Restart()
	require.NoError(t, err, , "Failed to restart the test network")
	net.Stop()
}
```
### Send transactions in parallel
```Go
package tests

func TestMultipleSessions_CanSendLegacyTransactionsInParallel(t *testing.T) {
	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	chainId := session.GetChainId()

	for i := range 5 {
		t.Run(fmt.Sprintf("parallel-%d", i), func(t *testing.T) {
			session := session.SpawnSession(t)
			t.Parallel()

			tx := SetTransactionDefaults(t, session, &types.LegacyTx{}, session.GetSessionSponsor())
			signedTx := SignTransaction(t, chainId, tx, session.GetSessionSponsor())
			err := client.SendTransaction(t.Context(), signedTx)
			require.NoError(t, err, "failed to send transaction")
		})
	}
}

```
