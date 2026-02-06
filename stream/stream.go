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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
			return fmt.Errorf("failed to write to output: \n%w", err)
		}
	}

	if payload, ok := entry.Payload.(string); ok {
		trimmed := strings.TrimSpace(payload)
		_, err := fmt.Fprintf(out, "[%v] [%s] (%s) %s\n", timestamp, severity, resourceType, trimmed)
		if err != nil {
			return fmt.Errorf("failed to write to output: \n%w", err)
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
			return fmt.Errorf("failed to write to output: \n%w", err)
		}
	}

	if payload := entry.GetTextPayload(); payload != "" {
		trimmed := strings.TrimSpace(payload)
		_, err := fmt.Fprintf(out, "[%v] [%s] (%s) %s\n", timestamp, severity, resourceType, trimmed)
		if err != nil {
			return fmt.Errorf("failed to write to output: \n%w", err)
		}
	}

	return nil
}

// GetEntries fetches and list log entries according to a filter
func GetEntries(out io.Writer, projectID string, filter string, limit int) error {
	ctx := context.Background()
	adminClient, err := logadmin.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to create logadmin client: \n%w", err)
	}
	defer adminClient.Close()

	options := []logadmin.EntriesOption{logadmin.Filter(filter)}
	if limit > 0 {
		options = append(options, logadmin.NewestFirst())
	}

	iter := adminClient.Entries(ctx, options...)

	counter := 0
	for {
		if limit > 0 && counter >= limit {
			break
		}

		entry, err := iter.Next()
		if err != nil {
			// No more log entries
			if errors.Is(err, iterator.Done) {

				if counter == 0 {
					fmt.Fprintln(os.Stderr, "No entries found.")
				}

				break
			}

			// Unexpected error
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

// TailLogs fetches and tail live log entries according to a filter
func TailLogs(out io.Writer, projectID string, filter string, limit int) error {
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
		return fmt.Errorf("NewClient error: \n%w", err)
	}
	defer client.Close()

	stream, err := client.TailLogEntries(ctx)
	if err != nil {
		return fmt.Errorf("TailLogEntries error: \n%w", err)
	}
	defer stream.CloseSend()

	req := &loggingpb.TailLogEntriesRequest{
		ResourceNames: []string{"projects/" + projectID},
		Filter:        filter,
	}

	if err := stream.Send(req); err != nil {
		return fmt.Errorf("stream.Send error: \n%w", err)
	}

	counter := 0
	for {
		resp, err := stream.Recv()
		if err != nil {
			// Respect context cancellation
			if status.Code(err) == codes.Canceled {
				fmt.Fprintln(out, "Streaming stopped successfully")
				break
			}

			// Stream is closed normally
			if errors.Is(err, io.EOF) {
				break
			}

			// Unexpected error
			return fmt.Errorf("stream.Recv error: \n%w", err)
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
		if limit > 0 && counter >= limit {
			break
		}

	}

	return nil
}
