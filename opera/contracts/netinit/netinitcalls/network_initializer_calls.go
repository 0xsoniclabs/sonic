package netinitcall

import (
	_ "embed"
	"math/big"
	"strings"

	"github.com/0xsoniclabs/consensus/inter/idx"
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
