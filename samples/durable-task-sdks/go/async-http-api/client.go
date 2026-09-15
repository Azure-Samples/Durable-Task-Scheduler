package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
)

func demo(ctx context.Context, handler http.Handler) (err error) {
	server, err := startHTTPServer(ctx, "127.0.0.1:0", handler)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, server.Close()) }()
	client := &http.Client{Timeout: 10 * time.Second}
	defer client.CloseIdleConnections()

	var started startResponse
	code, headers, err := requestJSON(ctx, client, http.MethodPost, server.URL+"/api/start-operation",
		operationRequest{ProcessingTime: 2}, &started)
	if err != nil {
		return err
	}
	if code != http.StatusAccepted || headers.Get("Location") == "" {
		return fmt.Errorf("start operation returned HTTP %d without a polling location", code)
	}
	location := headers.Get("Location")
	fmt.Printf("Accepted operation %s; polling %s\n", started.OperationID, location)

	var result *operationResult
	err = sample.Until(ctx, time.Second, func() (bool, error) {
		var status statusResponse
		code, _, err := requestJSON(ctx, client, http.MethodGet, server.URL+location, nil, &status)
		if err != nil {
			return false, err
		}
		if code == http.StatusAccepted {
			return false, nil
		}
		if code != http.StatusOK || status.Status != "Completed" || status.Result == nil {
			return false, fmt.Errorf("operation ended with HTTP %d, status %s: %s", code, status.Status, status.Error)
		}
		result = status.Result
		return true, nil
	})
	if err != nil {
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
