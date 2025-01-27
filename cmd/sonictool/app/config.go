package app

import (
	"fmt"
	"github.com/0xsoniclabs/sonic/config"
	"gopkg.in/urfave/cli.v1"
	"os"
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
func dumpConfig(ctx *cli.Context) error {
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
		defer dump.Close()
	}
	dump.WriteString(comment)
	dump.Write(out)

	return nil
}
