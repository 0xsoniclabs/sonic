package app

import (
	"os"

	"gopkg.in/urfave/cli.v1"
)

// Run starts sonicd with the regular command line arguments.
func Run() error {
	return RunWithArgs(os.Args, nil)
}

type AppControl struct {
	NodeIdAnnouncement   chan<- string
	HttpPortAnnouncement chan<- string
	Shutdown             <-chan struct{}
}

// RunWithArgs starts sonicd with the given command line arguments.
// An optional httpPortAnnouncement channel can be provided to announce the HTTP
// port used by the HTTP server of the started sonicd node. The channel is
// closed when the process stops.
// Another optional stop channel can be provided. By sending a message through
// this channel, or closing it, the shutdown of the process is triggered.
func RunWithArgs(
	args []string,
	control *AppControl,
) error {
	app := initApp()
	initAppHelp()

	// If present, take ownership and inject the control struct into the action.
	if control != nil {
		if control.NodeIdAnnouncement != nil {
			defer close(control.NodeIdAnnouncement)
		}
		if control.HttpPortAnnouncement != nil {
			defer close(control.HttpPortAnnouncement)
		}
		app.Action = func(ctx *cli.Context) error {
			return lachesisMainInternal(
				ctx,
				control,
			)
		}
	}

	return app.Run(args)
}
