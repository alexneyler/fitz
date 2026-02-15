package main

import (
	"fmt"
	"os"

	"fitz/internal/cli"
)

var version = "dev"

func main() {
	cli.Version = version
	if err := cli.Execute(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
