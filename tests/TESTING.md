# Sonic Integration Testing

This document is designed to help newcomers navigate our integration testing framework. This guide provides an introduction to the framework's core concepts, what kind of test can be written, how to compose them and use helper utilities to write clean and efficient tests.

To run the integration tests run the command `go test /path/to/0xSonicLabs/sonic/tests/... -timeout 30`, the last flag is needed because the default timeout is 10 minutes and the tests take longer than that.

## Framework Scope
The current implementation enables one to make test that:
- Send transactions and verify their effect on the blockchain.
- Simulate and query networks with multiple nodes.
- Deploy different network rules (including hard forks).
- Deploy custom contracts and interact with them.
- interact with the full json and websocket RPCs
- start, stop, and restart instances of the sonic client


## Key Concepts

Our integration tests simulate a running network with one or many validator nodes, aimed to verify end-to-end properties of the network. Before diving into the details, it's helpful to understand the primary building blocks of our testing infrastructure.

- `Network`: The main object representing a (possibly multi-node) network. It is a self-contained environment that can be started and stopped for a single test. Its life cycle is managed by the test where it is created.
- `Session`: A session is a view of a isolated set of addresses existing in the blockchain state which allows the execution of transactions and accounts modifications without collisions with other tests. Sessions can be executed in parallel.
- `Account`: Is a pair of address and key, where the address is used to signal sender/receiver and the key is used to sign the transactions.
- `Sponsor`: Is a special account with a significant balance that funds and signs transactions. Both `Network` and `Session` have a sponsor account that is used to send transactions.
- `Transactions`: They are the messages via which users can interact with the network. `Transactions` can be explicitly constructed or implicitly by using contracts ABI or account endowments.
- `Contracts`: Are programs stored on the blockchain. They have can have some logic processing inside and access to the blockchain status. To use them in a test they need to be compiled from solidity code into go. More about this on its own section below.

## Getting Started

Here are some considerations to keep in mind when adding new integration tests:
1) If there is already an integration test file with the same domain, consider adding your test there before creating a new file.
0) If your test needs to do one of the following actions:
	- The network is going to be restarted.
	- The network rules are going to be updated.
	- The network's epoch is going to be forcibly advanced.
	- The network is going to have more than one node.
	- The network is going to have have a specified config file, or a specific non-default configuration parameter.
	
	Then use an exclusive new network `net := StartIntegrationTestNet(t)`.
	For example:
	```Go
	import (
		"testing"
		"github.com/0xsoniclabs/sonic/tests"
		"github.com/stretchr/testify/require"
	)

	func TestNetworkRule_Update(t *testing.T){

		require := require.New(t)
		net := tests.StartIntegrationTestNet(t)

		current := tests.GetNetworkRules(t, net)
		modified := myRuleModifications(current)
		tests.UpdateNetworkRules(t, net, modified)
		AdvanceEpochAndWaitForBlocks(t, net)

		net.Restart()
		newConfig := tests.GetNetworkRules(t, net)
		require.Equal(modified, newConfig)
	}	
	```

	Otherwise we highly encourage you to use `session := getIntegrationTestNetSession(t, Upgrade)`

	`Upgrades` indicates which hard fork options the network uses. `opera.Sonic` hard fork is used as a default.

	```Go
	func TestMultipleSessions_CanSendLegacyTransactionsInBulk(t *testing.T) {
		session := getIntegrationTestNetSession(t, opera.GetAllegroUpgrades())

		chainId := session.GetChainId()
		txs := types.Transaction[]{}
		for i := range 5 {
			tx := SetTransactionDefaults(t, session, &types.LegacyTx{}, session.GetSessionSponsor())
			signedTx := SignTransaction(t, chainId, tx, session.GetSessionSponsor())
			txs = append(txs, signedTx)
		}
		receipts, err := session.RunAll(txs)
		require.NoError(t, err, "failed to send transaction")
		require.Equal(t, len(receipts), len(txs))
	}
	```
	Note that both sessions and networks can be started with 


0) If multiple properties or values need to be verified, analyze if it is possible to split them into sub cases using `t.Run` and even more, if each sub test can be parallelized with `t.Parallel()`. Keep in mind running nested tests in parallel might require you to use a `Session`.

	```Go
	func TestType_ManyProperties(t *testing.T){
		session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())

		t.Run("someProperty", func (t *testing.T){
			subSession := session.SpawnSession(t)
			t.Parallel()

			validateSomeProperty(t, session)
		})
		t.Run("anotherProperty", func (t *testing.T){
			subSession := session.SpawnSession(t)
			t.Parallel()

			validateAnotherProperty(t, session)
		})
	}
	```
	Note that `t.Parallel()` is always called after `SpawnSession`, that is to prevent concurrency issues. 

