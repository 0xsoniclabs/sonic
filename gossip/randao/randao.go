// prevrandao package provides the functionality
// to generate the prevrandao value for the sonic consensus
// protocol.
package randao

import (
	"crypto/sha256"

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/valkeystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// RandaoReveal contains the randao reveal value, which can be used to generate
// the next randao value. RandaoReveal can be sent in events to other peers
// to inform them of the new randao value, and allow them to verify it.
//
// The randao reveal is computed by a proposing validator
// using the following formula:
//   - hash = sha256(domainSeparator + previousRandao)
//   - randaoReveal = signature(proposerPrivateKey, hash)
//
// Where + is the concatenation operator, and signature is the concatenation of
// the R&S values of the ECDSA signature using Secp256k1 (64 bytes long).
//
// The next randao value is computed by the following formula:
//   - nextRandao = sha256(randaoReveal)
//
// Peers receiving a randao reveal can verify that the proposer followed the
// protocol by checking that the signature is valid for:
//   - hash = sha256(domainSeparator + previousRandao)
//
// Where both domainSeparator and previousRandao are known to every peer.
type RandaoReveal [64]byte

// GenerateNextRandaoReveal Constructs a new RandaoReveal
//   - previousRandao is the previous randao value
//   - proposerKey is the public key of the proposer originating this randao value
//   - Signer is the signer used to sign messages within the gossip package
func GenerateNextRandaoReveal(
	previousRandao common.Hash,
	validatorSigner valkeystore.SignerAuthority,
) (RandaoReveal, error) {
	hash := sha256.Sum256(append(domainSeparator[:], previousRandao[:]...))
	buff, err := validatorSigner.Sign(hash)
	if err != nil {
		return RandaoReveal{}, err
	}
	var result RandaoReveal
	copy(result[:], buff[:])
	return result, nil
}

// VerifyAndGetRandao verifies randaoReveal and extracts a the corresponding randao value.
// If the verification can be proven, this value is equal on all peers.
//   - previousRandao is the previous randao value
//   - proposerKey is the public key of the proposer originating this randao reveal
func (s RandaoReveal) VerifyAndGetRandao(
	previousRandao common.Hash,
	proposerPublicKey validatorpk.PubKey,
) (common.Hash, bool) {

	hash := sha256.Sum256(append(domainSeparator[:], previousRandao[:]...))

	// if the signature does not correspond to the input data
	// for the given proposerPublicKey, then randao cannot be generated.
	if ok := crypto.VerifySignature(proposerPublicKey.Raw, hash[:], s[:]); !ok {
		return common.Hash{}, false
	}

	// next randao is the hash of the reveal
	return sha256.Sum256(s[:]), true
}

// domainSeparator is the domain separator used to generate the randao value
// and is used to verify the randao reveal signature.
// https://en.wikipedia.org/wiki/Domain_separation
var domainSeparator = []byte("Sonic-Randao")
