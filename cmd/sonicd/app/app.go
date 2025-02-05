package app

import (
	"os"

	"gopkg.in/urfave/cli.v1"
)

// Run starts sonicd with the regular command line arguments.
func Run() error {
	return RunWithArgs(os.Args, nil)
}

// RunWithArgs starts sonicd with the given command line arguments.
// An optional httpPortAnnouncement channel can be provided to announce the HTTP port
// used by the HTTP server of the started sonicd node. The channel is closed when the
// when the process stops.
func RunWithArgs(
	args []string,
	httpPortAnnouncement chan<- string,
) error {
	app := initApp()
	initAppHelp()

	// If present, inject the http port announcement channel into the action.
	if httpPortAnnouncement != nil {
		defer close(httpPortAnnouncement)
		app.Action = func(ctx *cli.Context) error {
			return lachesisMainInternal(ctx, httpPortAnnouncement)
		}
	}

	return app.Run(args)
}
