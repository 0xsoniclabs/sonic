// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract BlockHash {
    event Seen(uint256 currentBlock, uint256 observedBlock, bytes32 blockHash);

    function observe() public {
        uint256 start = 0;
        uint256 end = block.number + 5;
        if (end > 260) {
            start = end - 261;
        }
        for (uint256 i = start; i <= end; i++) {
            bytes32 blockHash = blockhash(i);
            emit Seen(block.number, i, blockHash);
        }
    }

    function getBlockHash(uint256 nr) public view returns (bytes32) {
        return blockhash(nr);
    }
}
