// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract BlobBaseFee {
    event CurrentBlobBaseFee(uint256 fee);

    function logCurrentBlobBaseFee() public {
        emit CurrentBlobBaseFee(block.blobbasefee);
    }

    function getBlobBaseFee() public view returns (uint256) {
        return block.blobbasefee;
    }
}
