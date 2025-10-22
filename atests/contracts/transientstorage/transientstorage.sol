// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

contract TransientStorage {

    InnerContract inner;
    event StoredValue(uint256 value);

    constructor() {
        inner = new InnerContract();
    }

    function storeValue() public {
        inner.storeTransientValue(42);
        emit StoredValue(getValue());
    }

    function getValue() public view returns (uint256) {
        return inner.getTransientValue();
    }
}

contract InnerContract {
    uint transient tval;

    function storeTransientValue(uint256 val) public {
        tval = val;
        
    }

    function getTransientValue() public view returns (uint256) {
        return tval;
    }
}