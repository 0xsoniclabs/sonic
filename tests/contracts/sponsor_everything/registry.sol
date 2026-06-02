// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// SubsidiesRegistry is a stand-in contract for Sonic's on-chain subsidies
// registry to be used as a replacement to the development registry used in
// integration tests.
contract SubsidiesRegistry {

    // A fund tracks the total funds available for an individual sponsorship
    // grant. A fund tracks the remaining funds available for sponsorships and
    // the contributions made by individual sponsors.
    struct Fund {
      uint256 funds;
      uint256 totalContributions;
      mapping(address => uint256) contributors;
    }

    // All available sponsorship funds identified by an ID.
    mapping(bytes32 id => Fund) public sponsorships;

    // --- Functions for sponsors to add and withdraw funds ---

    // sponsor allows anyone to add funds to the sponsorship fund.
    function sponsor(bytes32 fundId) public payable {
        Fund storage fund = sponsorships[fundId];
        fund.funds += msg.value;
        fund.contributors[msg.sender] += msg.value;
        fund.totalContributions += msg.value;
    }

    // --- Funding infrastructure used by the Sonic client ---

    function getGasConfig() public pure returns (
        uint256 chooseFundLimit,
        uint256 deductFeesLimit,
        uint256 traceLimit,
        uint256 fundBackedOverheadCharge,
        uint256 networkTrackedOverheadCharge
    ) {
        uint256 getGasConfigCosts = 50_000;
        chooseFundLimit = 1_234_567;  // < different from default
        deductFeesLimit = 654_321;    // < different from default
        traceLimit = 707_707;         // < different from default
        fundBackedOverheadCharge = chooseFundLimit + deductFeesLimit + getGasConfigCosts;
        networkTrackedOverheadCharge = chooseFundLimit + traceLimit + getGasConfigCosts;
    }

    function chooseFund(
        address /*from*/,
        address /*to*/,
        uint256 /*value*/,
        uint256 /*nonce*/,
        bytes calldata /*callData*/,
        uint256 fee
    ) public view returns (uint256 mode, bytes32 fundId) {
        // Everything is funded if there is enough balance to cover the fee.
        if (address(this).balance >= fee) {
            return (1, bytes32(uint256(1)));
        }
        return (0, bytes32(0));
    }

    function deductFees(bytes32 fundId, uint256 fee) public {
        require(msg.sender == address(0)); // < only be called through internal transactions
        require(fundId != bytes32(0), "No sponsorship fund chosen");
        require(address(this).balance >= fee, "Not enough funds");
        feeBurner.burnNativeTokens{value: fee}();
    }

    function track(bytes32 /*trackingId*/, uint256 /*fee*/) public view {
        require(msg.sender == address(0)); // only callable through internal transactions
    }

    // --- Sponsor Policies ---

    // This type of sponsorship is used by unit tests.
    function accountSponsorshipFundId(address /*from*/) public pure returns (bool, bytes32) {
        return (true, bytes32(uint256(1)));
    }

    // --- Internal functions ---

    // Address of the FeeBurner contract used to burn native tokens.
    // In this contract, this is a hardcoded constant referring to the SFC.
    FeeBurner private constant feeBurner = FeeBurner(0xFC00FACE00000000000000000000000000000000);
}

// Minimal interface for the FeeBurner contract used to burn native tokens. This
// interface is required to be implemented by the SFC.
interface FeeBurner {
    function burnNativeTokens() external payable;
}
