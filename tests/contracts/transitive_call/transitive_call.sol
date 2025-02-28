// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// TransitiveCall is a contract that can call a chain of contracts.
// the value is passed from the account starting the chain of calls until it
// reaches the last one.
contract TransitiveCall {
    int private count = 0;

    function leafCall() external payable {
        count += 1;
    }

    function transitiveCall(address[] calldata call_chain) external payable {
        require(call_chain.length > 0, "call_chain is empty");
        count += 1;
        if (call_chain.length == 1) {
            TransitiveCall callable = TransitiveCall(call_chain[0]);
            callable.leafCall{value: msg.value}();
        } else {
            TransitiveCall callable = TransitiveCall(call_chain[0]);
            address[] memory tail = new address[](call_chain.length - 1);
            for (uint256 i = 1; i < call_chain.length; i++) {
                tail[i - 1] = call_chain[i];
            }
            callable.transitiveCall{value: msg.value}(tail);
        }
    }

    function getCount() public view returns (int) {
        return count;
    }
}
