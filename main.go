package main

import (
	"fmt"
	"os"

	"github.com/matthewfritsch/claudehopper/cmd"
)

// Version information — set via ldflags at build time:
//
//	-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
var version = "dev"
var commit = "none"
var date = "unknown"

func main() {
	cmd.SetVersionInfo(version, commit, date)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
