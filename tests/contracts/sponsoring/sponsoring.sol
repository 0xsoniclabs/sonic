// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// This contract is used to test Sponsoring usage of SetCode transactions,
// https://eips.ethereum.org/EIPS/eip-7702
//
// The code is for testing purposes and lacks of all the protections any production
// code should have.
contract SponsoringDelegate {
    function execute(
        address payable to,
        uint256 value,
        bytes calldata data
    ) external payable {
        // execute forwards the call to the provided address
        //  with the provided data and transferring the specified value
        (bool success, ) = to.call{value: value}(data);
        require(success, "call reverted");
    }
}
