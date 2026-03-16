// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract RevertContract {
    event SideEffect(string message);
    uint256 private count = 0;


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

    function probabilisticRevert() public {
        // reverts a transaction during execution, depending on previous history
        // and the address of the sender. This revert should not be reliably
        // statically predictable.
        count++;
        uint256 rand = uint256(uint160(msg.sender)) | count;
        if (rand % 2 == 0) {
            revert("Probabilistic revert");
        }
    }
}
