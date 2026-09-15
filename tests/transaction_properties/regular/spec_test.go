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
	"bytes"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core/contracts"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestTxTypes_RegistryAgreesWithEachTypeItBuilds(t *testing.T) {
	for txType, newSpec := range txTypes {
		spec := newSpec()
		require.Equal(t, txType, spec.TxType(),
			"the registry key must be the type the spec reports")
	}
	require.Len(t, txTypes, 5)
	require.Equal(t, []uint8{
		types.LegacyTxType, types.AccessListTxType, types.DynamicFeeTxType,
		types.BlobTxType, types.SetCodeTxType,
	}, knownTxTypes, "the known types must be in ascending order, so drawing is reproducible")
}

func TestPayload_HoldsOnlyZeroOrNonZeroBytes(t *testing.T) {
	require.Nil(t, core.Payload{DataLen: 0}.Data())
	require.Nil(t, core.Payload{DataLen: -1}.Data())

	zeros := core.Payload{DataLen: 4}.Data()
	require.Equal(t, []byte{0, 0, 0, 0}, zeros)

	ones := core.Payload{DataLen: 4, DataNonZero: true}.Data()
	require.Equal(t, []byte{1, 1, 1, 1}, ones)
	require.Zero(t, bytes.Count(ones, []byte{0}))
}

func TestPayload_ACallSuppliesTheDataInsteadOfTheDrawnLength(t *testing.T) {
	call := &contracts.Call{Args: []byte{9}}
	payload := core.Payload{DataLen: 4, Call: call}
	require.Equal(t, call.Data(), payload.Data())
	require.Equal(t, call, core.CallOf(&legacyTx{Payload: payload}))
}

func TestRecipient_OnlyAPointerRecipientCanBeACreation(t *testing.T) {
	require.True(t, core.OptionalRecipient{To: core.ToCreate}.IsCreate())
	require.False(t, core.OptionalRecipient{To: core.ToSelf}.IsCreate())

	// A type holding its recipient by value cannot express a creation whatever it was given.
	require.False(t, core.RequiredRecipient{To: core.ToCreate}.IsCreate())
	require.Equal(t, core.ToCreate, core.RequiredRecipient{To: core.ToCreate}.Recipient())
}

func TestValue_NarrowFieldsSaturateAtTheUint256Maximum(t *testing.T) {
	wide := new(big.Int).Lsh(big.NewInt(1), 300)

	require.Equal(t, wide, core.WideValue{Value: wide}.Amount(),
		"a big.Int field carries the value as drawn")
	require.Equal(t, 301, core.WideValue{Value: wide}.Amount().BitLen())

	require.Equal(t, 256, core.NarrowValue{Value: wide}.Amount().BitLen(),
		"a uint256 field cannot carry more than 256 bits")
	require.Equal(t, new(uint256.Int).SetAllOne().ToBig(), core.NarrowValue{Value: wide}.Amount())
}

func TestPrices_SinglePriceReportsOneFieldAsBothCaps(t *testing.T) {
	price := core.SinglePrice{GasPrice: big.NewInt(11)}

	require.Equal(t, big.NewInt(11), price.FeeCap())
	require.Equal(t, big.NewInt(11), price.TipCap())
	require.False(t, price.SeparateTipCap(),
		"a tip above the fee cap must be unrepresentable for this type")
}

func TestPrices_SeparateCapsAreReportedApart(t *testing.T) {
	wide := core.WidePrices{GasFeeCap: big.NewInt(9), GasTipCap: big.NewInt(4)}
	require.Equal(t, big.NewInt(9), wide.FeeCap())
	require.Equal(t, big.NewInt(4), wide.TipCap())
	require.True(t, wide.SeparateTipCap())

	tooWide := new(big.Int).Lsh(big.NewInt(1), 300)
	narrow := core.NarrowPrices{GasFeeCap: tooWide, GasTipCap: tooWide}
	require.Equal(t, 256, narrow.FeeCap().BitLen())
	require.Equal(t, 256, narrow.TipCap().BitLen())
}

