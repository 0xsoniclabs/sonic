// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// FailingPostTxRegistry is a test registry that approves all sponsorships
// using mode 3 (network sponsored with on-chain tracking) so the txpool
// pre-check passes, but whose track() function always reverts.
// This allows integration tests to verify that a failing post-transaction
// causes the sponsored transaction to be rolled back under the Brio rules.
contract FailingPostTxRegistry {

    function getGasConfig() public pure returns (
        uint256 gasLimitForChooseFund,
        uint256 gasLimitForDeductFees,
        uint256 gasLimitForTrack,
        uint256 overheadChargeForFundBackedSponsorships,
        uint256 overheadChargeForNetworkSponsorshipsWithTracking
    ) {
        uint256 getGasConfigCosts = 50_000;
        gasLimitForChooseFund = 100_000;
        gasLimitForDeductFees = 60_000;
        gasLimitForTrack = 80_000;
        overheadChargeForFundBackedSponsorships = gasLimitForChooseFund + gasLimitForDeductFees + getGasConfigCosts;
        overheadChargeForNetworkSponsorshipsWithTracking = gasLimitForChooseFund + gasLimitForTrack + getGasConfigCosts;
    }

    function chooseFund(
        address /*from*/,
        address /*to*/,
        uint256 /*value*/,
        uint256 /*nonce*/,
        bytes calldata /*callData*/,
        uint256 /*fee*/
    ) public pure returns (uint256 mode, bytes32 payload) {
        // Mode 3: network sponsored with tracking.
        return (3, bytes32(uint256(0xdeadbeef)));
    }

    function deductFees(bytes32 /*fundId*/, uint256 /*fee*/) public pure {
        revert("deductFees should not be called for mode 3");
    }

    function track(bytes32 /*trackingId*/, uint256 /*fee*/) public pure {
        revert("track always fails");
    }
}
