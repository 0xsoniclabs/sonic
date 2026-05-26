// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// NetworkSponsorRegistry is a test registry that sponsors all transactions
// using mode 2 (network sponsored, no on-chain tracking).
contract NetworkSponsorRegistry {

    function getGasConfig() public pure returns (
        uint256 chooseFundLimit,
        uint256 deductFeesLimit,
        uint256 overheadCharge,
        uint256 trackGasCost
    ) {
        chooseFundLimit = 100_000;
        deductFeesLimit = 60_000;
        overheadCharge = chooseFundLimit + deductFeesLimit + 50_000;
        trackGasCost = 0;
    }

    function chooseFund(
        address /*from*/,
        address /*to*/,
        uint256 /*value*/,
        uint256 /*nonce*/,
        bytes calldata /*callData*/,
        uint256 /*fee*/
    ) public pure returns (uint256 mode, bytes32 payload) {
        // Mode 2: network sponsored, no deduction, no tracking.
        return (2, bytes32(0));
    }

    function deductFees(bytes32 /*fundId*/, uint256 /*fee*/) public pure {
        revert("deductFees should not be called for mode 2");
    }

    function track(bytes32 /*trackingId*/, uint256 /*fee*/) public pure {
        revert("track should not be called for mode 2");
    }
}
