// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// DataReader is a contract that reads data from the transaction data field.
// It can be used to test transactions with different data input sizes.
contract DataReader {
    function sendData(bytes memory data) public {}
}
