package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestratorName = "GoAsyncHTTPAPI"
	activityName     = "GoAsyncHTTPProcessOperation"
)

var (
	serve  = flag.Bool("serve", false, "Serve the interactive HTTP API instead of running the verification demo")
	listen = flag.String("listen", "127.0.0.1:8000", "Loopback listen address for -serve")
)

type operationRequest struct {
	ProcessingTime int `json:"processing_time"`
}

type operationInput struct {
	OperationID    string `json:"operation_id"`
	ProcessingTime int    `json:"processing_time"`
}

type operationResult struct {
	OperationID string  `json:"operation_id"`
	Status      string  `json:"status"`
	Result      string  `json:"result"`
	ProcessedAt float64 `json:"processed_at"`
}

func main() { sample.Main("async-http-api", run) }

func run(ctx context.Context) error {
	if *serve {
		if _, err := loopbackAddress(*listen); err != nil {
			return err
		}
	}
	if !*serve {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 65*time.Second)
		defer cancel()
	}
	registry := task.NewTaskRegistry()
	if err := registry.AddOrchestratorN(orchestratorName, orchestrate); err != nil {
		return err
	}
	if err := registry.AddActivityN(activityName, processActivity); err != nil {
		return err
	}
	return sample.WithHost(ctx, registry, func(ctx context.Context, client *dts.Client) error {
		handler := newHandler(schedulerStore{client})
		if *serve {
			return serveHTTP(ctx, *listen, handler)
		}
		return demo(ctx, client, handler)
	})
}

func validateProcessingTime(seconds int) error {
	if seconds < 1 || seconds > 30 {
		return errors.New("processing_time must be an integer between 1 and 30 seconds")
	}
	return nil
}

func orchestrate(ctx *task.OrchestrationContext) (any, error) {
	var input operationInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if err := validateProcessingTime(input.ProcessingTime); err != nil {
		return nil, err
	}
	if input.OperationID != string(ctx.ID) {
		return nil, errors.New("operation ID must match the orchestration instance ID")
	}
	var result operationResult
	if err := ctx.CallActivity(activityName, task.WithActivityInput(input)).Await(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func processActivity(ctx task.ActivityContext) (any, error) {
	var input operationInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	return processOperation(ctx.Context(), input)
}

func processOperation(ctx context.Context, input operationInput) (operationResult, error) {
	if err := validateProcessingTime(input.ProcessingTime); err != nil {
		return operationResult{}, err
	}
	// Simulated external work belongs in an activity, not in replayed orchestration code.
	timer := time.NewTimer(time.Duration(input.ProcessingTime) * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return operationResult{}, ctx.Err()
	case <-timer.C:
	}
	return operationResult{
		OperationID: input.OperationID,
		Status:      "completed",
		Result:      fmt.Sprintf("Operation %s completed successfully", input.OperationID),
		ProcessedAt: float64(time.Now().UnixMilli()) / 1000,
	}, nil
}

func demo(ctx context.Context, client *dts.Client, handler http.Handler) error {
	server := httptest.NewServer(handler)
	defer server.Close()
	httpClient := &http.Client{Timeout: 10 * time.Second}
	var started startResponse
	code, headers, err := requestJSON(ctx, httpClient, http.MethodPost, server.URL+"/api/start-operation",
		operationRequest{ProcessingTime: 3}, &started)
	if err != nil {
		return err
	}
	if err := sample.Require(code == http.StatusAccepted && headers.Get("Location") == started.StatusURL &&
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
			return false, sample.Require(headers.Get("Retry-After") == "1" &&
				headers.Get("Location") == started.StatusURL && status.Result == nil,
				"invalid pending response: %+v", status)
		}
		return true, sample.Require(code == http.StatusOK && status.Status == "Completed",
			"unexpected terminal HTTP response: %d %+v", code, status)
	}); err != nil {
		return err
	}
	var result operationResult
	if err := sample.Wait(ctx, client, api.InstanceID(started.OperationID), &result); err != nil {
		return err
	}
	if err := sample.Require(sawPending && status.OperationID == started.OperationID &&
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
	if err := sample.Require(code == http.StatusAccepted && headers.Get("Location") == canceled.StatusURL,
		"terminate returned %d", code); err != nil {
		return err
	}
	metadata, err := client.WaitForOrchestrationCompletion(ctx, api.InstanceID(canceled.OperationID))
	if err != nil {
		return err
	}
	if err := sample.Require(metadata.RuntimeStatus == api.RUNTIME_STATUS_TERMINATED, "expected terminated, got %s", metadata.RuntimeStatus); err != nil {
		return err
	}
	var terminated statusResponse
	code, _, err = requestJSON(ctx, httpClient, http.MethodGet, server.URL+canceled.StatusURL, nil, &terminated)
	if err != nil {
		return err
	}
	if err := sample.Require(code == http.StatusOK && terminated.Status == "Terminated" && terminated.Result == nil,
		"invalid terminated status: %d %+v", code, terminated); err != nil {
		return err
	}
	code, _, err = requestJSON(ctx, httpClient, http.MethodGet,
		server.URL+"/api/operations/"+string(sample.ID("async-http-missing")), nil, nil)
	if err != nil {
		return err
	}
	if err := sample.Require(code == http.StatusNotFound, "missing operation returned %d", code); err != nil {
		return err
	}
	return sample.PrintJSON(result)
}

func requestJSON(ctx context.Context, client *http.Client, method, address string, input, output any) (int, http.Header, error) {
	var body io.Reader
	if input != nil {
		data, err := json.Marshal(input)
		if err != nil {
			return 0, nil, err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, address, body)
	if err != nil {
		return 0, nil, err
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024+1))
	if err != nil {
		return 0, nil, err
	}
	if len(data) > 64*1024 || resp.Header.Get("Content-Type") != "application/json" {
		return 0, nil, errors.New("invalid JSON HTTP response")
	}
	if output != nil {
		if err := json.Unmarshal(data, output); err != nil {
			return resp.StatusCode, resp.Header, err
		}
	}
	return resp.StatusCode, resp.Header, nil
}
