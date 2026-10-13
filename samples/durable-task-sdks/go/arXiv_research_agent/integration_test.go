package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

func TestIntegration(t *testing.T) {
	ctx := testutil.IntegrationContext(t)
	activities, err := newActivities("fixture")
	if err != nil {
		t.Fatal(err)
	}
	registry, err := newRegistry(activities)
	if err != nil {
		t.Fatal(err)
	}
	err = sample.WithHost(ctx, registry, func(ctx context.Context, client *dts.Client) error {
		app := &researchAPI{store: schedulerStore{client}, mode: "fixture"}
		return verifyResearchHTTP(ctx, client, app.handler())
	})
	if err != nil {
		t.Fatal(err)
	}
}

func verifyResearchHTTP(ctx context.Context, client *dts.Client, handler http.Handler) error {
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
	if err := testutil.Require(code == 200 && health.Status == "healthy" && health.Mode == "fixture", "invalid health response"); err != nil {
		return err
	}
	var start startResponse
	code, headers, err := requestJSON(ctx, httpClient, http.MethodPost, server.URL+"/agents",
		startRequest{Topic: demoTopic, MaxIterations: 2}, &start)
	if err != nil {
		return err
	}
	if err := testutil.Require(code == 202 && start.OK && start.Mode == "fixture" && instancePattern.MatchString(start.InstanceID) &&
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
	if err := testutil.Require(code == 200, "wait endpoint returned HTTP %d", code); err != nil {
		return err
	}
	if err := verifyFixtureResult(result); err != nil {
		return err
	}
	if err := testutil.Require(status.Report == expectedDemoReport && status.Iteration == 2 && status.FindingsCount == 3,
		"status endpoint did not return the completed report"); err != nil {
		return err
	}
	var durableResult researchResult
	if err := sample.Wait(ctx, client, api.InstanceID(start.InstanceID), &durableResult); err != nil {
		return err
	}
	if err := testutil.Require(reflect.DeepEqual(result, durableResult), "HTTP result differs from durable output"); err != nil {
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
	if err := testutil.Require(code == 202, "scheduled start returned %d", code); err != nil {
		return err
	}
	code, headers, err = requestJSON(ctx, httpClient, http.MethodDelete, server.URL+cancelJob.StatusURL, nil, nil)
	if err != nil {
		return err
	}
	if err := testutil.Require(code == 202 && headers.Get("Location") == cancelJob.StatusURL, "termination returned %d", code); err != nil {
		return err
	}
	metadata, err = client.WaitForOrchestrationCompletion(ctx, api.InstanceID(cancelJob.InstanceID))
	if err != nil {
		return err
	}
	if err := testutil.Require(metadata.RuntimeStatus == api.RUNTIME_STATUS_TERMINATED, "expected a terminated research job"); err != nil {
		return err
	}
	code, _, err = requestJSON(ctx, httpClient, http.MethodGet, server.URL+cancelJob.StatusURL, nil, &status)
	if err != nil {
		return err
	}
	if err := testutil.Require(code == 200 && status.Status == "TERMINATED", "wrong terminated status: %+v", status); err != nil {
		return err
	}
	code, _, err = requestJSON(ctx, httpClient, http.MethodGet, server.URL+"/agents/"+string(sample.ID("arxiv-missing")), nil, nil)
	if err != nil {
		return err
	}
	if err := testutil.Require(code == 404, "missing research returned %d", code); err != nil {
		return err
	}
	return nil
}
