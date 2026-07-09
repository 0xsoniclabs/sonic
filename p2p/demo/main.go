// Copyright 2026 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

// Command sonic-p2p-demo runs nodes on the Sonic p2p network for demonstration
// and hands-on diagnosis. It starts real validator or archive nodes on a faked
// chain that discover each other automatically (mDNS) and periodically log the
// health of their peer connections, and offers a one-shot scan of the network.
//
// Usage:
//
//	go run ./p2p/demo validator --id 0     # run validator 0 (IDs 0..N-1)
//	go run ./p2p/demo archive              # run an archive node
//	go run ./p2p/demo scan                 # crawl the network and print a report
//
// Start several validators (and an archive) in separate terminals; they form a
// mesh with no further configuration. See p2p/demo/README.md.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/log"
	cli "gopkg.in/urfave/cli.v1"

	"github.com/0xsoniclabs/sonic/p2p/networks"
)

var (
	debugFlag = cli.BoolFlag{
		Name:  "debug",
		Usage: "enable debug logging (shows discovery, dialing, and handshakes)",
	}
	idFlag = cli.IntFlag{
		Name:  "id",
		Usage: fmt.Sprintf("validator ID in [0,%d)", numValidators),
	}
	peerFlag = cli.StringSliceFlag{
		Name:  "peer",
		Usage: "multiaddr of a peer to bootstrap from, e.g. /ip4/1.2.3.4/udp/5/quic-v1/p2p/<id> (repeatable; not needed on one host)",
	}
	reportIntervalFlag = cli.DurationFlag{
		Name:  "report-interval",
		Usage: "how often to log the peer-health table",
		Value: 10 * time.Second,
	}
	waitFlag = cli.DurationFlag{
		Name:  "wait",
		Usage: "how long to discover peers before scanning",
		Value: 5 * time.Second,
	}
	maxNodesFlag = cli.IntFlag{
		Name:  "max-nodes",
		Usage: "maximum nodes to visit during a scan (0 = unbounded)",
		Value: 0,
	}
)

func main() {
	if err := newApp().Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newApp() *cli.App {
	app := cli.NewApp()
	app.Name = "sonic-p2p-demo"
	app.Usage = "run nodes on the Sonic p2p network for demonstration and diagnosis"
	app.Description = fmt.Sprintf(`Spin up validator and archive nodes on a faked chain to watch the Sonic p2p
network form and to diagnose peer connections.

The validator set is hard-coded and deterministic: there are %d validators with
IDs 0..%d, and every process derives the same set, so nodes agree on who the
validators are with no shared configuration. Nodes find each other automatically
on the local network via mDNS — just start several in separate terminals.

Each running node logs a table of its peers' health (round-trip time, jitter,
loss, and each peer's reported role and block height) every few seconds.`,
		numValidators, numValidators-1)
	app.HideVersion = true

	app.Flags = []cli.Flag{debugFlag}
	app.Before = func(ctx *cli.Context) error {
		level := log.LevelInfo
		if ctx.GlobalBool(debugFlag.Name) {
			level = log.LevelDebug
		}
		log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, level, true)))
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:      "validator",
			Usage:     "run a validator node with a given ID",
			ArgsUsage: " ",
			Description: fmt.Sprintf(`Run a validator node. --id selects which of the %d hard-coded validators this
process is (0..%d). Validators authenticate each other and form a full mesh.`,
				numValidators, numValidators-1),
			Flags:  []cli.Flag{idFlag, peerFlag, reportIntervalFlag},
			Action: validatorAction,
		},
		{
			Name:        "archive",
			Usage:       "run an archive node",
			ArgsUsage:   " ",
			Description: "Run an archive node: it joins the network and answers scan and health\nprobes, but runs no consensus and needs no key.",
			Flags:       []cli.Flag{peerFlag, reportIntervalFlag},
			Action:      archiveAction,
		},
		{
			Name:        "scan",
			Usage:       "crawl the network, print all discovered nodes, and exit",
			ArgsUsage:   " ",
			Description: "Stand up a temporary observer, discover peers, crawl the network with the\nscan protocol, print an aggregated report, and stop.",
			Flags:       []cli.Flag{peerFlag, waitFlag, maxNodesFlag},
			Action:      scanAction,
		},
	}
	return app
}

func validatorAction(ctx *cli.Context) error {
	if !ctx.IsSet(idFlag.Name) {
		return fmt.Errorf("--id is required (a validator ID in [0,%d))", numValidators)
	}
	id := ctx.Int(idFlag.Name)
	if id < 0 || id >= numValidators {
		return fmt.Errorf("invalid --id %d: must be in [0,%d)", id, numValidators)
	}
	runContext, stop := signalContext()
	defer stop()
	return runNode(runContext, networks.RoleValidator, id, ctx.StringSlice(peerFlag.Name), ctx.Duration(reportIntervalFlag.Name))
}

func archiveAction(ctx *cli.Context) error {
	runContext, stop := signalContext()
	defer stop()
	return runNode(runContext, networks.RoleArchive, 0, ctx.StringSlice(peerFlag.Name), ctx.Duration(reportIntervalFlag.Name))
}

func scanAction(ctx *cli.Context) error {
	runContext, stop := signalContext()
	defer stop()
	return runScan(runContext, ctx.StringSlice(peerFlag.Name), ctx.Duration(waitFlag.Name), ctx.Int(maxNodesFlag.Name))
}

// signalContext returns a context cancelled on the first SIGINT or SIGTERM, so
// Ctrl-C shuts a node down cleanly.
func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
