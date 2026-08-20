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

package tests

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/contracts/transitive_call"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// transfer is the decoded form of an EIP-7708 ETH transfer log.
type transfer struct {
	from  common.Address
	to    common.Address
	value int64
}

func TestEip7708_ValueTransfersEmitTransferLogsFromCantoOn(t *testing.T) {
	for name, upgrades := range map[string]opera.Upgrades{
		"brio":  opera.GetBrioUpgrades(),
		"canto": opera.GetCantoUpgrades(),
	} {
		t.Run(name, func(t *testing.T) {
			session := getIntegrationTestNetSession(t, upgrades)
			enabled := upgrades.Canto

			t.Run("transaction to an account", func(t *testing.T) {
				sender := makeFundedAccount(t, session)
				recipient := NewAccount().Address()
				receipt := runValueTransfer(t, session, sender, &recipient, 1234, nil)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
				requireTransferLogs(t, receipt, transfersIf(enabled,
					transfer{sender.Address(), recipient, 1234},
				))
			})

			t.Run("transaction without value", func(t *testing.T) {
				sender := makeFundedAccount(t, session)
				recipient := NewAccount().Address()
				receipt := runValueTransfer(t, session, sender, &recipient, 0, nil)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
				requireTransferLogs(t, receipt, nil)
			})

			t.Run("transaction to self", func(t *testing.T) {
				sender := makeFundedAccount(t, session)
				recipient := sender.Address()
				receipt := runValueTransfer(t, session, sender, &recipient, 1234, nil)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
				requireTransferLogs(t, receipt, nil)
			})

			t.Run("contract creation with value", func(t *testing.T) {
				sender := makeFundedAccount(t, session)
				// Init code returning an empty contract.
				initCode := []byte{byte(0x60), 0x00, byte(0x60), 0x00, byte(0xf3)}
				receipt := runValueTransfer(t, session, sender, nil, 1234, initCode)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
				requireTransferLogs(t, receipt, transfersIf(enabled,
					transfer{sender.Address(), receipt.ContractAddress, 1234},
				))
			})

			t.Run("nested calls", func(t *testing.T) {
				sender := session.GetSessionSponsor()
				first, firstAddress := deployTransitiveCall(t, session)
				_, secondAddress := deployTransitiveCall(t, session)

				receipt, err := session.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
					opts.Value = big.NewInt(1234)
					return first.TransitiveCall(opts, []common.Address{secondAddress})
				})
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
				requireTransferLogs(t, receipt, transfersIf(enabled,
					transfer{sender.Address(), firstAddress, 1234},
					transfer{firstAddress, secondAddress, 1234},
				))
			})

			t.Run("reverted call", func(t *testing.T) {
				first, _ := deployTransitiveCall(t, session)

				// An empty call chain makes the contract revert, undoing the
				// transaction's own value transfer as well.
				receipt, err := session.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
					opts.Value = big.NewInt(1234)
					opts.GasLimit = 100_000
					return first.TransitiveCall(opts, nil)
				})
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusFailed, receipt.Status)
				requireTransferLogs(t, receipt, nil)
			})
		})
	}
}

// TestEip7708_TransferLogsAreIndexedAndFilterable covers the receipt bloom
// filter and the log index, both of which grow by one entry per value transfer
// once EIP-7708 is active.
func TestEip7708_TransferLogsAreIndexedAndFilterable(t *testing.T) {
	session := getIntegrationTestNetSession(t, opera.GetCantoUpgrades())

	sender := makeFundedAccount(t, session)
	recipient := NewAccount().Address()
	receipt := runValueTransfer(t, session, sender, &recipient, 1234, nil)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	require.Len(t, receipt.Logs, 1)

	require.Equal(t, types.CreateBloom(receipt), receipt.Bloom,
		"the receipt bloom must cover the transfer log")
	require.True(t, types.BloomLookup(receipt.Bloom, params.SystemAddress))
	require.True(t, types.BloomLookup(receipt.Bloom, params.EthTransferLogEvent))

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	logs, err := client.FilterLogs(t.Context(), ethereum.FilterQuery{
		FromBlock: receipt.BlockNumber,
		ToBlock:   receipt.BlockNumber,
		Addresses: []common.Address{params.SystemAddress},
		Topics: [][]common.Hash{
			{params.EthTransferLogEvent},
			{common.BytesToHash(sender.Address().Bytes())},
		},
	})
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, receipt.Logs[0].Index, logs[0].Index)
	require.Equal(t, receipt.TxHash, logs[0].TxHash)
	require.Equal(t, recipient, common.BytesToAddress(logs[0].Topics[2][:]))
}

func makeFundedAccount(t *testing.T, session IntegrationTestNetSession) *Account {
	t.Helper()
	return MakeAccountWithBalance(t, session, big.NewInt(1e18))
}

// runValueTransfer sends a transaction moving the given value to the given
// recipient, or creating a contract from the given init code if to is nil.
func runValueTransfer(
	t *testing.T,
	session IntegrationTestNetSession,
	sender *Account,
	to *common.Address,
	value int64,
	data []byte,
) *types.Receipt {
	t.Helper()
	tx := CreateTransaction(t, session, &types.DynamicFeeTx{
		To:    to,
		Value: big.NewInt(value),
		Data:  data,
	}, sender)
	receipt, err := session.Run(tx)
	require.NoError(t, err)
	return receipt
}

func deployTransitiveCall(t *testing.T, session IntegrationTestNetSession) (*transitive_call.TransitiveCall, common.Address) {
	t.Helper()
	contract, receipt, err := DeployContract(session, transitive_call.DeployTransitiveCall)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	return contract, receipt.ContractAddress
}

func transfersIf(enabled bool, transfers ...transfer) []transfer {
	if !enabled {
		return nil
	}
	return transfers
}

// requireTransferLogs checks that the receipt contains exactly the given
// EIP-7708 transfer logs, in order, and no other logs from the system address.
func requireTransferLogs(t *testing.T, receipt *types.Receipt, want []transfer) {
	t.Helper()
	var got []transfer
	for _, log := range receipt.Logs {
		if log.Address != params.SystemAddress {
			continue
		}
		require.Len(t, log.Topics, 3)
		require.Equal(t, params.EthTransferLogEvent, log.Topics[0])
		require.Len(t, log.Data, 32)
		got = append(got, transfer{
			from:  common.BytesToAddress(log.Topics[1][:]),
			to:    common.BytesToAddress(log.Topics[2][:]),
			value: new(big.Int).SetBytes(log.Data).Int64(),
		})
	}
	require.Equal(t, want, got)
}
