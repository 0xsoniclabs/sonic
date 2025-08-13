// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

contract TestStorageCall {
    TestTransientStorage t;

    constructor() {
        t = new TestTransientStorage();
    }

    function transientStore(uint256 v) public returns (uint256) {
        t.store(v);
        return t.val();
    }

    function testVal() public view returns (uint256) {
        return t.val();
    }
}

contract TestTransientStorage {
    bytes32 constant SLOT = 0;

    function store(uint256 v) public {
        assembly {
            tstore(SLOT, v)
        }
    }

    function val() public view returns (uint256 v) {
        assembly {
            v := tload(SLOT)
        }
    }
}