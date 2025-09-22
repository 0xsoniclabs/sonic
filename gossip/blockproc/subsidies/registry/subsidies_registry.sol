// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

contract SubsidiesRegistry {

    struct Pot {
      uint256 funds;
      uint256 totalContributions;
      mapping(address => uint256) contributors;
    }

    FeeBurner private constant feeBurner = FeeBurner(0xFC00FACE00000000000000000000000000000000);

    // Global pot for any transaction
    Pot public globalSponsorship;

    // From -> Pot
    mapping(address => Pot) public accountSponsorships;

    // From -> To -> Pot
    mapping(address => mapping(address => Pot)) public userSponsorships;

    // From -> To -> Function -> Pot
    mapping(address => mapping(address => mapping(bytes4 => Pot))) public callSponsorships;

    // To -> Function -> Pot
    mapping(address => mapping(bytes4 => Pot)) public serviceSponsorships;

    // To -> Pot
    mapping(address => Pot) public contractSponsorships;

    function sponsorGlobal() public payable{
        _addFunds(globalSponsorship, msg.sender, msg.value);
    }

    function withdrawGlobalSponsorship(uint256 amount) public {
        _withdrawFunds(globalSponsorship, msg.sender, amount);
    }

    function sponsorAccount(address from) public payable{
        _addFunds(accountSponsorships[from], msg.sender, msg.value);
    }

    function withdrawAccountSponsorship(address from, uint256 amount) public {
        _withdrawFunds(accountSponsorships[from], msg.sender, amount);
    }

    function sponsorUser(address from, address to) public payable{
        _addFunds(userSponsorships[from][to], msg.sender, msg.value);
    }

    function withdrawUserSponsorship(address from, address to, uint256 amount) public {
        _withdrawFunds(userSponsorships[from][to], msg.sender, amount);
    }

    function sponsorCall(address from, address to, bytes4 functionSelector) public payable{
        _addFunds(callSponsorships[from][to][functionSelector], msg.sender, msg.value);
    }

    function withdrawCallSponsorship(address from, address to, bytes4 functionSelector, uint256 amount) public {
        _withdrawFunds(callSponsorships[from][to][functionSelector], msg.sender, amount);
    }

    function sponsorService(address to, bytes4 functionSelector) public payable{
        _addFunds(serviceSponsorships[to][functionSelector], msg.sender, msg.value);
    }

    function withdrawServiceSponsorship(address to, bytes4 functionSelector, uint256 amount) public {
        _withdrawFunds(serviceSponsorships[to][functionSelector], msg.sender, amount);
    }

    function sponsorContract(address to) public payable{
        _addFunds(contractSponsorships[to], msg.sender, msg.value);
    }

    function withdrawContractSponsorship(address to, uint256 amount) public {
        _withdrawFunds(contractSponsorships[to], msg.sender, amount);
    }

    function isCovered(address from, address to, bytes4 functionSelector, uint256 fee) public view returns(bool){
        ( , bool exists) = _getPot(from, to, functionSelector, fee);
        return exists;
    }

    function deductFees(address from, address to, bytes4 functionSelector, uint256 fee) public {
        require(msg.sender == address(0)); // < only be called through internal transactions

        (Pot storage pot, bool exists) = _getPot(from, to, functionSelector, fee);
        require(exists, "No sponsorship pot available");
        require(pot.funds >= fee, "Not enough funds");
        feeBurner.burnNativeTokens{value: fee}();
        pot.funds -= fee;
    }

    function _getPot(address from, address to, bytes4 functionSelector, uint256 fee) internal view returns (Pot storage, bool) {
        if (callSponsorships[from][to][functionSelector].funds >= fee) {
            return (callSponsorships[from][to][functionSelector], true);
        }
        if (userSponsorships[from][to].funds >= fee) {
            return (userSponsorships[from][to], true);
        }
        if (accountSponsorships[from].funds >= fee) {
            return (accountSponsorships[from], true);
        }
        if (serviceSponsorships[to][functionSelector].funds >= fee) {
            return (serviceSponsorships[to][functionSelector], true);
        }
        if (contractSponsorships[to].funds >= fee) {
            return (contractSponsorships[to], true);
        }
        if (globalSponsorship.funds >= fee) {
            return (globalSponsorship, true);
        }
        return (globalSponsorship, false);
    }

    function _addFunds(Pot storage pot, address sponsor, uint256 amount) internal {
        pot.funds += amount;
        pot.contributors[sponsor] += amount;
        pot.totalContributions += amount;
    }

    function _withdrawFunds(Pot storage pot, address sponsor, uint256 amount) internal {
        require(tx.gasprice > 0, "Withdrawals are not supported through sponsored transactions");
        require(pot.contributors[sponsor] >= amount, "Not enough contributions to withdraw");
        uint256 share = (amount * pot.funds) / pot.totalContributions;
        require(share <= pot.funds, "Not enough available funds to withdraw");
        (bool success, ) = sponsor.call{value: share}("");
        require(success, "Transfer failed");
        pot.contributors[sponsor] -= amount;
        pot.totalContributions -= amount;
        pot.funds -= share;
    }

    // TODO: define policies for the following features
    // - Precedence of sponsorship types
    // - Admin functions
    // - Events
}

interface FeeBurner {
    function burnNativeTokens() external payable;
}
