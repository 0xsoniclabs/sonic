// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract PrevRandao {
    event CurrentPrevRandao(uint256 prevrandao);


    function logCurrentPrevRandao() public {
        emit CurrentPrevRandao(block.prevrandao);
    }

    function getPrevRandao() public view returns (uint256) {
        return block.prevrandao;
    }

}
