// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract RevertContract {
    event SideEffect(string message);
    int private count = 0;


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
        count = count + 1;
        if (count % 2 == 0) {
            uint8 rand = uint8(bytes20(tx.origin)[0]);
            if (rand % 2 == 0) {
                revert("Probabilistic revert");
            }
        }
    }
}
