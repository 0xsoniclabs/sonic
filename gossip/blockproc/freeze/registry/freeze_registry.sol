// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

// FreezeRegistry is a stand-in contract for Sonic's on-chain account freeze
// registry to be used in local testing and development environments. It is
// deployed behind the EIP-1967 proxy at the registry address.
//
// State lives in an ERC-7201 namespace. The client reads `frozen` and
// `optedOut` directly from storage, so the namespace and the position of
// these fields are part of the protocol.
//
// isFrozen and isOptedOut are the contract's own reading of that state. Any
// registry replacing this one must provide them: the client's direct reads
// are tested against these views, which needs no knowledge of how a given
// implementation guards its setters.
contract FreezeRegistry {
    /// @custom:storage-location erc7201:sonic.freeze.registry
    struct FreezeStorage {
        // base + 0: whether transfers from/to this account are disabled
        mapping(address account => bool isFrozen) frozen;
        // base + 1: whether transfers of this token ignore account freezes
        mapping(address token => bool isOptedOut) optedOut;
    }

    // keccak256(abi.encode(uint256(keccak256("sonic.freeze.registry")) - 1)) & ~bytes32(uint256(0xff))
    bytes32 private constant STORAGE_LOCATION =
        0x79438978416e308cc48db3f2f690ad3d9d4d8848bddb35c7c4dfe2c0165d5900;

    function setFrozen(address account, bool value) external {
        _storage().frozen[account] = value;
    }

    function setOptedOut(address token, bool value) external {
        _storage().optedOut[token] = value;
    }

    function isFrozen(address account) external view returns (bool) {
        return _storage().frozen[account];
    }

    function isOptedOut(address token) external view returns (bool) {
        return _storage().optedOut[token];
    }

    function _storage() private pure returns (FreezeStorage storage $) {
        assembly {
            $.slot := STORAGE_LOCATION
        }
    }
}
