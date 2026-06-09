// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// NetworkSponsorRegistry is a test registry that can be configured to sponsor
// transactions from specific accounts using:
// - mode 1 (fund-backed)
// - mode 2 (network sponsored, no on-chain tracking)
// - mode 3 (network sponsored with on-chain tracking)

    // --- Sponsorship mode constants ---

    // MODE_NOT_COVERED is returned by chooseFund when no sponsorship covers the transaction.
    uint256 constant MODE_NOT_COVERED = 0;

    // MODE_FUND_BACKED is returned by chooseFund when a sponsor fund covers the gas fee.
    // The payload is the fundId to pass to deductFees.
    uint256 constant MODE_FUND_BACKED = 1;

    // MODE_NETWORK_SPONSORED is returned by chooseFund when the network absorbs the gas cost
    // directly. No post-execution transaction is inserted; the payload is ignored.
    uint256 constant MODE_NETWORK_SPONSORED = 2;

    // MODE_NETWORK_SPONSORED_WITH_TRACKING is returned by chooseFund when the network absorbs
    // the gas cost and the registry is notified via track(trackingId, fee) afterward.
    // The payload is the trackingId passed to track.
    uint256 constant MODE_NETWORK_SPONSORED_WITH_TRACKING = 3;


    uint256 constant ACCEPTED_FUND_ID = 1;
    uint256 constant ACCEPTED_TRACKING_ID = 2;

    // --- Configurable Sponsorship Modes ---

    mapping(address => bool) isFundSponsored;
    mapping(address => bool) isNetworkSponsored;
    mapping(address => bool) isNetworkSponsoredWithTrackingSponsored;

    // --- Configuration of Sponsorship Modes ---

    function setFundSponsored(address sponsored, bool value) public {
        isFundSponsored[sponsored] = value;
    }

    function setNetworkSponsored(address sponsored, bool value) public {
        isNetworkSponsored[sponsored] = value;
    }

    function setNetworkSponsoredWithTrackingSponsored(address sponsored, bool value) public {
        isNetworkSponsoredWithTrackingSponsored[sponsored] = value;
    }

    // --- Required interface ---

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
        address from,
        address /*to*/,
        uint256 /*value*/,
        uint256 /*nonce*/,
        bytes calldata /*callData*/,
        uint256 /*fee*/
    ) public view returns (uint256 mode, bytes32 payload) {
        if (isNetworkSponsored[from]) {
            return (MODE_NETWORK_SPONSORED, bytes32(0));
        } else if (isFundSponsored[from]) {
            return (MODE_FUND_BACKED, bytes32(ACCEPTED_FUND_ID));
        } else if (isNetworkSponsoredWithTrackingSponsored[from]) {
            return (MODE_NETWORK_SPONSORED_WITH_TRACKING, bytes32(ACCEPTED_TRACKING_ID));
        } else {
            return (MODE_NOT_COVERED, bytes32(0));
        }
    }

    function deductFees(bytes32 fundId, uint256 /*fee*/) public pure {
        if (fundId == bytes32(ACCEPTED_FUND_ID)) {
            // Accept the deduction without doing anything.
        } else {
            revert("deductFees: unknown fundId");
        }
    }

    function track(bytes32 trackingId, uint256 /*fee*/) public pure {
        if (trackingId == bytes32(ACCEPTED_TRACKING_ID)) {
            // Accept the tracking without doing anything.
        } else {
            revert("track: unknown trackingId");
        }
    }
}
