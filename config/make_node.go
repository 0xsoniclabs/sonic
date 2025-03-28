package config

import (
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/external"
	"github.com/ethereum/go-ethereum/accounts/scwallet"
	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/0xsoniclabs/sonic/config/flags"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/gossip/emitter"
	"github.com/0xsoniclabs/sonic/integration"
	"github.com/0xsoniclabs/sonic/utils/errlock"
	"github.com/0xsoniclabs/sonic/valkeystore"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"gopkg.in/urfave/cli.v1"
)

var (
	chainInfoGauge = metrics.GetOrRegisterGaugeInfo("chain/info", nil)
)

func MakeNode(ctx *cli.Context, cfg *Config) (*node.Node, *gossip.Service, func(), error) {
	var success bool
	var cleanup []func()
	defer func() { // if the function fails, clean-up in defer, otherwise return cleaning function
		if !success {
			for i := len(cleanup) - 1; i >= 0; i-- {
				cleanup[i]()
			}
		}
	}()

	// check errlock file
	errlock.SetDefaultDatadir(cfg.Node.DataDir)
	if err := errlock.Check(); err != nil {
		return nil, nil, nil, err
	}

	engine, dagIndex, gdb, cdb, blockProc, closeDBs, err := integration.MakeEngine(path.Join(cfg.Node.DataDir, "chaindata"), cfg.AppConfigs())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to make consensus engine: %w", err)
	}
	cleanup = append(cleanup, func() {
		if err := gdb.Close(); err != nil {
			log.Warn("Failed to close gossip store", "err", err)
		}
		if err := cdb.Close(); err != nil {
			log.Warn("Failed to close consensus database", "err", err)
		}
		if closeDBs != nil {
			if err := closeDBs(); err != nil {
				log.Warn("Failed to close databases", "err", err)
			}
		}
	})

	// substitute default bootnodes if requested
	networkName := ""
	if gdb.HasBlockEpochState() {
		networkName = gdb.GetRules().Name
	}
	if needDefaultBootnodes(cfg.Node.P2P.BootstrapNodes) {
		bootnodes := Bootnodes[networkName]
		if bootnodes == nil {
			bootnodes = []string{}
		}
		setBootnodes(ctx, bootnodes, &cfg.Node)
	}

	stack, err := MakeNetworkStack(ctx, &cfg.Node)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unlock validator key: %w", err)
	}
	cleanup = append(cleanup, func() {
		if err := stack.Close(); err != nil && err != node.ErrNodeStopped {
			log.Warn("Failed to close stack", "err", err)
		}
	})

	keystoreDir, err := cfg.Node.KeyDirConfig()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to setup account config: %w", err)
	}
	valKeystore := valkeystore.NewDefaultFileKeystore(path.Join(keystoreDir, "validator"))
	valPubkey := cfg.Emitter.Validator.PubKey
	if key := getFakeValidatorKey(ctx); key != nil && cfg.Emitter.Validator.ID != 0 {
		if err := addFakeValidatorKey(ctx, key, valPubkey, valKeystore); err != nil {
			return nil, nil, nil, err
		}
		coinbase := integration.SetAccountKey(stack.AccountManager(), key, "fakepassword")
		log.Info("Unlocked fake validator account", "address", coinbase.Address.Hex())
	}

	// unlock validator key
	if !valPubkey.Empty() {
		err := unlockValidatorKey(ctx, valPubkey, valKeystore)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to unlock validator key: %w", err)
		}
	}
	signer := valkeystore.NewSigner(valKeystore)

	// Create and register a gossip network service.
	newTxPool := func(reader evmcore.StateReader) gossip.TxPool {
		if cfg.TxPool.Journal != "" {
			cfg.TxPool.Journal = path.Join(cfg.Node.DataDir, cfg.TxPool.Journal)
		}
		return evmcore.NewTxPool(cfg.TxPool, reader.Config(), reader)
	}
	haltCheck := func(oldEpoch, newEpoch consensus.Epoch, age time.Time) bool {
		stop := ctx.GlobalIsSet(flags.ExitWhenAgeFlag.Name) && ctx.GlobalDuration(flags.ExitWhenAgeFlag.Name) >= time.Since(age)
		stop = stop || ctx.GlobalIsSet(flags.ExitWhenEpochFlag.Name) && consensus.Epoch(ctx.GlobalUint64(flags.ExitWhenEpochFlag.Name)) <= newEpoch
		if stop {
			go func() {
				// do it in a separate thread to avoid deadlock
				_ = stack.Close()
			}()
			return true
		}
		return false
	}
	svc, err := gossip.NewService(stack, cfg.Opera, gdb, blockProc, engine, dagIndex, newTxPool, haltCheck)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create the gossip service: %w", err)
	}
	err = engine.Bootstrap(svc.GetConsensusCallbacks())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to bootstrap the consensus engine: %w", err)
	}
	err = engine.Reset(gdb.GetEpoch(), gdb.GetValidators())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to reset the consensus engine: %w", err)
	}
	svc.ReprocessEpochEvents()

	// commit dbs to avoid dirty state when the rest of the startup fails
	if err := gdb.Commit(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to commit gossip store: %w", err)
	}

	if cfg.Emitter.Validator.ID != 0 {
		svc.RegisterEmitter(emitter.NewEmitter(
			cfg.Emitter,
			svc.EmitterWorld(signer),
			gdb.AsBaseFeeSource(),
		))
	}

	stack.RegisterAPIs(svc.APIs())
	stack.RegisterProtocols(svc.Protocols())
	stack.RegisterLifecycle(svc)

	rules, _ := gdb.GetEpochRules()
	chainInfoGauge.Update(metrics.GaugeInfoValue{"chain_id": strconv.FormatUint(rules.NetworkID, 10)})

	success = true // skip cleanup in defer - keep it for the returned cleanup function
	return stack, svc, func() {
		for i := len(cleanup) - 1; i >= 0; i-- {
			cleanup[i]()
		}
	}, nil
}

