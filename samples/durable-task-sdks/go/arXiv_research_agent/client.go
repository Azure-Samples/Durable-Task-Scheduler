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

const demoTopic = "durable workflow reliability"

func demo(ctx context.Context, handler http.Handler) (err error) {
	server, err := startHTTPServer(ctx, "127.0.0.1:0", handler)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, server.Close()) }()
	client := &http.Client{Timeout: 65 * time.Second}
	defer client.CloseIdleConnections()

	var started startResponse
	code, _, err := requestJSON(ctx, client, http.MethodPost, server.URL+"/agents",
		startRequest{Topic: demoTopic, MaxIterations: 2}, &started)
	if err != nil {
		return err
	}
	if code != http.StatusAccepted || started.StatusURL == "" {
		return fmt.Errorf("start research returned HTTP %d without a status URL", code)
	}
	fmt.Printf("Research %s started; waiting for its report\n", started.InstanceID)

	var result researchResult
	code, _, err = requestJSON(ctx, client, http.MethodGet,
		server.URL+started.StatusURL+"/wait?timeout=60", nil, &result)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("research wait returned HTTP %d; the job can be inspected at %s", code, started.StatusURL)
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
	response, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024+1))
	if err != nil {
		return 0, nil, err
	}
	if len(data) > 1024*1024 || response.Header.Get("Content-Type") != "application/json" {
		return 0, nil, errors.New("invalid JSON HTTP response")
	}
	if output != nil {
		if err := json.Unmarshal(data, output); err != nil {
			return response.StatusCode, response.Header, err
		}
	}
	return response.StatusCode, response.Header, nil
}
