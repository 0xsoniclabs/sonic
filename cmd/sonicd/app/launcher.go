package app

import (
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/0xsoniclabs/sonic/cmd/sonicd/diskusage"
	"github.com/0xsoniclabs/sonic/cmd/sonicd/metrics"
	"github.com/0xsoniclabs/sonic/cmd/sonicd/tracing"
	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/config/flags"
	"github.com/0xsoniclabs/sonic/version"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/console/prompt"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover/discfilter"
	"gopkg.in/urfave/cli.v1"

	ethmetrics "github.com/ethereum/go-ethereum/metrics"

	"github.com/0xsoniclabs/sonic/debug"

	// Force-load the tracer engines to trigger registration
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/live"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
)

var (
	// The app that holds all commands and flags.
	app *cli.App

	nodeFlags        []cli.Flag
	testFlags        []cli.Flag
	gpoFlags         []cli.Flag
	accountFlags     []cli.Flag
	performanceFlags []cli.Flag
	networkingFlags  []cli.Flag
	txpoolFlags      []cli.Flag
	operaFlags       []cli.Flag
	rpcFlags         []cli.Flag
	metricsFlags     []cli.Flag
)

func initFlags() {
	// Flags for testing purpose.
	testFlags = []cli.Flag{
		config.FakeNetFlag,
		flags.SuppressFramePanicFlag,
	}

	// Flags that configure the node.
	gpoFlags = []cli.Flag{}
	accountFlags = []cli.Flag{
		flags.UnlockedAccountFlag,
		flags.PasswordFileFlag,
		flags.ExternalSignerFlag,
		flags.InsecureUnlockAllowedFlag,
	}
	performanceFlags = []cli.Flag{
		flags.CacheFlag,
		flags.LiveDbCacheFlag,
		flags.ArchiveCacheFlag,
	}
	networkingFlags = []cli.Flag{
		flags.BootnodesFlag,
		flags.ListenPortFlag,
		flags.MaxPeersFlag,
		flags.MaxPendingPeersFlag,
		flags.NATFlag,
		flags.NoDiscoverFlag,
		flags.DiscoveryV4Flag,
		flags.DiscoveryV5Flag,
		flags.NetrestrictFlag,
		flags.NodeKeyFileFlag,
		flags.NodeKeyHexFlag,
	}
	txpoolFlags = []cli.Flag{
		flags.TxPoolLocalsFlag,
		flags.TxPoolNoLocalsFlag,
		flags.TxPoolJournalFlag,
		flags.TxPoolRejournalFlag,
		flags.TxPoolPriceLimitFlag,
		flags.TxPoolPriceBumpFlag,
		flags.TxPoolAccountSlotsFlag,
		flags.TxPoolGlobalSlotsFlag,
		flags.TxPoolAccountQueueFlag,
		flags.TxPoolGlobalQueueFlag,
		flags.TxPoolLifetimeFlag,
	}
	operaFlags = []cli.Flag{
		flags.IdentityFlag,
		flags.DataDirFlag,
		flags.MinFreeDiskSpaceFlag,
		flags.KeyStoreDirFlag,
		flags.USBFlag,
		flags.SmartCardDaemonPathFlag,
		flags.ExitWhenAgeFlag,
		flags.ExitWhenEpochFlag,
		flags.LightKDFFlag,
		flags.ConfigFileFlag,
		flags.ValidatorIDFlag,
		flags.ValidatorPubkeyFlag,
		flags.ValidatorPasswordFlag,
		flags.ModeFlag,
	}

	rpcFlags = []cli.Flag{
		flags.HTTPEnabledFlag,
		flags.HTTPListenAddrFlag,
		flags.HTTPPortFlag,
		flags.HTTPCORSDomainFlag,
		flags.HTTPVirtualHostsFlag,
		flags.HTTPApiFlag,
		flags.HTTPPathPrefixFlag,
		flags.WSEnabledFlag,
		flags.WSListenAddrFlag,
		flags.WSPortFlag,
		flags.WSApiFlag,
		flags.WSAllowedOriginsFlag,
		flags.WSPathPrefixFlag,
		flags.IPCDisabledFlag,
		flags.IPCPathFlag,
		flags.RPCGlobalGasCapFlag,
		flags.RPCGlobalEVMTimeoutFlag,
		flags.RPCGlobalTxFeeCapFlag,
		flags.RPCGlobalTimeoutFlag,
		flags.BatchRequestLimit,
		flags.BatchResponseMaxSize,
		flags.MaxResponseSizeFlag,
		flags.StructLogLimitFlag,
	}

	metricsFlags = []cli.Flag{
		metrics.MetricsEnabledFlag,
		metrics.MetricsEnabledExpensiveFlag,
		metrics.MetricsHTTPFlag,
		metrics.MetricsPortFlag,
		metrics.MetricsEnableInfluxDBFlag,
		metrics.MetricsInfluxDBEndpointFlag,
		metrics.MetricsInfluxDBDatabaseFlag,
		metrics.MetricsInfluxDBUsernameFlag,
		metrics.MetricsInfluxDBPasswordFlag,
		metrics.MetricsInfluxDBTagsFlag,
		metrics.MetricsEnableInfluxDBV2Flag,
		metrics.MetricsInfluxDBTokenFlag,
		metrics.MetricsInfluxDBBucketFlag,
		metrics.MetricsInfluxDBOrganizationFlag,
		tracing.EnableFlag,
	}

	nodeFlags = []cli.Flag{}
	nodeFlags = append(nodeFlags, gpoFlags...)
	nodeFlags = append(nodeFlags, accountFlags...)
	nodeFlags = append(nodeFlags, performanceFlags...)
	nodeFlags = append(nodeFlags, networkingFlags...)
	nodeFlags = append(nodeFlags, txpoolFlags...)
	nodeFlags = append(nodeFlags, operaFlags...)
}

