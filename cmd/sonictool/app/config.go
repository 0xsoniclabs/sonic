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

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"gopkg.in/urfave/cli.v1"
)

func checkConfig(ctx *cli.Context) error {
	if len(ctx.Args()) < 1 {
		return fmt.Errorf("this command requires an argument - the config toml file")
	}
	configFile := ctx.Args().Get(0)
	_, err := config.MakeAllConfigsFromFile(ctx, configFile)
	return err
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) (err error) {
	cfg, err := config.MakeAllConfigs(ctx)
	if err != nil {
		return err
	}
	comment := ""

	out, err := config.TomlSettings.Marshal(&cfg)
	if err != nil {
		return err
	}

	dump := os.Stdout
	if ctx.NArg() > 0 {
		dump, err = os.OpenFile(ctx.Args().Get(0), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer caution.CloseAndReportError(&err, dump, "failed to close config file")
	}
	_, err = dump.WriteString(comment)
	if err != nil {
		return err
	}
	_, err = dump.Write(out)
	return err
}
