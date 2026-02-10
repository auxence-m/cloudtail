package cmd

import (
	"context"
	"os"

	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

func defaultColorScheme(c lipgloss.LightDarkFunc) fang.ColorScheme {
	base := c(lipgloss.Black, lipgloss.White)
	return fang.ColorScheme{
		Base:           base,
		Title:          lipgloss.BrightMagenta,
		Description:    base, // flag and command descriptions
		Codeblock:      lipgloss.Color("238"),
		Program:        lipgloss.Color("#00005f"),
		DimmedArgument: c(lipgloss.BrightWhite, lipgloss.BrightBlack),
		Comment:        c(lipgloss.BrightWhite, lipgloss.BrightBlack),
		Flag:           lipgloss.Color("#5faf5f"),
		FlagDefault:    c(lipgloss.BrightWhite, lipgloss.BrightBlack), // flag default values in descriptions
		Command:        lipgloss.Color("#ffff00"),
		QuotedString:   base,
		Argument:       base,
		ErrorHeader:    [2]color.Color{lipgloss.Black, lipgloss.Red},
		ErrorDetails:   lipgloss.Red,
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "cloudtail",
	Version: "1.0",
	Short:   "cloudtail displays or tail logs from Google Cloud Logging",
	Long: `cloudtail is a lightweight cloud-native command-line tool written in Golang that allows users to view or tail logs from Google Cloud Logging (similar to kubectl logs).

It connects to the Google Cloud Logging API, fetches logs for a specific project based on filters like severity, resource type, or time range. 

It displays the logs or continuously streams them to the terminal or to an output file in near real-time.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Add charmbracelet/fang
	// charmbracelet/fang calls root.ExecuteContext()
	err := fang.Execute(context.Background(), rootCmd,
		fang.WithVersion(rootCmd.Version), fang.WithoutManpage(), fang.WithNotifySignal(os.Interrupt, os.Kill), fang.WithColorSchemeFunc(defaultColorScheme))

	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "help message for toggle")
}
