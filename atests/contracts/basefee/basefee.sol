// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract BaseFee {

    event CurrentFee(uint256 fee);

    function logCurrentBaseFee() public {
        emit CurrentFee(block.basefee);
    }

    function getBaseFee() public view returns (uint256) {
        return block.basefee;
    }
}