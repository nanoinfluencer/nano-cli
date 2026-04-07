package main

import (
	"os"

	"github.com/nanoinfluencer/nano-cli/cmd/nanoinf"
)

var version = "dev"

func main() {
	cmd := nanoinf.NewRootCommand()
	cmd.Version = version
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
