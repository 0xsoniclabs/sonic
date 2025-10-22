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
