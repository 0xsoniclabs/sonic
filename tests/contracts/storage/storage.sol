// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract Storage {
    int public a = 1;
    int public b = 2;
    int public c = 3;

    function getA() public view returns (int) {
        return a;
    }

    function getB() public view returns (int) {
        return b;
    }

    function getC() public view returns (int) {
        return c;
    }

     function sumABC() public view returns (int){
         return a + b + c;
     }
}
