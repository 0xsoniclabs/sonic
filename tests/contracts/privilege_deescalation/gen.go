package privilege_deescalation

//go:generate solc --bin privilege_deescalation.sol --abi privilege_deescalation.sol -o build --overwrite
//go:generate abigen --bin=build/PrivilegeDeescalation.bin --abi=build/PrivilegeDeescalation.abi --pkg=privilege_deescalation --out=privilege_deescalation.go
