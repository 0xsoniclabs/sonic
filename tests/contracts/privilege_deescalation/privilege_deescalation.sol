// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// This contract is used to test PrivilegeDeescalation usage of SetCode transactions,
// https://eips.ethereum.org/EIPS/eip-7702
//
// The code is for testing purposes and lacks of all the protections any production
// code should have.
contract PrivilegeDeescalation {
    address private authorizedAddress;

    function do_payment(address to, uint256 value) external {
        require(
            authorizedAddress == msg.sender,
            "only allowed addresses can transfer founds"
        );
        (bool success, ) = to.call{value: value}("");
        require(success, "call reverted");
    }

    function allow_payment(address account) external {
        require(
            address(this) == msg.sender,
            "only the own account can change access list"
        );
        authorizedAddress = account;
    }
}
