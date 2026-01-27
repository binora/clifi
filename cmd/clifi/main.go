package main

import (
	"os"

	"github.com/yolodolo42/clifi/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
