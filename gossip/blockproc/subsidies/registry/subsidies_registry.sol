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

    // --- Funding infrastructure used by the Sonic client ---

    function chooseFund(
        address from,
        address to,
        uint256 /*value*/,
        uint256 nonce,
        bytes calldata callData,
        uint256 fee
    ) public view returns (bytes32 fundId) {
        // Check all possible sponsorship funds in order of precedence.
        bool covered;
        (covered, fundId) = approvalSponsorshipFundId(from, to, callData);
        if (covered && sponsorships[fundId].funds >= fee) {
            return fundId;
        }
        (covered, fundId) = callSponsorshipFundId(from, to, callData);
        if (covered && sponsorships[fundId].funds >= fee) {
            return fundId;
        }
        (covered, fundId) = accountSponsorshipFundId(from);
        if (covered && sponsorships[fundId].funds >= fee) {
            return fundId;
        }
        (covered, fundId) = contractSponsorshipFundId(to);
        if (covered && sponsorships[fundId].funds >= fee) {
            return fundId;
        }
        (covered, fundId) = bootstrapSponsorshipFund(nonce);
        if (covered && sponsorships[fundId].funds >= fee) {
            return fundId;
        }
        (covered, fundId) = globalSponsorshipFundId();
        if (covered && sponsorships[fundId].funds >= fee) {
            return fundId;
        }
        // No sponsorship found to cover the fee, returning the 0 fund.
        return bytes32(0);
    }

    function deductFees(bytes32 fundId, uint256 fee) public {
        require(msg.sender == address(0)); // < only be called through internal transactions
        require(fundId != bytes32(0), "No sponsorship fund chosen");
        Fund storage fund = sponsorships[fundId];
        require(fund.funds >= fee, "Not enough funds");
        feeBurner.burnNativeTokens{value: fee}();
        fund.funds -= fee;
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
