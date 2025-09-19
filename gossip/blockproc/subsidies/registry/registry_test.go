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

package registry

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestGetCode_CodeIsNotEmpty(t *testing.T) {
	require.NotEmpty(t, GetCode())
}

func TestContractAddressMatchesCreatorAddress(t *testing.T) {
	require := require.New(t)
	_, creator := GetDeploymentTransaction()
	want := crypto.CreateAddress(creator, 0)
	got := GetAddress()
	require.Equal(want, got)
}

func TestGetDeploymentTransaction_ReturnsValidTransaction(t *testing.T) {
	require := require.New(t)

	tx, creator := GetDeploymentTransaction()
	require.NotNil(tx, "transaction is nil")

	signer := types.LatestSignerForChainID(nil)
	from, err := types.Sender(signer, tx)
	require.NoError(err, "failed to derive sender from transaction")
	require.Equal(creator, from, "creator address does not match derived sender address")

	want := getUnsignedDeploymentTransaction()

	require.Equal(big.NewInt(0), tx.ChainId())
	require.Equal(want.Nonce, tx.Nonce())
	require.Equal(want.GasPrice, tx.GasPrice())
	require.Equal(want.Gas, tx.Gas())
	require.Equal(want.Data, tx.Data())
}

func TestGenerateDeploymentTransaction(t *testing.T) {
	// This function was used to generate the deployment transaction for the
	// subsidies registry contract. The output of this function was copied to
	// into the implementation. It is kept in case the deployed code needs to
	// be updated.

	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}

	creator := crypto.PubkeyToAddress(key.PublicKey)
	addr := crypto.CreateAddress(creator, 0)

	signer := types.LatestSignerForChainID(nil)
	tx := types.MustSignNewTx(key, signer, getUnsignedDeploymentTransaction())

	fmt.Printf("Creator address: %s\n", creator.Hex())
	fmt.Printf("Contract address: %s\n", addr.Hex())

	fmt.Printf("Transaction data:\n")
	fmt.Printf("  Nonce: %d\n", tx.Nonce())
	fmt.Printf("  Gas Price: 0x%x = %d\n", tx.GasPrice().Uint64(), tx.GasPrice().Uint64())
	fmt.Printf("  Gas Limit: 0x%x = %d\n", tx.Gas(), tx.Gas())
	v, r, s := tx.RawSignatureValues()
	fmt.Printf("  V: 0x%x\n", v)
	fmt.Printf("  R: 0x%x\n", r)
	fmt.Printf("  S: 0x%x\n", s)

	//t.Fail() // uncomment to see the output when running "go test"
}
