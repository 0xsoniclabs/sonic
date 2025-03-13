// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package flags

import (
	"fmt"
	"strings"

	"github.com/0xsoniclabs/consensus/utils/cachescale"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip"
	pcsclite "github.com/gballet/go-libpcsclite"
	"gopkg.in/urfave/cli.v1"

	"github.com/ethereum/go-ethereum/node"
)

// These are all the command line flags we support.
// If you add to this list, please remember to include the
// flag in the appropriate command definition.
//
// The flags are defined here so their names and help texts
// are the same for all commands.

var (
	// General settings
	DataDirFlag = cli.StringFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases and keystore",
	}
	MinFreeDiskSpaceFlag = cli.StringFlag{
		Name:  "datadir.minfreedisk",
		Usage: "Minimum free disk space in MB, once reached triggers auto shut down (default = 8192 MB, 0 = disabled)",
	}
	KeyStoreDirFlag = cli.StringFlag{
		Name:  "keystore",
		Usage: "Directory for the keystore (default = inside the datadir)",
	}
	USBFlag = cli.BoolFlag{
		Name:  "usb",
		Usage: "Enable monitoring and management of USB hardware wallets",
	}
	SmartCardDaemonPathFlag = cli.StringFlag{
		Name:  "pcscdpath",
		Usage: "Path to the smartcard daemon (pcscd) socket file",
		Value: pcsclite.PCSCDSockName,
	}
	IdentityFlag = cli.StringFlag{
		Name:  "identity",
		Usage: "Custom node name",
	}
	LightKDFFlag = cli.BoolFlag{
		Name:  "lightkdf",
		Usage: "Reduce key-derivation RAM & CPU usage at some expense of KDF strength",
	}
	// Transaction pool settings
	TxPoolLocalsFlag = cli.StringFlag{
		Name:  "txpool.locals",
		Usage: "Comma separated accounts to treat as locals (no flush, priority inclusion)",
	}
	TxPoolNoLocalsFlag = cli.BoolFlag{
		Name:  "txpool.nolocals",
		Usage: "Disables price exemptions for locally submitted transactions",
	}
	TxPoolJournalFlag = cli.StringFlag{
		Name:  "txpool.journal",
		Usage: "Disk journal for local transaction to survive node restarts",
		Value: evmcore.DefaultTxPoolConfig.Journal,
	}
	TxPoolRejournalFlag = cli.DurationFlag{
		Name:  "txpool.rejournal",
		Usage: "Time interval to regenerate the local transaction journal",
		Value: evmcore.DefaultTxPoolConfig.Rejournal,
	}
	TxPoolPriceLimitFlag = cli.Uint64Flag{
		Name:  "txpool.pricelimit",
		Usage: "Minimum gas price limit to enforce for acceptance into the pool",
		Value: evmcore.DefaultTxPoolConfig.PriceLimit,
	}
	TxPoolPriceBumpFlag = cli.Uint64Flag{
		Name:  "txpool.pricebump",
		Usage: "Price bump percentage to replace an already existing transaction",
		Value: evmcore.DefaultTxPoolConfig.PriceBump,
	}
	TxPoolAccountSlotsFlag = cli.Uint64Flag{
		Name:  "txpool.accountslots",
		Usage: "Minimum number of executable transaction slots guaranteed per account",
		Value: evmcore.DefaultTxPoolConfig.AccountSlots,
	}
	TxPoolGlobalSlotsFlag = cli.Uint64Flag{
		Name:  "txpool.globalslots",
		Usage: "Maximum number of executable transaction slots for all accounts",
		Value: evmcore.DefaultTxPoolConfig.GlobalSlots,
	}
	TxPoolAccountQueueFlag = cli.Uint64Flag{
		Name:  "txpool.accountqueue",
		Usage: "Maximum number of non-executable transaction slots permitted per account",
		Value: evmcore.DefaultTxPoolConfig.AccountQueue,
	}
	TxPoolGlobalQueueFlag = cli.Uint64Flag{
		Name:  "txpool.globalqueue",
		Usage: "Maximum number of non-executable transaction slots for all accounts",
		Value: evmcore.DefaultTxPoolConfig.GlobalQueue,
	}
	TxPoolLifetimeFlag = cli.DurationFlag{
		Name:  "txpool.lifetime",
		Usage: "Maximum amount of time non-executable transaction are queued",
		Value: evmcore.DefaultTxPoolConfig.Lifetime,
	}
	// Account settings
	UnlockedAccountFlag = cli.StringFlag{
		Name:  "unlock",
		Usage: "Comma separated list of accounts to unlock",
		Value: "",
	}
	PasswordFileFlag = cli.StringFlag{
		Name:  "password",
		Usage: "Password file to use for non-interactive password input",
		Value: "",
	}
	ExternalSignerFlag = cli.StringFlag{
		Name:  "signer",
		Usage: "External signer (url or path to ipc file)",
		Value: "",
	}
	InsecureUnlockAllowedFlag = cli.BoolFlag{
		Name:  "allow-insecure-unlock",
		Usage: "Allow insecure account unlocking when account-related RPCs are exposed by http",
	}
	// RPC settings
	IPCDisabledFlag = cli.BoolFlag{
		Name:  "ipcdisable",
		Usage: "Disable the IPC-RPC server",
	}
	IPCPathFlag = cli.StringFlag{
		Name:  "ipcpath",
		Usage: "Filename for IPC socket/pipe within the datadir (explicit paths escape it)",
	}
	HTTPEnabledFlag = cli.BoolFlag{
		Name:  "http",
		Usage: "Enable the HTTP-RPC server",
	}
	HTTPListenAddrFlag = cli.StringFlag{
		Name:  "http.addr",
		Usage: "HTTP-RPC server listening interface",
		Value: node.DefaultHTTPHost,
	}
	HTTPPortFlag = cli.IntFlag{
		Name:  "http.port",
		Usage: "HTTP-RPC server listening port",
		Value: 18545,
	}
	HTTPCORSDomainFlag = cli.StringFlag{
		Name:  "http.corsdomain",
		Usage: "Comma separated list of domains from which to accept cross origin requests (browser enforced)",
		Value: "",
	}
	HTTPVirtualHostsFlag = cli.StringFlag{
		Name:  "http.vhosts",
		Usage: "Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard.",
		Value: strings.Join(node.DefaultConfig.HTTPVirtualHosts, ","),
	}
	HTTPApiFlag = cli.StringFlag{
		Name:  "http.api",
		Usage: "API's offered over the HTTP-RPC interface",
		Value: "",
	}
	HTTPPathPrefixFlag = cli.StringFlag{
		Name:  "http.rpcprefix",
		Usage: "HTTP path path prefix on which JSON-RPC is served. Use '/' to serve on all paths.",
		Value: "",
	}
	WSEnabledFlag = cli.BoolFlag{
		Name:  "ws",
		Usage: "Enable the WS-RPC server",
	}
	WSListenAddrFlag = cli.StringFlag{
		Name:  "ws.addr",
		Usage: "WS-RPC server listening interface",
		Value: node.DefaultWSHost,
	}
	WSPortFlag = cli.IntFlag{
		Name:  "ws.port",
		Usage: "WS-RPC server listening port",
		Value: 18546,
	}
	WSApiFlag = cli.StringFlag{
		Name:  "ws.api",
		Usage: "API's offered over the WS-RPC interface",
		Value: "",
	}
	WSAllowedOriginsFlag = cli.StringFlag{
		Name:  "ws.origins",
		Usage: "Origins from which to accept websockets requests",
		Value: "",
	}
	WSPathPrefixFlag = cli.StringFlag{
		Name:  "ws.rpcprefix",
		Usage: "HTTP path prefix on which JSON-RPC is served. Use '/' to serve on all paths.",
		Value: "",
	}

	// Network Settings
	MaxPeersFlag = cli.IntFlag{
		Name:  "maxpeers",
		Usage: "Maximum number of network peers (network disabled if set to 0)",
		Value: node.DefaultConfig.P2P.MaxPeers,
	}
	MaxPendingPeersFlag = cli.IntFlag{
		Name:  "maxpendpeers",
		Usage: "Maximum number of pending connection attempts (defaults used if set to 0)",
		Value: node.DefaultConfig.P2P.MaxPendingPeers,
	}
	ListenPortFlag = cli.IntFlag{
		Name:  "port",
		Usage: "Network listening port",
		Value: 5050,
	}
	BootnodesFlag = cli.StringFlag{
		Name:  "bootnodes",
		Usage: "Comma separated enode URLs for P2P discovery bootstrap",
		Value: "",
	}
	NodeKeyFileFlag = cli.StringFlag{
		Name:  "nodekey",
		Usage: "P2P node key file",
	}
	NodeKeyHexFlag = cli.StringFlag{
		Name:  "nodekeyhex",
		Usage: "P2P node key as hex (for testing)",
	}
	NATFlag = cli.StringFlag{
		Name:  "nat",
		Usage: "NAT port mapping mechanism (any|none|upnp|pmp|pmp:<IP>|extip:<IP>)",
		Value: "any",
	}
	NoDiscoverFlag = cli.BoolFlag{
		Name:  "nodiscover",
		Usage: "Disables the peer discovery mechanism (manual peer addition)",
	}
	DiscoveryV4Flag = cli.BoolFlag{
		Name:  "discovery.v4",
		Usage: "Enables the legacy V4 discovery mechanism",
	}
	DiscoveryV5Flag = cli.BoolFlag{
		Name:  "discovery.v5",
		Usage: "Enables the RLPx V5 (Topic Discovery) mechanism",
	}
	NetrestrictFlag = cli.StringFlag{
		Name:  "netrestrict",
		Usage: "Restricts network communication to the given IP networks (CIDR masks)",
	}

	ConfigFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
	}
	CacheFlag = cli.IntFlag{
		Name:  "cache",
		Usage: "Megabytes of memory allocated to internal caching",
	}
	RPCGlobalGasCapFlag = cli.Uint64Flag{
		Name:  "rpc.gascap",
		Usage: "Sets a cap on gas that can be used in ftm_call/estimateGas (0=infinite)",
		Value: gossip.DefaultConfig(cachescale.Identity).RPCGasCap,
	}
	RPCGlobalEVMTimeoutFlag = &cli.DurationFlag{
		Name:  "rpc.evmtimeout",
		Usage: "Sets a timeout used for eth_call (0=infinite)",
		Value: gossip.DefaultConfig(cachescale.Identity).RPCEVMTimeout,
	}
	RPCGlobalTxFeeCapFlag = cli.Float64Flag{
		Name:  "rpc.txfeecap",
		Usage: "Sets a cap on transaction fee (in FTM) that can be sent via the RPC APIs (0 = no cap)",
		Value: gossip.DefaultConfig(cachescale.Identity).RPCTxFeeCap,
	}
	RPCGlobalTimeoutFlag = cli.DurationFlag{
		Name:  "rpc.timeout",
		Usage: "Time limit for RPC calls execution",
		Value: gossip.DefaultConfig(cachescale.Identity).RPCTimeout,
	}
	BatchRequestLimit = &cli.IntFlag{
		Name:  "rpc.batch-request-limit",
		Usage: "Maximum number of requests in a batch",
		Value: node.DefaultConfig.BatchRequestLimit,
	}
	BatchResponseMaxSize = &cli.IntFlag{
		Name:  "rpc.batch-response-max-size",
		Usage: "Maximum number of bytes returned from a batched call",
		Value: node.DefaultConfig.BatchResponseMaxSize,
	}
	MaxResponseSizeFlag = cli.IntFlag{
		Name:  "rpc.maxresponsesize",
		Usage: "Limit maximum size in some RPC calls execution",
		Value: gossip.DefaultConfig(cachescale.Identity).MaxResponseSize,
	}
	StructLogLimitFlag = cli.IntFlag{
		Name:  "rpc.structloglimit",
		Usage: "Limit maximum number of debug logs for structured EVM logs, 0=unlimited, negative value means no log results",
		Value: gossip.DefaultConfig(cachescale.Identity).StructLogLimit,
	}
	ModeFlag = cli.StringFlag{
		Name:  "mode",
		Usage: `Mode of the node ("rpc" or "validator")`,
		Value: "rpc",
	}
	ExitWhenAgeFlag = cli.DurationFlag{
		Name:  "exitwhensynced.age",
		Usage: "Exits after synchronisation reaches the required age",
	}
	ExitWhenEpochFlag = cli.Uint64Flag{
		Name:  "exitwhensynced.epoch",
		Usage: "Exits after synchronisation reaches the required epoch",
	}

	// Validator
	ValidatorIDFlag = cli.UintFlag{
		Name:  "validator.id",
		Usage: "ID of a validator to create events from",
		Value: 0,
	}
	ValidatorPubkeyFlag = cli.StringFlag{
		Name:  "validator.pubkey",
		Usage: "Public key of a validator to create events from",
		Value: "",
	}
	ValidatorPasswordFlag = cli.StringFlag{
		Name:  "validator.password",
		Usage: "Password to unlock validator private key",
		Value: "",
	}

	// Consensus
	SuppressFramePanicFlag = cli.BoolFlag{
		Name:  "lachesis.suppress-frame-panic",
		Usage: "Suppress frame missmatch error (when testing on historical imported/synced events)",
	}

	// StateDb
	LiveDbCacheFlag = cli.Int64Flag{
		Name: "statedb.livecache",
		Usage: fmt.Sprintf("Size of live db cache in bytes. Leaving this blank (which is generally recommended),"+
			"or setting this to <1 will automatically allocate cache size depending on how much cache you use with %s."+
			"Setting this value to <=2000 will result in pre-configured cache capacity of 2KB", CacheFlag.Name),
		Value: 0,
	}
	ArchiveCacheFlag = cli.IntFlag{
		Name: "statedb.archivecache",
		Usage: fmt.Sprintf("Size of archive cache in bytes. Leaving this blank (which is generally recommended),"+
			"or setting this to <1 will automatically allocate cache size depending on how much cache you use with %s."+
			"Setting this value to <=2000 will result in pre-configured cache capacity of 2KB", CacheFlag.Name),
		Value: 0,
	}
	StateDbCacheCapacityFlag = cli.IntFlag{
		Name: "statedb.cache",
		Usage: "The number of cached data elements by each StateDb instance. Since this is an experimental feature " +
			"used mainly by tests it is generally recommended leaving this blank. In tests no value lower than 1024 " +
			"is recommended. Setting this to <1 will automatically set the cache capacity to a DB defined default value.",
		Value: 0,
	}
	StateDbCheckPointInterval = cli.IntFlag{
		Name:   "statedb.checkpointinterval",
		Hidden: true, // Intended for testing
		Usage: "The number of blocks after which a db healing checkpoint will be created. " +
			"Setting this to <1 will automatically set the value to a DB defined default.",
		Value: 0,
	}
)
