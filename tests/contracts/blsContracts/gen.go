package blsContracts

//go:generate solc --combined-json abi,bin --abi --bin --overwrite -o build BLS.sol
//go:generate abigen --combined-json build/combined.json --pkg=blsContracts --out=blsContracts.go
