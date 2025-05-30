package valkeystore

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/valkeystore/encryption"
)

//go:generate mockgen -source=signer.go -destination=signer_mock.go  -package=valkeystore

type SignerAuthority interface {
	Sign(digest common.Hash) ([]byte, error)
	PublicKey() validatorpk.PubKey
}

type signerAuthorityImpl struct {
	backend KeystoreI
	pubkey  validatorpk.PubKey
}

func NewSignerAuthority(store KeystoreI, pubkey validatorpk.PubKey) SignerAuthority {
	return &signerAuthorityImpl{
		backend: store,
		pubkey:  pubkey,
	}
}

func (s *signerAuthorityImpl) Sign(digest common.Hash) ([]byte, error) {
	if s.pubkey.Type != validatorpk.Types.Secp256k1 {
		return nil, encryption.ErrNotSupportedType
	}
	key, err := s.backend.GetUnlocked(s.pubkey)
	if err != nil {
		return nil, err
	}

	secp256k1Key := key.Decoded.(*ecdsa.PrivateKey)

	sigRSV, err := crypto.Sign(digest[:], secp256k1Key)
	if err != nil {
		return nil, err
	}
	sigRS := sigRSV[:64]
	return sigRS, err
}

func (s *signerAuthorityImpl) PublicKey() validatorpk.PubKey {
	return s.pubkey.Copy()
}

func VerifySignature(digest common.Hash, signature []byte, pubkey validatorpk.PubKey) bool {
	if pubkey.Type != validatorpk.Types.Secp256k1 {
		return false
	}
	return crypto.VerifySignature(pubkey.Raw, digest[:], signature)
}
