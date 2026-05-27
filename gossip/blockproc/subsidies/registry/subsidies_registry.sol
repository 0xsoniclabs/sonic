// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

// SubsidiesRegistry is a stand-in contract for Sonic's on-chain subsidies
// registry to be used in local testing and development environments.
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

    // Allows a sponsor to add funds to a specific fund identified by its ID.
    // The contributed sponsorship amount becomes available for sponsored
    // transactions. Remaining sponsorship funds may be withdrawn by sponsors
    // at any time using the `withdraw` function.
    function sponsor(bytes32 fundId) public payable {
        Fund storage fund = sponsorships[fundId];
        fund.funds += msg.value;
        fund.contributors[msg.sender] += msg.value;
        fund.totalContributions += msg.value;
    }

    // Allows a sponsor to withdraw their contributions from a fund
    // proportionally to their share of total contributions.
    // TODO: this policy allows past sponsors to consume fresh funds added by
    // other sponsors. Think about a better policy preventing this.
    function withdraw(bytes32 fundId, uint256 amount) public {
        require(tx.gasprice > 0, "Withdrawals are not supported through sponsored transactions");

        Fund storage fund = sponsorships[fundId];
        address payable contributor = payable(msg.sender);
        require(fund.contributors[contributor] >= amount, "Not enough contributions to withdraw");

        // Scale the withdrawal amount based on the current fund balance.
        uint256 share = (amount * fund.funds) / fund.totalContributions;
        require(share <= fund.funds, "Not enough available funds to withdraw");

        // Re-entrance protection: update state before transfer
        fund.contributors[contributor] -= amount;
        fund.totalContributions -= amount;
        fund.funds -= share;

        // Transfer the share to the contributor
        contributor.transfer(share);
    }

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

    // --- Funding infrastructure used by the Sonic client ---

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
        address to,
        uint256 /*value*/,
        uint256 nonce,
        bytes calldata callData,
        uint256 fee
    ) public view returns (uint256 mode, bytes32 payload) {
        // Check all possible sponsorship funds in order of precedence.
        bool covered;
        bytes32 fundId;
        (covered, fundId) = approvalSponsorshipFundId(from, to, callData);
        if (covered && sponsorships[fundId].funds >= fee) {
            return (MODE_FUND_BACKED, fundId);
        }
        (covered, fundId) = callSponsorshipFundId(from, to, callData);
        if (covered && sponsorships[fundId].funds >= fee) {
            return (MODE_FUND_BACKED, fundId);
        }
        (covered, fundId) = accountSponsorshipFundId(from);
        if (covered && sponsorships[fundId].funds >= fee) {
            return (MODE_FUND_BACKED, fundId);
        }
        (covered, fundId) = contractSponsorshipFundId(to);
        if (covered && sponsorships[fundId].funds >= fee) {
            return (MODE_FUND_BACKED, fundId);
        }
        (covered, fundId) = bootstrapSponsorshipFund(nonce);
        if (covered && sponsorships[fundId].funds >= fee) {
            return (MODE_FUND_BACKED, fundId);
        }
        (covered, fundId) = globalSponsorshipFundId();
        if (covered && sponsorships[fundId].funds >= fee) {
            return (MODE_FUND_BACKED, fundId);
        }
        // No sponsorship found to cover the fee.
        return (MODE_NOT_COVERED, bytes32(0));
    }

    function deductFees(bytes32 fundId, uint256 fee) public {
        require(msg.sender == address(0)); // < only be called through internal transactions
        require(fundId != bytes32(0), "No sponsorship fund chosen");
        Fund storage fund = sponsorships[fundId];
        require(fund.funds >= fee, "Not enough funds");
        feeBurner.burnNativeTokens{value: fee}();
        fund.funds -= fee;
    }

    // track is called after a network-sponsored transaction with tracking (mode 3).
    // The trackingId identifies the sponsorship context; fee is the gas cost that
    // would have been charged. In this test registry the call is a no-op; production
    // registries use it to enforce on-chain rate limits and quotas.
    function track(bytes32 /*trackingId*/, uint256 /*fee*/) public view {
        require(msg.sender == address(0)); // only callable through internal transactions
    }

    // --- Fund Identifiers ---

    // Global sponsorships cover all transactions. They may be used for
    // Sonic wide marketing campaigns.
    function globalSponsorshipFundId() public pure returns (bool, bytes32) {
        return (true, keccak256(abi.encodePacked("g")));
    }

    // Account sponsorships cover all transactions sent from a specific
    // account. All sponsorship requests from this account will be covered.
    function accountSponsorshipFundId(address from) public pure returns (bool, bytes32) {
        return (true, keccak256(abi.encodePacked("a", from)));
    }

    // Contract sponsorships cover all transactions sent to a specific
    // contract. All sponsorship requests for transactions targeting this
    // contract will be covered.
    function contractSponsorshipFundId(address to) public pure returns (bool, bytes32) {
        return (true, keccak256(abi.encodePacked("c", to)));
    }

    // Call sponsorships cover all transactions calling a specific
    // function on a specific contract.
    function callSponsorshipFundId(address from, address to, bytes calldata callData) public pure returns (bool, bytes32) {
        // Ignore create contract calls (to is zero address) and calls with too short
        // call data (less than 4 bytes, not covering the function selector).
        if (to == address(0) || callData.length < 4) {
            return (false, bytes32(0));
        }
        bytes4 selector = bytes4(callData[:4]);
        return (true, keccak256(abi.encodePacked("c", from, to, selector)));
    }

    // Approval sponsorships cover all ERC20 approve calls from a specific
    // account to a specific token contract and spender with a non-zero
    // approval amount.
    function approvalSponsorshipFundId(address from, address to, bytes calldata callData) public pure returns (bool, bytes32) {
        if (to == address(0) || callData.length != 2*32+4) {
            return (false, bytes32(0));
        }
        bytes4 selector = bytes4(callData[:4]);
        if (selector != 0x095ea7b3) { // ERC20 approve
            return (false, bytes32(0));
        }
        (address spender, uint256 value) = abi.decode(callData[4:], (address, uint256));
        if (value < 1) { // we do not sponsor zero-amount approvals
            return (false, bytes32(0));
        }
        return (true, keccak256(abi.encodePacked("a", from, to, spender)));
    }

    // Bootstrap sponsorships cover the first few transactions from a new
    // account. This allows new users to get started without having to
    // acquire native tokens first.
    function bootstrapSponsorshipFund(uint256 nonce) public pure returns (bool, bytes32) {
        if (nonce < 3) {
            return (true, keccak256(abi.encodePacked("b")));
        }
        return (false, bytes32(0));
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
