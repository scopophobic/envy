package main

import (
	"os"

	"github.com/envo/cli/internal/commands"
)

func main() {
	os.Exit(commands.Execute())
}