// init the CLI app.
func initApp() {
	discfilter.Enable()

	initFlags()

	app = cli.NewApp()
	app.Name = "sonicd"
	app.Usage = "the Sonic network client"
	app.Version = version.VersionWithCommit(config.GitCommit, config.GitDate)
	app.Action = lachesisMain
	app.HideVersion = true // we have a command to print the version
	app.Commands = []cli.Command{
		versionCommand,
	}
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = append(app.Flags, testFlags...)
	app.Flags = append(app.Flags, nodeFlags...)
	app.Flags = append(app.Flags, rpcFlags...)
	app.Flags = append(app.Flags, debug.Flags...)
	app.Flags = append(app.Flags, metricsFlags...)

	app.Before = func(ctx *cli.Context) error {
		if err := debug.Setup(ctx); err != nil {
			return err
		}

		// Start metrics export if enabled
		err := metrics.SetupMetrics(ctx)
		if err != nil {
			return fmt.Errorf("failed to setup metrics: %w", err)
		}
		// Start system runtime metrics collection
		go ethmetrics.CollectProcessMetrics(3 * time.Second)
		return nil
	}

	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		prompt.Stdin.Close() // Resets terminal mode.

		return nil
	}
}

// opera is the main entry point into the system if no special subcommand is ran.
// It creates a default node based on the command line arguments and runs it in
// blocking mode, waiting for it to be shut down.
func lachesisMain(ctx *cli.Context) error {
	if args := ctx.Args(); len(args) > 0 {
		return fmt.Errorf("invalid command: %q", args[0])
	}

	cfg, err := config.MakeAllConfigs(ctx)
	if err != nil {
		return err
	}

	metrics.SetDataDir(cfg.Node.DataDir) // report disk space usage into metrics
	liveCache := ctx.GlobalInt64(flags.LiveDbCacheFlag.Name)
	if liveCache > 0 {
		cfg.OperaStore.EVM.StateDb.LiveCache = liveCache
	}

	archiveCache := ctx.GlobalInt64(flags.ArchiveCacheFlag.Name)
	if archiveCache > 0 {
		cfg.OperaStore.EVM.StateDb.ArchiveCache = archiveCache
	}

	node, _, nodeClose, err := config.MakeNode(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize the node: %w", err)
	}
	defer nodeClose()

	if err := startNode(ctx, node); err != nil {
		return fmt.Errorf("failed to start the node: %w", err)
	}
	node.Wait()
	return nil
}

// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces.
func startNode(ctx *cli.Context, stack *node.Node) error {
	// Start up the node itself
	if err := stack.Start(); err != nil {
		return fmt.Errorf("error starting protocol stack: %w", err)
	}
	go func() {
		stopNodeSig := make(chan os.Signal, 1)
		signal.Notify(stopNodeSig, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(stopNodeSig)

		startFreeDiskSpaceMonitor(ctx, stopNodeSig, stack.InstanceDir())

		<-stopNodeSig
		log.Info("Got interrupt, shutting down...")
		done := make(chan struct{})
		go func() {
			defer close(done)
			if err := stack.Close(); err != nil {
				log.Warn("Error during shutdown", "err", err)
			}
		}()
		for i := 10; i > 0; i-- {
			select {
			case <-stopNodeSig:
				if i > 1 {
					log.Warn("Already shutting down, interrupt more to panic.", "times", i-1)
				}
			case <-done:
				log.Info("Shutdown complete.")
				return
			}
		}
		// received 10 interrupts - kill the node forcefully
		debug.Exit() // ensure trace and CPU profile data is flushed.
		debug.LoudPanic("boom")
	}()

	// Unlock any account specifically requested
	err := unlockAccounts(ctx, stack)
	if err != nil {
		return fmt.Errorf("failed to unlock accounts: %w", err)
	}

	// Register wallet event handlers to open and auto-derive wallets
	events := make(chan accounts.WalletEvent, 16)
	stack.AccountManager().Subscribe(events)

	// Create a client to interact with local opera node.
	rpcClient := stack.Attach()
	ethClient := ethclient.NewClient(rpcClient)
	go func() {
		// Open any wallets already attached
		for _, wallet := range stack.AccountManager().Wallets() {
			if err := wallet.Open(""); err != nil {
				log.Warn("Failed to open wallet", "url", wallet.URL(), "err", err)
			}
		}
		// Listen for wallet event till termination
		for event := range events {
			switch event.Kind {
			case accounts.WalletArrived:
				if err := event.Wallet.Open(""); err != nil {
					log.Warn("New wallet appeared, failed to open", "url", event.Wallet.URL(), "err", err)
				}
			case accounts.WalletOpened:
				status, _ := event.Wallet.Status()
				log.Info("New wallet appeared", "url", event.Wallet.URL(), "status", status)

				var derivationPaths []accounts.DerivationPath
				if event.Wallet.URL().Scheme == "ledger" {
					derivationPaths = append(derivationPaths, accounts.LegacyLedgerBaseDerivationPath)
				}
				derivationPaths = append(derivationPaths, accounts.DefaultBaseDerivationPath)

				event.Wallet.SelfDerive(derivationPaths, ethClient)

			case accounts.WalletDropped:
				log.Info("Old wallet dropped", "url", event.Wallet.URL())
				event.Wallet.Close()
			}
		}
	}()

	return nil
}

func startFreeDiskSpaceMonitor(ctx *cli.Context, stopNodeSig chan os.Signal, path string) {
	var minFreeDiskSpace int
	if ctx.GlobalIsSet(flags.MinFreeDiskSpaceFlag.Name) {
		minFreeDiskSpace = ctx.GlobalInt(flags.MinFreeDiskSpaceFlag.Name)
	} else {
		minFreeDiskSpace = 8192
	}
	if minFreeDiskSpace > 0 {
		go diskusage.MonitorFreeDiskSpace(stopNodeSig, path, uint64(minFreeDiskSpace)*1024*1024)
	}
}
