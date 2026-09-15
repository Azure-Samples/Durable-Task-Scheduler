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
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

var (
	serve  = flag.Bool("serve", false, "Serve the interactive API instead of the bounded fixture demonstration")
	listen = flag.String("listen", "127.0.0.1:8000", "Loopback listen address for -serve")
	mode   = flag.String("mode", defaultMode(), "Research mode: fixture or real (real requires -serve)")
)

func defaultMode() string {
	if value := strings.TrimSpace(os.Getenv("RESEARCH_MODE")); value != "" {
		return value
	}
	return "fixture"
}

func main() { sample.Main("arXiv_research_agent", run) }

func run(ctx context.Context) error {
	if *serve {
		if _, err := loopbackAddress(*listen); err != nil {
			return err
		}
	}
	if *mode != "fixture" && *mode != "real" {
		return errors.New("-mode must be fixture or real")
	}
	if !*serve && *mode != "fixture" {
		return errors.New("the bounded verification demo uses fixtures; use -serve -mode real for real research")
	}
	if !*serve {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 65*time.Second)
		defer cancel()
	}
	activities := &activities{mode: *mode}
	if *mode == "real" {
		config, err := loadModelConfig(os.Getenv)
		if err != nil {
			return err
		}
		model, err := newOpenAIModel(config)
		if err != nil {
			return err
		}
		source, err := newArxivClient(strings.TrimSpace(os.Getenv("ARXIV_API_ENDPOINT")))
		if err != nil {
			return err
		}
		activities.model, activities.source = model, source
	}
	registry, err := newRegistry(activities)
	if err != nil {
		return err
	}
	fmt.Printf("Research mode: %s (fixture papers and reports are synthetic, not academic evidence)\n", *mode)
	return sample.WithHost(ctx, registry, func(ctx context.Context, client *dts.Client) error {
		app := &researchAPI{store: schedulerStore{client}, mode: *mode}
		if *serve {
			return serveHTTP(ctx, *listen, app.handler())
		}
		return demo(ctx, client, app.handler())
	})
}

