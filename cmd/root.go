package cmd

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "cloudtail",
	Version: "1.0",
	Short:   "cloudtail - display or tail logs from Google Cloud Logging",
	Long: `cloudtail is a lightweight cloud-native command-line tool written in Golang that allows users to display or tail logs from Google Cloud Logging (similar to Kubectl logs).
It connects to the Google Cloud Logging API, fetches logs for a specific project based on filters (like severity, resource, or time range). 
It displays the logs or continuously streams them to the terminal or to an output file in near real-time.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Add charmbracelet/fang
	// charmbracelet/fang calls root.ExecuteContext()
	err := fang.Execute(context.Background(), rootCmd,
		fang.WithVersion(rootCmd.Version), fang.WithoutManpage(), fang.WithNotifySignal(os.Interrupt, os.Kill))
	if err != nil {
		os.Exit(1)
	}
}

func Root() *cobra.Command {
	return rootCmd
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "help message for toggle")
}
