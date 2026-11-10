# Sonic Integration Testing

This document is designed to help newcomers navigate our integration testing framework. This guide provides an introduction to the framework's core concepts, what kinds of tests can be written, how to compose them, and how to use helper utilities to write clean and efficient tests.

To run the integration tests, run the command `go test /path/to/0xSonicLabs/sonic/tests/... -timeout 30m`; the last flag is needed because the default timeout is 10 minutes and the tests take longer than that.

## Framework Scope
The current implementation lets you write tests that:
- Send transactions and verify their effect on the blockchain.
- Simulate and query networks with multiple nodes.
- Deploy different network rules (including hard forks).
- Deploy custom contracts and interact with them.
- Interact with the full JSON and WebSocket RPCs.
- Start, stop, and restart instances of the Sonic client.

## Key Concepts

Our integration tests simulate a running network with one or many validator nodes, aimed at verifying end-to-end properties of the network. Before diving into the details, it's helpful to understand the primary building blocks of our testing infrastructure.

- `Network`: The main object representing a (possibly multi-node) network. It is a self-contained environment that can be started and stopped for a single test. Its life cycle is managed by the test where it is created.
- `Session`: A view of an isolated set of addresses existing in the blockchain state, which allows the execution of transactions and account modifications without collisions with other tests. Sessions can be used in parallel.
- `Account`: A pair of an address and a key, where the address is used to identify the sender or receiver and the key is used to sign transactions.
- `Sponsor`: A special account with a significant balance that funds and signs transactions. Both `Network` and `Session` have a sponsor account that is used to send transactions.
- `Transactions`: The messages through which users interact with the network. `Transactions` can be constructed explicitly, or implicitly by using a contract's ABI or account endowments.
- `Contracts`: Programs stored on the blockchain. They can contain logic and access the blockchain state. To use them in a test, they need to be compiled from Solidity code into Go. More about this in its own section below.

## Getting Started

Here are some considerations to keep in mind when adding new integration tests:
1) If there is already an integration test file with the same domain, consider adding your test there before creating a new file.
2) If one of the following applies to your test:
	- The network is going to be restarted.
	- The network rules are going to be updated.
	- The network's epoch is going to be forcibly advanced.
	- The network is going to have more than one node.
	- The network is going to have a specified config file, or a specific non-default configuration parameter.

	then use a new, exclusive network `net := StartIntegrationTestNet(t)`.
	For example:
	```Go
	import (
		"testing"
		"github.com/0xsoniclabs/sonic/tests"
		"github.com/stretchr/testify/require"
	)

	func TestNetworkRule_Update(t *testing.T) {

		require := require.New(t)
		net := tests.StartIntegrationTestNet(t)

		current := tests.GetNetworkRules(t, net)
		modified := myRuleModifications(current)
		tests.UpdateNetworkRules(t, net, modified)
		net.AdvanceEpoch(t, 1)

		require.NoError(net.Restart())
		newConfig := tests.GetNetworkRules(t, net)
		require.Equal(modified, newConfig)
	}
	```

	Otherwise, we highly encourage you to use `session := getIntegrationTestNetSession(t, upgrades)`.

	`upgrades` indicates which hard forks the network uses.

	```Go
	func TestMultipleSessions_CanSendLegacyTransactionsInBulk(t *testing.T) {
		session := getIntegrationTestNetSession(t, opera.GetAllegroUpgrades())

		client, err := session.GetClient()
		require.NoError(t, err)
		defer client.Close()
		nonce, err := client.PendingNonceAt(t.Context(), session.GetSessionSponsor().Address())
		require.NoError(t, err)

		chainId := session.GetChainId()
		txs := []*types.Transaction{}
		for i := range 5 {
			tx := SetTransactionDefaults(t, session, &types.LegacyTx{Nonce: nonce + uint64(i)}, session.GetSessionSponsor())
			signedTx := SignTransaction(t, chainId, tx, session.GetSessionSponsor())
			txs = append(txs, signedTx)
		}
		receipts, err := session.RunAll(txs)
		require.NoError(t, err, "failed to send transaction")
		require.Equal(t, len(receipts), len(txs))
	}
	```
	Note that both sessions and networks can be started with specific upgrades: sessions through the `upgrades` argument, and networks through the `Upgrades` field of `IntegrationTestNetOptions`, which defaults to `opera.GetSonicUpgrades()`.

