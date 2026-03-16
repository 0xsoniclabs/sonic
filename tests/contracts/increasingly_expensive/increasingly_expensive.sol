// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract IncreasinglyExpensive {
    // This variable is kept in the contract's storage
    uint256 public counter;

    function incrementAndLoop() public {
        // Read from storage, increments by 1, and save back to storage
        counter += 1;
        
        // Cache the storage variable to a local memory variable
        uint256 count = counter;
        
        // 4. Loop as many iterations as the current counter value
        for (uint256 i = count; i > 0; i--) {
        }
    }
}