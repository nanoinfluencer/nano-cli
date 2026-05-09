package main

import (
	"fmt"
	"io"
	"os"

	"github.com/nanoinfluencer/nano-cli/cmd/nanoinf"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	cmd := nanoinf.NewRootCommand()
	cmd.Version = version
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
