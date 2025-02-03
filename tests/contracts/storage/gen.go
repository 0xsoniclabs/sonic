package storage

//go:generate solc --bin storage.sol --abi storage.sol -o build --overwrite
//go:generate abigen --bin=build/Storage.bin --abi=build/Storage.abi --pkg=storage --out=Storage.go
