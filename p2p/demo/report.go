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

	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/networks"
	"github.com/0xsoniclabs/sonic/p2p/protocols"
)

// reportLoop periodically prints the node's view of its peers' health until ctx
// is cancelled.
func reportLoop(ctx context.Context, out log.Logger, monitor *protocols.HealthMonitor, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reportHealth(out, monitor)
		}
	}
}

// reportHealth logs a compact, human-readable table of the current per-peer
// health, one row per peer, sorted by peer ID for stable output.
func reportHealth(out log.Logger, monitor *protocols.HealthMonitor) {
	snapshot := monitor.Snapshot()
	if len(snapshot) == 0 {
		out.Info("peer health: no peers yet - start more demo nodes")
		return
	}
	sort.Slice(snapshot, func(i, j int) bool {
		return snapshot[i].Peer.String() < snapshot[j].Peer.String()
	})

	var table strings.Builder
	table.WriteString("peer health\n")
	fmt.Fprintf(&table, "  %-12s  %-9s  %10s  %10s  %6s  %10s  %7s\n",
		"PEER", "ROLE", "RTT", "JITTER", "LOSS", "HEIGHT", "PROBES")
	for _, entry := range snapshot {
		sample := entry.Sample
		fmt.Fprintf(&table, "  %-12s  %-9s  %10s  %10s  %5.0f%%  %10d  %7d\n",
			shortPeerID(entry.Peer),
			roleName(sample.Role),
			formatDuration(sample.Average),
			formatDuration(sample.Jitter),
			sample.LossRate()*100,
			sample.BlockHeight,
			sample.Probes,
		)
	}
	out.Info(strings.TrimRight(table.String(), "\n"))
}

// shortPeerID returns a shortened, recognisable form of a peer ID.
func shortPeerID(id p2p.PeerID) string {
	full := id.String()
	if len(full) <= 12 {
		return full
	}
	return full[len(full)-12:]
}

// roleName renders a role for display.
func roleName(role networks.Role) string {
	switch role {
	case networks.RoleValidator:
		return "validator"
	case networks.RoleArchive:
		return "archive"
	case networks.RoleObserver:
		return "observer"
	default:
		return "unknown"
	}
}

// formatDuration renders a probe duration in milliseconds, or a dash if unset.
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000)
}
