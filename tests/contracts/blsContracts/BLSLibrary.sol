// SPDX-License-Identifier: MIT
pragma solidity >=0.8.26;

import "./BigNumber.sol";
import "./Elements.sol";

library BLSLibrary {
    using BigNumbers for *;

    bytes constant moduloBytes =
        hex"1a0111ea397fe69a4b1ba7b6434bacd764774b84f38512bf6730d2a0f6b0f6241eabfffeb153ffffb9feffffffffaaab";
    
    // Negated generator of G1 to be used for pairing function
    bytes constant negG1 =
        hex"0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb00000000000000000000000000000000114d1d6855d545a8aa7d76c8cf2e21f267816aef1db507c96655b9d5caac42364e6f38ba0ecb751bad54dcd6b939c2ca";

    // This defines how many fp elements have to be created when hashing message
    // For EncodeToG2 this is 2
    // For HashToG2 this is 2*2
    uint16 constant elCount = 2;

    // Domain separation tag 
    // TODO: need to be specified according to spec recommendation
    bytes constant dst = "";

    // Main function to hash message into G2 point
    function EncodeToG2(bytes memory message)
        public
        view
        returns (bytes memory)
    {
        Elements.Element[] memory elArray = hashToFieldPoints(message);
        bytes memory fpBytes;
        uint64 zero;
        for (uint8 i; i < elArray.length; i++) {
            fpBytes = abi.encodePacked(
                fpBytes,
                zero,
                zero,
                elArray[i].val[5],
                elArray[i].val[4],
                elArray[i].val[3],
                elArray[i].val[2],
                elArray[i].val[1],
                elArray[i].val[0]
            );
        }
        return precompileMaptoG2(fpBytes);
    }

    // All input parameters are points on G1 or G2 as bytes
    function CheckSignature(
        bytes memory pubKey,
        bytes memory signature,
        bytes memory messageHash
    ) public view returns (bool) {
        bytes memory res = precompilePair(
                abi.encodePacked(negG1, signature, pubKey, messageHash)
            );
        require(res.length == 32, "Invalid result length");
        return res[31] != 0;
    }

    // All input parameters are points on G1 or G2 as bytes
    function CheckAggregatedSignature(
        bytes memory pubKeys,
        bytes memory signature,
        bytes memory messageHash
    ) public view returns (bool) {
        
        require(pubKeys.length%128 == 0, "Invalid public keys length");
        require(signature.length == 256, "Invalid signature length");

        bytes memory input = abi.encodePacked(negG1, signature);

        // Split public keys and add them into the input data
        for (uint256 i = 0; i < pubKeys.length/128; i++) {
            bytes memory pk = new bytes(128);
            uint256 chunkStart = i * 128;
            assembly {
                let chunkPtr := add(pk, 0x20) // Points to the start of the `chunk` data
                let srcPtr := add(add(pubKeys, 0x20), chunkStart) // Points to the correct position in `pseudoRandomBytes`
                mstore(chunkPtr, mload(srcPtr)) // Copy 32 bytes
                mstore(add(chunkPtr, 0x20), mload(add(srcPtr, 0x20))) // Copy the next 32 bytes
                mstore(add(chunkPtr, 0x40), mload(add(srcPtr, 0x40))) // Copy the next 32 bytes
                mstore(add(chunkPtr, 0x60), mload(add(srcPtr, 0x60))) // Copy the next 32 bytes
            }
            input = abi.encodePacked(input, pk, messageHash);
        }

        // Call precompile contract to pair signatures
        bytes memory res = precompilePair(input);
        require(res.length == 32, "Invalid result length");
        return res[31] != 0;
    }

    // Use precompile contract to map hashed message as a field points into G2 point
    function precompileMaptoG2(bytes memory input)
        private
        view
        returns (bytes memory)
    {
        (bool ok, bytes memory out) = address(0x11).staticcall(input);
        require(ok, "BLS mapToG2 call failed");
        return out;
    }

    // Use precompile contract for pairing
    function precompilePair(bytes memory input)
        private
        view
        returns (bytes memory)
    {
        (bool ok, bytes memory out) = address(0x0f).staticcall(input);
        require(ok, "BLS pair call failed");
        
        return out;
    }

    // https://datatracker.ietf.org/doc/html/draft-irtf-cfrg-hash-to-curve-09#name-expand_message_xmd
    //
    // Z_pad = I2OSP(0, r_in_bytes)   64
    // l_i_b_str = I2OSP(len_in_bytes, 2)
    // DST_prime = DST ∥ I2OSP(len(DST), 1)
    // b₀ = H(Z_pad ∥ msg ∥ l_i_b_str ∥ I2OSP(0, 1) ∥ DST_prime)

    // count := 2 * 2  // for hashToG2 count = 2*2      for encodeToG2 count = 2
    // // 128 bits of security
    // // L = ceil((ceil(log2(p)) + k) / 8), where k is the security parameter = 128
    // Bits  = 381 // number of bits needed to represent a Element
    // const Bytes = 1 + (Bits-1)/8     //48
    // const L = 16 + Bytes             //64
    // lenInBytes := count * L          //256
    // lenInBytes for ElementCount 2*2 (G2) is 256 and that is 10 in []byte16
    // lenInBytes = 128;  // for encodeToG2 count = 2       for hashToG2 count = 2*2

    // ell := (lenInBytes + h.Size() - 1) / h.Size() // ceil(len_in_bytes / b_in_bytes)     h.Size 32
    // ell := (256 + 32 -1)/32 = 8

    function hash(
        bytes memory message,
        bytes memory _dst,
        uint16 count
    ) private pure returns (bytes memory) {
        require(_dst.length == 0, "DST not implemented yet");

        //require(l <= type(uint16).max, "length exceeds uint16 range");
        uint16 length = 64 * count; // L = 64 as explained above
        uint64 ell = (length + 31) / 32;
        bytes32 zero;
        uint8 zbyte;
        uint8 one = 1;

        // Z_pad = I2OSP(0, r_in_bytes)   64
        // l_i_b_str = I2OSP(len_in_bytes, 2)
        // DST_prime = DST ∥ I2OSP(len(DST), 1)

        // b₀ = H(Z_pad ∥ msg ∥ l_i_b_str ∥ I2OSP(0, 1) ∥ DST_prime)
        bytes32 b0 = sha256(
            abi.encodePacked(zero, zero, message, length, zbyte, zbyte)
        );
        // b₁ = H(b₀ ∥ I2OSP(1, 1) ∥ DST_prime)
        bytes32 b1 = sha256(abi.encodePacked(b0, one, zbyte));

        require(b1 != sha256(abi.encodePacked(message)));

        bytes memory res = abi.encodePacked(b1);
        bytes32 strxor;
        for (uint8 i = 2; i <= ell; i++) {
            // b_i = H(strxor(b₀, b_(i - 1)) ∥ I2OSP(i, 1) ∥ DST_prime)
            strxor = b0 ^ b1;
            b1 = sha256(abi.encodePacked(strxor, i, zbyte));
            res = abi.encodePacked(res, b1);
        }
        return res;
    }

    // https://datatracker.ietf.org/doc/html/draft-irtf-cfrg-hash-to-curve-09#name-hash_to_field-implementatio
    // hashToFieldPoints is a function that takes a message and returns an array of field points
    function hashToFieldPoints(bytes memory message)
        private
        view
        returns (Elements.Element[] memory)
    {
        // expand message
        bytes memory pseudoRandomBytes = hash(
            abi.encodePacked(message),
            abi.encodePacked(dst),
            elCount
        );

        require(pseudoRandomBytes.length != 0, "No data");

        Elements.Element[] memory res = new Elements.Element[](elCount);

        BigNumber memory modulo;
        modulo = BigNumbers.init(moduloBytes, false);

        // for number of field points points
        for (uint256 i = 0; i < elCount; i++) {
            bytes memory chunk = new bytes(64);
            uint256 chunkStart = i * 64;

            assembly {
                let chunkPtr := add(chunk, 0x20) // Points to the start of the `chunk` data
                let srcPtr := add(add(pseudoRandomBytes, 0x20), chunkStart) // Points to the correct position in `pseudoRandomBytes`
                mstore(chunkPtr, mload(srcPtr)) // Copy 32 bytes
                mstore(add(chunkPtr, 0x20), mload(add(srcPtr, 0x20))) // Copy the next 32 bytes
            }

            BigNumber memory v = BigNumbers.init(chunk, false);
            res[i] = setBigIntToElement(v, modulo);
        }

        return res;
    }

    function setBigIntToElement(BigNumber memory v, BigNumber memory modulo) private view returns (Elements.Element memory) {

        BigNumber memory zero = BigNumbers.zero();
        int c = v.cmp(modulo, false);
        if (c == 0) {
            // v == 0
            Elements.Element memory zeroEl;
            return zeroEl;
        } else if (c != 1 && v.cmp(zero, false) != -1) {
            // 0 < v < q
            return Elements.ElementFromBytes(v.val);
        }

        v = v.mod(modulo);

        // Other BLS libraries like gnark are returning field points with Montgomery reduction.
        // This step is not needed for mapping field points to G2 as in the gnark library
        // field elements are recovered from their Montgomery representation before mapping.
        // Motngommery reduction can be achieved by calling Elements.toMont(Elements.ElementFromBytes(vv.val));

        return Elements.ElementFromBytes(v.val);
    }  
}
