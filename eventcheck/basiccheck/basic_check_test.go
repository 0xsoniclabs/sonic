package basiccheck

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/evmcore"
)

func TestIntrinsicGas_IsCompatibleWithLegacyFunction(t *testing.T) {

	type testData struct {
		data               []byte
		accessList         types.AccessList
		isContractCreation bool
	}

	tests := []testData{}
	for _, dataSize := range []int{
		0, 1, 16, 1024, 2048, 4096, 8192, 16384,
	} {
		for _, zeroesRatio := range []int{1, 2, 4} {
			for _, accessList := range []types.AccessList{
				nil,
				{},
				{
					{Address: common.Address{}, StorageKeys: []common.Hash{}},
				},
				{
					{Address: common.Address{0x42}, StorageKeys: []common.Hash{{1}}},
					{Address: common.Address{0x42}, StorageKeys: []common.Hash{{1}}},
				},
			} {

				for _, contractCreation := range []bool{true, false} {
					tests = append(tests, testData{
						data:               makeData(dataSize, dataSize/zeroesRatio),
						accessList:         accessList,
						isContractCreation: contractCreation,
					})
				}
			}
		}
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("test%d", i), func(t *testing.T) {

			legacyValue, legacyError := legacyIntrinsicGas(test.data, test.accessList, test.isContractCreation)
			newValue, newError := core.IntrinsicGas(test.data, test.accessList, nil, test.isContractCreation, true, true, false)

			require.Equal(t, legacyError, newError, "both implementation shall return the same error with the same input")

			if legacyError != nil {
				require.Equal(t, legacyValue, newValue, "both implementation shall return the same value with the same input")
			}
		})
	}
}

// makeData creates a byte slice of the given size with the given number of zeroes at the end.
// This is relevant for the intrinsic gas calculation, as zero bytes are priced differently.
func makeData(size, zeroesCount int) []byte {
	zeroes := bytes.Repeat([]byte{0}, zeroesCount)
	nonZeroes := bytes.Repeat([]byte{1}, size)
	copy(nonZeroes, zeroes)
	return nonZeroes
}

// legacyIntrinsicGas is the function used for basiccheck.validateTx before using
// core.IntrinsicGas from Geth's core package.
// Do not change this function, the code here is only for comparison with previous implementations.
func legacyIntrinsicGas(data []byte, accessList types.AccessList, isContractCreation bool) (uint64, error) {
	// Set the starting gas for the raw transaction
	var gas uint64
	if isContractCreation {
		gas = params.TxGasContractCreation
	} else {
		gas = params.TxGas
	}
	// Bump the required gas by the amount of transactional data
	if len(data) > 0 {

		// Zero and non-zero bytes are priced differently
		var nz uint64
		for _, byt := range data {
			if byt != 0 {
				nz++
			}
		}
		// Make sure we don't exceed uint64 for all data combinations
		if (math.MaxUint64-gas)/params.TxDataNonZeroGasEIP2028 < nz {
			return 0, vm.ErrOutOfGas
		}
		gas += nz * params.TxDataNonZeroGasEIP2028

		z := uint64(len(data)) - nz
		if (math.MaxUint64-gas)/params.TxDataZeroGas < z {
			return 0, evmcore.ErrGasUintOverflow
		}
		gas += z * params.TxDataZeroGas
	}
	if accessList != nil {
		gas += uint64(len(accessList)) * params.TxAccessListAddressGas
		gas += uint64(accessList.StorageKeys()) * params.TxAccessListStorageKeyGas
	}
	return gas, nil
}
