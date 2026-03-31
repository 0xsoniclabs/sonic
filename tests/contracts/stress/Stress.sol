// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Stress {
    event ComputationDone(uint256 result);

    // This function performs a computationally intensive task by repeatedly hashing
    // The number of rounds has to be carefully chosen if not running out of gas
    // is desired.
    function computeHeavySum(uint256 rounds) public returns (uint256) {
        bytes32 hash = keccak256(abi.encodePacked(uint256(0)));
        for (uint256 i = 1; i <= rounds; i++) {
            hash = keccak256(abi.encodePacked(hash, i));
        }
        emit ComputationDone(uint256(hash));
        return uint256(hash);
    }
}
