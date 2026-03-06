package main

import (
	"os"

	"github.com/anishalle/hack/internal/cli"
)

var version = "dev"

func main() {
	os.Exit(cli.Execute(version))
}
