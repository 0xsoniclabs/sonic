// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract BlockOverride {
    event Seen(uint256 currentBlock, uint256 observedBlock, bytes32 blockHash);

    function observe() public {
        uint256 start = 0;
        uint256 end = block.number + 5;
        if (end > 260) {
            start = end - 270;
        }
        for (uint256 i = start; i <= end; i++) {
            bytes32 blockHash = blockhash(i);
            emit Seen(block.number, i, blockHash);
        }
    }

    function getBlockHash(uint256 nr) public view returns (bytes32) {
        return blockhash(nr);
    }

    function getBlockParameters()
        public
        view
        returns (
            uint256 number,
            uint256 difficulty,
            uint256 time,
            uint256 gaslimit,
            address coinbase,
            uint256 random,
            uint256 basefee,
            uint256 blobbasefee
        )
    {
        number = block.number;
        difficulty = block.difficulty;
        time = block.timestamp;
        gaslimit = block.gaslimit;
        coinbase = block.coinbase;
        random = block.prevrandao;
        basefee = block.basefee;
        blobbasefee = block.blobbasefee;
    }
}
