// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

// PriorityRegistry is a stand-in contract for Sonic's on-chain transaction
// priority registry to be used in local testing and development environments.
//
// For each transaction the node queries `getPriority`, which returns a
// (level, weight, id) triple:
//   - level:  0 = no priority; > 0 = prioritized (higher level scheduled first).
//   - weight: tie-breaker within a level (higher first).
//   - id:     entity identifier used for per-entity rate limiting.
//
// `getPriorityConfig` returns the per-entity rate limits enforced by the node
// during block formation and event emission.
//
// This stand-in is storage-configurable so tests can register priorities and
// limits. Production registries are governed and upgradeable behind a proxy;
// the node depends only on the ABI shape, not on this implementation.
contract PriorityRegistry {

    struct Priority {
        uint256 level;
        uint256 weight;
        bytes32 id;
    }

    // Priority assigned to transactions by sender. Configurable for testing.
    mapping(address => Priority) public senderPriority;

    // Transactions whose gas limit exceeds maxGas are never prioritized.
    // A value of zero disables the gas filter.
    uint256 public maxGas;

    // Per-entity rate limits. Zero selects the built-in defaults below.
    uint256 private maxTxsPerEntityPerBlockValue;
    uint256 private maxTxsPerEntityPerEventValue;

    uint256 constant DEFAULT_MAX_PER_BLOCK = 16;
    uint256 constant DEFAULT_MAX_PER_EVENT = 4;

    // --- configuration (test/development helpers) ---

    function setSenderPriority(address from, uint256 level, uint256 weight, bytes32 id) external {
        senderPriority[from] = Priority(level, weight, id);
    }

    function setMaxGas(uint256 g) external {
        maxGas = g;
    }

    function setConfig(uint256 perBlock, uint256 perEvent) external {
        maxTxsPerEntityPerBlockValue = perBlock;
        maxTxsPerEntityPerEventValue = perEvent;
    }

    // --- interface consumed by the Sonic client ---

    function getPriority(
        address from,
        address /*to*/,
        uint256 /*value*/,
        uint256 /*nonce*/,
        bytes calldata /*data*/,
        uint256 gas
    ) external view returns (uint256 level, uint256 weight, bytes32 id) {
        if (maxGas != 0 && gas > maxGas) {
            return (0, 0, bytes32(0));
        }
        Priority storage p = senderPriority[from];
        return (p.level, p.weight, p.id);
    }

    function getPriorityConfig() external view returns (
        uint256 maxTxsPerEntityPerBlock,
        uint256 maxTxsPerEntityPerEvent
    ) {
        maxTxsPerEntityPerBlock = maxTxsPerEntityPerBlockValue == 0
            ? DEFAULT_MAX_PER_BLOCK : maxTxsPerEntityPerBlockValue;
        maxTxsPerEntityPerEvent = maxTxsPerEntityPerEventValue == 0
            ? DEFAULT_MAX_PER_EVENT : maxTxsPerEntityPerEventValue;
    }
}
