package main

import (
	"os"

	"github.com/auxence-m/cloudtail/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	cmd.Execute()

	rootCmd := cmd.Root()

	// Add doc generation
	err := doc.GenMarkdownTree(rootCmd, "./docs")
	if err != nil {
		os.Exit(1)
	}

}
