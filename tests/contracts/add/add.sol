// SPDX-License-Identifier: MIT
pragma solidity ^0.8.4;

contract Add {
    function add(uint iter) public pure returns (uint) {
        uint count = 0;
        for (uint i = 0; i < iter; i++) {
            count++;
        }
        return count;
    }
}
