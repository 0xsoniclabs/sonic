// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// NoncePriorityRegistry is a stand-in for Sonic's on-chain transaction priority registry that
// answers per transaction rather than per sender: it keys a priority on the (sender, nonce) pair,
// the only pair among getPriority's arguments that names one transaction of a sender. The
// transaction property tests replace the development registry with this one so that two
// transactions of the same sender can carry different priorities, which is what the per-sender
// nonce ordering of the block formation ordering step is decided by.
//
// A slot nobody registered answers level zero, which is what "not prioritized" is.
contract NoncePriorityRegistry {
    struct Priority {
        uint64 level;
        uint64 weight;
        uint128 id;
    }

    mapping(address => mapping(uint256 => Priority)) private priorities;

    uint256 private maxGasPerEntityPerBlockValue;
    uint256 private maxPiggybackTxsPerEntityPerEventValue;

    // --- configuration (test helpers) ---

    // setPriorities registers the priorities of the consecutive nonce slots firstNonce,
    // firstNonce + 1, ... of one sender, so a whole batch's worth of slots costs one transaction.
    function setPriorities(
        address from,
        uint256 firstNonce,
        Priority[] calldata window
    ) external {
        for (uint256 i = 0; i < window.length; i++) {
            priorities[from][firstNonce + i] = window[i];
        }
    }

    function setConfig(uint256 perBlockGas, uint256 perEventTxs) external {
        maxGasPerEntityPerBlockValue = perBlockGas;
        maxPiggybackTxsPerEntityPerEventValue = perEventTxs;
    }

    // --- interface consumed by the Sonic client ---

    function getPriority(
        address from,
        address /*to*/,
        uint256 /*value*/,
        uint256 nonce,
        bytes calldata /*data*/,
        uint256 /*gas*/
    ) external view returns (uint64 level, uint64 weight, uint128 id) {
        Priority storage p = priorities[from][nonce];
        return (p.level, p.weight, p.id);
    }

    function getPriorityConfig()
        external
        view
        returns (
            uint256 maxGasPerEntityPerBlock,
            uint256 maxPiggybackTxsPerEntityPerEvent
        )
    {
        maxGasPerEntityPerBlock = maxGasPerEntityPerBlockValue;
        maxPiggybackTxsPerEntityPerEvent = maxPiggybackTxsPerEntityPerEventValue;
    }
}
