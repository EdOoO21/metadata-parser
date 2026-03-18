package main

import (
	"fmt"
	"os"

	"github.com/EdOoO21/metadata-parser/internal/infrastructure/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
