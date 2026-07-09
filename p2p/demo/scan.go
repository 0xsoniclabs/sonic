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

package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/p2p/networks"
	"github.com/0xsoniclabs/sonic/p2p/protocols"
)

// runScan stands up a throwaway observer node, discovers peers via mDNS (and any
// explicit --peer addresses), crawls the network with the scan protocol, prints
// the aggregated report, and stops. It does not run consensus.
func runScan(ctx context.Context, peers []string, wait time.Duration, maxNodes int) error {
	out := log.Root()

	node, err := newNode(out, peers)
	if err != nil {
		return err
	}
	if err := node.Start(); err != nil {
		return fmt.Errorf("failed to start scan node: %w", err)
	}
	defer func() { _ = node.Stop() }()

	discovery, err := startDiscovery(ctx, node, out)
	if err != nil {
		return err
	}
	defer func() { _ = discovery.Close() }()

	out.Info("discovering peers before scanning", "wait", wait)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(wait):
	}

	seeds := nodePeerSource{node: node}.Peers()
	if len(seeds) == 0 {
		out.Warn("no peers discovered - start some demo nodes, then scan again")
		return nil
	}

	report := protocols.NewScanner(node, maxNodes).Scan(ctx, seeds)
	out.Info(renderReport(report))
	return nil
}

// renderReport formats a NetworkReport as a readable multi-line summary.
func renderReport(report protocols.NetworkReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "network scan complete - %d node(s) discovered\n", report.NodeCount)

	b.WriteString("  roles:\n")
	for _, role := range []networks.Role{networks.RoleValidator, networks.RoleArchive, networks.RoleObserver} {
		if count := report.RoleCounts[role]; count > 0 {
			fmt.Fprintf(&b, "    %-9s %d\n", roleName(role), count)
		}
	}

	b.WriteString("  client versions:\n")
	for _, version := range sortedKeys(report.ClientVersions) {
		fmt.Fprintf(&b, "    %-24s %d\n", version, report.ClientVersions[version])
	}

	b.WriteString("  block heights:\n")
	for _, height := range sortedHeights(report.HeightHistogram) {
		fmt.Fprintf(&b, "    %-12d %d node(s)\n", height, report.HeightHistogram[height])
	}
	return strings.TrimRight(b.String(), "\n")
}

func sortedKeys(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedHeights(counts map[uint64]int) []uint64 {
	heights := make([]uint64, 0, len(counts))
	for height := range counts {
		heights = append(heights, height)
	}
	sort.Slice(heights, func(i, j int) bool { return heights[i] < heights[j] })
	return heights
}
