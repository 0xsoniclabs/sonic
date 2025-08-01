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

package gossip

// compile SFC with truffle
//go:generate bash -c "cd ../../opera-sfc && git checkout e86e5db95f98965f4489ad962565a3850126023a"
//go:generate bash -c "docker run --name go-opera-sfc-compiler -v $(pwd)/contract/solc:/src/build/contracts -v $(pwd)/../../opera-sfc:/src -w /src node:23.2.0 bash -c 'export NPM_CONFIG_PREFIX=~; npm install --no-save; npm install --no-save truffle@5.11.5' && docker commit go-opera-sfc-compiler go-opera-sfc-compiler-image && docker rm go-opera-sfc-compiler"
//go:generate bash -c "docker run --rm -v $(pwd)/contract/solc:/src/artifacts/contracts/sfc -v $(pwd)/../../opera-sfc:/src -w /src go-opera-sfc-compiler-image bash -c 'export NPM_CONFIG_PREFIX=~; rm -f /src/artifacts/contracts/*; npm run compile; chmod -R a+w /src/artifacts/contracts'"
//go:generate bash -c "cd ./contract/solc && for f in SFC.sol/SFC.json NetworkInitializer.sol/NetworkInitializer.json NodeDriver.sol/NodeDriver.json NodeDriverAuth.sol/NodeDriverAuth.json; do jq -j .bytecode $DOLLAR{f} > $DOLLAR{f%.json}.bin; jq -j .deployedBytecode $DOLLAR{f} > $DOLLAR{f%.json}.bin-runtime; jq -c .abi $DOLLAR{f} > $DOLLAR{f%.json}.abi; done"

// wrap SFC with golang
//go:generate mkdir -p ./contract/sfc100
//go:generate go run github.com/ethereum/go-ethereum/cmd/abigen --bin=./contract/solc/SFC.sol/SFC.bin --abi=./contract/solc/SFC.sol/SFC.abi --pkg=sfc100 --type=Contract --out=contract/sfc100/contract.go
//go:generate bash -c "(echo -ne '\nvar ContractBinRuntime = \"'; cat contract/solc/SFC.sol/SFC.bin-runtime; echo '\"') >> contract/sfc100/contract.go"
// wrap NetworkInitializer with golang
//go:generate mkdir -p ./contract/netinit100
//go:generate go run github.com/ethereum/go-ethereum/cmd/abigen --bin=./contract/solc/NetworkInitializer.sol/NetworkInitializer.bin --abi=./contract/solc/NetworkInitializer.sol/NetworkInitializer.abi --pkg=netinit100 --type=Contract --out=contract/netinit100/contract.go
//go:generate bash -c "(echo -ne '\nvar ContractBinRuntime = \"'; cat contract/solc/NetworkInitializer.sol/NetworkInitializer.bin-runtime; echo '\"') >> contract/netinit100/contract.go"
// wrap NodeDriver with golang
//go:generate mkdir -p ./contract/driver100
//go:generate go run github.com/ethereum/go-ethereum/cmd/abigen --bin=./contract/solc/NodeDriver.sol/NodeDriver.bin --abi=./contract/solc/NodeDriver.sol/NodeDriver.abi --pkg=driver100 --type=Contract --out=contract/driver100/contract.go
//go:generate bash -c "(echo -ne '\nvar ContractBinRuntime = \"'; cat contract/solc/NodeDriver.sol/NodeDriver.bin-runtime; echo '\"') >> contract/driver100/contract.go"
// wrap NodeDriverAuth with golang
//go:generate mkdir -p ./contract/driverauth100
//go:generate go run github.com/ethereum/go-ethereum/cmd/abigen --bin=./contract/solc/NodeDriverAuth.sol/NodeDriverAuth.bin --abi=./contract/solc/NodeDriverAuth.sol/NodeDriverAuth.abi --pkg=driverauth100 --type=Contract --out=contract/driverauth100/contract.go
//go:generate bash -c "(echo -ne '\nvar ContractBinRuntime = \"'; cat contract/solc/NodeDriverAuth.sol/NodeDriverAuth.bin-runtime; echo '\"') >> contract/driverauth100/contract.go"

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/gossip/contract/driver100"
	"github.com/0xsoniclabs/sonic/gossip/contract/driverauth100"
	"github.com/0xsoniclabs/sonic/gossip/contract/netinit100"
	"github.com/0xsoniclabs/sonic/gossip/contract/sfc100"
	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/0xsoniclabs/sonic/opera/contracts/evmwriter"
	"github.com/0xsoniclabs/sonic/opera/contracts/netinit"
	"github.com/0xsoniclabs/sonic/opera/contracts/sfc"
	"github.com/0xsoniclabs/sonic/utils"
)

