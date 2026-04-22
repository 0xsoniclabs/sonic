// SPDX-License-Identifier: MIT
pragma solidity ^0.8.4;

contract Store {
    int private count = 0;
    mapping(address => mapping(int => int)) private data;

    function put(int key, int value) public {
        data[msg.sender][key] = value;
        count++;
    }

    function fill(int from, int to, int value) public {
        for (int key = from; key < to; key++) {
            data[msg.sender][key] = value;
        }
        count++;
    }

    function get(int key) public view returns (int) {
        return data[msg.sender][key];
    }

    function getCount() public view returns (int) {
        return count;
    }
}
