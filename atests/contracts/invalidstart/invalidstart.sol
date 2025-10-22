// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract InvalidStart {
    // byteccode source: https://eips.ethereum.org/EIPS/eip-3541#test-cases
    bytes public invalidBytecode = hex"60ef60005360016000f3";
    bytes public validBytecode = hex"60fe60005360016000f3";
    function createContractWithInvalidCode() public {
        createOrRevert(invalidBytecode);
    }

    function createContractWithValidCode() public {
        createOrRevert(validBytecode);
    }

    function createOrRevert(bytes memory bytecode) private {
        assembly {
            let emptyContract := create(0, add(bytecode, 0x20), mload(bytecode))
            if iszero(emptyContract) {
                revert(0, 0)
            }
        }
    }

    function create2ContractWithInvalidCode() public {
        create2OrRevert(invalidBytecode);
    }

    function create2ContractWithValidCode() public {
        create2OrRevert(validBytecode);
    }

    function create2OrRevert(bytes memory bytecode) private {
        assembly {
            // Deploy the contract that self-destructs
            let emptyContract := create2(0, add(bytecode, 0x20), mload(bytecode), 0)
            if iszero(emptyContract) {
                revert(0, 0)
            }
        }
    }
}
