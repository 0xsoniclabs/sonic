// SPDX-License-Identifier: MIT
pragma solidity >=0.8.26;

// Library for operations with BLS12-381 field elements.
// These elements are product of hash to field function and can be mapped to G2.
// https://datatracker.ietf.org/doc/html/draft-irtf-cfrg-hash-to-curve-09#name-hash_to_field-implementatio
library Elements {

    // Element represents a field element stored on 6 words (uint64)
    struct Element {
        uint64[6] val; // Fixed-size array of 6 uint64 values
    }
    
    // // Field modulus q
    // const 
    uint64 private constant q0  = 13402431016077863595;
	uint64 private constant q1  = 2210141511517208575;
	uint64 private constant q2  = 7435674573564081700;
	uint64 private constant q3  = 7239337960414712511;
	uint64 private constant q4  = 5412103778470702295;
	uint64 private constant q5  = 1873798617647539866;

    // q + r'.r = 1, i.e., qInvNeg = - q⁻¹ mod r
    // used for Montgomery reduction
    uint64 private constant qInvNeg = 9940570264628428797;

    function ElementFromBytes(bytes memory data) public pure returns (Elements.Element memory) {
        uint256 chunkCount = 8;
        Elements.Element memory chunks;

        for (uint256 i = 2; i < chunkCount; i++) {
            uint64 chunk = 0;

            for (uint256 j = 0; j < 8; j++) {
                uint256 index = i * 8 + j;
                if (index < data.length) {
                    chunk |= uint64(uint8(data[index])) << uint64(8 * (7 - j));
                }
            }

            chunks.val[7-i] = chunk;
        }

        return chunks;
    }

    // rSquare where r is the Montgommery constant
    // see section 2.3.2 of Tolga Acar's thesis
    // https://www.microsoft.com/en-us/research/wp-content/uploads/1998/06/97Acar.pdf
    // rSquare = Element{
    //     17644856173732828998,
    //     754043588434789617,
    //     10224657059481499349,
    //     7488229067341005760,
    //     11130996698012816685,
    //     1267921511277847466,
    // }

    // toMont converts z to Montgomery form
    // sets and returns z = z * r²
    function toMont(Element memory z) public pure returns (Element memory){
        Element memory rSquare;
        rSquare.val[0] = 17644856173732828998;
        rSquare.val[1] = 754043588434789617;
        rSquare.val[2] = 10224657059481499349;
        rSquare.val[3] = 7488229067341005760;
        rSquare.val[4] = 11130996698012816685;
        rSquare.val[5] = 1267921511277847466;
        return mul(z,rSquare);
    }

    function mul(Element memory x, Element memory y) public pure returns (Element memory z) {

        MulHelper memory me;   
        // not important result
        uint64 na;

        {
            uint64 c0;
            uint64 c1;
            uint64 c2; 
            (me.u0, me.t0) = Mul64(x.val[0], y.val[0]);
            (me.u1, me.t1) = Mul64(x.val[0], y.val[1]);
            (me.u2, me.t2) = Mul64(x.val[0], y.val[2]);
            (me.u3, me.t3) = Mul64(x.val[0], y.val[3]);
            (me.u4, me.t4) = Mul64(x.val[0], y.val[4]);
            (me.u5, me.t5) = Mul64(x.val[0], y.val[5]);
            (me.t1, c0) = Add64(me.u0, me.t1, 0);
            (me.t2, c0) = Add64(me.u1, me.t2, c0);
            (me.t3, c0) = Add64(me.u2, me.t3, c0);
            (me.t4, c0) = Add64(me.u3, me.t4, c0);
            (me.t5, c0) = Add64(me.u4, me.t5, c0);
            (c2, na) = Add64(me.u5, 0, c0);

            uint64 m;
            (na, m) = Mul64(qInvNeg, me.t0);

            (me.u0, c1) = Mul64(m, q0);
            (na, c0) = Add64(me.t0, c1, 0);
            (me.u1, c1) = Mul64(m, q1);
            (me.t0, c0) = Add64(me.t1, c1, c0);
            (me.u2, c1) = Mul64(m, q2);
            (me.t1, c0) = Add64(me.t2, c1, c0);
            (me.u3, c1) = Mul64(m, q3);
            (me.t2, c0) = Add64(me.t3, c1, c0);
            (me.u4, c1) = Mul64(m, q4);
            (me.t3, c0) = Add64(me.t4, c1, c0);
            (me.u5, c1) = Mul64(m, q5);

            (me.t4, c0) = Add64(0, c1, c0);
            (me.u5, na) = Add64(me.u5, 0, c0);
            (me.t0, c0) = Add64(me.u0, me.t0, 0);
            (me.t1, c0) = Add64(me.u1, me.t1, c0);
            (me.t2, c0) = Add64(me.u2, me.t2, c0);
            (me.t3, c0) = Add64(me.u3, me.t3, c0);
            (me.t4, c0) = Add64(me.u4, me.t4, c0);
            (c2, na) = Add64(c2, 0, c0);
            (me.t4, c0) = Add64(me.t5, me.t4, 0);
            (me.t5, na) = Add64(me.u5, c2, c0);
        }{
            uint64 c0;
            uint64 c1;
            uint64 c2;
            (me.u0, c1) = Mul64(x.val[1], y.val[0]);
            (me.t0, c0) = Add64(c1, me.t0, 0);
            (me.u1, c1) = Mul64(x.val[1], y.val[1]);
            (me.t1, c0) = Add64(c1, me.t1, c0);
            (me.u2, c1) = Mul64(x.val[1], y.val[2]);
            (me.t2, c0) = Add64(c1, me.t2, c0);
            (me.u3, c1) = Mul64(x.val[1], y.val[3]);
            (me.t3, c0) = Add64(c1, me.t3, c0);
            (me.u4, c1) = Mul64(x.val[1], y.val[4]);
            (me.t4, c0) = Add64(c1, me.t4, c0);
            (me.u5, c1) = Mul64(x.val[1], y.val[5]);
            (me.t5, c0) = Add64(c1, me.t5, c0);

            (c2, na) = Add64(0, 0, c0);
            (me.t1, c0) = Add64(me.u0, me.t1, 0);
            (me.t2, c0) = Add64(me.u1, me.t2, c0);
            (me.t3, c0) = Add64(me.u2, me.t3, c0);
            (me.t4, c0) = Add64(me.u3, me.t4, c0);
            (me.t5, c0) = Add64(me.u4, me.t5, c0);
            (c2, na) = Add64(me.u5, c2, c0);

            uint64 m;
            (na, m) = Mul64(qInvNeg, me.t0);

            (me.u0, c1) = Mul64(m, q0);
            (na, c0) = Add64(me.t0, c1, 0);
            (me.u1, c1) = Mul64(m, q1);
            (me.t0, c0) = Add64(me.t1, c1, c0);
            (me.u2, c1) = Mul64(m, q2);
            (me.t1, c0) = Add64(me.t2, c1, c0);
            (me.u3, c1) = Mul64(m, q3);
            (me.t2, c0) = Add64(me.t3, c1, c0);
            (me.u4, c1) = Mul64(m, q4);
            (me.t3, c0) = Add64(me.t4, c1, c0);
            (me.u5, c1) = Mul64(m, q5);

            (me.t4, c0) = Add64(0, c1, c0);
            (me.u5, na) = Add64(me.u5, 0, c0);
            (me.t0, c0) = Add64(me.u0, me.t0, 0);
            (me.t1, c0) = Add64(me.u1, me.t1, c0);
            (me.t2, c0) = Add64(me.u2, me.t2, c0);
            (me.t3, c0) = Add64(me.u3, me.t3, c0);
            (me.t4, c0) = Add64(me.u4, me.t4, c0);
            (c2, na) = Add64(c2, 0, c0);
            (me.t4, c0) = Add64(me.t5, me.t4, 0);
            (me.t5, na) = Add64(me.u5, c2, c0);
        }{
            uint64 c0;
            uint64 c1;
            uint64 c2;
            (me.u0, c1) = Mul64(x.val[2], y.val[0]);
            (me.t0, c0) = Add64(c1, me.t0, 0);
            (me.u1, c1) = Mul64(x.val[2], y.val[1]);
            (me.t1, c0) = Add64(c1, me.t1, c0);
            (me.u2, c1) = Mul64(x.val[2], y.val[2]);
            (me.t2, c0) = Add64(c1, me.t2, c0);
            (me.u3, c1) = Mul64(x.val[2], y.val[3]);
            (me.t3, c0) = Add64(c1, me.t3, c0);
            (me.u4, c1) = Mul64(x.val[2], y.val[4]);
            (me.t4, c0) = Add64(c1, me.t4, c0);
            (me.u5, c1) = Mul64(x.val[2], y.val[5]);
            (me.t5, c0) = Add64(c1, me.t5, c0);

            (c2, na) = Add64(0, 0, c0);
            (me.t1, c0) = Add64(me.u0, me.t1, 0);
            (me.t2, c0) = Add64(me.u1, me.t2, c0);
            (me.t3, c0) = Add64(me.u2, me.t3, c0);
            (me.t4, c0) = Add64(me.u3, me.t4, c0);
            (me.t5, c0) = Add64(me.u4, me.t5, c0);
            (c2, na) = Add64(me.u5, c2, c0);

            uint64 m;
            (na, m) = Mul64(qInvNeg, me.t0);

            (me.u0, c1) = Mul64(m, q0);
            (na, c0) = Add64(me.t0, c1, 0);
            (me.u1, c1) = Mul64(m, q1);
            (me.t0, c0) = Add64(me.t1, c1, c0);
            (me.u2, c1) = Mul64(m, q2);
            (me.t1, c0) = Add64(me.t2, c1, c0);
            (me.u3, c1) = Mul64(m, q3);
            (me.t2, c0) = Add64(me.t3, c1, c0);
            (me.u4, c1) = Mul64(m, q4);
            (me.t3, c0) = Add64(me.t4, c1, c0);
            (me.u5, c1) = Mul64(m, q5);

            (me.t4, c0) = Add64(0, c1, c0);
            (me.u5, na) = Add64(me.u5, 0, c0);
            (me.t0, c0) = Add64(me.u0, me.t0, 0);
            (me.t1, c0) = Add64(me.u1, me.t1, c0);
            (me.t2, c0) = Add64(me.u2, me.t2, c0);
            (me.t3, c0) = Add64(me.u3, me.t3, c0);
            (me.t4, c0) = Add64(me.u4, me.t4, c0);
            (c2, na) = Add64(c2, 0, c0);
            (me.t4, c0) = Add64(me.t5, me.t4, 0);
            (me.t5, na) = Add64(me.u5, c2, c0);

        }{
            uint64 c0;
            uint64 c1;
            uint64 c2;
            (me.u0, c1) = Mul64(x.val[3], y.val[0]);
            (me.t0, c0) = Add64(c1, me.t0, 0);
            (me.u1, c1) = Mul64(x.val[3], y.val[1]);
            (me.t1, c0) = Add64(c1, me.t1, c0);
            (me.u2, c1) = Mul64(x.val[3], y.val[2]);
            (me.t2, c0) = Add64(c1, me.t2, c0);
            (me.u3, c1) = Mul64(x.val[3], y.val[3]);
            (me.t3, c0) = Add64(c1, me.t3, c0);
            (me.u4, c1) = Mul64(x.val[3], y.val[4]);
            (me.t4, c0) = Add64(c1, me.t4, c0);
            (me.u5, c1) = Mul64(x.val[3], y.val[5]);
            (me.t5, c0) = Add64(c1, me.t5, c0);

            (c2, na) = Add64(0, 0, c0);
            (me.t1, c0) = Add64(me.u0, me.t1, 0);
            (me.t2, c0) = Add64(me.u1, me.t2, c0);
            (me.t3, c0) = Add64(me.u2, me.t3, c0);
            (me.t4, c0) = Add64(me.u3, me.t4, c0);
            (me.t5, c0) = Add64(me.u4, me.t5, c0);
            (c2, na) = Add64(me.u5, c2, c0);

            uint64 m;
            (na, m) = Mul64(qInvNeg, me.t0);

            (me.u0, c1) = Mul64(m, q0);
            (na, c0) = Add64(me.t0, c1, 0);
            (me.u1, c1) = Mul64(m, q1);
            (me.t0, c0) = Add64(me.t1, c1, c0);
            (me.u2, c1) = Mul64(m, q2);
            (me.t1, c0) = Add64(me.t2, c1, c0);
            (me.u3, c1) = Mul64(m, q3);
            (me.t2, c0) = Add64(me.t3, c1, c0);
            (me.u4, c1) = Mul64(m, q4);
            (me.t3, c0) = Add64(me.t4, c1, c0);
            (me.u5, c1) = Mul64(m, q5);

            (me.t4, c0) = Add64(0, c1, c0);
            (me.u5, na) = Add64(me.u5, 0, c0);
            (me.t0, c0) = Add64(me.u0, me.t0, 0);
            (me.t1, c0) = Add64(me.u1, me.t1, c0);
            (me.t2, c0) = Add64(me.u2, me.t2, c0);
            (me.t3, c0) = Add64(me.u3, me.t3, c0);
            (me.t4, c0) = Add64(me.u4, me.t4, c0);
            (c2, na) = Add64(c2, 0, c0);
            (me.t4, c0) = Add64(me.t5, me.t4, 0);
            (me.t5, na) = Add64(me.u5, c2, c0);

        }{
            uint64 c0;
            uint64 c1;
            uint64 c2;
            (me.u0, c1) = Mul64(x.val[4], y.val[0]);
            (me.t0, c0) = Add64(c1, me.t0, 0);
            (me.u1, c1) = Mul64(x.val[4], y.val[1]);
            (me.t1, c0) = Add64(c1, me.t1, c0);
            (me.u2, c1) = Mul64(x.val[4], y.val[2]);
            (me.t2, c0) = Add64(c1, me.t2, c0);
            (me.u3, c1) = Mul64(x.val[4], y.val[3]);
            (me.t3, c0) = Add64(c1, me.t3, c0);
            (me.u4, c1) = Mul64(x.val[4], y.val[4]);
            (me.t4, c0) = Add64(c1, me.t4, c0);
            (me.u5, c1) = Mul64(x.val[4], y.val[5]);
            (me.t5, c0) = Add64(c1, me.t5, c0);

            (c2, na) = Add64(0, 0, c0);
            (me.t1, c0) = Add64(me.u0, me.t1, 0);
            (me.t2, c0) = Add64(me.u1, me.t2, c0);
            (me.t3, c0) = Add64(me.u2, me.t3, c0);
            (me.t4, c0) = Add64(me.u3, me.t4, c0);
            (me.t5, c0) = Add64(me.u4, me.t5, c0);
            (c2, na) = Add64(me.u5, c2, c0);

            uint64 m;
            (na, m) = Mul64(qInvNeg, me.t0);

            (me.u0, c1) = Mul64(m, q0);
            (na, c0) = Add64(me.t0, c1, 0);
            (me.u1, c1) = Mul64(m, q1);
            (me.t0, c0) = Add64(me.t1, c1, c0);
            (me.u2, c1) = Mul64(m, q2);
            (me.t1, c0) = Add64(me.t2, c1, c0);
            (me.u3, c1) = Mul64(m, q3);
            (me.t2, c0) = Add64(me.t3, c1, c0);
            (me.u4, c1) = Mul64(m, q4);
            (me.t3, c0) = Add64(me.t4, c1, c0);
            (me.u5, c1) = Mul64(m, q5);

            (me.t4, c0) = Add64(0, c1, c0);
            (me.u5, na) = Add64(me.u5, 0, c0);
            (me.t0, c0) = Add64(me.u0, me.t0, 0);
            (me.t1, c0) = Add64(me.u1, me.t1, c0);
            (me.t2, c0) = Add64(me.u2, me.t2, c0);
            (me.t3, c0) = Add64(me.u3, me.t3, c0);
            (me.t4, c0) = Add64(me.u4, me.t4, c0);
            (c2, na) = Add64(c2, 0, c0);
            (me.t4, c0) = Add64(me.t5, me.t4, 0);
            (me.t5, na) = Add64(me.u5, c2, c0);
        }{
            uint64 c0;
            uint64 c1;
            uint64 c2;
            (me.u0, c1) = Mul64(x.val[5], y.val[0]);
            (me.t0, c0) = Add64(c1, me.t0, 0);
            (me.u1, c1) = Mul64(x.val[5], y.val[1]);
            (me.t1, c0) = Add64(c1, me.t1, c0);
            (me.u2, c1) = Mul64(x.val[5], y.val[2]);
            (me.t2, c0) = Add64(c1, me.t2, c0);
            (me.u3, c1) = Mul64(x.val[5], y.val[3]);
            (me.t3, c0) = Add64(c1, me.t3, c0);
            (me.u4, c1) = Mul64(x.val[5], y.val[4]);
            (me.t4, c0) = Add64(c1, me.t4, c0);
            (me.u5, c1) = Mul64(x.val[5], y.val[5]);
            (me.t5, c0) = Add64(c1, me.t5, c0);

            (c2, na) = Add64(0, 0, c0);
            (me.t1, c0) = Add64(me.u0, me.t1, 0);
            (me.t2, c0) = Add64(me.u1, me.t2, c0);
            (me.t3, c0) = Add64(me.u2, me.t3, c0);
            (me.t4, c0) = Add64(me.u3, me.t4, c0);
            (me.t5, c0) = Add64(me.u4, me.t5, c0);
            (c2, na) = Add64(me.u5, c2, c0);

            uint64 m;
            (na, m) = Mul64(qInvNeg, me.t0);

            (me.u0, c1) = Mul64(m, q0);
            (na, c0) = Add64(me.t0, c1, 0);
            (me.u1, c1) = Mul64(m, q1);
            (me.t0, c0) = Add64(me.t1, c1, c0);
            (me.u2, c1) = Mul64(m, q2);
            (me.t1, c0) = Add64(me.t2, c1, c0);
            (me.u3, c1) = Mul64(m, q3);
            (me.t2, c0) = Add64(me.t3, c1, c0);
            (me.u4, c1) = Mul64(m, q4);
            (me.t3, c0) = Add64(me.t4, c1, c0);
            (me.u5, c1) = Mul64(m, q5);

            (me.t4, c0) = Add64(0, c1, c0);
            (me.u5, na) = Add64(me.u5, 0, c0);
            (me.t0, c0) = Add64(me.u0, me.t0, 0);
            (me.t1, c0) = Add64(me.u1, me.t1, c0);
            (me.t2, c0) = Add64(me.u2, me.t2, c0);
            (me.t3, c0) = Add64(me.u3, me.t3, c0);
            (me.t4, c0) = Add64(me.u4, me.t4, c0);
            (c2, na) = Add64(c2, 0, c0);
            (me.t4, c0) = Add64(me.t5, me.t4, 0);
            (me.t5, na) = Add64(me.u5, c2, c0);
        }
        z.val[0] = me.t0;
        z.val[1] = me.t1;
        z.val[2] = me.t2;
        z.val[3] = me.t3;
        z.val[4] = me.t4;
        z.val[5] = me.t5;
    }

    function Mul64(uint64 a, uint64 b) public pure returns (uint64 high,uint64 low) {
        // Perform 64-bit multiplication
        uint128 result = uint128(a) * uint128(b);

        // Extract the low and high 64-bit parts
        low = uint64(result);
        high = uint64(result >> 64);
    }

    function Add64(uint64 a, uint64 b, uint64 carryIn) public pure returns (uint64 sum, uint64 carryOut) {
        unchecked {
            uint64 tempSum = a + b + carryIn;
            sum = tempSum;
            carryOut = (tempSum < a || tempSum < b || tempSum < carryIn) ? 1 : 0;
        }
    }

    // Helper struct is needed to avoid stack to deep compilation error
    // as there is a limit of defined variables in single function
    struct MulHelper {
        uint64 t0;
        uint64 t1;
        uint64 t2;
        uint64 t3;
        uint64 t4;
        uint64 t5;
        
	    uint64 u0;
        uint64 u1;
	    uint64 u2;
	    uint64 u3;
	    uint64 u4;
	    uint64 u5;
    }
}