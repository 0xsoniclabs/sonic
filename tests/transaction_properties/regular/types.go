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

// Package regular holds the ordinary transactions of the network: the five transaction types, each
// declared as the capabilities it carries, how each draws and assembles itself, and the rules
// deciding whether one can pay its way from its sender's balance. It is the domain every other one
// builds on -- see the subsidies package, which takes this generator and these rules and layers a
// second pricing regime over them.
package regular

import (
	"fmt"

	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/ethereum/go-ethereum/core/types"
	"pgregory.net/rapid"
)

// The transaction types, each declared as the capabilities it carries.

type legacyTx struct {
	core.Envelope
	core.Payload
	core.OptionalRecipient
	core.WideValue
	core.SinglePrice
}

type accessListTx struct {
	core.Envelope
	core.Payload
	core.OptionalRecipient
	core.WideValue
	core.SinglePrice
	core.AccessListEntries
}

type dynamicFeeTx struct {
	core.Envelope
	core.Payload
	core.OptionalRecipient
	core.WideValue
	core.WidePrices
	core.AccessListEntries
}

type blobTx struct {
	core.Envelope
	core.Payload
	core.RequiredRecipient
	core.NarrowValue
	core.NarrowPrices
	core.AccessListEntries
	core.BlobHashList
}

type setCodeTx struct {
	core.Envelope
	core.Payload
	core.RequiredRecipient
	core.NarrowValue
	core.NarrowPrices
	core.AccessListEntries
	core.AuthList
}

func (legacyTx) TxType() uint8     { return types.LegacyTxType }
func (accessListTx) TxType() uint8 { return types.AccessListTxType }
func (dynamicFeeTx) TxType() uint8 { return types.DynamicFeeTxType }
func (blobTx) TxType() uint8       { return types.BlobTxType }
func (setCodeTx) TxType() uint8    { return types.SetCodeTxType }

// Each transaction type draws the capabilities it carries, and nothing else. The recipient comes
// before the payload, because what the payload is depends on whether it is addressed to code; the gas
// limit comes last, because it is drawn relative to the intrinsic cost of everything above it.

func (s *legacyTx) Draw(t *rapid.T, env core.GenEnv) {
	s.Envelope.Draw(t, env)
	s.OptionalRecipient.Draw(t, env)
	s.Payload.DrawFor(t, env, s.To)
	s.WideValue.Draw(t, env)
	s.SinglePrice.Draw(t, env)
	s.GasLimit = DrawGasLimit(t, env, s)
}

func (s *accessListTx) Draw(t *rapid.T, env core.GenEnv) {
	s.Envelope.Draw(t, env)
	s.OptionalRecipient.Draw(t, env)
	s.Payload.DrawFor(t, env, s.To)
	s.WideValue.Draw(t, env)
	s.SinglePrice.Draw(t, env)
	s.AccessListEntries.Draw(t, env)
	s.GasLimit = DrawGasLimit(t, env, s)
}

func (s *dynamicFeeTx) Draw(t *rapid.T, env core.GenEnv) {
	s.Envelope.Draw(t, env)
	s.OptionalRecipient.Draw(t, env)
	s.Payload.DrawFor(t, env, s.To)
	s.WideValue.Draw(t, env)
	s.WidePrices.Draw(t, env)
	s.AccessListEntries.Draw(t, env)
	s.GasLimit = DrawGasLimit(t, env, s)
}

func (s *blobTx) Draw(t *rapid.T, env core.GenEnv) {
	s.Envelope.Draw(t, env)
	s.RequiredRecipient.Draw(t, env)
	s.Payload.DrawFor(t, env, s.To)
	s.NarrowValue.Draw(t, env)
	s.NarrowPrices.Draw(t, env)
	s.AccessListEntries.Draw(t, env)
	s.BlobHashList.Draw(t, env)
	s.GasLimit = DrawGasLimit(t, env, s)
}

