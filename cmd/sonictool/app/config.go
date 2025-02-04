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