func TestAccessList_HasOneStorageKeyPerEntry(t *testing.T) {
	require.Empty(t, core.AccessListEntries{AccessListLen: 0}.AccessList())

	list := core.AccessListEntries{AccessListLen: 3}.AccessList()
	require.Len(t, list, 3)

	addresses := map[common.Address]bool{}
	for _, entry := range list {
		require.Len(t, entry.StorageKeys, 1)
		addresses[entry.Address] = true
	}
	require.Len(t, addresses, 3, "every entry must touch a distinct address")
}

func TestBlobHashes_CarryTheVersionByteEachTime(t *testing.T) {
	hashes := core.BlobHashList{BlobHashLen: 2}.BlobHashes()

	require.Len(t, hashes, 2)
	for i, hash := range hashes {
		require.Equal(t, uint8(core.BlobHashVersion), hash[0],
			"hash %d must be structurally valid, not only the first", i)
	}
	require.NotEqual(t, hashes[0], hashes[1])
}

func TestOptionalCapabilities_AbsenceIsExpressedByAbsence(t *testing.T) {
	tests := map[string]struct {
		spec       core.TxSpec
		accessList bool
		blobHashes bool
		auths      bool
	}{
		"legacy carries none of them":    {spec: &legacyTx{}},
		"access list":                    {spec: &accessListTx{}, accessList: true},
		"dynamic fee":                    {spec: &dynamicFeeTx{}, accessList: true},
		"blob carries hashes":            {spec: &blobTx{}, accessList: true, blobHashes: true},
		"set code carries authorization": {spec: &setCodeTx{}, accessList: true, auths: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, hasAccessList := test.spec.(core.WithAccessList)
			_, hasBlobHashes := test.spec.(core.WithBlobHashes)
			_, hasAuths := test.spec.(core.WithAuths)

			require.Equal(t, test.accessList, hasAccessList)
			require.Equal(t, test.blobHashes, hasBlobHashes)
			require.Equal(t, test.auths, hasAuths)

			if !test.accessList {
				require.Nil(t, core.AccessListOf(test.spec))
			}
			if !test.blobHashes {
				require.Nil(t, core.BlobHashesOf(test.spec))
			}
			if !test.auths {
				require.Nil(t, core.AuthsOf(test.spec))
			}
		})
	}
}

func TestToUint256_SaturatesRatherThanWrapping(t *testing.T) {
	require.Equal(t, uint256.NewInt(5), core.ToUint256(big.NewInt(5)))

	maximum := new(uint256.Int).SetAllOne()
	require.Equal(t, maximum, core.ToUint256(maximum.ToBig()))
	require.Equal(t, maximum, core.ToUint256(new(big.Int).Add(maximum.ToBig(), big.NewInt(1))))
}

func TestFormat_NamesEveryCapabilityTheSpecCarries(t *testing.T) {
	spec := &setCodeTx{
		Envelope:          core.Envelope{SenderIdx: 2, Nonce: core.NonceGap, NonceGapSize: 3, GasLimit: 99, Signing: core.SignZeroR},
		Payload:           core.Payload{DataLen: 7},
		RequiredRecipient: core.RequiredRecipient{To: core.ToPrecompile},
		NarrowValue:       core.NarrowValue{Value: big.NewInt(5)},
		NarrowPrices:      core.NarrowPrices{GasFeeCap: big.NewInt(3), GasTipCap: big.NewInt(1)},
		AccessListEntries: core.AccessListEntries{AccessListLen: 2},
		AuthList:          core.AuthList{Entries: []core.AuthChoice{core.AuthSelf}},
	}

	rendered := core.Format(spec)
	for _, want := range []string{
		"type=4", "sender=2", "nonce=Gap(+3)", "gas=99", "signing=ZeroR",
		"data=7B", "to=Precompile", "accessList=2", "auths=[Self]",
	} {
		require.Contains(t, rendered, want)
	}

	// A capability the type does not carry is not mentioned at all.
	require.NotContains(t, core.Format(&legacyTx{}), "accessList")
	require.NotContains(t, core.Format(&legacyTx{}), "blobHashes")
}
