// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

contract B {
    uint public num;


    function setVars(uint _num) public payable {
        num = _num;
    }

    function getNum() public view returns (uint){
        return num;
    }
}

contract A {
    B public bContract;
    
    event Log(string message);

    constructor() {
        bContract = new B();
    }

    function InnerCall(uint _num) public payable {

        // Normal inner call
        bContract.setVars(_num);

    }
}