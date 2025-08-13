// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

contract DataStore {
    bytes32 data;
    uint256 data2;

    function storeData(bytes32 _data, uint256 _data2) public {
        data = _data;
        data2 = _data2;
    }

    function getData() public view returns (bytes32, uint256) {
        return (data, data2);
    }
}