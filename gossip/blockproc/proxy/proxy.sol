
// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

// Proxy is a minimal proxy contract that delegates all calls to an implementation
// contract. The implementation address is stored in a specific storage slot
// as per EIP-1967. This proxy does not include any access control for updating
// the implementation. It is only intended for local testing and development.
//
// This is a slimed down version of these two OpenZeppelin contracts:
// https://github.com/OpenZeppelin/openzeppelin-contracts/tree/master/contracts/proxy/ERC1967
// https://github.com/OpenZeppelin/openzeppelin-contracts/blob/master/contracts/proxy/Proxy.sol
contract Proxy {

    // The storage location of the implementation, as per EIP-1967
    bytes32 internal constant IMPLEMENTATION_SLOT = 0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;

    // Updates the implementation.
    function update(address newImplementation) external {
        assembly {
            sstore(IMPLEMENTATION_SLOT, newImplementation)
        }
    }

    // Delegates the current call to the implementation.
    function _delegate() internal virtual {
        assembly {
            calldatacopy(0x00, 0x00, calldatasize())
            let result := delegatecall(gas(), sload(IMPLEMENTATION_SLOT), 0x00, calldatasize(), 0x00, 0x00)
            returndatacopy(0x00, 0x00, returndatasize())

            switch result
            case 0 {
                revert(0x00, returndatasize())
            }
            default {
                return(0x00, returndatasize())
            }
        }
    }

    fallback() external payable virtual {
        _delegate();
    }

}