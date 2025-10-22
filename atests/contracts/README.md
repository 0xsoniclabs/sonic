# Test Contracts

This directory contains a set of smart contracts used by integration tests.

Contracts are grouped in applications, which may each consist of multiple 
contracts. Each application is retained in its own directory.

Within each application directory, the following files should be present:
 - *.sol  ... the Solidity code for the application
 - gen.go ... a go file with a generator rule producing Go bindings for the app
 - *.go   ... the generated Go binding files (checked in in the code repo)

For an example application, see the `counter` directory.

## Code Generation Tools

For compiling Solidity code and generating the Go bindings the following tools
need to be installed on your system:
- the `solc` compiler; on Ubuntu you an install it using `sudo snap install solc --edge`
- the `abigen` tool; this can be installed using `go install github.com/ethereum/go-ethereum/cmd/abigen@latest`

Background information on those tools can be found [here](https://goethereumbook.org/en/smart-contract-compile/).
