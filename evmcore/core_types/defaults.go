package coretypes

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// GetCoinbase returns the coinbase to be used by blocks on Sonic networks.
func GetCoinbase() common.Address {
	return common.Address{}
}

// GetBlobBaseFee returns the blob base fee to be used by blocks on Sonic networks.
func GetBlobBaseFee() uint256.Int {
	return uint256.Int{}
}
