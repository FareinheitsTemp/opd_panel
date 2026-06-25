package main

import (
	"os"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/delivery/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
