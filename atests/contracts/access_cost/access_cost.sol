// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract AccessCost {
    uint256 value;
    event LogCost(uint256 cost);

    // This method measures the gas cost of accessing an addresss by
    // queriying its balance.
    // The cost of the access can be found in the LogCost event.
    function touchAddress(address addr) public {
        uint256 before = gasleft();
        uint256 v = addr.balance;
        emit LogCost(before - gasleft());

        // store it, to prevent optimizations and warnings
        value = v;
    }

    function touchCoinBase() public {
        touchAddress(getCoinBaseAddress());
    }

    function touchOrigin() public {
        touchAddress(getOrigin());
    }

    function getOrigin() public view returns (address) {
        return tx.origin;
    }

    function getCoinBaseAddress() public view returns (address) {
        return block.coinbase;
    }

    function getAddressAccessCost(
        address addr
    ) public view returns (uint256, uint256) {
        uint256 before = gasleft();
        uint256 b = addr.balance;
        uint256 cost = before - gasleft();
        return (b, cost);
    }
}
