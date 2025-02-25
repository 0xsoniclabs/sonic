package evmcore

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestIntrinsicGas_IsCompatibleWithGethImplementation(t *testing.T) {

	// The intention of this test is to detect deviations between the geth
	// implementation of intrinsic gas calculation and the sonic implementation.
	//
	// This test alerts us if the intrinsic gas calculation is changed.

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

				for _, isEIP3860 := range []bool{true, false} {
					for _, contractCreation := range []bool{true, false} {
						tests = append(tests, testData{
							data:               makeData(dataSize, dataSize/zeroesRatio),
							accessList:         accessList,
							isContractCreation: contractCreation,
							isEIP3860:          isEIP3860,
						})
					}
				}
			}
		}
	}

	for _, test := range tests {

		sonicValue, sonicError := IntrinsicGas(test.data, test.accessList, nil, test.isContractCreation, test.isEIP3860)
		gethValue, gethError := core.IntrinsicGas(test.data, test.accessList, nil, test.isContractCreation, true, true, test.isEIP3860)

		require.Equal(t, sonicError, gethError, "both implementation shall return the same error with the same input")
		if sonicError != nil {
			require.Equal(t, sonicValue, gethValue, "both implementation shall return the same value with the same input")
		}
	}
}

type testData struct {
	data               []byte
	accessList         types.AccessList
	isContractCreation bool
	isEIP3860          bool
}

// makeData creates a byte slice of the given size with the given number of zeroes at the end.
// This is relevant for the intrinsic gas calculation, as zero bytes are priced differently.
func makeData(size, zeroesCount int) []byte {
	zeroes := bytes.Repeat([]byte{0}, zeroesCount)
	nonZeroes := bytes.Repeat([]byte{1}, size)
	copy(nonZeroes, zeroes)
	return nonZeroes
}
