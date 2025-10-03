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

package gas_subsidies

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestGasSubsidies_HelperFunctions(t *testing.T) {

	upgrades := opera.GetAllegroUpgrades()
	upgrades.GasSubsidies = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	sponsee := tests.NewAccount()
	receiver := tests.NewAccount()
	receiverAddress := receiver.Address()

	donation := big.NewInt(1e18)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Donate some money into a fund for a specific account
	registry := Fund(t, net, sponsee.Address(), donation)

	tx := types.LegacyTx{
		To:       &receiverAddress,
		Gas:      21000,
		GasPrice: big.NewInt(1e9),
	}

	// make a sponsorship request transaction from the sponsee account
	sponsoredTx := makeSponsorRequestTransaction(t, &tx, net.GetChainId(), sponsee)
	require.Equal(t, sponsoredTx.GasPrice(), big.NewInt(0))

	// need to wait for subsidies to be implemented.
	receipt, err := net.Run(sponsoredTx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	validateSponsoredTxInBlock(t, net, sponsoredTx.Hash())

	// check that the sponsorship funds got deducted
	ok, fundId, err := registry.AccountSponsorshipFundId(nil, sponsee.Address())
	require.NoError(t, err)
	require.True(t, ok, "registry should have a fund ID")

	tests.WaitForProofOf(t, client, int(receipt.BlockNumber.Int64()))

	sponsorship, err := registry.Sponsorships(nil, fundId)
	require.NoError(t, err)
	require.Less(t, sponsorship.Funds.Uint64(), donation.Uint64())

	// check that the transaction after the sponsored one is the payment internal tx
	txIndex, block := getTransactionIndexInBlock(t, client, receipt)
	require.GreaterOrEqual(t, len(block.Transactions()), txIndex+1)
	require.Equal(t, receipt.TxHash, block.Transactions()[txIndex].Hash())
	require.True(t, internaltx.IsInternal(block.Transactions()[txIndex+1])) // this check is only for subsidized transactions
}

func TestMakeSponsorRequestTransaction_CanHandleAllTransactionTypes(t *testing.T) {

	txs := []types.TxData{
		&types.LegacyTx{GasPrice: big.NewInt(1), To: &common.Address{42}},
		&types.AccessListTx{GasPrice: big.NewInt(1), To: &common.Address{42}},
		&types.DynamicFeeTx{GasFeeCap: big.NewInt(1), GasTipCap: big.NewInt(1), To: &common.Address{42}},
		&types.BlobTx{GasFeeCap: uint256.NewInt(1), GasTipCap: uint256.NewInt(1)},
		&types.SetCodeTx{GasFeeCap: uint256.NewInt(1), GasTipCap: uint256.NewInt(1)},
	}

	for _, tx := range txs {
		t.Run(fmt.Sprintf("%T", tx), func(t *testing.T) {
			sponsoredTx := makeSponsorRequestTransaction(t, tx, big.NewInt(1), tests.NewAccount())
			require.Equal(t, sponsoredTx.GasPrice(), big.NewInt(0))
			require.Equal(t, sponsoredTx.GasTipCap(), big.NewInt(0))
			require.True(t, subsidies.IsSponsorshipRequest(sponsoredTx))
		})
	}
}
