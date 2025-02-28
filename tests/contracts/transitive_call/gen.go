package transitive_call

//go:generate solc --bin transitive_call.sol --abi transitive_call.sol -o build --overwrite
//go:generate abigen --bin=build/TransitiveCall.bin --abi=build/TransitiveCall.abi --pkg=transitive_call --out=transitive_call.go
