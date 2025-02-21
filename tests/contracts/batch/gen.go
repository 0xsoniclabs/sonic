package batch

//go:generate solc --bin batch.sol --abi batch.sol -o build --overwrite
//go:generate abigen --bin=build/BatchCallDelegation.bin --abi=build/BatchCallDelegation.abi --pkg=batch --out=batch.go
