# Sonic Integration Testing

Given that the integration testing infrastructure has grown in size and complexity, the team decided to put together a guide/documentation of how to use the existing tools, where to look for alternatives and where to add them. This document is intended as an introduction into the tool, not a complete guide.

## Table of content
 - [Starting a Test Net](#starting-a-test-network)
    - [Options](#options)
    - [Frequently used Functionalities](#frequently-used-functionalities)
 - [Client](#client)
 - [Session](#session)
    - [When NOT to use a Session?](#when-not-to-use-a-session)
    - [Session from shared network](#session-from-shared-network)
- [Require](#require)
- [Miscellaneous utilities](#miscellaneous-utilities)
- [Example](#example)
    - [Init and Restart](#init-and-restart)
    - [Send transactions in parallel](#send-transactions-in-parallel)



## Starting a test network

Currently at `integration_test_net.go` the backbone of this infrastructure is declared. The main goal of this tool is to simulate a network with one or multiple nodes connected. 

The first step to instantiate a new network is to call one of the following:
- `net := StartIntegrationTestNet(t)`
- `net := StartIntegrationTestNetWithFakeGenesis(t)`
- `net := StartIntegrationTestNetWithJsonGenesis(t)`

the latter one is actually called from the first option as well. For clarification on the differences refer to the docstring on each function, but unless your test case requires a fake genesis instance, you are probably safe using `StartIntegrationTestNet`.

### Options

All 3 alternatives provide a second optional parameter `options := IntegrationTestNetOptions`. This options allow you to customize the network, mainly with:
- `Upgrades`: which hard fork should the network be (for now only options are `Sonic` or `Allegro`).
- `NumNodes`: number of nodes the network should start.
- `ClientExtraArguments`: additional command line arguments for starting the Sonic Client.
- `ModifyConfig`: to play around with toml config file.
- `Accounts`: To add specific accounts into the test genesis.
- `SkipCleanup`: To tell the network that it should not add its cleanup to the current testing clean up.

`Upgrades` and `Nodes` being the most commonly used of all, feel free to play around with these two but be CAREFUL about modifying any of the other options. Next is an exampl
```Go
net := StartIntegrationTestNetWithJsonGenesis(t,
           tests.IntegrationTestNetOptions{
			   Upgrades: AsPointer(opera.GetAllegroUpgrades()),
		   })
```


### Frequently used Functionalities

As we have developed more tests we noticed certain patterns and made functions to shorten or replace such patterns. The following is a list of those:
- `net.EndowAccount(address, value)`: transfers `value` tokens to the `address` from the account sponsoring the validator. (There is an alternative `EndowAccounts`, which does the same but to a list of addresses)
- `net.Run(tx)` will send the tx to a node, wait for its execution, wait for the receipt of the tx and return its receipt. (There is an alternative `RunAll`, which does the same but with a list of transactions)
- `net.GetReceipt(txHash)` queries the network until it gets a receipt for the given tx hash, or returns timeout. (There is an alternative `GetReceipts`, which does the same but for a list of transactions)
- `net.Apply(issue)` sends a transaction and waits for it to be processed, usually used for contracts interactions.
- `net.SessionSponsor()` returns the account sponsoring the network or session, which is the account used for signing all the transactions (more on sessions in [Session](#session) section)
- `net.GetClient()` returns a pooled client connected to the node 0 of the network. More on this in [Client](#client)
- `net.GetChainId` returns the chain ID of the network.
- `net.SpawnSession` returns a new session with a fresh account. More on this in [Session](#session)
- If your test case requires it there is `net.Stop` and `net.Restart`.

Some of these methods are implemented using the functionalities provided by the a client.


## Client

Once you have a running test network the main way to interact with it would be to get an RPC client to one of the nodes:
```Go
client, err := net.GetClient()
require.NoError(t, err)
defer client.Close()
```
the error must be checked. The clients are pooled per node and thus must be closed to be return to the pool. Note that pooled clients are basically [ethclient](https://github.com/ethereum/go-ethereum/tree/master/ethclient)s and have all of their functionalities. Amongst which the most commonly used are:
- `client.BlockByNumber(ctx, number)` queries for the specified block, returning the block and an error
- `client.BlockReceipt(ctx,BlockNumberOrHash)` queries for the receipts of the transactions included in the block of the number/hash given.
- `client.NonceAt(ctx, addr, blockNum)` returns the nonce of the given address at the given block height.
- A big flexible tool is `client.Client().Call()` which is using the `Call()` of the underlying [`rpcClient`](https://github.com/ethereum/go-ethereum/blob/master/rpc/client.go#L79), which can be used to query all the RPC functions provided by Sonic.

## Session 

When attempting to run multiple network operations or transactions in parallel, there could be a race condition on the nonces or other network resources, hence came the session. Sessions provide almost the same functionalities as Network (with a few exceptions). They can be generated by calling `net.SpawnSession()`. 

### When NOT to use a Session?
While Networks and Sessions offer almost the same functionality there are cases when a network could be shared amongst many sessions in parallel. 
If your test case has any of the following requirements it should stick to using only ONE network :
- The network needs to be restarted
- The network rules need to be updated
- The networks epoch needs to be forcibly advanced.

### Session from shared network
Sessions can be produced by the usual `net := StartIntegrationTestNet(t)` by simply calling `session := net.SpawnSession(t)`, in which case we can use either `net` or `session` to submit the transactions (but you should avoid spawning sessions if you do not need parallelism), or they can be produced by `session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())`. 
This method requires the `Upgrades` (unlike the optional options from `StartIntegrationTestNet`) to be provided and spawns a session in a network using that specif Upgrade. The network that this new session belongs to will be shared amongst multiple tests, so it is safe to run in parallel, just be mindful that calling `t.Parallel()` should always be done after spawning new sessions (regardless of whether they come from a shared network or not).

## Require
To verify values instead of the more basic pattern
```Go
if got != want {
    t.Error("something went wrong, wanted %v but got %v", want, got)
}
```
We opted to use [`require`](https://github.com/stretchr/testify/blob/master/require/doc.go), to improve readability of the tests, resulting instead on code similar to:
```Go
require.Equal(want, got, "something went wrong")
```
Similarly this package provides a broad variety of methods to simplify checking.


## Miscellaneous utilities
The following are a list of common utilities defined in `test_utils.go` utilized amongst many integration tests:
- `SignTransaction(t, chainId, tx, from)` signs tx as if it came `from` to be used on the network with the `chainId`
- `SetTransactionDefaults(t, netOrSess, tx, sender)` uses a client to query the network for the next Nonce and an acceptable gas price.
- `UpdateNetworkRules(t, net, changes)` is used to update the network rules
- `GetNetworkRules(t, net)` returns the set of active rules in the given network
- `GetEpochOfBlock(t, client, blockNumber)` uses the given client to query the block info and returns the epoch number of that block
- `MakeAccountWithBalance(t, netOrSession, balance)` creates a new account giving it `balance`.
- `WaitFor(ctx, predicate)` is a function that will execute `predicate` over some incremental delay until a timeout is reached.
- `AdvanceEpochAndWaitForBlocks(t, net)` advances the epoch of the provided network and waits for the first two blocks of the new epoch to be produced.

The idea for these functions is to prevent multiple integration tests to have their own implementations of helper functions but instead implement them in a common space to be used by other tests as well.

Note that since some integration tests are in sub-packages, some of these functions need to be public, but to enforce that they are only used in tests, they take a `t *testing.T` parameter. Keep this in mind when implementing new helper functions.

## Example
### Init and Restart
```Go
package tests

func TestIntegrationTestNet_CanStartRestartAndStopIntegrationTestNet(t *testing.T) {
	net := StartIntegrationTestNet(t)
	require.NoError(t, net.Restart(), "Failed to restart the test network")

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
