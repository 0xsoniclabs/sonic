// Copyright 2026 Sonic Operations Ltd
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

package core

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// PrecompileAddress is the ecrecover precompile, a recipient that exists but is not an ordinary
// account.
var PrecompileAddress = common.Address{0x01}

// BuildContext supplies what a spec needs to become a transaction, kept apart from the spec so that
// the spec stays pure data.
type BuildContext struct {
	ChainId     *big.Int
	Accounts    []PooledAccount
	UnfundedKey *ecdsa.PrivateKey
}

func (c BuildContext) Sender(spec TxSpec) PooledAccount {
	return c.Accounts[spec.Sender()%len(c.Accounts)]
}

// bystander returns an address that can receive value but is never itself under observation, so every
// observed account's balance stays a function of its own transactions alone.
func (c BuildContext) Bystander(spec TxSpec) common.Address {
	index := spec.Sender() % len(c.Accounts)
	address := common.Address{0xb7, 0x57, 0xa4, 0xde}
	address[18] = byte(index >> 8)
	address[19] = byte(index)
	return address
}

// recipient resolves a drawn recipient to an address, or to nil for a contract creation.
func (c BuildContext) RecipientAddress(spec TxSpec) *common.Address {
	if spec.IsCreate() {
		return nil
	}
	switch spec.Recipient() {
	case ToContract:
		if call := CallOf(spec); call != nil {
			return &call.Address
		}
		address := c.Bystander(spec) // a spec naming a contract it never drew has nowhere else to go
		return &address
	case ToPrecompile:
		return &PrecompileAddress
	case ToOther:
		address := c.Bystander(spec)
		return &address
	default: // ToSelf
		address := c.Sender(spec).Address()
		return &address
	}
}

// BuildTx materialises a spec into a signed transaction, taking the nonce from the caller because it
// has to match what the model assumed. It deliberately avoids tests.SetTransactionDefaults and
// tests.CreateTransaction, which fill in zero-valued nonce, gas and fee fields and so would repair the
// very defects being generated. An error means the spec is not representable at all, which is a
// harness limit rather than a finding.
func BuildTx(spec TxSpec, nonce uint64, ctx BuildContext) (*types.Transaction, error) {
	inner, err := spec.TxData(nonce, ctx)
	if err != nil {
		return nil, err
	}
	return SignSpec(spec, inner, ctx)
}

// SignSpec signs according to the spec's signing choice. It signs the hash directly rather than
// through types.SignNewTx, because that is the only way to corrupt the raw 65-byte signature before
// attaching it and so produce a sender that cannot be recovered.
func SignSpec(spec TxSpec, inner types.TxData, ctx BuildContext) (*types.Transaction, error) {
	key := ctx.Sender(spec).PrivateKey
	signer := types.NewPragueSigner(ctx.ChainId)

	switch spec.SigningMode() {
	case SignWrongChainID:
		wrongChainId := new(big.Int).Add(ctx.ChainId, big.NewInt(1))
		if spec.TxType() == types.LegacyTxType {
			return sign(inner, types.NewEIP155Signer(wrongChainId), key, nil)
		}
		wrongInner, err := WithChainId(inner, wrongChainId)
		if err != nil {
			return nil, err
		}
		return sign(wrongInner, types.NewPragueSigner(wrongChainId), key, nil)

	case SignUnfundedKey:
		return sign(inner, signer, ctx.UnfundedKey, nil)

	case SignZeroR:
		return sign(inner, signer, key, func(sig []byte) { clear(sig[0:32]) })

	case SignZeroS:
		return sign(inner, signer, key, func(sig []byte) { clear(sig[32:64]) })

	case SignForeign:
		return signHash(inner, signer, key, func(hash common.Hash) common.Hash {
			return crypto.Keccak256Hash(hash[:])
		})

	default: // SignCorrect
		return sign(inner, signer, key, nil)
	}
}

// sign hashes, signs and attaches, applying mutate to the raw signature in between when it is not nil.
func sign(
	inner types.TxData,
	signer types.Signer,
	key *ecdsa.PrivateKey,
	mutate func(sig []byte),
) (*types.Transaction, error) {
	tx := types.NewTx(inner)
	sig, err := crypto.Sign(signer.Hash(tx).Bytes(), key)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}
	if mutate != nil {
		mutate(sig)
	}
	signed, err := tx.WithSignature(signer, sig)
	if err != nil {
		return nil, fmt.Errorf("failed to attach signature: %w", err)
	}
	return signed, nil
}

// signHash signs a hash derived from the transaction's own, producing a valid signature over the wrong
// payload.
func signHash(
	inner types.TxData,
	signer types.Signer,
	key *ecdsa.PrivateKey,
	derive func(common.Hash) common.Hash,
) (*types.Transaction, error) {
	tx := types.NewTx(inner)
	sig, err := crypto.Sign(derive(signer.Hash(tx)).Bytes(), key)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}
	return tx.WithSignature(signer, sig)
}

// WithChainId rewrites the chain ID of a typed transaction payload.
func WithChainId(inner types.TxData, chainId *big.Int) (types.TxData, error) {
	switch typed := inner.(type) {
	case *types.AccessListTx:
		copied := *typed
		copied.ChainID = chainId
		return &copied, nil
	case *types.DynamicFeeTx:
		copied := *typed
		copied.ChainID = chainId
		return &copied, nil
	case *types.BlobTx:
		copied := *typed
		copied.ChainID = ToUint256(chainId)
		return &copied, nil
	case *types.SetCodeTx:
		copied := *typed
		copied.ChainID = ToUint256(chainId)
		return &copied, nil
	}
	return nil, fmt.Errorf("transaction type %T has no chain ID field", inner)
}
