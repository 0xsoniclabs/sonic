// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// This contract is used to test Batching usage of SetCode transactions,
// https://eips.ethereum.org/EIPS/eip-7702
//
// It is inspired by the example described in https://viem.sh/experimental/eip7702.
// The code is for testing purposes and lacks of all the protections any production
// code should have.
contract BatchCallDelegation {
    struct Call {
        address payable to;
        uint256 value;
    }

    function execute(Call[] calldata calls) external payable {
        for (uint256 i = 0; i < calls.length; i++) {
            Call memory call = calls[i];
            (bool success, ) = call.to.call{value: call.value}("");
            require(success, "call reverted");
        }
    }
}
