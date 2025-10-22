// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

contract IndexedLogs {
    event Event1(uint256 id);
    event Event2(uint256 id);
    event Event3(uint256 id, string text);

    function emitEvents() public {
        for (uint256 i = 0; i < 5; i++) {
            emit Event1(i);
            emit Event2(i);
            emit Event3(i, "test string");
        }
    }
}