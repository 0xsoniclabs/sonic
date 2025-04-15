package valkeystore

import (
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/valkeystore/encryption"
)

//go:generate mockgen -source=keystore.go -destination=keystore_mock.go -package=valkeystore

type RawKeystoreI interface {
	Has(pubkey validatorpk.PubKey) bool
	Add(pubkey validatorpk.PubKey, key []byte, auth string) error
	Get(pubkey validatorpk.PubKey, auth string) (*encryption.PrivateKey, error)
}

type KeystoreI interface {
	RawKeystoreI
	Unlock(pubkey validatorpk.PubKey, auth string) error
	Unlocked(pubkey validatorpk.PubKey) bool
	GetUnlocked(pubkey validatorpk.PubKey) (*encryption.PrivateKey, error)
}
