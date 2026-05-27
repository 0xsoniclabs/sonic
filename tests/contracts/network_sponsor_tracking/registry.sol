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
        uint256 chooseFundLimit,
        uint256 deductFeesLimit,
        uint256 overheadCharge,
        uint256 trackGasCost
    ) {
        chooseFundLimit = 100_000;
        deductFeesLimit = 60_000;
        overheadCharge = chooseFundLimit + deductFeesLimit + 50_000;
        trackGasCost = 60_000;
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
