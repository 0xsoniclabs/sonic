// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.20;

contract counter_event_emitter {
    int private totalCount = 0;
    mapping(address => int) public perAddrCount;
    event Count(int totalCount, int perAddrCount);

    function increment() public {
        // We need to check correct order per account
        perAddrCount[msg.sender] += 1;
        totalCount += 1;
        emit Count(totalCount, perAddrCount[msg.sender]);
    }

    function getTotalCount() public view returns (int) {
        return totalCount;
    }
}
