// Copyright 2025 Sonic Operations Ltd
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

// The package daemon implements tooling for interacting with systemd
// and running the sonic client as a daemon.

package daemon

import (
	"log/slog"

	"github.com/coreos/go-systemd/daemon"
)

// NotifyReady notifies systemd that the service is ready.
// This function should be called when the service is fully initialized and
// ready to accept requests.
func NotifyReady() {
	_, err := daemon.SdNotify(false, daemon.SdNotifyReady)
	if err != nil {
		slog.Error("Failed to notify ready to systemd", "error", err)
	}
}

// NotifyStopping notifies systemd that the service is stopping.
// This function should be called when the service is about to stop, changing
// daemon status to stopping while it is in the process of shutting down: data
// is being flushed and resources released.
func NotifyStopping() {
	_, err := daemon.SdNotify(false, daemon.SdNotifyStopping)
	if err != nil {
		slog.Error("Failed to notify stop to systemd", "error", err)
	}
}

// NotifyHeartbeat notifies systemd that the service is healthy and responsive.
// This allows the systemd watchdog to monitor the service and restart it if
// it becomes unresponsive.
//
// The watchdog timeout should be configured to at least twice the interval
// at which this function is called. Watchdog can operate at a much larger
// interval, but this is the minimum requirement.
func NotifyHeartbeat() {
	_, err := daemon.SdNotify(false, daemon.SdNotifyWatchdog)
	if err != nil {
		slog.Error("Failed to notify heartbeat to systemd", "error", err)
	}
}