func TestSFC(t *testing.T) {
	logger.SetTestMode(t)
	logger.SetLevel("debug")

	env := newTestEnv(2, 3, t)
	t.Cleanup(func() {
		err := env.Close()
		require.NoError(t, err)
	})

	var (
		sfc10 *sfc100.Contract
		err   error
	)

	authDriver10, err := driverauth100.NewContract(driverauth.ContractAddress, env)
	require.NoError(t, err)
	rootDriver10, err := driver100.NewContract(driver.ContractAddress, env)
	require.NoError(t, err)

	admin := idx.ValidatorID(1)
	adminAddr := env.Address(admin)

	_ = true &&
		t.Run("Genesis SFC", func(t *testing.T) {
			require := require.New(t)

			exp := sfc.GetContractBin()
			got, err := env.CodeAt(context.TODO(), sfc.ContractAddress, nil)
			require.NoError(err)
			require.Equal(exp, got, "genesis SFC contract")
			require.Equal(exp, hexutil.MustDecode(sfc100.ContractBinRuntime), "genesis SFC contract version")
		}) &&
		t.Run("Genesis Driver", func(t *testing.T) {
			require := require.New(t)

			exp := driver.GetContractBin()
			got, err := env.CodeAt(context.TODO(), driver.ContractAddress, nil)
			require.NoError(err)
			require.Equal(exp, got, "genesis Driver contract")
			require.Equal(exp, hexutil.MustDecode(driver100.ContractBinRuntime), "genesis Driver contract version")
		}) &&
		t.Run("Genesis DriverAuth", func(t *testing.T) {
			require := require.New(t)

			exp := driverauth.GetContractBin()
			got, err := env.CodeAt(context.TODO(), driverauth.ContractAddress, nil)
			require.NoError(err)
			require.Equal(exp, got, "genesis DriverAuth contract")
			require.Equal(exp, hexutil.MustDecode(driverauth100.ContractBinRuntime), "genesis DriverAuth contract version")
		}) &&
		t.Run("Network initializer", func(t *testing.T) {
			require := require.New(t)

			exp := netinit.GetContractBin()
			got, err := env.CodeAt(context.TODO(), netinit.ContractAddress, nil)
			require.NoError(err)
			require.NotEmpty(exp, "genesis NetworkInitializer contract")
			require.Equal(exp, got, "genesis NetworkInitializer contract") // not destructed after EIP-6780
			require.Equal(exp, hexutil.MustDecode(netinit100.ContractBinRuntime), "genesis NetworkInitializer contract version")
		}) &&
		t.Run("Builtin EvmWriter", func(t *testing.T) {
			require := require.New(t)

			exp := []byte{0}
			got, err := env.CodeAt(context.TODO(), evmwriter.ContractAddress, nil)
			require.NoError(err)
			require.Equal(exp, got, "builtin EvmWriter contract")
		}) &&
		t.Run("Some transfers I", func(t *testing.T) {
			circleTransfers(t, env, 1)
		}) &&
		t.Run("Check owners", func(t *testing.T) {
			require := require.New(t)

			opts := env.ReadOnly()

			actualOwner, err := authDriver10.Owner(opts)
			require.NoError(err)
			require.Equal(adminAddr, actualOwner)
		}) &&
		t.Run("SFC upgrade", func(t *testing.T) {
			require := require.New(t)

			// create new
			rr, err := env.ApplyTxs(nextEpoch,
				env.Contract(admin, utils.ToFtm(0), sfc100.ContractMetaData.Bin),
			)
			require.NoError(err)
			require.Equal(1, rr.Len())
			require.Equal(types.ReceiptStatusSuccessful, rr[0].Status)
			newImpl := rr[0].ContractAddress
			require.NotEqual(sfc.ContractAddress, newImpl)
			newSfcContractBinRuntime, err := env.CodeAt(context.TODO(), newImpl, nil)
			require.NoError(err)
			require.Equal(hexutil.MustDecode(sfc100.ContractBinRuntime), newSfcContractBinRuntime)

			tx, err := authDriver10.CopyCode(env.Pay(admin), sfc.ContractAddress, newImpl)
			require.NoError(err)
			rr, err = env.ApplyTxs(sameEpoch, tx)
			require.NoError(err)
			require.Equal(1, rr.Len())
			require.Equal(types.ReceiptStatusSuccessful, rr[0].Status)
			got, err := env.CodeAt(context.TODO(), sfc.ContractAddress, nil)
			require.NoError(err)
			require.Equal(newSfcContractBinRuntime, got, "new SFC contract")

			sfc10, err = sfc100.NewContract(sfc.ContractAddress, env)
			require.NoError(err)
			sfcEpoch, err := sfc10.CurrentEpoch(env.ReadOnly())
			require.NoError(err)
			require.Equal(0, sfcEpoch.Cmp(big.NewInt(3)), "current SFC epoch %s", sfcEpoch.String())
		})

	t.Run("Direct driver", func(t *testing.T) {
		require := require.New(t)

		// create new
		anyContractBin := driver100.ContractBin
		rr, err := env.ApplyTxs(nextEpoch,
			env.Contract(admin, utils.ToFtm(0), anyContractBin),
		)
		require.NoError(err)
		require.Equal(1, rr.Len())
		require.Equal(types.ReceiptStatusSuccessful, rr[0].Status)
		newImpl := rr[0].ContractAddress

		tx, err := rootDriver10.CopyCode(env.Pay(admin), sfc.ContractAddress, newImpl)
		require.NoError(err)
		rr, err = env.ApplyTxs(sameEpoch, tx)
		require.NoError(err)
		require.Equal(1, rr.Len())
		require.NotEqual(types.ReceiptStatusSuccessful, rr[0].Status)
	})

}

