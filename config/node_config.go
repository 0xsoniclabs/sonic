package config

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"strings"

	"github.com/0xsoniclabs/sonic/config/flags"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/p2p/netutil"
	"github.com/ethereum/go-ethereum/params"
	"gopkg.in/urfave/cli.v1"
)

// SetNodeConfig applies node-related command line flags to the config.
func SetNodeConfig(ctx *cli.Context, cfg *node.Config) error {
	if err := setP2PConfig(ctx, &cfg.P2P); err != nil {
		return err
	}
	if err := setIPC(ctx, cfg); err != nil {
		return err
	}
	setHTTP(ctx, cfg)
	setWS(ctx, cfg)
	setSmartCard(ctx, cfg)

	if identity := ctx.GlobalString(flags.IdentityFlag.Name); len(identity) > 0 {
		cfg.UserIdent = identity
	}
	if ctx.GlobalIsSet(flags.DataDirFlag.Name) {
		cfg.DataDir = ctx.GlobalString(flags.DataDirFlag.Name)
	} else {
		return fmt.Errorf("flag --%s is missing", flags.DataDirFlag.Name)
	}

	if ctx.GlobalIsSet(flags.ExternalSignerFlag.Name) {
		cfg.ExternalSigner = ctx.GlobalString(flags.ExternalSignerFlag.Name)
	}
	if ctx.GlobalIsSet(flags.KeyStoreDirFlag.Name) {
		cfg.KeyStoreDir = ctx.GlobalString(flags.KeyStoreDirFlag.Name)
	}
	if ctx.GlobalIsSet(flags.LightKDFFlag.Name) {
		cfg.UseLightweightKDF = ctx.GlobalBool(flags.LightKDFFlag.Name)
	}
	if ctx.GlobalIsSet(flags.USBFlag.Name) {
		cfg.USB = ctx.GlobalBool(flags.USBFlag.Name)
	}
	if ctx.GlobalIsSet(flags.InsecureUnlockAllowedFlag.Name) {
		cfg.InsecureUnlockAllowed = ctx.GlobalBool(flags.InsecureUnlockAllowedFlag.Name)
	}
	return nil
}

// MakePasswordList reads password lines from the file specified by the global --password flag.
func MakePasswordList(ctx *cli.Context) ([]string, error) {
	path := ctx.String(flags.PasswordFileFlag.Name)
	if path == "" {
		return nil, nil
	}
	text, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read password file: %w", err)
	}
	lines := strings.Split(string(text), "\n")
	// Sanitise DOS line endings.
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	return lines, nil
}

// setNodeKey creates a node key from set command line flags, either loading it
// from a file or as a specified hex value. If neither flags were provided, this
// method returns nil and an emphemeral key is to be generated.
func setNodeKey(ctx *cli.Context, cfg *p2p.Config) error {
	var (
		hex  = ctx.GlobalString(flags.NodeKeyHexFlag.Name)
		file = ctx.GlobalString(flags.NodeKeyFileFlag.Name)
		key  *ecdsa.PrivateKey
		err  error
	)
	switch {
	case file != "" && hex != "":
		return fmt.Errorf("options %q and %q are mutually exclusive", flags.NodeKeyFileFlag.Name, flags.NodeKeyHexFlag.Name)
	case file != "":
		if key, err = crypto.LoadECDSA(file); err != nil {
			return fmt.Errorf("option %q: %v", flags.NodeKeyFileFlag.Name, err)
		}
		cfg.PrivateKey = key
	case hex != "":
		if key, err = crypto.HexToECDSA(hex); err != nil {
			return fmt.Errorf("option %q: %v", flags.NodeKeyHexFlag.Name, err)
		}
		cfg.PrivateKey = key
	}
	return nil
}

// setBootstrapNodes creates a list of bootstrap nodes from the command line
// flags, reverting to pre-configured ones if none have been specified.
func setBootstrapNodes(ctx *cli.Context, cfg *p2p.Config) {
	urls := params.MainnetBootnodes
	switch {
	case ctx.GlobalIsSet(flags.BootnodesFlag.Name):
		urls = splitAndTrim(ctx.GlobalString(flags.BootnodesFlag.Name))
	case cfg.BootstrapNodes != nil:
		return // already set, don't apply defaults.
	}

	cfg.BootstrapNodes = make([]*enode.Node, 0, len(urls))
	for _, url := range urls {
		if url != "" {
			node, err := enode.Parse(enode.ValidSchemes, url)
			if err != nil {
				log.Crit("Bootstrap URL invalid", "enode", url, "err", err)
				continue
			}
			cfg.BootstrapNodes = append(cfg.BootstrapNodes, node)
		}
	}
}

