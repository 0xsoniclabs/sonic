// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract BlockParameterSource {

    // Parameters lists all block-context parameters that are accessible within
    // the EVM. To be extended with additional parameters as needed.
    struct Parameters {
        uint256 chainId;
        uint256 number;
        uint256 time;
        address coinbase;
        uint256 gasLimit;
        uint256 baseFee;
        uint256 blobBaseFee;
        uint256 prevRandao;
    }

    event Log(Parameters parameters);

    function logBlockParameters() public {
        emit Log(getBlockParameters());
    }

    function getBlockParameters() public view returns (Parameters memory) {
        Parameters memory params = Parameters({
            chainId: block.chainid,
            number: block.number,
            time: block.timestamp,
            coinbase: block.coinbase,
            gasLimit: block.gaslimit,
            baseFee: block.basefee,
            blobBaseFee: block.blobbasefee,
            prevRandao: block.prevrandao
        });
        return params;
    }

}
