// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// Payer holds native S and pays it out, or forwards calls, on request, for
// tests only.
contract Payer {
    receive() external payable {}

    function pay(address payable to, uint256 amount) external {
        (bool ok, ) = to.call{value: amount}("");
        require(ok, "payment failed");
    }

    // forward calls to with data and reverts if the call fails.
    function forward(address to, bytes calldata data) external {
        (bool ok, ) = to.call(data);
        // A real router would bubble up the inner revert data;
        // test code omits it for simplicity.
        require(ok, "forward failed");
    }
}
