package sample

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/microsoft/durabletask-go/api"
)

func TestConnectionString(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"default", nil, DefaultConnectionString},
		{"localhost without scheme", map[string]string{"ENDPOINT": "localhost:8080"}, DefaultConnectionString},
		{"loopback", map[string]string{"ENDPOINT": "127.0.0.1:8080"}, "Endpoint=http://127.0.0.1:8080;TaskHub=default;Authentication=None"},
		{"IPv6 loopback", map[string]string{"ENDPOINT": "[::1]:8080"}, "Endpoint=http://[::1]:8080;TaskHub=default;Authentication=None"},
		{"Azure", map[string]string{"ENDPOINT": "example.durabletask.io", "TASKHUB": "go"}, "Endpoint=https://example.durabletask.io;TaskHub=go;Authentication=DefaultAzure"},
		{"CLI", map[string]string{"ENDPOINT": "https://example.durabletask.io", "DTS_AUTHENTICATION": "AzureCLI"}, "Endpoint=https://example.durabletask.io;TaskHub=default;Authentication=AzureCLI"},
		{"remote HTTP does not disable auth", map[string]string{"ENDPOINT": "http://example.com"}, "Endpoint=http://example.com;TaskHub=default;Authentication=DefaultAzure"},
		{"precedence", map[string]string{"DTS_CONNECTION_STRING": DefaultConnectionString, "TASKHUB": "ignored"}, DefaultConnectionString},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := connectionString(func(key string) string { return tt.env[key] })
			if err != nil || got != tt.want {
				t.Fatalf("connectionString = %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestOptionsRejectInvalidConfiguration(t *testing.T) {
	t.Setenv("DTS_CONNECTION_STRING", "Endpoint=http://localhost:8080;TaskHub=default")
	if _, err := Options(); err == nil {
		t.Fatal("expected missing authentication to fail")
	}
}

func TestIDsAreUnique(t *testing.T) {
	first, second := ID("test"), ID("test")
	if first == second || !strings.HasPrefix(string(first), "go-test-") {
		t.Fatalf("invalid IDs: %q, %q", first, second)
	}
}

func TestUntil(t *testing.T) {
	calls := 0
	if err := Until(t.Context(), time.Millisecond, func() (bool, error) {
		calls++
		return calls == 2, nil
	}); err != nil || calls != 2 {
		t.Fatalf("Until: %v, calls=%d", err, calls)
	}
	expected := errors.New("poll failed")
	if err := Until(t.Context(), time.Millisecond, func() (bool, error) {
		return false, expected
	}); !errors.Is(err, expected) {
		t.Fatalf("expected polling error, got %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := Until(ctx, time.Millisecond, func() (bool, error) {
		t.Fatal("condition called after cancellation")
		return false, nil
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

type completedClient struct {
	metadata *api.OrchestrationMetadata
	err      error
}

func (c completedClient) WaitForOrchestrationCompletion(context.Context, api.InstanceID, ...api.FetchOrchestrationMetadataOptions) (*api.OrchestrationMetadata, error) {
	return c.metadata, c.err
}

func TestWaitRequiresSuccessfulCompletion(t *testing.T) {
	for _, status := range []api.OrchestrationStatus{
		api.RUNTIME_STATUS_FAILED,
		api.RUNTIME_STATUS_TERMINATED,
		api.RUNTIME_STATUS_CANCELED,
		api.RUNTIME_STATUS_RUNNING,
	} {
		t.Run(status.String(), func(t *testing.T) {
			client := completedClient{metadata: &api.OrchestrationMetadata{RuntimeStatus: status}}
			if err := Wait(t.Context(), client, "test", nil); err == nil {
				t.Fatalf("accepted %s as successful", status)
			}
		})
	}
}

func TestWaitReadsAndValidatesOutput(t *testing.T) {
	client := completedClient{metadata: &api.OrchestrationMetadata{
		RuntimeStatus:    api.RUNTIME_STATUS_COMPLETED,
		SerializedOutput: `{"value":42}`,
	}}
	var output struct {
		Value int `json:"value"`
	}
	if err := Wait(t.Context(), client, "test", &output); err != nil || output.Value != 42 {
		t.Fatalf("output=%+v, err=%v", output, err)
	}
	client.metadata.SerializedOutput = "{"
	if err := Wait(t.Context(), client, "test", &output); err == nil {
		t.Fatal("accepted malformed output")
	}
}

func TestWaitPreservesErrors(t *testing.T) {
	if err := Wait(t.Context(), completedClient{err: context.DeadlineExceeded}, "test", nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lost client error: %v", err)
	}
	if err := Wait(t.Context(), completedClient{}, "test", nil); err == nil {
		t.Fatal("accepted missing metadata")
	}
}
