// Copyright 2025 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

pragma solidity ^0.8.24;

struct config {
    bool enabled; // kill-switch for this feature (optional)
    uint256 overheadCharge; // fees for processing bundled transactions
    uint256 perTxCharge; // fees charged for each transaction in a bundle
}

contract BundleService {
    // Provides configuration parameters for the bundling service offered by Sonic,
    // including gas pricing parameters.
    function getConfig() public pure returns (config memory) {
        return
            config({
                enabled: true,
                overheadCharge: 10000000000000000, // 0.01 $S
                perTxCharge: 1000000000000000 // 0.001 $S
            });
    }

    // TODO: extend interface to allow payback of basefee delta.
    // ARGS:  receiver of payback, total gas
    // receive function to accept native token payments for bundling fees.
    receive() external payable {
        // TODO: log the payment, for the block processor to be able to identify
        // that it was done.
        feeBurner.burnNativeTokens{value: msg.value}();
    }

    // --- Internal functions ---

    // Address of the FeeBurner contract used to burn native tokens.
    // In this contract, this is a hardcoded constant referring to the SFC.
    FeeBurner private constant feeBurner =
        FeeBurner(0xFC00FACE00000000000000000000000000000000);
}

// Minimal interface for the FeeBurner contract used to burn native tokens. This
// interface is required to be implemented by the SFC.
interface FeeBurner {
    function burnNativeTokens() external payable;
}
