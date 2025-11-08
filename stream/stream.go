package stream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"cloud.google.com/go/logging"
	loggingv2 "cloud.google.com/go/logging/apiv2"
	"cloud.google.com/go/logging/apiv2/loggingpb"
	"cloud.google.com/go/logging/logadmin"
	"google.golang.org/api/iterator"
)

func printLogEntry(out io.Writer, entry *logging.Entry) error {
	timestamp := entry.Timestamp.Format(time.RFC3339)
	severity := formatSeverity(entry.Severity.String())
	resourceType := entry.Resource.Type

	if req := entry.HTTPRequest; req != nil {
		reqUrl := ""
		if req.Request != nil {
			reqUrl = req.Request.URL.String()
		}
		_, err := fmt.Fprintf(out, "[%v] [%s] (%s) %s %s %d %dms\n", timestamp, severity, resourceType, req.Request.Method, reqUrl, req.Status, req.Latency.Milliseconds())
		if err != nil {
			return fmt.Errorf("failed to write to output: %w", err)
		}
	}

	if payload, ok := entry.Payload.(string); ok {
		trimmed := strings.TrimSpace(payload)
		_, err := fmt.Fprintf(out, "[%v] [%s] (%s) %s\n", timestamp, severity, resourceType, trimmed)
		if err != nil {
			return fmt.Errorf("failed to write to output: %w", err)
		}
	}

	return nil
}

func printTailLogEntry(out io.Writer, entry *loggingpb.LogEntry) error {
	timestamp := entry.Timestamp.AsTime().Format(time.RFC3339)
	severity := formatSeverity(entry.Severity.String())
	resourceType := entry.Resource.Type

	if req := entry.HttpRequest; req != nil {
		_, err := fmt.Fprintf(out, "[%v] [%s] (%s) %s %s %d %dms\n", timestamp, severity, resourceType, req.RequestMethod, req.RequestUrl, req.Status, req.Latency.AsDuration().Milliseconds())
		if err != nil {
			return fmt.Errorf("failed to write to output: %w", err)
		}
	}

	if payload := entry.GetTextPayload(); payload != "" {
		trimmed := strings.TrimSpace(payload)
		_, err := fmt.Fprintf(out, "[%v] [%s] (%s) %s\n", timestamp, severity, resourceType, trimmed)
		if err != nil {
			return fmt.Errorf("failed to write to output: %w", err)
		}
	}

	return nil
}

// getEntries fetches and list log entries according to a filter
func getEntries(out io.Writer, projectID string, filter string, maxLogs int) error {
	ctx := context.Background()
	adminClient, err := logadmin.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to create logadmin client: %w", err)
	}
	defer adminClient.Close()

	options := []logadmin.EntriesOption{logadmin.Filter(filter)}
	if maxLogs > 0 {
		options = append(options, logadmin.NewestFirst())
	}

	iter := adminClient.Entries(ctx, options...)

	counter := 0
	for {
		if maxLogs > 0 && counter >= maxLogs {
			break
		}

		entry, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}

		// Print log entries
		err = printLogEntry(out, entry)
		if err != nil {
			return err
		}

		counter++
	}

	return nil
}

// tailLogs fetches and tail live log entries according to a filter
func tailLogs(out io.Writer, projectID string, filter string, maxLogs int) error {
	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up channel to catch OS signals (like Ctrl+C)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signalChan)

	go func() {
		<-signalChan
		fmt.Println("\nReceived an interrupt signal, stopping stream...")
		cancel() // stop receiving logs
	}()

	client, err := loggingv2.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("NewClient error: %w", err)
	}
	defer client.Close()

	stream, err := client.TailLogEntries(ctx)
	if err != nil {
		return fmt.Errorf("TailLogEntries error: %w", err)
	}
	defer stream.CloseSend()

	req := &loggingpb.TailLogEntriesRequest{
		ResourceNames: []string{"projects/" + projectID},
		Filter:        filter,
	}

	if err := stream.Send(req); err != nil {
		return fmt.Errorf("stream.Send error: %w", err)
	}

	counter := 0
	for {
		// Respect context cancellation
		if ctx.Err() != nil {
			fmt.Fprintln(out, "Streaming stopped successfully")
			return nil
		}

		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("stream.Recv error: %w", err)
		}

		entries := resp.GetEntries()
		if len(entries) == 0 {
			continue
		}

		for _, entry := range entries {
			err = printTailLogEntry(out, entry)
			if err != nil {
				return err
			}
		}

		counter += len(resp.GetEntries())
		if maxLogs > 0 && counter >= maxLogs {
			break
		}

	}

	return nil
}
