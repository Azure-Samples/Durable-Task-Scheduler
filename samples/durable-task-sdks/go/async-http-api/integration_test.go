package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

func TestIntegration(t *testing.T) {
	ctx := testutil.IntegrationContext(t)
	registry, err := newRegistry()
	if err != nil {
		t.Fatal(err)
	}
	err = sample.WithHost(ctx, registry, func(ctx context.Context, client *dts.Client) error {
		return verifyHTTPAPI(ctx, client, newHandler(schedulerStore{client}))
	})
	if err != nil {
		t.Fatal(err)
	}
}

func verifyHTTPAPI(ctx context.Context, client *dts.Client, handler http.Handler) error {
	server := httptest.NewServer(handler)
	defer server.Close()
	httpClient := &http.Client{Timeout: 10 * time.Second}
	var started startResponse
	code, headers, err := requestJSON(ctx, httpClient, http.MethodPost, server.URL+"/api/start-operation",
		operationRequest{ProcessingTime: 3}, &started)
	if err != nil {
		return err
	}
	if err := testutil.Require(code == http.StatusAccepted && headers.Get("Location") == started.StatusURL &&
		headers.Get("Retry-After") == "1" && started.OperationID != "", "invalid start response: %d %+v", code, headers); err != nil {
		return err
	}
	var status statusResponse
	sawPending := false
	if err := sample.Until(ctx, 150*time.Millisecond, func() (bool, error) {
		code, headers, err := requestJSON(ctx, httpClient, http.MethodGet, server.URL+started.StatusURL, nil, &status)
		if err != nil {
			return false, err
		}
		if code == http.StatusAccepted {
			sawPending = true
			return false, testutil.Require(headers.Get("Retry-After") == "1" &&
				headers.Get("Location") == started.StatusURL && status.Result == nil,
				"invalid pending response: %+v", status)
		}
		return true, testutil.Require(code == http.StatusOK && status.Status == "Completed",
			"unexpected terminal HTTP response: %d %+v", code, status)
	}); err != nil {
		return err
	}
	var result operationResult
	if err := sample.Wait(ctx, client, api.InstanceID(started.OperationID), &result); err != nil {
		return err
	}
	if err := testutil.Require(sawPending && status.OperationID == started.OperationID &&
		status.Result != nil && *status.Result == result && result.OperationID == started.OperationID &&
		result.Status == "completed" && result.ProcessedAt > 0 &&
		result.Result == fmt.Sprintf("Operation %s completed successfully", started.OperationID),
		"HTTP and durable results differ: HTTP=%+v durable=%+v", status, result); err != nil {
		return err
	}
	var canceled startResponse
	code, _, err = requestJSON(ctx, httpClient, http.MethodPost, server.URL+"/api/start-operation",
		operationRequest{ProcessingTime: 4}, &canceled)
	if err != nil {
		return err
	}
	if code != http.StatusAccepted {
		return fmt.Errorf("second start returned %d", code)
	}
	code, headers, err = requestJSON(ctx, httpClient, http.MethodDelete, server.URL+canceled.StatusURL, nil, nil)
	if err != nil {
		return err
	}
	if err := testutil.Require(code == http.StatusAccepted && headers.Get("Location") == canceled.StatusURL,
		"terminate returned %d", code); err != nil {
		return err
	}
	metadata, err := client.WaitForOrchestrationCompletion(ctx, api.InstanceID(canceled.OperationID))
	if err != nil {
		return err
	}
	if err := testutil.Require(metadata.RuntimeStatus == api.RUNTIME_STATUS_TERMINATED, "expected terminated, got %s", metadata.RuntimeStatus); err != nil {
		return err
	}
	var terminated statusResponse
	code, _, err = requestJSON(ctx, httpClient, http.MethodGet, server.URL+canceled.StatusURL, nil, &terminated)
	if err != nil {
		return err
	}
	if err := testutil.Require(code == http.StatusOK && terminated.Status == "Terminated" && terminated.Result == nil,
		"invalid terminated status: %d %+v", code, terminated); err != nil {
		return err
	}
	code, _, err = requestJSON(ctx, httpClient, http.MethodGet,
		server.URL+"/api/operations/"+string(sample.ID("async-http-missing")), nil, nil)
	if err != nil {
		return err
	}
	if err := testutil.Require(code == http.StatusNotFound, "missing operation returned %d", code); err != nil {
		return err
	}
	return nil
}