// setBootstrapNodesV5 creates a list of bootstrap nodes from the command line
// flags, reverting to pre-configured ones if none have been specified.
func setBootstrapNodesV5(ctx *cli.Context, cfg *p2p.Config) {
	urls := params.V5Bootnodes
	switch {
	case ctx.GlobalIsSet(flags.BootnodesFlag.Name):
		urls = splitAndTrim(ctx.GlobalString(flags.BootnodesFlag.Name))
	case cfg.BootstrapNodesV5 != nil:
		return // already set, don't apply defaults.
	}

	cfg.BootstrapNodesV5 = make([]*enode.Node, 0, len(urls))
	for _, url := range urls {
		if url != "" {
			node, err := enode.Parse(enode.ValidSchemes, url)
			if err != nil {
				log.Error("Bootstrap URL invalid", "enode", url, "err", err)
				continue
			}
			cfg.BootstrapNodesV5 = append(cfg.BootstrapNodesV5, node)
		}
	}
}

// setListenAddress creates a TCP listening address string from set command
// line flags.
func setListenAddress(ctx *cli.Context, cfg *p2p.Config) {
	if ctx.GlobalIsSet(flags.ListenPortFlag.Name) {
		cfg.ListenAddr = fmt.Sprintf(":%d", ctx.GlobalInt(flags.ListenPortFlag.Name))
	}
}

// splitAndTrim splits input separated by a comma
// and trims excessive white space from the substrings.
func splitAndTrim(input string) (ret []string) {
	l := strings.Split(input, ",")
	for _, r := range l {
		if r = strings.TrimSpace(r); r != "" {
			ret = append(ret, r)
		}
	}
	return ret
}

// setHTTP creates the HTTP RPC listener interface string from the set
// command line flags, returning empty if the HTTP endpoint is disabled.
func setHTTP(ctx *cli.Context, cfg *node.Config) {
	if ctx.GlobalBool(flags.HTTPEnabledFlag.Name) && cfg.HTTPHost == "" {
		cfg.HTTPHost = "127.0.0.1"
		if ctx.GlobalIsSet(flags.HTTPListenAddrFlag.Name) {
			cfg.HTTPHost = ctx.GlobalString(flags.HTTPListenAddrFlag.Name)
		}
	}
	if ctx.GlobalIsSet(flags.HTTPPortFlag.Name) {
		cfg.HTTPPort = ctx.GlobalInt(flags.HTTPPortFlag.Name)
	}
	if ctx.GlobalIsSet(flags.HTTPCORSDomainFlag.Name) {
		cfg.HTTPCors = splitAndTrim(ctx.GlobalString(flags.HTTPCORSDomainFlag.Name))
	}
	if ctx.GlobalIsSet(flags.HTTPApiFlag.Name) {
		cfg.HTTPModules = splitAndTrim(ctx.GlobalString(flags.HTTPApiFlag.Name))
	}
	if ctx.GlobalIsSet(flags.HTTPVirtualHostsFlag.Name) {
		cfg.HTTPVirtualHosts = splitAndTrim(ctx.GlobalString(flags.HTTPVirtualHostsFlag.Name))
	}
	if ctx.GlobalIsSet(flags.HTTPPathPrefixFlag.Name) {
		cfg.HTTPPathPrefix = ctx.GlobalString(flags.HTTPPathPrefixFlag.Name)
	}
	if ctx.IsSet(flags.BatchRequestLimit.Name) {
		cfg.BatchRequestLimit = ctx.Int(flags.BatchRequestLimit.Name)
	}
	if ctx.IsSet(flags.BatchResponseMaxSize.Name) {
		cfg.BatchResponseMaxSize = ctx.Int(flags.BatchResponseMaxSize.Name)
	}
}