func circleTransfers(t *testing.T, env *testEnv, count uint64) {
	require := require.New(t)
	validatorsNum := env.store.GetValidators().Len()

	// save start balances
	balances := make([]*uint256.Int, validatorsNum)
	for i := range balances {
		balances[i] = env.State().GetBalance(env.Address(idx.ValidatorID(i + 1)))
	}

	for i := uint64(0); i < count; i++ {
		// transfers
		txs := make([]*types.Transaction, validatorsNum)
		for i := idx.Validator(0); i < validatorsNum; i++ {
			from := (i) % validatorsNum
			to := (i + 1) % validatorsNum
			txs[i] = env.Transfer(idx.ValidatorID(from+1), idx.ValidatorID(to+1), utils.ToFtm(100))
		}

		rr, err := env.ApplyTxs(sameEpoch, txs...)
		require.NoError(err)
		for i, r := range rr {
			fee := uint256.NewInt(0).Mul(new(uint256.Int).SetUint64(r.GasUsed), utils.BigIntToUint256(txs[i].GasPrice()))
			balances[i] = uint256.NewInt(0).Sub(balances[i], fee)
		}
	}

	// check balances
	for i := range balances {
		require.Equal(
			balances[i],
			env.State().GetBalance(env.Address(idx.ValidatorID(i+1))),
			fmt.Sprintf("account%d", i),
		)
	}
}
