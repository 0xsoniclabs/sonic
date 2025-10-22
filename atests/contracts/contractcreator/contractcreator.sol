// SPDX-License-Identifier: MIT
pragma solidity ^0.8.8;

contract ContractCreator {
    event LogCost(uint256 cost);
    
    function CreatetWith(uint codeSize) public {
        // save gas before attempting to create
        uint256 before = gasleft();
        uint256 result = 0;
        assembly {
            // in these assembly blocks, results from function calls cannot be ignored
            // so we assign it to result even if it is not used after. 
            result := create(0, 0, codeSize)
        }
        // report how much gas was used during create call
        emit LogCost(before - gasleft());
    }

    function Create2With(uint codeSize) public {
        // save gas before attempting to create
        uint256 before = gasleft();
        uint256 result = 0;
        assembly {
            // in these assembly blocks, results from function calls cannot be ignored
            // so we assign it to result even if it is not used after. 
            result := create2(0, 0, codeSize, 0)
        }
        // report how much gas was used during create call
        emit LogCost(before - gasleft());
    }

    // GetOverheadCost is only used to measure the cost of creating a variable and assign to it.
    function GetOverheadCost(uint someValue) public {
        uint256 before = gasleft();
        uint256 result = 0;
        assembly {
            result := someValue
        }
        emit LogCost(before - gasleft());
    }
}