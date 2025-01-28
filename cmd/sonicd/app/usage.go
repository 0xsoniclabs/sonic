// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// Contains the geth command usage template and generator.

package app

import (
	"github.com/0xsoniclabs/sonic/cmd/sonicd/cmdhelper"
	"io"
	"sort"

	cli "gopkg.in/urfave/cli.v1"

	"github.com/0xsoniclabs/sonic/debug"
)

// AppHelpFlagGroups is the application flags, grouped by functionality.
var AppHelpFlagGroups = calcAppHelpFlagGroups()

func calcAppHelpFlagGroups() []cmdhelper.FlagGroup {
	initFlags()
	return []cmdhelper.FlagGroup{
		{
			Name:  "OPERA",
			Flags: operaFlags,
		},
		{
			Name:  "TRANSACTION POOL",
			Flags: txpoolFlags,
		},
		{
			Name:  "PERFORMANCE TUNING",
			Flags: performanceFlags,
		},
		{
			Name:  "ACCOUNT",
			Flags: accountFlags,
		},
		{
			Name:  "API",
			Flags: rpcFlags,
		},
		{
			Name:  "NETWORKING",
			Flags: networkingFlags,
		},
		{
			Name:  "GAS PRICE ORACLE",
			Flags: gpoFlags,
		},
		{
			Name:  "METRICS AND STATS",
			Flags: metricsFlags,
		},
		{
			Name:  "TESTING",
			Flags: testFlags,
		},
		{
			Name:  "LOGGING AND DEBUGGING",
			Flags: debug.Flags,
		},
		{
			Name: "MISC",
			Flags: []cli.Flag{
				cli.HelpFlag,
			},
		},
	}
}

func initAppHelp() {
	// Override the default app help template
	cli.AppHelpTemplate = cmdhelper.AppHelpTemplate

	// Override the default app help printer, but only for the global app help
	originalHelpPrinter := cli.HelpPrinter
	cli.HelpPrinter = func(w io.Writer, tmpl string, data interface{}) {
		if tmpl == cmdhelper.AppHelpTemplate {
			// Iterate over all the flags and add any uncategorized ones
			categorized := make(map[string]struct{})
			for _, group := range AppHelpFlagGroups {
				for _, flag := range group.Flags {
					categorized[flag.String()] = struct{}{}
				}
			}
			var uncategorized []cli.Flag
			for _, flag := range data.(*cli.App).Flags {
				if _, ok := categorized[flag.String()]; !ok {
					uncategorized = append(uncategorized, flag)
				}
			}
			if len(uncategorized) > 0 {
				// Append all uncategorized options to the misc group
				miscs := len(AppHelpFlagGroups[len(AppHelpFlagGroups)-1].Flags)
				AppHelpFlagGroups[len(AppHelpFlagGroups)-1].Flags = append(AppHelpFlagGroups[len(AppHelpFlagGroups)-1].Flags, uncategorized...)

				// Make sure they are removed afterwards
				defer func() {
					AppHelpFlagGroups[len(AppHelpFlagGroups)-1].Flags = AppHelpFlagGroups[len(AppHelpFlagGroups)-1].Flags[:miscs]
				}()
			}
			// Render out custom usage screen
			originalHelpPrinter(w, tmpl, cmdhelper.HelpData{App: data, FlagGroups: AppHelpFlagGroups})
		} else if tmpl == cmdhelper.CommandHelpTemplate {
			// Iterate over all command specific flags and categorize them
			categorized := make(map[string][]cli.Flag)
			for _, flag := range data.(cli.Command).Flags {
				if _, ok := categorized[flag.String()]; !ok {
					categorized[cmdhelper.FlagCategory(flag, AppHelpFlagGroups)] = append(categorized[cmdhelper.FlagCategory(flag, AppHelpFlagGroups)], flag)
				}
			}

			// sort to get a stable ordering
			sorted := make([]cmdhelper.FlagGroup, 0, len(categorized))
			for cat, flgs := range categorized {
				sorted = append(sorted, cmdhelper.FlagGroup{Name: cat, Flags: flgs})
			}
			sort.Sort(cmdhelper.ByCategory(sorted))

			// add sorted array to data and render with default printer
			originalHelpPrinter(w, tmpl, map[string]interface{}{
				"cmd":              data,
				"categorizedFlags": sorted,
			})
		} else {
			originalHelpPrinter(w, tmpl, data)
		}
	}
}