// setWS creates the WebSocket RPC listener interface string from the set
// command line flags, returning empty if the HTTP endpoint is disabled.
func setWS(ctx *cli.Context, cfg *node.Config) {
	if ctx.GlobalBool(flags.WSEnabledFlag.Name) && cfg.WSHost == "" {
		cfg.WSHost = "127.0.0.1"
		if ctx.GlobalIsSet(flags.WSListenAddrFlag.Name) {
			cfg.WSHost = ctx.GlobalString(flags.WSListenAddrFlag.Name)
		}
	}
	if ctx.GlobalIsSet(flags.WSPortFlag.Name) {
		cfg.WSPort = ctx.GlobalInt(flags.WSPortFlag.Name)
	}
	if ctx.GlobalIsSet(flags.WSAllowedOriginsFlag.Name) {
		cfg.WSOrigins = splitAndTrim(ctx.GlobalString(flags.WSAllowedOriginsFlag.Name))
	}
	if ctx.GlobalIsSet(flags.WSApiFlag.Name) {
		cfg.WSModules = splitAndTrim(ctx.GlobalString(flags.WSApiFlag.Name))
	}
	if ctx.GlobalIsSet(flags.WSPathPrefixFlag.Name) {
		cfg.WSPathPrefix = ctx.GlobalString(flags.WSPathPrefixFlag.Name)
	}
}

// setIPC creates an IPC path configuration from the set command line flags,
// returning an empty string if IPC was explicitly disabled, or the set path.
func setIPC(ctx *cli.Context, cfg *node.Config) error {
	if ctx.GlobalIsSet(flags.IPCDisabledFlag.Name) && ctx.GlobalIsSet(flags.IPCPathFlag.Name) {
		return fmt.Errorf("flags --%s and --%s can't be used at the same time", flags.IPCDisabledFlag.Name, flags.IPCPathFlag.Name)
	}
	switch {
	case ctx.GlobalBool(flags.IPCDisabledFlag.Name):
		cfg.IPCPath = ""
	case ctx.GlobalIsSet(flags.IPCPathFlag.Name):
		cfg.IPCPath = ctx.GlobalString(flags.IPCPathFlag.Name)
	}
	return nil
}

func setP2PConfig(ctx *cli.Context, cfg *p2p.Config) error {
	if err := setNodeKey(ctx, cfg); err != nil {
		return err
	}
	if ctx.GlobalIsSet(flags.NATFlag.Name) {
		natif, err := nat.Parse(ctx.GlobalString(flags.NATFlag.Name))
		if err != nil {
			return fmt.Errorf("option %s: %v", flags.NATFlag.Name, err)
		}
		cfg.NAT = natif
	}
	setListenAddress(ctx, cfg)
	setBootstrapNodes(ctx, cfg)
	setBootstrapNodesV5(ctx, cfg)

	if ctx.GlobalIsSet(flags.MaxPeersFlag.Name) {
		cfg.MaxPeers = ctx.GlobalInt(flags.MaxPeersFlag.Name)
	}
	log.Info("Maximum peer count", "total", cfg.MaxPeers)

	if ctx.GlobalIsSet(flags.MaxPendingPeersFlag.Name) {
		cfg.MaxPendingPeers = ctx.GlobalInt(flags.MaxPendingPeersFlag.Name)
	}
	if ctx.GlobalIsSet(flags.NoDiscoverFlag.Name) {
		cfg.NoDiscovery = true
	}
	if ctx.GlobalIsSet(flags.DiscoveryV4Flag.Name) {
		cfg.DiscoveryV4 = ctx.GlobalBool(flags.DiscoveryV4Flag.Name)
	}
	if ctx.GlobalIsSet(flags.DiscoveryV5Flag.Name) {
		cfg.DiscoveryV5 = ctx.GlobalBool(flags.DiscoveryV5Flag.Name)
	}

	if netrestrict := ctx.GlobalString(flags.NetrestrictFlag.Name); netrestrict != "" {
		list, err := netutil.ParseNetlist(netrestrict)
		if err != nil {
			return fmt.Errorf("option %q: %v", flags.NetrestrictFlag.Name, err)
		}
		cfg.NetRestrict = list
	}

	return nil
}

func setSmartCard(ctx *cli.Context, cfg *node.Config) {
	// Skip enabling smartcards if no path is set
	path := ctx.GlobalString(flags.SmartCardDaemonPathFlag.Name)
	if path == "" {
		return
	}
	// Sanity check that the smartcard path is valid
	fi, err := os.Stat(path)
	if err != nil {
		log.Info("Smartcard socket not found, disabling", "err", err)
		return
	}
	if fi.Mode()&os.ModeType != os.ModeSocket {
		log.Error("Invalid smartcard daemon path", "path", path, "type", fi.Mode().String())
		return
	}
	// Smartcard daemon path exists and is a socket, enable it
	cfg.SmartCardDaemonPath = path
}
