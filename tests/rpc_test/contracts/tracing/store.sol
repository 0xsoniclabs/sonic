// SPDX-License-Identifier: MIT

pragma solidity ^0.8.26;

contract Storage {

    uint256 number;

    event TestLog(uint256 indexed num);

    function store(uint256 num) public {
        number = num;
        emit TestLog(num);
    }


    function retrieve() public view returns (uint256){
        return number;
    }
}