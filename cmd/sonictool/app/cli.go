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
	"strings"

	"github.com/0xsoniclabs/sonic/config/flags"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/rpc"
	"gopkg.in/urfave/cli.v1"
)

var (
	JSpathFlag = cli.StringFlag{
		Name:  "jspath",
		Usage: "JavaScript root path for `loadScript`",
		Value: ".",
	}
	PreloadJSFlag = cli.StringFlag{
		Name:  "preload",
		Usage: "Comma separated list of JavaScript files to preload into the console",
	}
	ExecFlag = cli.StringFlag{
		Name:  "exec",
		Usage: "Execute JavaScript statement",
	}
)

// remoteConsole will connect to a remote sonic instance, attaching a JavaScript
// console to it.
func remoteConsole(ctx *cli.Context) (err error) {
	// Attach to a remotely running sonic instance and start the JavaScript console
	endpoint := ctx.Args().First()
	if endpoint == "" {
		if !ctx.GlobalIsSet(flags.DataDirFlag.Name) {
			return fmt.Errorf("the --%s flag is missing and the IPC endpoint path is not specified", flags.DataDirFlag.Name)
		}
		endpoint = fmt.Sprintf("%s/sonic.ipc", ctx.GlobalString(flags.DataDirFlag.Name))
	}
	client, err := rpc.Dial(endpoint)
	if err != nil {
		return fmt.Errorf("unable to attach to the node: %v", err)
	}
	defer client.Close()

	if !ctx.GlobalIsSet(flags.DataDirFlag.Name) {
		return fmt.Errorf("please specify the --%s flag to a directory, where should be the console history stored", flags.DataDirFlag.Name)
	}

	config := console.Config{
		DataDir: ctx.GlobalString(flags.DataDirFlag.Name), // console history will be stored here
		DocRoot: ctx.String(JSpathFlag.Name),              // from where to load scripts
		Client:  client,
		Preload: makeConsolePreloads(ctx.String(PreloadJSFlag.Name)),
	}

	console, err := console.New(config)
	if err != nil {
		return fmt.Errorf("failed to start the JavaScript console: %v", err)
	}
	defer caution.ExecuteAndReportError(&err,
		func() error { return console.Stop(false) },
		"failed to stop the JavaScript console")

	if script := ctx.String(ExecFlag.Name); script != "" {
		console.Evaluate(script)
		return nil
	}

	// Otherwise print the welcome screen and enter interactive mode
	console.Welcome()
	console.Interactive()

	return nil
}

// makeConsolePreloads retrieves the absolute paths for the console JavaScript
// scripts to preload before starting.
func makeConsolePreloads(preloadsStr string) []string {
	if preloadsStr == "" {
		return nil
	}
	var preloads []string
	for _, file := range strings.Split(preloadsStr, ",") {
		preloads = append(preloads, strings.TrimSpace(file))
	}
	return preloads
}
