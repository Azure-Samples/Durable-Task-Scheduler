// Package sample shares connection setup and CLI helpers across samples.
package sample

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/client"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/task"
)

const DefaultConnectionString = "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None"

func Main(name string, run func(context.Context) error) {
	timeout := flag.Duration("timeout", 2*time.Minute, "Maximum sample runtime")
	flag.Parse()
	if *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "timeout must be positive")
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
		os.Exit(1)
	}
}

func Options() (*dts.Options, error) {
	connectionString, err := connectionString(os.Getenv)
	if err != nil {
		return nil, err
	}
	return dts.NewOptionsFromConnectionString(connectionString)
}

func connectionString(getenv func(string) string) (string, error) {
	if value := strings.TrimSpace(getenv("DTS_CONNECTION_STRING")); value != "" {
		return value, nil
	}
	endpoint := strings.TrimSpace(getenv("ENDPOINT"))
	if endpoint == "" {
		endpoint = "http://localhost:8080"
	}
	if !strings.Contains(endpoint, "://") {
		address, err := url.Parse("//" + endpoint)
		if err != nil {
			return "", fmt.Errorf("parse ENDPOINT: %w", err)
		}
		scheme := "https"
		if isLoopback(address.Hostname()) {
			scheme = "http"
		}
		endpoint = scheme + "://" + endpoint
	}
	address, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse ENDPOINT: %w", err)
	}
	hub := strings.TrimSpace(getenv("TASKHUB"))
	if hub == "" {
		hub = "default"
	}
	auth := strings.TrimSpace(getenv("DTS_AUTHENTICATION"))
	if auth == "" {
		auth = "DefaultAzure"
		if address.Scheme == "http" && isLoopback(address.Hostname()) {
			auth = "None"
		}
	}
	return fmt.Sprintf("Endpoint=%s;TaskHub=%s;Authentication=%s", endpoint, hub, auth), nil
}

func isLoopback(host string) bool {
	return strings.EqualFold(host, "localhost") || net.ParseIP(host).IsLoopback()
}

func Logger() api.Logger {
	return api.NewSlogLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})))
}

type Host struct {
	Client *dts.Client
	Worker *client.TaskHubGrpcWorker
}

func Start(ctx context.Context, registry *task.TaskRegistry, options *dts.Options, workerOptions ...client.TaskHubGrpcWorkerOption) (*Host, error) {
	return StartWithWorkerContext(ctx, ctx, registry, options, workerOptions...)
}

// StartWithWorkerContext keeps a worker alive for cleanup that itself needs
// durable execution, while connection setup still honors the caller's context.
func StartWithWorkerContext(ctx, workerCtx context.Context, registry *task.TaskRegistry, options *dts.Options, workerOptions ...client.TaskHubGrpcWorkerOption) (*Host, error) {
	if options == nil {
		var err error
		options, err = Options()
		if err != nil {
			return nil, err
		}
	}
	logger := Logger()
	c, err := dts.NewClient(ctx, options, logger)
	if err != nil {
		return nil, fmt.Errorf("connect to DTS: %w", err)
	}
	workerOptions = append([]client.TaskHubGrpcWorkerOption{client.WithAutoWorkItemFilters()}, workerOptions...)
	worker, err := dts.NewWorker(options, registry, logger, workerOptions...)
	if err != nil {
		return nil, errors.Join(err, c.Close())
	}
	if err := worker.Start(workerCtx); err != nil {
		return nil, errors.Join(err, c.Close())
	}
	return &Host{Client: c, Worker: worker}, nil
}

func (h *Host) Close() error {
	// The run context may have expired; draining requires a fresh deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return errors.Join(h.Worker.Shutdown(ctx), h.Client.Close())
}

func WithHost(ctx context.Context, registry *task.TaskRegistry, run func(context.Context, *dts.Client) error) (err error) {
	host, err := Start(ctx, registry, nil)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, host.Close()) }()
	return run(ctx, host.Client)
}

func ID(prefix string) api.InstanceID {
	return api.InstanceID("go-" + prefix + "-" + strings.ToLower(rand.Text()))
}

type completionClient interface {
	WaitForOrchestrationCompletion(context.Context, api.InstanceID, ...api.FetchOrchestrationMetadataOptions) (*api.OrchestrationMetadata, error)
}

func Wait(ctx context.Context, c completionClient, id api.InstanceID, output any) error {
	metadata, err := c.WaitForOrchestrationCompletion(ctx, id, api.WithFetchPayloads(true))
	if err != nil {
		return fmt.Errorf("wait for %s: %w", id, err)
	}
	if metadata == nil {
		return fmt.Errorf("wait for %s returned no metadata", id)
	}
	if metadata.RuntimeStatus != api.RUNTIME_STATUS_COMPLETED {
		return fmt.Errorf("%s ended with status %s: %+v", id, metadata.RuntimeStatus, metadata.FailureDetails)
	}
	if output != nil {
		if err := metadata.ReadOutput(output); err != nil {
			return fmt.Errorf("read output of %s: %w", id, err)
		}
	}
	return nil
}

func Until(ctx context.Context, interval time.Duration, condition func() (bool, error)) error {
	if interval <= 0 {
		return errors.New("poll interval must be positive")
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		done, err := condition()
		if err != nil || done {
			return err
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func PrintJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
