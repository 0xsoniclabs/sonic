package block_override

//go:generate solc --bin block_override.sol --abi block_override.sol -o build --overwrite
//go:generate abigen --bin=build/BlockOverride.bin --abi=build/BlockOverride.abi --pkg=block_override --out=block_override.go