3) If multiple properties or values need to be verified, check whether they can be split into subtests using `t.Run` and, further, whether each subtest can run in parallel using `t.Parallel()`. Keep in mind that running nested tests in parallel might require you to use a `Session`.

	```Go
	func TestType_ManyProperties(t *testing.T) {
		session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())

		t.Run("someProperty", func(t *testing.T) {
			subSession := session.SpawnSession(t)
			t.Parallel()

			validateSomeProperty(t, subSession)
		})
		t.Run("anotherProperty", func(t *testing.T) {
			subSession := session.SpawnSession(t)
			t.Parallel()

			validateAnotherProperty(t, subSession)
		})
	}
	```
	Note that `t.Parallel()` is always called after `SpawnSession` to prevent concurrency issues.

4) If all the tests in the new file take over 2 minutes, consider moving them to their own sub-package so that Go can automatically run them in parallel with tests from other packages.

## Client

Networks or sessions can produce `Client`s connected to the different nodes. They can be used to interact with the JSON-RPC API and need to be closed after use.

```Go
func TestSendTransaction_Asynchronously(t *testing.T) {
	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	chainId := session.GetChainId()

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	hashes := []common.Hash{}
	for range 5 {
		tx := SetTransactionDefaults(t, session, &types.LegacyTx{}, session.GetSessionSponsor())
		signedTx := SignTransaction(t, chainId, tx, session.GetSessionSponsor())
		err := client.SendTransaction(t.Context(), signedTx)
		require.NoError(t, err, "failed to send transaction")
		hashes = append(hashes, signedTx.Hash())
	}
	receipts, err := session.GetReceipts(hashes)
	require.NoError(t, err)
	require.Equal(t, len(receipts), len(hashes))
}
```
In this example, all 5 transactions are submitted one after the other without waiting for their receipts. This means that by the time the `for` loop ends, we do not know whether the transactions have been processed and added to a block.
`session.GetReceipts` waits until all of the transactions have been executed (and their receipts reported via RPC).

Transactions can also be sent via `session.RunAll`, which waits until it gets the receipts for all the transactions, or with `client.SendTransaction`, which does not wait for the transactions to be executed, hence allowing transactions to be sent asynchronously. The use of `RunAll` can be observed in `TestMultipleSessions_CanSendLegacyTransactionsInBulk` above.

One can also get a WebSocket client to subscribe to notifications, as done in `TestBlockInArchive`.

## Solidity Contracts

Solidity code can be hand-crafted and then used to generate the corresponding Go code. For an example, look at [`sonic/tests/contracts/counter`](https://github.com/0xsoniclabs/sonic/tree/main/tests/contracts/counter/). One must write the `.sol` file, such as
```Solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract Counter {
    int private count = 0;

    function incrementCounter() public {
        count += 1;
    }

    function getCount() public view returns (int) {
        return count;
    }
}
```

and a `gen.go` file to generate the corresponding `counter.go`. The `gen.go` file must follow this pattern:
```Go
package mycontract

//go:generate solc --bin mycontract.sol --abi mycontract.sol -o build --overwrite
//go:generate abigen --bin=build/MyContract.bin --abi=build/MyContract.abi --pkg=mycontract --out=mycontract.go
```

where the abigen `--bin` and `--abi` paths follow the casing of the contract name, while the other parameters are lower case.

The following commands install the required tools:
- `sudo snap install solc` to install the Solidity compiler.
- `go install github.com/ethereum/go-ethereum/cmd/abigen@latest` to install the binding generator.

The generated bindings can then be used in a test, as in `sonic/tests/counter_test.go`:
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

There is an optional tool to get heap memory reports per test; it was added in [PR #350](https://github.com/0xsoniclabs/sonic/pull/350).

To get a report of a test's peak heap memory usage, run it with the environment variable set:
`SONIC_TEST_HEAP_PROFILE=on go test ./tests`
The resulting report can be inspected with pprof:
`go tool pprof -http "localhost:8000" build/profile/mem_myTestName.pprof`

## Require

Please use [testify/require](https://github.com/stretchr/testify/blob/master/require/doc.go) for improved readability.
