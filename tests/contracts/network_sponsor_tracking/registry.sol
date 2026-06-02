// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// NetworkSponsorTrackingRegistry is a test registry that sponsors all
// transactions using mode 3 (network sponsored with on-chain tracking).
// It emits a Track event so tests can verify the trackingId and fee.
contract NetworkSponsorTrackingRegistry {

    event Tracked(bytes32 indexed trackingId, uint256 fee);

    // Fixed tracking ID returned to all chooseFund callers, so tests can
    // assert that the same ID arrives in the track() call.
    bytes32 public constant TRACKING_ID = bytes32(uint256(0xdeadbeef));

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
        return (3, TRACKING_ID);
    }

    function deductFees(bytes32 /*fundId*/, uint256 /*fee*/) public pure {
        revert("deductFees should not be called for mode 3");
    }

    function track(bytes32 trackingId, uint256 fee) public {
        require(msg.sender == address(0), "only internal transactions");
        emit Tracked(trackingId, fee);
    }
}
