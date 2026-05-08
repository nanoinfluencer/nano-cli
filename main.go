package main

import (
	"fmt"
	"os"

	"github.com/nanoinfluencer/nano-cli/cmd/nanoinf"
)

var version = "dev"

func main() {
	cmd := nanoinf.NewRootCommand()
	cmd.Version = version
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
