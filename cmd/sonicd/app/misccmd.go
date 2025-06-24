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

package app

import (
	"fmt"
	"os"
	"runtime"

	"github.com/0xsoniclabs/sonic/version"

	"github.com/0xsoniclabs/sonic/config"
	"gopkg.in/urfave/cli.v1"

	"github.com/0xsoniclabs/sonic/gossip"
)

var (
	versionCommand = cli.Command{
		Action:    versionAction,
		Name:      "version",
		Usage:     "Print version numbers",
		ArgsUsage: " ",
		Category:  "MISCELLANEOUS COMMANDS",
		Description: `
The output of this command is supposed to be machine-readable.
`,
	}
)

func versionAction(ctx *cli.Context) error {
	fmt.Println(config.ClientIdentifier)
	fmt.Println("Version:", version.String())
	if commit := version.GitCommit(); commit != "" {
		fmt.Println("Git Commit:", commit)
	}
	if date := version.GitDate(); date != "" {
		fmt.Println("Git Commit Date:", date)
	}
	fmt.Println("Architecture:", runtime.GOARCH)
	fmt.Println("Protocol Versions:", gossip.ProtocolVersions)
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("Operating System:", runtime.GOOS)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", os.Getenv("GOROOT"))
	return nil
}