func (s *setCodeTx) Draw(t *rapid.T, env core.GenEnv) {
	s.Envelope.Draw(t, env)
	s.RequiredRecipient.Draw(t, env)
	s.Payload.DrawFor(t, env, s.To)
	s.NarrowValue.Draw(t, env)
	s.NarrowPrices.Draw(t, env)
	s.AccessListEntries.Draw(t, env)
	s.AuthList.Draw(t, env)
	s.GasLimit = DrawGasLimit(t, env, s)
}

// Each transaction type assembles its own payload out of the capabilities it carries. types.NewTx
// copies what it is given, so the fields may be shared with the spec.

func (s *legacyTx) TxData(nonce uint64, ctx core.BuildContext) (types.TxData, error) {
	return &types.LegacyTx{
		Nonce:    nonce,
		GasPrice: s.GasPrice,
		Gas:      s.GasLimit,
		To:       ctx.RecipientAddress(s),
		Value:    s.Amount(),
		Data:     s.Data(),
	}, nil
}

func (s *accessListTx) TxData(nonce uint64, ctx core.BuildContext) (types.TxData, error) {
	return &types.AccessListTx{
		ChainID:    ctx.ChainId,
		Nonce:      nonce,
		GasPrice:   s.GasPrice,
		Gas:        s.GasLimit,
		To:         ctx.RecipientAddress(s),
		Value:      s.Amount(),
		Data:       s.Data(),
		AccessList: s.AccessList(),
	}, nil
}

func (s *dynamicFeeTx) TxData(nonce uint64, ctx core.BuildContext) (types.TxData, error) {
	return &types.DynamicFeeTx{
		ChainID:    ctx.ChainId,
		Nonce:      nonce,
		GasTipCap:  s.GasTipCap,
		GasFeeCap:  s.GasFeeCap,
		Gas:        s.GasLimit,
		To:         ctx.RecipientAddress(s),
		Value:      s.Amount(),
		Data:       s.Data(),
		AccessList: s.AccessList(),
	}, nil
}

func (s *blobTx) TxData(nonce uint64, ctx core.BuildContext) (types.TxData, error) {
	return &types.BlobTx{
		ChainID:    core.ToUint256(ctx.ChainId),
		Nonce:      nonce,
		GasTipCap:  core.ToUint256(s.TipCap()),
		GasFeeCap:  core.ToUint256(s.FeeCap()),
		Gas:        s.GasLimit,
		To:         *ctx.RecipientAddress(s),
		Value:      core.ToUint256(s.Amount()),
		Data:       s.Data(),
		AccessList: s.AccessList(),
		BlobFeeCap: core.ToUint256(s.FeeCap()),
		BlobHashes: s.BlobHashes(),
	}, nil
}

func (s *setCodeTx) TxData(nonce uint64, ctx core.BuildContext) (types.TxData, error) {
	auths, err := s.authorizations(ctx)
	if err != nil {
		return nil, err
	}
	return &types.SetCodeTx{
		ChainID:    core.ToUint256(ctx.ChainId),
		Nonce:      nonce,
		GasTipCap:  core.ToUint256(s.TipCap()),
		GasFeeCap:  core.ToUint256(s.FeeCap()),
		Gas:        s.GasLimit,
		To:         *ctx.RecipientAddress(s),
		Value:      core.ToUint256(s.Amount()),
		Data:       s.Data(),
		AccessList: s.AccessList(),
		AuthList:   auths,
	}, nil
}

// authorizations signs the drawn authorization list.
func (s *setCodeTx) authorizations(ctx core.BuildContext) ([]types.SetCodeAuthorization, error) {
	sender := ctx.Sender(s)
	signed := make([]types.SetCodeAuthorization, 0, len(s.Entries))
	for _, choice := range s.Entries {
		authorization := types.SetCodeAuthorization{
			ChainID: *core.ToUint256(ctx.ChainId),
			Address: sender.Address(),
		}
		switch choice {
		case core.AuthOther:
			authorization.Address = ctx.Bystander(s)
		case core.AuthWrongChainID:
			authorization.ChainID.AddUint64(&authorization.ChainID, 1)
		}
		auth, err := types.SignSetCode(sender.PrivateKey, authorization)
		if err != nil {
			return nil, fmt.Errorf("failed to sign authorization: %w", err)
		}
		signed = append(signed, auth)
	}
	return signed, nil
}
