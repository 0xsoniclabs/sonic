// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract BlockOverride {
    event BlockNumber(uint256 currentBlock);

    function logBlockNumber() public {
        emit BlockNumber(block.number);
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
