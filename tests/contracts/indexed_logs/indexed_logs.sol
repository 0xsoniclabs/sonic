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

    event Log1();
    event Log2(uint256 indexed a);
    event Log3(uint256 indexed a, uint256 indexed b);
    event Log4(uint256 indexed a, uint256 indexed b, uint256 indexed c);

    // Emits a Cartesian product of logs with up to three indexed parameters.
    function emitCartesianProduct(uint256 n) public {
        assembly {
            log0(0, 0) // emit Log0() with no topics
        }
        emit Log1(); // log1, using the event id as topic 1
        for (uint256 i = 0; i < n; i++) {
            emit Log2(i);
            for (uint256 j = 0; j < n; j++) {
                emit Log3(i, j);
                for (uint256 k = 0; k < n; k++) {
                    emit Log4(i, j, k);
                }
            }
        }
    }

    // A simple function to emit a given number of log messages.
    function emitLogs(uint256 n) public {
        for (uint256 i = 0; i < n; i++) {
            emit Log2(i);
        }
    }
}