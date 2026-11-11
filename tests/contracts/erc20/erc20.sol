// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// A minimal ERC-20 token, https://eips.ethereum.org/EIPS/eip-20
// The entire supply is created for the deployer. The total supply is the sum
// of the balances of all holders, so that it reflects every balance update.
contract ERC20 {
    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;
    address[] holders;
    mapping(address => bool) isHolder;

    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);

    constructor(uint256 supply) {
        _addHolder(msg.sender);
        balanceOf[msg.sender] = supply;
        emit Transfer(address(0), msg.sender, supply);
    }

    function totalSupply() external view returns (uint256 sum) {
        for (uint256 i = 0; i < holders.length; i++) {
            sum += balanceOf[holders[i]];
        }
    }

    function transfer(address to, uint256 value) external returns (bool) {
        _transfer(msg.sender, to, value);
        return true;
    }

    function approve(address spender, uint256 value) external returns (bool) {
        allowance[msg.sender][spender] = value;
        emit Approval(msg.sender, spender, value);
        return true;
    }

    function transferFrom(address from, address to, uint256 value) external returns (bool) {
        allowance[from][msg.sender] -= value;
        _transfer(from, to, value);
        return true;
    }

    function _transfer(address from, address to, uint256 value) internal {
        balanceOf[from] -= value;
        balanceOf[to] += value;
        _addHolder(to);
        emit Transfer(from, to, value);
    }

    function _addHolder(address account) internal {
        if (!isHolder[account]) {
            isHolder[account] = true;
            holders.push(account);
        }
    }
}