0) If all the tests in the new file take over 2 minutes consider moving it to its own sub-package so that go can automatically choose to run it in parallel with tests from other packages. 

## Client

Networks or sessions can produce `Client`s connected to the different nodes. These need to be closed and they can be used to interact with the JSON-RPC API.

```Go
func TestSendTransaction_Asynchronously(t *testing.T){
	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	chainId := session.GetChainId()

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	hashes := hash[]
	for i := range 5 {
		tx := SetTransactionDefaults(t, session, &types.LegacyTx{}, session.GetSessionSponsor())
		signedTx := SignTransaction(t, chainId, tx, session.GetSessionSponsor())
		err := client.SendTransaction(t.Context(), signedTx)
		require.NoError(t, err, "failed to send transaction")
		hash = append(hash, signedTx.Hash())
	}
	receipts, err := session.GetReceipts(hashes)
	require.NoError(t, err)
	require.Equal(t, len(receipts), len(hashes))
}

```
In this example all 5 transactions are submitted one after the other without waiting for the receipts, that means that by the time the `for` loop ends, we do not know if the transactions have been processed and added into a new block or not. 
`session.GetReceipts` will wait until all of the transactions have been executed (and their receipts reported via RPC).

Transactions can also be sent via `session.RunAll` which wait until it gets the receipts for all the transactions or with `client.SendTransaction` which does not wait for the transactions to be executed, hence enabling to send transactions asynchronously. The use `RunAll` can be observed in `TestMultipleSessions_CanSendLegacyTransactionsInBulk`


One can also get *websocket* client to subscribe to different methods like `TestBlockInArchive`

## Solidty Contracts

Solidy code can be hand crafted and then used to generate the corresponding Go code. For examples on this please look at [`sonic/tests/contracts/counter`](https://github.com/0xsoniclabs/sonic/tree/main/tests/contracts/counter/), one must write the `.sol` file, such as
```Solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract Counter {
    int private count = 0;

    function incrementCounter() public {
        count += 1;
    }
}
```

and `gen.go` to generate the corresponding `counter.go`. Then it can be used as shown in `sonic/tests/counter_test.go`
the file `gen.go` must comply to the following pattern:
```Go
package mycontract

//go:generate solc --bin mycontract.sol --abi mycontract.sol -o build --overwrite
//go:generate abigen --bin=build/MyContract.bin --abi=build/MyContract.abi --pkg=mycontract --out=mycontract.go
```

where abigen `--bin` and `--abi` follow the casing of the contract name, while the other parameters are lower case.

The following commands can be used to install the needed dependencies are needed for this:
- `sudo snap install solc` to install solidity 0.8.30.
- `go install github.com/ethereum/go-ethereum/cmd/abigen@latest` to install the binding generator.

```Go
func TestCounter_CanIncrementAndReadCounterFromHead(t *testing.T) {

	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	t.Parallel()

	// Deploy the counter contract.
	contract, receipt, err := DeployContract(session, counter.DeployCounter)
	require.NoError(t, err, "failed to deploy contract; %v", err)
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)

	// Increment the counter a few times and check that the value is as expected.
	for i := 0; i < 10; i++ {
		counter, err := contract.GetCount(nil)
		require.NoError(t, err, "failed to get counter value")
		require.Equal(t, int64(i), counter.Int64(), "unexpected counter value")

		_, err = session.Apply(contract.IncrementCounter)
		require.NoError(t, err, "failed to apply increment counter contract")
	}
}
```


## Memory Analysis 

There is an optional tool to get heap memory reports per test, it was added in [PR#350](https://github.com/0xsoniclabs/sonic/pull/350)

To run a test and get a report of its memory peak heap memory usage it must run with the env var:
`SONIC_TEST_HEAP_PROFILE=on go test ./tests`
and the resulting report can be inspected with pprof
`go tool pprof -http "localhost:8000" build/profile/mem_myTestName.pprof`

## Require

Please use [testify/require]((https://github.com/stretchr/testify/blob/master/require/doc.go) ) for improved readability.