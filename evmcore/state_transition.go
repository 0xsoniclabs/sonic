package evmcore

import (
	"bytes"
	"math"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// IntrinsicGas computes the minimum amount of gas required to process a transaction
// before executing any smart contract logic.
// This cost ensures th data transmission and transaction validation, are covered.
//
// This implementation is based in:
// go-ethereum@v0.0.0-20241022121122-7063a6b506bd/core/state_transition.go
//
// The minimum revision considered is Istanbul, any previous revision cost differences are ignored.
func IntrinsicGas(
	data []byte,
	accessList types.AccessList,
	authList []types.SetCodeAuthorization,
	isContractCreation bool,
	isEIP3860 bool,
) (uint64, error) {
	// Set the starting gas for the raw transaction
	gas := params.TxGas
	if isContractCreation {
		gas = params.TxGasContractCreation
	}
	dataLen := uint64(len(data))
	// Bump the required gas by the amount of transactional data
	if dataLen > 0 {
		// Zero and non-zero bytes are priced differently
		z := uint64(bytes.Count(data, []byte{0}))
		nz := dataLen - z

		// Make sure we don't exceed uint64 for all data combinations
		nonZeroGas := params.TxDataNonZeroGasEIP2028
		if (math.MaxUint64-gas)/nonZeroGas < nz {
			return 0, ErrGasUintOverflow
		}
		gas += nz * nonZeroGas

		if (math.MaxUint64-gas)/params.TxDataZeroGas < z {
			return 0, ErrGasUintOverflow
		}
		gas += z * params.TxDataZeroGas

		if isContractCreation && isEIP3860 {
			lenWords := toWordSize(dataLen)
			if (math.MaxUint64-gas)/params.InitCodeWordGas < lenWords {
				return 0, ErrGasUintOverflow
			}
			gas += lenWords * params.InitCodeWordGas
		}
	}
	if accessList != nil {
		gas += uint64(len(accessList)) * params.TxAccessListAddressGas
		gas += uint64(accessList.StorageKeys()) * params.TxAccessListStorageKeyGas
	}
	if authList != nil {
		gas += uint64(len(authList)) * params.CallNewAccountGas
	}
	return gas, nil
}

// toWordSize returns the ceiled word size required for init code payment calculation.
func toWordSize(size uint64) uint64 {
	if size > math.MaxUint64-31 {
		return math.MaxUint64/32 + 1
	}

	return (size + 31) / 32
}
