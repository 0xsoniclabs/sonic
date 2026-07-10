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

package gossip

import (
	"flag"
	"io"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

// TestMain is the entry point for the gossip test suite.
// It silences the go-ethereum global logger by default to keep test output clean.
func TestMain(m *testing.M) {
	var enableLogs bool
	flag.BoolVar(&enableLogs, "gossip.logs", false, "enable go-ethereum global logger output in gossip tests")
	flag.Parse()

	if !enableLogs {
		log.SetDefault(log.NewLogger(log.NewTerminalHandler(io.Discard, false)))
	}

	os.Exit(m.Run())
}
