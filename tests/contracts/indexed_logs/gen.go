package indexed_logs

//go:generate solc --bin indexed_logs.sol --abi indexed_logs.sol -o build --overwrite
//go:generate abigen --bin=build/IndexedLogs.bin --abi=build/IndexedLogs.abi --pkg=indexed_logs --out=indexed_logs.go
