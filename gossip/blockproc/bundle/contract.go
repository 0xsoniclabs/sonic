package bundle

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
)

//go:generate solc --optimize --optimize-runs 200 --bin --bin-runtime bundles_contract.sol --abi bundles_contract.sol -o build --overwrite
//go:generate abigen --bin=build/Bundles.bin --abi=build/Bundles.abi --pkg=bundle --out=bundles_abigen.go

var BundleContractCode []byte = hexutil.MustDecode(BundleMetaData.Bin)
