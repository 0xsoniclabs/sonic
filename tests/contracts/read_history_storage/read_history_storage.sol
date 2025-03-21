// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// ReadHistoryStorage uses the history storage contract to read block hashes
// https://eips.ethereum.org/EIPS/eip-2935
contract ReadHistoryStorage {
    event BlockHash(
        uint256 queriedBlock,
        bytes32 blockHash,
        bytes32 builtinBlockHash
    );

    function readHistoryStorage(uint256 blockNumber) public {
        address historyStorageAddress = 0x0000F90827F1C53a10cb7A02335B175320002935;

        // A call to the history storage from any address other than system-address
        // is a read operation, the argument is the block number of the hash queried
        (bool success, bytes memory data) = historyStorageAddress.call(
            abi.encode(blockNumber)
        );
        require(success, "call failed");

        bytes32 blockHash = abi.decode(data, (bytes32));
        bytes32 builtinBlockHash = blockhash(blockNumber);
        emit BlockHash(blockNumber, blockHash, builtinBlockHash);
    }
}
