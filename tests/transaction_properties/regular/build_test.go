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

package regular

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core/contracts"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// testBuildContext is a build context over deterministic keys, needing no network.
func testBuildContext(t *testing.T) core.BuildContext {
	t.Helper()

	accounts := make([]core.PooledAccount, 2)
	for i := range accounts {
		key, err := core.DeriveKey(i)
		require.NoError(t, err)
		accounts[i] = core.PooledAccount{Account: &tests.Account{PrivateKey: key}, Index: i}
	}

	return core.BuildContext{
		ChainId:     big.NewInt(4242),
		Accounts:    accounts,
		UnfundedKey: core.UnfundedKey,
	}
}

// specOfEveryType returns one buildable spec per transaction type, with fields a builder can carry.
func specOfEveryType() map[uint8]core.TxSpec {
	price := big.NewInt(1_000)
	value := big.NewInt(7)

	return map[uint8]core.TxSpec{
		types.LegacyTxType: &legacyTx{
			Envelope:          core.Envelope{GasLimit: 100_000},
			Payload:           core.Payload{DataLen: 3, DataNonZero: true},
			OptionalRecipient: core.OptionalRecipient{To: core.ToSelf},
			WideValue:         core.WideValue{Value: value},
			SinglePrice:       core.SinglePrice{GasPrice: price},
		},
		types.AccessListTxType: &accessListTx{
			Envelope:          core.Envelope{GasLimit: 100_000},
			OptionalRecipient: core.OptionalRecipient{To: core.ToOther},
			WideValue:         core.WideValue{Value: value},
			SinglePrice:       core.SinglePrice{GasPrice: price},
			AccessListEntries: core.AccessListEntries{AccessListLen: 2},
		},
		types.DynamicFeeTxType: &dynamicFeeTx{
			Envelope:          core.Envelope{GasLimit: 100_000},
			OptionalRecipient: core.OptionalRecipient{To: core.ToCreate},
			WideValue:         core.WideValue{Value: value},
			WidePrices:        core.WidePrices{GasFeeCap: price, GasTipCap: big.NewInt(1)},
			AccessListEntries: core.AccessListEntries{AccessListLen: 1},
		},
		types.BlobTxType: &blobTx{
			Envelope:          core.Envelope{GasLimit: 100_000},
			RequiredRecipient: core.RequiredRecipient{To: core.ToSelf},
			NarrowValue:       core.NarrowValue{Value: value},
			NarrowPrices:      core.NarrowPrices{GasFeeCap: price, GasTipCap: big.NewInt(1)},
			BlobHashList:      core.BlobHashList{BlobHashLen: 1},
		},
		types.SetCodeTxType: &setCodeTx{
			Envelope:          core.Envelope{GasLimit: 100_000},
			RequiredRecipient: core.RequiredRecipient{To: core.ToPrecompile},
			NarrowValue:       core.NarrowValue{Value: value},
			NarrowPrices:      core.NarrowPrices{GasFeeCap: price, GasTipCap: big.NewInt(1)},
			AuthList:          core.AuthList{Entries: []core.AuthChoice{core.AuthSelf, core.AuthOther, core.AuthWrongChainID}},
		},
	}
}

func TestBuildTx_CarriesTheSpecIntoATransactionOfThatType(t *testing.T) {
	ctx := testBuildContext(t)

	for txType, spec := range specOfEveryType() {
		t.Run(core.Format(spec), func(t *testing.T) {
			tx, err := core.BuildTx(spec, 17, ctx)
			require.NoError(t, err)

			require.Equal(t, txType, tx.Type())
			require.Equal(t, uint64(17), tx.Nonce())
			require.Equal(t, spec.Gas(), tx.Gas())
			require.Equal(t, spec.Amount(), tx.Value())
			require.Equal(t, spec.Data(), tx.Data())
			require.Equal(t, spec.FeeCap(), tx.GasFeeCap())
			require.Equal(t, spec.TipCap(), tx.GasTipCap())
			require.Len(t, tx.AccessList(), len(core.AccessListOf(spec)))
			require.Len(t, tx.BlobHashes(), len(core.BlobHashesOf(spec)))
			require.Len(t, tx.SetCodeAuthorizations(), len(core.AuthsOf(spec)))
		})
	}
}

// deployedContract stands in for a contract put on chain before a run, which is all a spec holds of
// one: where it is, and what to send it.
var deployedContract = &contracts.Call{Address: common.Address{0xc0, 0xde}}

func TestBuildTx_ResolvesTheRecipientOfEachChoice(t *testing.T) {
	ctx := testBuildContext(t)
	sender := ctx.Accounts[0].Address()

	tests := map[core.ToChoice]func(tx *types.Transaction){
		core.ToSelf: func(tx *types.Transaction) {
			require.Equal(t, sender, *tx.To())
		},
		core.ToOther: func(tx *types.Transaction) {
			require.NotEqual(t, sender, *tx.To(), "a bystander must not be an observed account")
			for _, account := range ctx.Accounts {
				require.NotEqual(t, account.Address(), *tx.To())
			}
		},
		core.ToPrecompile: func(tx *types.Transaction) {
			require.Equal(t, core.PrecompileAddress, *tx.To())
		},
		core.ToCreate: func(tx *types.Transaction) {
			require.Nil(t, tx.To(), "an empty recipient is what makes it a creation")
		},
		core.ToContract: func(tx *types.Transaction) {
			require.Equal(t, deployedContract.Address, *tx.To(),
				"a contract call must reach the contract whose call data it carries")
			require.Equal(t, deployedContract.Data(), tx.Data())
		},
	}

	for choice, check := range tests {
		t.Run(choice.String(), func(t *testing.T) {
			spec := &legacyTx{
				Envelope:          core.Envelope{GasLimit: 100_000},
				Payload:           core.Payload{Call: deployedContract},
				OptionalRecipient: core.OptionalRecipient{To: choice},
				WideValue:         core.WideValue{Value: big.NewInt(0)},
				SinglePrice:       core.SinglePrice{GasPrice: big.NewInt(1)},
			}
			tx, err := core.BuildTx(spec, 0, ctx)
			require.NoError(t, err)
			check(tx)
		})
	}
}

