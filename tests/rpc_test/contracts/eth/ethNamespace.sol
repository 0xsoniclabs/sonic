// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

/** * @title Storage * @dev Store & retrieve value in a variable * @custom:dev-run-script ./scripts/deploy_with_ethers.ts */
contract Storage {
    uint256 number;
    event TestLog(uint256 indexed num);

    /** * @dev Store value in variable * @param num value to store */
    function store(uint256 num) public {
        number = num;
        emit TestLog(num);
    }

    /** * @dev Return value * @return value of 'number' */
    function retrieve() public view returns (uint256) {
        return number;
    }
}