func demo(ctx context.Context, client *dts.Client, handler http.Handler) error {
	server := httptest.NewServer(handler)
	defer server.Close()
	httpClient := &http.Client{Timeout: 40 * time.Second}
	var health struct {
		Status string `json:"status"`
		Mode   string `json:"mode"`
	}
	code, _, err := requestJSON(ctx, httpClient, http.MethodGet, server.URL+"/health", nil, &health)
	if err != nil {
		return err
	}
	if err := sample.Require(code == 200 && health.Status == "healthy" && health.Mode == "fixture", "invalid health response"); err != nil {
		return err
	}
	var start startResponse
	code, headers, err := requestJSON(ctx, httpClient, http.MethodPost, server.URL+"/agents",
		startRequest{Topic: demoTopic, MaxIterations: 2}, &start)
	if err != nil {
		return err
	}
	if err := sample.Require(code == 202 && start.OK && start.Mode == "fixture" && instancePattern.MatchString(start.InstanceID) &&
		headers.Get("Location") == start.StatusURL && headers.Get("Retry-After") == "1" &&
		headers.Get("X-Research-Mode") == "fixture", "invalid start response: %d %+v", code, start); err != nil {
		return err
	}
	var status statusResponse
	if err := sample.Until(ctx, 200*time.Millisecond, func() (bool, error) {
		code, _, err := requestJSON(ctx, httpClient, http.MethodGet, server.URL+start.StatusURL, nil, &status)
		if err != nil {
			return false, err
		}
		if code != 200 || status.Topic != demoTopic || status.Mode != "fixture" {
			return false, fmt.Errorf("unexpected research status: %d %+v", code, status)
		}
		switch status.Status {
		case "COMPLETED":
			return true, nil
		case "PENDING", "RUNNING", "CONTINUED_AS_NEW":
			return false, nil
		default:
			return false, fmt.Errorf("research did not complete successfully: %+v", status)
		}
	}); err != nil {
		return err
	}
	var result researchResult
	code, _, err = requestJSON(ctx, httpClient, http.MethodGet, server.URL+start.StatusURL+"/wait?timeout=5", nil, &result)
	if err != nil {
		return err
	}
	if err := sample.Require(code == 200, "wait endpoint returned HTTP %d", code); err != nil {
		return err
	}
	if err := verifyFixtureResult(result); err != nil {
		return err
	}
	if err := sample.Require(status.Report == expectedDemoReport && status.Iteration == 2 && status.FindingsCount == 3,
		"status endpoint did not return the completed report"); err != nil {
		return err
	}
	var durableResult researchResult
	if err := sample.Wait(ctx, client, api.InstanceID(start.InstanceID), &durableResult); err != nil {
		return err
	}
	if err := sample.Require(reflect.DeepEqual(result, durableResult), "HTTP result differs from durable output"); err != nil {
		return err
	}
	metadata, err := client.FetchOrchestrationMetadata(ctx, api.InstanceID(start.InstanceID), api.WithFetchPayloads(true))
	if err != nil {
		return err
	}
	if err := verifyFixtureCheckpoint(ctx, client, metadata); err != nil {
		return err
	}
	var cancelJob startResponse
	code, _, err = requestJSON(ctx, httpClient, http.MethodPost, server.URL+"/agents",
		startRequest{Topic: "fixture cancellation", MaxIterations: 2, StartDelaySeconds: 30}, &cancelJob)
	if err != nil {
		return err
	}
	if err := sample.Require(code == 202, "scheduled start returned %d", code); err != nil {
		return err
	}
	code, headers, err = requestJSON(ctx, httpClient, http.MethodDelete, server.URL+cancelJob.StatusURL, nil, nil)
	if err != nil {
		return err
	}
	if err := sample.Require(code == 202 && headers.Get("Location") == cancelJob.StatusURL, "termination returned %d", code); err != nil {
		return err
	}
	metadata, err = client.WaitForOrchestrationCompletion(ctx, api.InstanceID(cancelJob.InstanceID))
	if err != nil {
		return err
	}
	if err := sample.Require(metadata.RuntimeStatus == api.RUNTIME_STATUS_TERMINATED, "expected a terminated research job"); err != nil {
		return err
	}
	code, _, err = requestJSON(ctx, httpClient, http.MethodGet, server.URL+cancelJob.StatusURL, nil, &status)
	if err != nil {
		return err
	}
	if err := sample.Require(code == 200 && status.Status == "TERMINATED", "wrong terminated status: %+v", status); err != nil {
		return err
	}
	code, _, err = requestJSON(ctx, httpClient, http.MethodGet, server.URL+"/agents/"+string(sample.ID("arxiv-missing")), nil, nil)
	if err != nil {
		return err
	}
	if err := sample.Require(code == 404, "missing research returned %d", code); err != nil {
		return err
	}
	return sample.PrintJSON(map[string]any{
		"instance_id": start.InstanceID, "mode": result.Mode, "iterations": result.Iterations,
		"findings_count": result.FindingsCount, "paper_ids": result.PaperIDs, "report": result.Report,
	})
}

func verifyFixtureResult(result researchResult) error {
	if err := sample.Require(result.Mode == "fixture" && result.Topic == demoTopic && result.Iterations == 2 &&
		result.FindingsCount == 3 && len(result.Findings) == 3 && len(result.Papers) == 3 &&
		reflect.DeepEqual(result.PaperIDs, []string{"fixture-001", "fixture-002", "fixture-003"}) &&
		result.Report == expectedDemoReport, "fixture report, iterations or paper IDs differ: %+v", result); err != nil {
		return err
	}
	for _, item := range result.Papers {
		expected, err := fixturePaper(item.ID)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(item, expected) {
			return fmt.Errorf("fetched fixture metadata differs: %+v", item)
		}
	}
	expectedQueries := []string{demoTopic, demoTopic + " methods", demoTopic + " evaluation"}
	expectedIDs := [][]string{{"fixture-001", "fixture-002"}, {"fixture-002", "fixture-003"}, {"fixture-001", "fixture-003"}}
	for index, finding := range result.Findings {
		if finding.Query != expectedQueries[index] || finding.RelevanceScore != 8 ||
			!reflect.DeepEqual(finding.PaperIDs, expectedIDs[index]) ||
			finding.Summary != "Synthetic analysis of "+strings.Join(expectedIDs[index], ", ")+"." {
			return fmt.Errorf("unexpected fixture analysis: %+v", finding)
		}
	}
	return nil
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
