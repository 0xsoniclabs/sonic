package read_history_storage

//go:generate solc --bin read_history_storage.sol --abi read_history_storage.sol -o build --overwrite
//go:generate abigen --bin=build/ReadHistoryStorage.bin --abi=build/ReadHistoryStorage.abi --pkg=read_history_storage --out=read_history_storage.go
