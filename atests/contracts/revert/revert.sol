// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract RevertContract {
    event SideEffect(string message);

    function doRevert() public {
        // event is a side effect to avoid this function from being pure
        emit SideEffect("Before revert");

        revert("Reverted");
    }

    function doCrash() public {
        // event is a side effect to avoid this function from being pure
        emit SideEffect("Before crash");

        assembly {
            invalid()
        }
    }
}
