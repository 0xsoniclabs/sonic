package block_parameters

//go:generate solc --bin block_parameters.sol --abi block_parameters.sol -o build --overwrite
//go:generate abigen --bin=build/BlockParameters.bin --abi=build/BlockParameters.abi --pkg=block_parameters --out=block_parameters.go
