// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// MisbehavingPriorityRegistry is a stand-in contract for Sonic's on-chain
// priority registry to be used as a replacement to the development registry
// used in integration tests. Like MagicValuePriority it classifies a
// transaction as prioritized iff its `value` field equals a fixed magic
// constant, but it can be told with setMode to fail delivering that
// classification to the node in the ways a real registry might.
contract MisbehavingPriorityRegistry {
    enum Mode {
        Correct, // delivers the classification as expected
        OutOfGas, // needs more gas than the node grants the call
        OutOfRange, // returns values exceeding the expected types
        Truncated, // returns fewer values than expected
        Reverting // does not return at all
    }

    // The current mode. Defaults to Mode.Correct.
    Mode public mode;

    // The exact value a tx must carry to be prioritized. Public so tests can
    // read it instead of duplicating it.
    uint256 public constant MAGIC_VALUE = 123456789;

    // A fixed non-zero level assigned to prioritized (magic-value) txs.
    uint64 constant PRIORITY_LEVEL = 1;

    // A fixed weight assigned to prioritized txs.
    uint64 constant PRIORITY_WEIGHT = 0;

    // A large per-block gas budget so rate-limiting cannot trim any
    // prioritized transaction during tests.
    uint256 constant PER_BLOCK_GAS = 1_000_000_000;

    // A large per-event tx budget so all prioritized events fit.
    uint256 constant PER_EVENT_TXS = 1_000;

    function setMode(Mode newMode) external {
        mode = newMode;
    }

    function getPriority(
        address /*from*/,
        address /*to*/,
        uint256 value,
        uint256 /*nonce*/,
        bytes calldata /*data*/,
        uint256 /*gas*/
    ) external view returns (uint64, uint64, uint128) {
        Mode current = mode;

        if (current == Mode.Reverting) {
            revert("getPriority always fails");
        }

        if (current == Mode.OutOfGas) {
            bytes32 burner;
            for (uint256 i = 0; i < 10_000; i++) {
                burner = keccak256(abi.encodePacked(burner, i));
            }
            // Reading the result keeps the loop from being optimized away.
            if (burner == bytes32(0)) {
                return (0, 0, 0);
            }
        }

        (uint64 level, uint64 weight, uint128 id) = classify(value);

        if (current == Mode.OutOfRange) {
            // The level, weight and id exceed the uint64, uint64 and uint128
            // the node expects.
            assembly {
                mstore(0x00, not(0))
                mstore(0x20, not(0))
                mstore(0x40, not(0))
                return(0x00, 0x60)
            }
        }

        if (current == Mode.Truncated) {
            // The id is missing from the response.
            assembly {
                mstore(0x00, level)
                mstore(0x20, weight)
                return(0x00, 0x40)
            }
        }

        return (level, weight, id);
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

    function classify(
        uint256 value
    ) internal pure returns (uint64 level, uint64 weight, uint128 id) {
        if (value == MAGIC_VALUE) {
            return (PRIORITY_LEVEL, PRIORITY_WEIGHT, 0);
        }
        return (0, 0, 0);
    }
}