func MakeNetworkStack(ctx *cli.Context, cfg *node.Config) (*node.Node, error) {
	stack, err := node.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create the protocol stack: %w", err)
	}

	keystoreDir, err := cfg.KeyDirConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to setup account config: %w", err)
	}
	err = setAccountManagerBackends(cfg, stack.AccountManager(), keystoreDir)
	if err != nil {
		return nil, fmt.Errorf("failed to setup account manager: %w", err)
	}

	return stack, nil
}

func setAccountManagerBackends(conf *node.Config, am *accounts.Manager, keydir string) error {
	scryptN := keystore.StandardScryptN
	scryptP := keystore.StandardScryptP
	if conf.UseLightweightKDF {
		scryptN = keystore.LightScryptN
		scryptP = keystore.LightScryptP
	}

	// Assemble the supported backends
	if len(conf.ExternalSigner) > 0 {
		log.Info("Using external signer", "url", conf.ExternalSigner)
		if extBackend, err := external.NewExternalBackend(conf.ExternalSigner); err == nil {
			am.AddBackend(extBackend)
			return nil
		} else {
			return fmt.Errorf("error connecting to external signer: %v", err)
		}
	}

	// For now, we're using EITHER external signer OR local signers.
	// If/when we implement some form of lockfile for USB and keystore wallets,
	// we can have both, but it's very confusing for the user to see the same
	// accounts in both externally and locally, plus very racey.
	am.AddBackend(keystore.NewKeyStore(keydir, scryptN, scryptP))
	if conf.USB {
		// Start a USB hub for Ledger hardware wallets
		if ledgerhub, err := usbwallet.NewLedgerHub(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start Ledger hub, disabling: %v", err))
		} else {
			am.AddBackend(ledgerhub)
		}
		// Start a USB hub for Trezor hardware wallets (HID version)
		if trezorhub, err := usbwallet.NewTrezorHubWithHID(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start HID Trezor hub, disabling: %v", err))
		} else {
			am.AddBackend(trezorhub)
		}
		// Start a USB hub for Trezor hardware wallets (WebUSB version)
		if trezorhub, err := usbwallet.NewTrezorHubWithWebUSB(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start WebUSB Trezor hub, disabling: %v", err))
		} else {
			am.AddBackend(trezorhub)
		}
	}
	if len(conf.SmartCardDaemonPath) > 0 {
		// Start a smart card hub
		if schub, err := scwallet.NewHub(conf.SmartCardDaemonPath, scwallet.Scheme, keydir); err != nil {
			log.Warn(fmt.Sprintf("Failed to start smart card hub, disabling: %v", err))
		} else {
			am.AddBackend(schub)
		}
	}

	return nil
}
