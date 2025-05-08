package sponsoring

//go:generate solc --bin sponsoring.sol --abi sponsoring.sol -o build --overwrite
//go:generate abigen --bin=build/SponsoringDelegate.bin --abi=build/SponsoringDelegate.abi --pkg=sponsoring --out=sponsoring.go
