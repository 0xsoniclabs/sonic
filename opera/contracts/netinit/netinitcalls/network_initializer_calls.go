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

package netinitcall

import (
	_ "embed"
	"math/big"
	"strings"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/sonic/utils"
)

//go:embed NetworkInitializerAbi.json
var ContractABI string

var (
	sAbi, _ = abi.JSON(strings.NewReader(ContractABI))
)

// Methods

func InitializeAll(sealedEpoch idx.Epoch, totalSupply *big.Int, sfcAddr common.Address, driverAuthAddr common.Address, driverAddr common.Address, evmWriterAddr common.Address, owner common.Address) []byte {
	data, _ := sAbi.Pack("initializeAll", utils.U64toBig(uint64(sealedEpoch)), totalSupply, sfcAddr, driverAuthAddr, driverAddr, evmWriterAddr, owner)
	return data
}
