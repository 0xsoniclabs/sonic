// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// DataReader is a contract that reads data from the transaction data field.
// It can be used to test transactions with different data input sizes.
contract DataReader {
    event DataSize(uint64 size, uint64 bufferSize);

    function sendData(bytes memory data) public {
        // Emit both the size of the raw transaction data field and
        // the size of the data passed to this function.
        // This helps tuning the caller side to approach limits.
        emit DataSize(uint64(msg.data.length), uint64(data.length));
    }
}
