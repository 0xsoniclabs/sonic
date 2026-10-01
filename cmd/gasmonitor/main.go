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

// gasmonitor follows the blocks of an RPC endpoint and reports blocks whose
// header gasUsed does not match the gas accounted for by their receipts
// (eth_getBlockReceipts), and whether these values change when the same
// blocks are queried again deeper in history.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func main() {
	cfg, err := parseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func parseFlags(args []string) (config, error) {
	fs := flag.NewFlagSet("gasmonitor", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `Usage: gasmonitor [flags]

Follows new blocks on an RPC endpoint and, for each block, compares the header
gasUsed with the receipts returned by eth_getBlockReceipts. Every block is
queried again at the given depths behind the head to detect values that
change between queries. A Markdown report is kept up to date in -out.

Ctrl-C once stops following new blocks and waits for pending re-checks;
Ctrl-C twice quits immediately. The report is written in both cases.

Flags:
`)
		fs.PrintDefaults()
	}
	var cfg config
	var depths string
	fs.StringVar(&cfg.rpcURL, "rpc", "https://rpc.soniclabs.com", "JSON-RPC HTTP endpoint; anything after the host (e.g. an API key) is masked in the console and report")
	fs.StringVar(&cfg.reportPath, "out", "", "report file (default gas-report-<timestamp>.md)")
	fs.StringVar(&depths, "recheck", "10,100,1000", "comma-separated depths (blocks behind head) at which each block is re-checked; empty disables")
	fs.IntVar(&cfg.workers, "workers", 4, "number of concurrent RPC workers")
	fs.DurationVar(&cfg.poll, "poll", 500*time.Millisecond, "head polling interval")
	fs.DurationVar(&cfg.reportEvery, "report-interval", 10*time.Second, "how often the report file is rewritten")
	fs.DurationVar(&cfg.timeout, "timeout", 15*time.Second, "timeout of a single RPC request")
	fs.IntVar(&cfg.retries, "retries", 3, "retries of a block query that returned an RPC error")
	fs.Uint64Var(&cfg.from, "from", 0, "first block to check (default: current head)")
	fs.Uint64Var(&cfg.maxBlocks, "blocks", 0, "stop following after this many blocks (0 = unlimited)")
	fs.DurationVar(&cfg.duration, "duration", 0, "stop following after this time (0 = unlimited)")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if fs.NArg() > 0 {
		return cfg, fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	for _, s := range strings.Split(depths, ",") {
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		d, err := strconv.ParseUint(s, 10, 64)
		if err != nil || d == 0 {
			return cfg, fmt.Errorf("invalid re-check depth %q: must be a positive integer", s)
		}
		cfg.depths = append(cfg.depths, d)
	}
	slices.Sort(cfg.depths)
	cfg.depths = slices.Compact(cfg.depths)
	if cfg.workers < 1 {
		return cfg, fmt.Errorf("-workers must be at least 1")
	}
	if cfg.reportPath == "" {
		cfg.reportPath = defaultReportPath()
	}
	return cfg, nil
}

func run(cfg config) error {
	con := newConsole(os.Stdout)
	m := newMonitor(cfg, con)
	con.event("writing report to %s", cfg.reportPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopFollowing := make(chan struct{})
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		<-signals
		close(stopFollowing)
		<-signals
		con.event("quitting")
		cancel()
	}()

	// Background status line and report refresh.
	done := make(chan struct{})
	var bg sync.WaitGroup
	bg.Add(1)
	go func() {
		defer bg.Done()
		status := time.NewTicker(time.Second)
		defer status.Stop()
		report := time.NewTicker(cfg.reportEvery)
		defer report.Stop()
		for {
			select {
			case <-done:
				return
			case <-status.C:
				con.status(m.statusLine())
			case <-report.C:
				if err := m.writeReport(); err != nil {
					con.event("error: writing report: %v", err)
				}
			}
		}
	}()

	runErr := m.run(ctx, stopFollowing)
	close(done)
	bg.Wait()

	m.mu.Lock()
	m.finished = true
	m.mu.Unlock()
	con.status(m.statusLine())
	con.finish()
	if err := m.writeReport(); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}
	fmt.Printf("report written to %s\n", cfg.reportPath)
	return runErr
}

// console prints event lines and keeps a single, continuously updated status
// line at the bottom when attached to a terminal.
type console struct {
	mu         sync.Mutex
	out        io.Writer
	tty        bool
	line       string
	lastPlain  time.Time
	plainEvery time.Duration
}

func newConsole(f *os.File) *console {
	tty := false
	if info, err := f.Stat(); err == nil {
		tty = info.Mode()&os.ModeCharDevice != 0
	}
	return &console{out: f, tty: tty, plainEvery: 30 * time.Second}
}

func (c *console) event(format string, args ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	msg := time.Now().Format(time.TimeOnly) + " " + fmt.Sprintf(format, args...)
	if c.tty {
		fmt.Fprintf(c.out, "\r\033[K%s\n%s", msg, c.line)
	} else {
		fmt.Fprintln(c.out, msg)
	}
}

func (c *console) status(line string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.line = line
	if c.tty {
		fmt.Fprintf(c.out, "\r\033[K%s", line)
	} else if time.Since(c.lastPlain) >= c.plainEvery {
		c.lastPlain = time.Now()
		fmt.Fprintln(c.out, time.Now().Format(time.TimeOnly)+" "+line)
	}
}

func (c *console) finish() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tty {
		fmt.Fprintln(c.out)
	}
}
