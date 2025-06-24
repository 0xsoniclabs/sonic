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

// prevrandao package provides the functionality
// to generate the prevrandao value for the sonic consensus
// protocol.
package randao

import (
	"crypto/sha256"
	"fmt"

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/valkeystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

//go:generate mockgen -source=randao.go -destination=randao_mock.go -package=randao

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

// RandaoMixer is an interface to abstract the randao mixing process.
// It provides a method to return the reveal and mix hash for the next
// block without exposing the need for this module to know about the
// validator keys.
type RandaoMixer interface {
	MixRandao(prevRandao common.Hash) (RandaoReveal, common.Hash, error)
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

type RandaoMixerAdapter struct {
	signer valkeystore.SignerAuthority
}

func NewRandaoMixerAdapter(signer valkeystore.SignerAuthority) *RandaoMixerAdapter {
	return &RandaoMixerAdapter{
		signer: signer,
	}
}

func (r *RandaoMixerAdapter) MixRandao(prevRandao common.Hash) (RandaoReveal, common.Hash, error) {
	reveal, err := generateNextRandaoReveal(prevRandao, r.signer)
	if err != nil {
		return reveal, common.Hash{}, err
	}

	mix, ok := reveal.VerifyAndGetRandao(prevRandao, r.signer.PublicKey())
	if !ok {
		return reveal, common.Hash{}, fmt.Errorf("failed to generate next randao reveal, randao reveal verification failed")
	}
	return reveal, mix, nil
}

// generateNextRandaoReveal Constructs a new RandaoReveal
//   - previousRandao is the previous randao value
//   - proposerKey is the public key of the proposer originating this randao value
//   - Signer is the signer used to sign messages within the gossip package
func generateNextRandaoReveal(
	previousRandao common.Hash,
	validatorSigner valkeystore.SignerAuthority,
) (RandaoReveal, error) {
	hash := sha256.Sum256(append(domainSeparator[:], previousRandao[:]...))
	buff, err := validatorSigner.Sign(hash)
	if err != nil {
		return RandaoReveal{}, fmt.Errorf("failed to generate next randao reveal: %w", err)
	}
	var result RandaoReveal
	copy(result[:], buff[:])
	return result, nil
}
