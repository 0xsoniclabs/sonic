// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

contract SubsidiesRegistry {

    struct Pot {
      uint256 funds;
      uint256 totalContributions;
      mapping(address => uint256) contributors;
    }

    FeeBurner feeBurner;

    // From -> To -> Deposit Amount
    mapping(address => mapping(address => Pot)) public userSponsorships;

    // To -> Function Selector -> Deposit Amount
    mapping(address => mapping(bytes4 => Pot)) public operationSponsorships;

    // To -> Deposit Amount
    mapping(address => Pot) public contractSponsorships;

    constructor(FeeBurner feeBurner_) {
        feeBurner = feeBurner_;
    }

    function sponsorUser(address from, address to) public payable{
        _addFunds(userSponsorships[from][to], msg.sender, msg.value);
    }

    function withdrawUserSponsorship(address from, address to, uint256 amount) public {
        _withdrawFunds(userSponsorships[from][to], msg.sender, amount);
    }

    function sponsorMethod(address to, bytes4 functionSelector) public payable{
        _addFunds(operationSponsorships[to][functionSelector], msg.sender, msg.value);
    }

    function withdrawMethodSponsorship(address to, bytes4 functionSelector, uint256 amount) public {
        _withdrawFunds(operationSponsorships[to][functionSelector], msg.sender, amount);
    }

    function sponsorContract(address to) public payable{
        _addFunds(contractSponsorships[to], msg.sender, msg.value);
    }

    function withdrawContractSponsorship(address to, uint256 amount) public {
        _withdrawFunds(contractSponsorships[to], msg.sender, amount);
    }

    function isCovered(address from, address to, bytes4 functionSelector, uint256 fee) public view returns(bool){
        if(userSponsorships[from][to].funds >= fee){
            return true;
        }
        if(operationSponsorships[to][functionSelector].funds >= fee){
            return true;
        }
        if(contractSponsorships[to].funds >= fee){
            return true;
        }
        return false;
    }

    function deductFees(address from, address to, bytes4 functionSelector, uint256 fee) public {
        require(msg.sender == address(0)); // < only be called through internal transactions
        require(isCovered(from, to, functionSelector, fee));
        feeBurner.burnNativeTokens{value: fee}();
        if(userSponsorships[from][to].funds >= fee){
            userSponsorships[from][to].funds -= fee;
            return;
        }
        if(operationSponsorships[to][functionSelector].funds >= fee){
            operationSponsorships[to][functionSelector].funds -= fee;
            return;
        }
        if(contractSponsorships[to].funds >= fee){
            contractSponsorships[to].funds -= fee;
            return;
        }
    }

    function _addFunds(Pot storage pot, address sponsor, uint256 amount) internal {
        pot.funds += amount;
        pot.contributors[sponsor] += amount;
        pot.totalContributions += amount;
    }

    function _withdrawFunds(Pot storage pot, address sponsor, uint256 amount) internal {
        require(pot.contributors[sponsor] >= amount, "Not enough contributions to withdraw");
        uint256 share = (amount * pot.funds) / pot.totalContributions;
        (bool success, ) = sponsor.call{value: share}("");
        require(success, "Transfer failed");
        pot.contributors[sponsor] -= amount;
        pot.totalContributions -= amount;
        pot.funds -= share;
    }

    receive() external payable {
        require(false, "Use sponsor functions to add funds");
    }

    // TODO: define policies for the following features
    // - Precedence of sponsorship types
    // - Admin functions
    // - Events
}

interface FeeBurner {
    function burnNativeTokens() external payable;
}
