package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/auxence-m/cloudtail/stream"
	"github.com/spf13/cobra"
)

type Options struct {
	LogName      string
	ResourceType string
	Severity     string
	Since        string
	SinceTime    string
	CustomFilter string
	Follow       bool
	Limit        int
	Output       string
}

// tailCmd represents the tail command
var tailCmd = &cobra.Command{
	Use:          "tail [projectID]",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	Short:        "Stream Google Cloud Logging entries directly into the terminal in real time",
	Long:         `The tail command will fetch and stream all Google Cloud Logging entries from the last 24 hours by default unless specified otherwise with the available flags`,
	Example: `
# List logs from the "cloudbuild" log within the last hour 
cloudtail tail projectID --log-name=projects/projectID/logs/cloudbuild --since=1h

# List the 50 most recent logs associated with the "cloud_run_revision" label
cloudtail tail projectID --resource-type=cloud_run_revision --limit=50

# List all the logs with a severity of "WARNING" starting from July 31, 2025, at 06:00:00 UTC
cloudtail tail projectID --severity=WARNING --since-time=2025-07-30T06:00:00Z

# Begin streaming the logs from the "log-name" and save them in the "output.log" file
cloudtail tail projectID --log-name=projects/projectID/logs/log-name --follow --output=output.log

# List logs using a more custom or complex log filter expression
cloudtail tail projectID --filter="severity >= "ERROR" AND resource.type = "k8s_cluster" AND timestamp >= "2023-01-01T00:00:00Z" AND timestamp <= "2023-01-31T23:59:59Z""
`,
	RunE: tailRun,
}

func tailRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing required argument: projectID")
	}

	flags := cmd.Flags()

	options := Options{}

	// Read flags
	options.LogName, _ = flags.GetString("log-name")
	options.ResourceType, _ = flags.GetString("resource-type")
	options.Severity, _ = flags.GetString("severity")
	options.Since, _ = flags.GetString("since")
	options.SinceTime, _ = flags.GetString("since-time")
	options.Follow, _ = flags.GetBool("follow")
	options.Limit, _ = flags.GetInt("limit")
	options.Output, _ = flags.GetString("output")
	options.CustomFilter, _ = flags.GetString("filter")

	projectID := args[0]

	return fetchAndTailLogs(options, projectID)
}

// validateSeverityFlag ensures the --severity flag has a valid value
func validateSeverityFlag(severity string) (string, error) {
	upper := strings.ToUpper(severity)

	validSeverities := map[string]struct{}{
		"INFO":    {},
		"DEBUG":   {},
		"WARNING": {},
		"NOTICE":  {},
		"ERROR":   {},
	}

	_, found := validSeverities[upper]
	if !found {
		return "", fmt.Errorf("invalid value for --severity flag: %q. (valid values: INFO, WARNING, ERROR, etc.)", severity)
	}

	return upper, nil
}

// validateSinceFlag validates that the --since flag is a duartion (e.g. "1h", "30m", or "20s") and converts it into a time.Duration.
func validateSinceFlag(since string) (time.Duration, error) {
	parseDuration, err := time.ParseDuration(since)
	if err != nil {
		return 0, fmt.Errorf("invalid value for --since flag: %q (valid values: 1h, 30m, 20s, 1h15m30s, etc.): \n%w", since, err)
	}

	if parseDuration < 0 {
		return 0, fmt.Errorf("the --since flag duration must be positive (got %q)", since)
	}

	return parseDuration, nil
}

// validateSinceTimeFlag validates that the --since-time flag is a valid RFC3339 timestamp (e.g. 2024-01-09T10:30:00Z).
func validateSinceTimeFlag(sinceTime string) (time.Time, error) {
	parsedTime, err := time.Parse(time.RFC3339, sinceTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid value for --sinceTime flag: %q (must be RFC3339 format): \n%w", sinceTime, err)
	}

	return parsedTime, nil
}

func fetchAndTailLogs(options Options, projectID string) error {
	var (
		parseDuration time.Duration
		parseTime     time.Time
		parseSeverity string
		err           error
	)

	// Trim options
	logName := strings.TrimSpace(options.LogName)
	resourceType := strings.TrimSpace(options.ResourceType)
	severity := strings.TrimSpace(options.Severity)
	since := strings.TrimSpace(options.Since)
	sinceTime := strings.TrimSpace(options.SinceTime)
	output := strings.TrimSpace(options.Output)
	customFilter := strings.TrimSpace(options.CustomFilter)

	// Validate severity flag
	if severity != "" {
		parseSeverity, err = validateSeverityFlag(severity)
		if err != nil {
			return err
		}
	}

	// Validate since flag
	if since != "" {
		parseDuration, err = validateSinceFlag(since)
		if err != nil {
			return err
		}
	}

	// Validate sinceTime flag
	if sinceTime != "" {
		parseTime, err = validateSinceTimeFlag(sinceTime)
		if err != nil {
			return err
		}
	}

	// Validate limit flag, make sure default value (-1) is ingnored
	if options.Limit != -1 && options.Limit < 0 {
		return fmt.Errorf("invalid value for --limit flag: %d. (must be positive)", options.Limit)

	}

	// Build filter object
	filter := stream.Filter{
		LogName:      logName,
		ResourceType: resourceType,
		Severity:     parseSeverity,
		Since:        parseDuration,
		SinceTime:    parseTime,
		CustomFilter: customFilter,
	}
	filterStr := stream.BuildFilterString(&filter)
	//fmt.Println(filterStr)

	// Set proper output
	out := os.Stdout
	if output != "" {
		file, err := os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("could not open output file: \n%w", err)
		}
		defer file.Close()
		out = file
	}

	// Fetch logs
	err = stream.GetEntries(out, projectID, filterStr, options.Limit)
	if err != nil {
		return fmt.Errorf("error fetching log entries \n%w", err)
	}

	// Tail logs if --follow is set
	if options.Follow {
		err = stream.TailLogs(out, projectID, filterStr, options.Limit)
		if err != nil {
			return fmt.Errorf("error tailing log entries \n%w", err)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(tailCmd)

	tailCmd.Flags().String("log-name", "", "Retrives the logs with the specified log name")
	tailCmd.Flags().String("resource-type", "", "Retrives the logs with the specified resource type")
	tailCmd.Flags().String("severity", "", "Retrives the logs with the specified severity level. (e.g., INFO, WARNING, ERROR)")
	tailCmd.Flags().String("since", "", "Retrieves logs newer than a specified relative duration (e.g., 1h, 30m, 20s, 1h15m30s). Only one of --since-time or --since may be used")
	tailCmd.Flags().String("since-time", "", "Retrieves logs newer than a specific timestamp in RFC3339 format (e.g., YYYY-MM-DDTHH:MM:SSZ). Only one of --since-time or --since may be used")
	tailCmd.Flags().String("filter", "", `Retrieves logs with the specified filter. This flag allows the use of raw filter for more customized or complex querries (e.g., severity >= "ERROR" AND severity <= "EMERGENCY" AND resource.type="gce_instance" AND timestamp >= "2025-12-18T12:00:00Z")`)

	tailCmd.MarkFlagsMutuallyExclusive("since", "since-time")

	tailCmd.Flags().Bool("follow", false, "Specify if the logs should be streamed in real-time as they are generated")
	tailCmd.Flags().Int("limit", -1, "Number of recent logs to display. Defaults to -1 with no effect, showing all logs")
	tailCmd.Flags().String("output", "", "Specify the output file to write the logs to")
}
