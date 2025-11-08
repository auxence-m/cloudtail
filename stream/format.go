package stream

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

type Filter struct {
	Severity     string
	Since        time.Duration
	SinceTime    time.Time
	ResourceType string
	LogName      string
}

type Options struct {
	Follow bool
	Tail   int
	Output string
}

func formatFilter(filter *Filter) string {
	var options []string

	if filter == nil {
		return ""
	}

	if filter.Severity != "" {
		options = append(options, fmt.Sprintf(`severity = "%s"`, filter.Severity))
	}

	if filter.Since != 0 {
		sinceTime := time.Now().Add(-filter.Since).Format(time.RFC3339)
		options = append(options, fmt.Sprintf(`timestamp >= "%s"`, sinceTime))
	}

	if !filter.SinceTime.IsZero() {
		options = append(options, fmt.Sprintf(`timestamp >= "%s"`, filter.SinceTime.Format(time.RFC3339)))
	}

	if filter.ResourceType != "" {
		options = append(options, fmt.Sprintf(`resource.type = "%s"`, filter.ResourceType))
	}

	if filter.LogName != "" {
		options = append(options, fmt.Sprintf(`logName = "%s"`, filter.LogName))
	}

	return strings.Join(options, " AND ")
}

func formatSeverity(severity string) string {
	colors := map[string]string{
		"INFO":    "\033[34m", // Blue
		"DEBUG":   "\033[34m", // Blue
		"WARNING": "\033[33m", // Yellow
		"NOTICE":  "\033[34m", // Blue
		"ERROR":   "\033[31m", // Red
	}

	upper := strings.ToUpper(severity)

	// Check if stdout is a terminal
	// If not, return severity without color
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return upper
	}

	color, ok := colors[upper]
	if ok {
		return fmt.Sprintf("%s%s\033[0m", color, upper)
	}

	return upper
}
