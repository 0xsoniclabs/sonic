# Bundles RPC API



The bundles RPC API exposes the following methods:


## sonic_getBundleInfo

## sonic_prepareBundle

### Parameters
#### transactionArgs
__object array__

The transaction response object which contains the following fields:
- from: The address the transaction is sent from
- to: The address the transaction is directed to
- gas: The integer of the gas provided for the transaction execution
- gasPrice: The integer of the gasPrice used for each paid gas, in wei
- maxFeePerGas: The maximum gas price to use per gas unit when gasPrice is not defined, in wei
- maxPriorityFeePerGas: The maximum priority fee to pay per gas when gasPrice is not defined, in wei
- value: The integer of the value sent with this transaction, in wei
- data: The compiled code of a contract or the hash of the invoked method signature and encoded parameters
- nonce: The next expected nonce for the sender
- accessList: An optional list of accessed addresses and storages during 
contract execution

#### executionFlags
reserved integer
#### earliestBlock
The earliest block number when the bundle execution can be attempted.
#### latestBlock 
The latest block number when the bundle execution can be attempted. After this block the bundle will be discarded if execution did not happen yet.

### Returns 

__object array__
The transaction arguments in the same format as provided by in the parameters, ready to be signed. Note: for the bundle to be correct, transactions need to be signed using the exact values of all the returned fields. 

## sonic_submitBundle

The submit bundle method receives an array of signed transactions and submits the for execution in the network.

### Parameters
- array data __REQUIRED__: All the signed transactions returned from the call to prepare using the same executionFlags, earliestBlock, and latestBlock.


#### executionFlags
reserved integer
#### earliestBlock
The earliest block number when the bundle execution can be attempted.
#### latestBlock 
The latest block number when the bundle execution can be attempted. After this block the bundle will be discarded if execution did not happen yet.


### Returns

__hash__ 
The Bundle hash, which execution can be tracked using `sonic_getBundleInfo`
