# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
This changelog documents updates relevant to users and downstream integrations. The scope of each version includes changes to the following interfaces:
- Interactions with other nodes in the Sonic network.
- Command-line interface and configuration files.
- Execution support for Ethereum transactions and Ethereum-compatible contracts.
- RPC interfaces, including JSON-RPC, WebSockets, and Unix sockets.

## [Unreleased]

Unreleased features are staged for future releases but have not yet undergone 
thorough testing or integration. These changes are experimental and may 
cause unexpected behavior or malfunctions in the client.

For optimal compatibility and stability, it is recommended to use the most recent released version.

### Added
- Introduce support for the Brio hard-fork upgrade.
- Implement CLZ VM instruction [EIP-7939](https://eips.ethereum.org/EIPS/eip-7939) when Brio upgrade is enabled.
- Add precompiled contract for secp256r1 Curve Support [EIP-7951](https://eips.ethereum.org/EIPS/eip-7951)  when Brio upgrade is enabled.
- Introduce eth_Config RPC method tailored for the Sonic network.

### Fixed

### Changed
- Bump minimum required Go version to 1.25.0.
- Increase gas cost for the ModExp precompiled contract in accordance with [EIP-7883](https://eips.ethereum.org/EIPS/eip-7883) when the Brio upgrade is enabled.
- Restrict maximum input length for ModExp precompiled contract [EIP-7823](https://eips.ethereum.org/EIPS/eip-7823) when Brio upgrade is enabled.
- Introduce protocol-level upper bound gas usage per transaction (à la [EIP-7825](https://eips.ethereum.org/EIPS/eip-7825)) when Brio upgrade is enabled.
- Introduce protocol level maximum RLP encoded block size of 10 MiB [EIP-7934](https://eips.ethereum.org/EIPS/eip-7934) when Brio upgrade us enabled.

### Removed

## [2.1.4] - 25 Nov 2025

### Fixed
- Fix issue affecting log timestamps serialization (RPC)
- Fix consistency issue of log transaction indices when querying logs using block hash (RPC)
### Changed
- Transaction Pool rejects transactions from non-EOA accounts.

## [2.1.3] - 5 Nov 2025

- Reduce overhead produced by handling of sponsored transactions by the transaction pool under heavy load.
- Reuse transactions sender to reduce transactions signature overheads.
- Implement mitigation measures regarding P2P events broadcasting lag which stalls block processing.

## [2.1.2] - 2 Oct 2025

- Added Gas Subsidy support
- Fixed backward compatibility of gas limit updates

## [2.1.1] - 23 Sep 2025

-  RPC api: Fix empty logs bloom
-  Add CreateTransaction helper function
-  Set chain ID for internal transaction in RPC
-  Remove deprecated default command arguments
-  Improve unit tests and integration tests

## [2.1.0] - 20 Aug 2025

- Fix error handling in debug_traceBlock

## [2.0.6] - 26 May 2025

- Synchronize access to cached hash in Block - fixes incorrect block hash returned by RPC rarely
- Check NoArchiveError when sending block notifications
- Update Tosca dependency

## [2.0.5] - 28 Apr 2025

- Check archive block height before sending subscribers notification
- Use GETH for eth_call simulation with large code in state override

## [2.0.4] - 26 Mar 2025

- Replay transaction on empty blocks
- Add block overrides for RPC calls
- Change the NoBaseFee VM configuration parameter for replaying transactions

## [2.0.3] - 20 Feb 2025

- Security fix of the CVE-2025-24883 (public key validity check on P2P).
- Storage override for eth_call RPC interface.
- Various fixes and improvements.

## [2.0.2] - 10 Feb 2025

- Implements eth_getAccount RPC method (Fantom-foundation/Sonic#370)
- Handle "finalized" and "safe" block number tags (Fantom-foundation/Sonic#388)
- Events fetching improved (Fantom-foundation/Sonic#372)
- Fixes gas capping in RPC calls (Fantom-foundation/Sonic#391)


## [2.0.1] - 10 Feb 2025
-  Adds the new Sonic VM for faster contract code processing.
-  Improves upon the Sonic DB performance with additional optimizations, especially for the new features.
-  Adds support for Cancun/Deneb including transient storage and new VM opcodes.
-  The Prevrandao is now fully supported in the VM and is ready to be used in contracts. Please note Prevrandao on Sonic can not be influenced by a validator not proposing a block.
-  Offers limited support for Type 3 transactions. The BLOB storage has not been implemented and non-empty BLOB transactions are rejected if submitted.
-  Includes an updated consensus control for stable TTF with improved security of the blocks building.
-  Built-in topology heuristics optimizes the network responsiveness.
- We also included number of smaller bug fixes and improvements across different parts of the system.


## [2.0.0] - 10 Feb 2025

Initial release of the Sonic client for the Sonic network.