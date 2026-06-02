// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// LegacyRegistry is a test registry with the old interface:
// - chooseFund returns a single bytes32 fundId (32-byte response)
// - getGasConfig returns three values (96-byte response)
// This is used to verify backward-compatibility of the node's parsing logic.
contract LegacyRegistry {

    struct Fund {
        uint256 funds;
    }

    mapping(bytes32 => Fund) public sponsorships;

    function sponsor(bytes32 fundId) public payable {
        sponsorships[fundId].funds += msg.value;
    }

    // Legacy 3-field getGasConfig (96-byte response).
    function getGasConfig() public pure returns (
        uint256 chooseFundLimit,
        uint256 deductFeesLimit,
        uint256 overheadCharge
    ) {
        chooseFundLimit = 100_000;
        deductFeesLimit = 60_000;
        overheadCharge = chooseFundLimit + deductFeesLimit + 50_000;
    }

    // Legacy single-word chooseFund (32-byte response).
    function chooseFund(
        address /*from*/,
        address /*to*/,
        uint256 /*value*/,
        uint256 /*nonce*/,
        bytes calldata /*callData*/,
        uint256 fee
    ) public view returns (bytes32 fundId) {
        bytes32 id = bytes32(uint256(1)); // fixed fund ID
        if (sponsorships[id].funds >= fee) {
            return id;
        }
        return bytes32(0);
    }

    function deductFees(bytes32 fundId, uint256 fee) public {
        require(msg.sender == address(0));
        require(sponsorships[fundId].funds >= fee, "Not enough funds");
        feeBurner.burnNativeTokens{value: fee}();
        sponsorships[fundId].funds -= fee;
    }

    FeeBurner private constant feeBurner = FeeBurner(0xFC00FACE00000000000000000000000000000000);
}

interface FeeBurner {
    function burnNativeTokens() external payable;
}
