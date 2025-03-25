// SPDX-License-Identifier: MIT
pragma solidity >=0.8.26;

import "./BLSLibrary.sol";

contract BLS {

    bytes public Signature;
    
    function EncodeToG2(bytes memory message) public view returns (bytes memory){
        return BLSLibrary.EncodeToG2(message);
    }

    function CheckSignature(bytes memory pubKey, bytes memory signature, bytes memory message) public view returns (bool){
        // check length of input parameters
        require(pubKey.length == 128, "Invalid public key length");
        require(signature.length == 256, "Invalid signature length");

        // hash message and do pairing
        bytes memory msgHashG2 = BLSLibrary.EncodeToG2(message);
        return BLSLibrary.Pair(pubKey,signature,msgHashG2);
    }

    function CheckAndUpdate(bytes memory pubKey, bytes memory signature, bytes memory message) public {
        bool res  = CheckSignature(pubKey, signature, message);
        if (res == true) {
            Signature = signature;
        }
    }

    function CheckAgregatedSignature(bytes memory pubKeys, bytes memory signature, bytes memory message) public view returns (bool){
        // check length of input parameters
        require(pubKeys.length%128 == 0, "Invalid public keys length");
        require(signature.length == 256, "Invalid signature length");

        // hash message and do pairing
        bytes memory msgHashG2 = BLSLibrary.EncodeToG2(message);
        return BLSLibrary.PairAgregateSignature(pubKeys,signature,msgHashG2);
    }

    function CheckAndUpdateAgregatedSignature(bytes memory pubKeys, bytes memory signature, bytes memory message) public{
        bool res  = CheckAgregatedSignature(pubKeys, signature, message);
        if (res == true) {
            Signature = signature;
        }
    }
}
