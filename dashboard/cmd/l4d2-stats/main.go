package main

import (
	"fmt"
	"os"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
