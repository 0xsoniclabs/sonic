package evmcore

import (
	"bytes"
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

var (
	accessListInputs = []types.AccessList{
		nil,
		{},
		{
			{Address: common.Address{}, StorageKeys: nil},
		},
		{
			{Address: common.Address{}, StorageKeys: []common.Hash{}},
		},
		{
			{Address: common.Address{0x42}, StorageKeys: []common.Hash{{1}}},
			{Address: common.Address{0x42}, StorageKeys: []common.Hash{{1}, {2}}},
		},
	}

	authorizationListInputs = [][]types.SetCodeAuthorization{
		nil,
		{},
		{{Address: common.Address{0x42}}},
		{{Address: common.Address{0x42}}, {Address: common.Address{0x43}}},
	}
)

// This test is a fuzz test for the intrinsic gas calculation.
// It tests the sonic implementation against the geth implementation for a variety of inputs.
// This test alerts us if the intrinsic gas calculation is changed.
// It additionally checks the legacy intrinsic gas calculation for compatibility with the
// changes introduced by the Prague update.
func FuzzTestIntrinsicGas(f *testing.F) {

	for _, dataSize := range []uint{
		0, 1, 16, 1024, 2048, 4096, 8192, 16384,
	} {
		for _, zeroesRatio := range []uint{1, 2, 4} {
			for accessListIndex := range len(accessListInputs) {
				for authorizationListIndex := range len(authorizationListInputs) {
					for _, isEIP3860 := range []bool{true, false} {
						for _, contractCreation := range []bool{true, false} {
							f.Add(dataSize, zeroesRatio, accessListIndex, authorizationListIndex, isEIP3860, contractCreation)
						}
					}
				}
			}
		}
	}

	// for _, test := range tests {
	f.Fuzz(func(t *testing.T, dataSize, zeroesRation uint, accessListIndex, authorizationListIndex int, isEIP3860, isContractCreation bool) {
		if zeroesRation == 0 {
			t.Skip()
		}
		data := makeData(dataSize, dataSize/zeroesRation)

		if accessListIndex < 0 || accessListIndex >= len(accessListInputs) {
			t.Skip()
		}
		accessList := accessListInputs[accessListIndex]

		if authorizationListIndex < 0 || authorizationListIndex >= len(authorizationListInputs) {
			t.Skip()
		}
		authList := authorizationListInputs[authorizationListIndex]

		sonicValue, sonicError := IntrinsicGas(data, accessList, authList, isContractCreation, isEIP3860)
		gethValue, gethError := core.IntrinsicGas(data, accessList, authList, isContractCreation, true, true, isEIP3860)

		require.Equal(t, sonicError, gethError, "both implementation shall return the same error with the same input")
		if sonicError != nil {
			require.Equal(t, sonicValue, gethValue, "both implementation shall return the same value with the same input")
		}

		if isEIP3860 || len(authList) > 0 {
			// we only test legacy compatibility when EIP-3860 is disabled and no authorization list is provided
			// we do not skip, as the test is still valid
			return
		}

		legacyValue, legacyError := LegacyIntrinsicGas(data, accessList, isContractCreation)
		require.Equal(t, legacyError, gethError, "both implementation shall return the same error with the same input")
		if legacyError != nil {
			require.Equal(t, legacyValue, gethValue, "both implementation shall return the same value with the same input")
		}
	})
}

// makeData creates a byte slice of the given size with the given number of zeroes at the end.
// This is relevant for the intrinsic gas calculation, as zero bytes are priced differently.
func makeData(size, zeroesCount uint) []byte {
	zeroes := bytes.Repeat([]byte{0}, int(zeroesCount))
	nonZeroes := bytes.Repeat([]byte{1}, int(size))
	copy(nonZeroes, zeroes)
	return nonZeroes
}

// LegacyIntrinsicGas is the Intrinsic gas computation used by the sonic before the Prague update.
// The behavior of the new implementation, without the EIP-3860 flag, and empty authorization list
// must remain the same as the legacy implementation.
// DO NOT MODIFY THIS FUNCTION.
func LegacyIntrinsicGas(data []byte, accessList types.AccessList, isContractCreation bool) (uint64, error) {
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
			return 0, ErrGasUintOverflow
		}
		gas += z * params.TxDataZeroGas
	}
	if accessList != nil {
		gas += uint64(len(accessList)) * params.TxAccessListAddressGas
		gas += uint64(accessList.StorageKeys()) * params.TxAccessListStorageKeyGas
	}
	return gas, nil
}
