// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract SelfDestruct {
    uint256 public someData = 123;
    event LogAfterDestruct(uint256 balance);

    // Constructor with optional selfdestruct
    // If destroyNow is true, the contract will selfdestruct during construction
    // This enables to generate a transaction where the contract both created and destroyed
    // If selfBeneficiary is true, the address of this new instance will be used as the recipient
    // for the selfdestruct call. Otherwise, the recipient address will be used.
    constructor(
        bool destroyNow,
        bool selfBeneficiary,
        address payable recipient
    ) payable {
        if (destroyNow) destroyContract(selfBeneficiary, recipient);
    }

    // destroyContract is a public function that can be called to selfdestruct the contract.
    // If selfBeneficiary is true, the address of this instance will be used as the recipient
    // for the selfdestruct call. Otherwise, the recipient address will be used.
    function destroyContract(
        bool selfBeneficiary,
        address payable recipient
    ) public {
        if (selfBeneficiary) recipient = payable(address(this));
        selfdestruct(recipient);

        // The test checks that this log is never emitted.
        emit LogAfterDestruct(address(this).balance);
    }
}

contract SelfDestructFactory {
    SelfDestruct lastContract;

    // LogDeployed is emitted when a new contract is deployed, used to read the
    // address of the last created contract
    event LogDeployed(address addr);
    // LogContractStorage is emitted to read the interal state of the last
    // created contract
    event LogContractStorage(uint256 value);

    function create() public payable {
        // construct and transfer value to new contract
        lastContract = (new SelfDestruct){value: msg.value}(
            false, // do not selfdestruct immediately
            false, // beneficiary is not self
            payable(address(0)) // any value, because selfdestruct is not called
        );
        emit LogDeployed(address(lastContract));
    }

    function destroy(address payable beneficiary) public {
        lastContract.destroyContract(false, beneficiary);
    }

    function destroyWithoutBeneficiary() public {
        lastContract.destroyContract(
            true, // beneficiary is self
            payable(address(0)) // any value, not used
        );
    }

    function createAndDestroy(address payable beneficiary) public payable {
        create();
        destroy(beneficiary);
        emit LogContractStorage(lastContract.someData());
    }

    function createAndDestroyWithoutBeneficiary() public payable {
        create();
        destroy(payable(address(lastContract)));
        emit LogContractStorage(lastContract.someData());
    }
}
