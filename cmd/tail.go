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
The following examples demonstrate common usage patterns for tail.

# Stream all logs in real time
cloudtail tail projectID --follow

# Stream logs from a specific resource type
cloudtail tail projectID --resource-type=gce_instance --follow

# Stream only ERROR severity logs
cloudtail tail projectID --severity=ERROR --follow

# Display the most recent 100 log entries
cloudtail tail projectID --limit=100

# Display logs from the last 30 minutes
cloudtail tail projectID --since=30m

# Display logs newer than a specific point in time
cloudtail tail projectID --since-time=2026-02-12T12:30:00Z

# Filter logs by log name and resource type
cloudtail tail projectID \
	--log-name=projects/projectID/logs/cloudbuild \
	--resource-type=k8s_container

# Combine severity and time-based filtering
cloudtail tail projectID --severity=WARNING --since=1h

# Use an advanced filter expression for complex queries
cloudtail tail projectID \
	--filter='severity>="ERROR" AND timestamp>="2026-01-01T00:00:00Z" AND timestamp<="2023-01-31T12:00:00Z"'

# Combine advanced filtering with a result limit
cloudtail tail projectID \
	--filter='severity>="ERROR" AND timestamp>="2026-01-01T00:00:00Z" AND timestamp<="2023-01-31T12:00:00Z"' \
	--limit=100

# Stream logs using an advanced filter expression
cloudtail tail projectID --filter='severity>="CRITICAL"' --follow

# Write log output to a file instead of stdout
cloudtail tail projectID --severity=INFO --output=logs.txt

# Stream logs and write them to a file
cloudtail tail projectID --follow --output=logs.txt

# Combine multiple filters for a focused query
cloudtail tail projectID \
	--log-name=projects/projectID/logs/cloudbuild \
	--resource-type=k8s_container \
	--severity=ERROR \
	--since=15m

# Retrieve recent logs using a fixed timestamp and save them
cloudtail tail projectID \
	--since-time=2026-01-13T12:30:00Z \
	--limit=200 \
	--output=incident.log
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
		"DEFAULT":   {},
		"DEBUG":     {},
		"INFO":      {},
		"NOTICE":    {},
		"WARNING":   {},
		"ERROR":     {},
		"CRITICAL":  {},
		"ALERT":     {},
		"EMERGENCY": {},
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

	tailCmd.Flags().String("log-name", "", "Filter logs by log name")
	tailCmd.Flags().String("resource-type", "", "Filter logs by resource type")
	tailCmd.Flags().String("severity", "", "Filter logs by severity level (DEFAULT, DEBUG, INFO, NOTICE, WARNING, ERROR, CRITICAL, ALERT, EMERGENCY)")
	tailCmd.Flags().String("since", "", "Show logs newer than a relative duration (e.g. 1h, 30m, 20s, 1h15m30s). Only one of since-time / since may be used")
	tailCmd.Flags().String("since-time", "", "Show logs newer than an RFC3339 timestamp (e.g. 2026-01-13T12:30:00Z). Only one of since-time / since may be used")
	tailCmd.Flags().String("filter", "", `Apply a raw filter expression for advanced queries (e.g. severity>="WARNING" AND severity<="ERROR" AND timestamp>="2026-05-18T12:00:00Z")`)

	tailCmd.MarkFlagsMutuallyExclusive("since", "since-time")

	tailCmd.Flags().Bool("follow", false, "Stream logs in real time")
	tailCmd.Flags().Int("limit", -1, "Maximum number of logs to display (defaults to -1, showing all logs).")
	tailCmd.Flags().String("output", "", "Write logs to the specified file (defaults to stdout).")
}
