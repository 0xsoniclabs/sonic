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

package contracts

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/tests"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

const (
	// deployerBalance is what the deployer is funded with in the genesis block, enough for every
	// deployment several times over.
	deployerBalance = 1e18
)

// deploymentGasPrice is comfortably above the base fee of an idle network, which is where a network
// under test sits when its contracts go up.
var deploymentGasPrice = big.NewInt(1e11)

// Deployer owns every contract this package puts on chain. It is nobody the harness observes, so
// what it spends on deployments is accounted for by nothing.
var Deployer = &tests.Account{PrivateKey: mustDeriveDeployerKey()}

func mustDeriveDeployerKey() *ecdsa.PrivateKey {
	key, err := crypto.ToECDSA(crypto.Keccak256([]byte("txprop-contract-deployer")))
	if err != nil {
		panic(fmt.Sprintf("failed to derive the deployer key: %v", err))
	}
	return key
}

// GenesisAccount funds the deployer in the genesis block, to be passed to the test network alongside
// whatever else it is given.
func GenesisAccount() makefakegenesis.Account {
	return makefakegenesis.Account{
		Name:    "txprop-contract-deployer",
		Address: Deployer.Address(),
		Balance: uint256.NewInt(deployerBalance),
	}
}

// Deploy puts every registered contract on the network and reports where each one landed. The
// transactions go out together, so this costs a handful of blocks rather than one per contract.
//
// A contract whose deployment fails is left out rather than failing the run: these applications are
// written for the whole test suite, and one built for a fork's features does not necessarily deploy
// on an older fork.
func Deploy(ctx context.Context, session tests.IntegrationTestNetSession) ([]Deployed, error) {
	client, err := session.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the network: %w", err)
	}
	defer client.Close()

	nonce, err := client.NonceAt(ctx, Deployer.Address(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read the deployer's nonce: %w", err)
	}

	chainId := session.GetChainId()
	signer := types.LatestSignerForChainID(chainId)

	txs := make([]*types.Transaction, 0, len(Registry))
	deploying := make([]int, 0, len(Registry))
	for i, contract := range Registry {
		// What the deployment costs is asked rather than assumed, because a gas limit generous
		// enough for the largest of these would let only one or two of them into a block and this
		// would take longer than the run it prepares for. A contract the node will not even price is
		// one this fork cannot run, and is left out.
		gas, err := client.EstimateGas(ctx, ethereum.CallMsg{
			From: Deployer.Address(),
			Data: contract.Code,
		})
		if err != nil {
			continue
		}

		tx, err := types.SignNewTx(Deployer.PrivateKey, signer, &types.DynamicFeeTx{
			ChainID:   chainId,
			Nonce:     nonce + uint64(len(txs)),
			GasTipCap: big.NewInt(0),
			GasFeeCap: deploymentGasPrice,
			Gas:       gas + gas/4, // room for what the estimate cannot see
			Data:      contract.Code,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to sign the deployment of %s: %w", contract.Name, err)
		}
		txs = append(txs, tx)
		deploying = append(deploying, i)
	}

	receipts, err := session.RunAll(txs)
	if err != nil {
		return nil, fmt.Errorf("failed to run the deployments: %w", err)
	}

	deployed := make([]Deployed, 0, len(receipts))
	for i, receipt := range receipts {
		if receipt.Status != types.ReceiptStatusSuccessful {
			continue
		}
		deployed = append(deployed, Deployed{Contract: deploying[i], Address: receipt.ContractAddress})
	}
	return deployed, nil
}
