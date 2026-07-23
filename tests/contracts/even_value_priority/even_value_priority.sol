// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// EvenValuePriority is a stand-in contract for Sonic's on-chain priority
// registry to be used as a replacement to the development registry used in
// integration tests. It classifies a transaction as prioritized iff its
// `value` field is even, so tests can exercise the classifier on the
// consensus path by submitting a mix of even- and odd-value transactions
// and observing that only the even ones are hoisted to the front of a block.
contract EvenValuePriority {
    // A fixed non-zero level assigned to prioritized (even-value) txs.
    uint64 constant PRIORITY_LEVEL = 1;

    // A fixed weight assigned to prioritized txs.
    uint64 constant PRIORITY_WEIGHT = 0;

    // A large per-block gas budget so rate-limiting cannot trim any
    // prioritized transaction during tests.
    uint256 constant PER_BLOCK_GAS = 1_000_000_000;

    // A large per-event tx budget so all prioritized events fit.
    uint256 constant PER_EVENT_TXS = 1_000;

    function getPriority(
        address /*from*/,
        address /*to*/,
        uint256 value,
        uint256 /*nonce*/,
        bytes calldata /*data*/,
        uint256 /*gas*/
    ) external pure returns (uint64 level, uint64 weight, uint128 id) {
        if (value % 2 == 0) {
            return (PRIORITY_LEVEL, PRIORITY_WEIGHT, 0);
        }
        return (0, 0, 0);
    }

    function getPriorityConfig()
        external
        pure
        returns (
            uint256 maxGasPerEntityPerBlock,
            uint256 maxPiggybackTxsPerEntityPerEvent
        )
    {
        return (PER_BLOCK_GAS, PER_EVENT_TXS);
    }
}