func TestBuildTx_ATypeThatCannotExpressACreationSendsToItsSenderInstead(t *testing.T) {
	ctx := testBuildContext(t)

	spec := &blobTx{
		Envelope:          core.Envelope{GasLimit: 100_000},
		RequiredRecipient: core.RequiredRecipient{To: core.ToCreate},
		NarrowValue:       core.NarrowValue{Value: big.NewInt(0)},
		NarrowPrices:      core.NarrowPrices{GasFeeCap: big.NewInt(1), GasTipCap: big.NewInt(1)},
	}

	tx, err := core.BuildTx(spec, 0, ctx)
	require.NoError(t, err)
	require.NotNil(t, tx.To())
	require.Equal(t, ctx.Accounts[0].Address(), *tx.To())
	require.False(t, spec.IsCreate())
}

// TestSignSpec_EveryDefectiveChoiceProducesADefect checks the point of the signing choices: each one
// other than core.SignCorrect must fail to recover the sender, or recover somebody who can pay nothing.
func TestSignSpec_EveryDefectiveChoiceProducesADefect(t *testing.T) {
	ctx := testBuildContext(t)
	signer := types.NewPragueSigner(ctx.ChainId)
	sender := ctx.Accounts[0].Address()
	unfunded := crypto.PubkeyToAddress(core.UnfundedKey.PublicKey)

	for txType, spec := range specOfEveryType() {
		for _, choice := range []core.SigningChoice{
			core.SignCorrect, core.SignWrongChainID, core.SignUnfundedKey, core.SignZeroR, core.SignZeroS, core.SignForeign,
		} {
			t.Run(choice.String(), func(t *testing.T) {
				tx, err := core.BuildTx(withSigning(spec, choice), 0, ctx)
				if err != nil {
					// A signature that cannot be attached at all is a defect of the intended kind.
					require.NotEqual(t, core.SignCorrect, choice)
					return
				}

				recovered, err := types.Sender(signer, tx)
				switch choice {
				case core.SignCorrect:
					require.NoError(t, err, "type %d must sign correctly", txType)
					require.Equal(t, sender, recovered)
				case core.SignUnfundedKey:
					require.NoError(t, err)
					require.Equal(t, unfunded, recovered,
						"the sender must be recoverable but hold nothing")
				default:
					if err == nil {
						require.NotEqual(t, sender, recovered,
							"%s must not recover to the funded sender", choice)
					}
				}
			})
		}
	}
}

// withSigning returns a copy of the spec signed differently, so one table can drive every choice.
func withSigning(spec core.TxSpec, choice core.SigningChoice) core.TxSpec {
	switch typed := spec.(type) {
	case *legacyTx:
		copied := *typed
		copied.Signing = choice
		return &copied
	case *accessListTx:
		copied := *typed
		copied.Signing = choice
		return &copied
	case *dynamicFeeTx:
		copied := *typed
		copied.Signing = choice
		return &copied
	case *blobTx:
		copied := *typed
		copied.Signing = choice
		return &copied
	case *setCodeTx:
		copied := *typed
		copied.Signing = choice
		return &copied
	}
	panic("unhandled transaction type")
}

func TestBuildTx_AuthorizationsFollowTheirChoices(t *testing.T) {
	ctx := testBuildContext(t)
	sender := ctx.Accounts[0].Address()

	spec := &setCodeTx{
		Envelope:          core.Envelope{GasLimit: 100_000},
		RequiredRecipient: core.RequiredRecipient{To: core.ToSelf},
		NarrowValue:       core.NarrowValue{Value: big.NewInt(0)},
		NarrowPrices:      core.NarrowPrices{GasFeeCap: big.NewInt(1), GasTipCap: big.NewInt(1)},
		AuthList:          core.AuthList{Entries: []core.AuthChoice{core.AuthSelf, core.AuthOther, core.AuthWrongChainID}},
	}

	tx, err := core.BuildTx(spec, 0, ctx)
	require.NoError(t, err)

	auths := tx.SetCodeAuthorizations()
	require.Len(t, auths, 3)
	require.Equal(t, sender, auths[0].Address, "AuthSelf delegates the sender")
	require.NotEqual(t, sender, auths[1].Address, "AuthOther delegates a bystander")
	require.Equal(t, ctx.ChainId.Uint64(), auths[0].ChainID.Uint64())
	require.Equal(t, ctx.ChainId.Uint64()+1, auths[2].ChainID.Uint64(),
		"AuthWrongChainID signs for another chain")
}

func TestWithChainId_RewritesEveryTypeThatHasTheField(t *testing.T) {
	other := big.NewInt(99)

	for txType, spec := range specOfEveryType() {
		inner, err := spec.TxData(0, testBuildContext(t))
		require.NoError(t, err)

		rewritten, err := core.WithChainId(inner, other)
		if txType == types.LegacyTxType {
			require.Error(t, err, "a legacy transaction has no chain ID field")
			continue
		}
		require.NoError(t, err)
		require.Equal(t, other, types.NewTx(rewritten).ChainId())

		// The original is left alone, so a spec can be built more than once.
		require.NotEqual(t, other, types.NewTx(inner).ChainId())
	}
}
