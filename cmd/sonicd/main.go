package main

import (
	"fmt"
	"os"

	"github.com/0xsoniclabs/sonic/cmd/sonicd/app"
)

func main() {
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
