package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
	"github.com/microsoft/durabletask-go/api"
)

func TestIntegration(t *testing.T) {
	ctx := testutil.IntegrationContext(t)
	if err := verifyVersions(ctx); err != nil {
		t.Fatal(err)
	}
}

func verifyVersions(ctx context.Context) (err error) {
	host, err := startWorker(ctx)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, host.Close()) }()

	for _, version := range versions {
		id := sample.ID("versioning")
		if _, err := host.Client.ScheduleNewOrchestration(ctx, orchestrationName,
			api.WithInstanceID(id), api.WithInput("World"), api.WithVersion(version)); err != nil {
			return err
		}
		var result versionResult
		if err := sample.Wait(ctx, host.Client, id, &result); err != nil {
			return err
		}
		metadata, err := host.Client.FetchOrchestrationMetadata(ctx, id)
		if err != nil {
			return err
		}
		if metadata.Version != version {
			return fmt.Errorf("persisted version = %q, want %q", metadata.Version, version)
		}
		if err := validateResult(result, version); err != nil {
			return err
		}
	}
	return nil
}

func validateResult(result versionResult, version string) error {
	steps, err := stepsForVersion(version)
	if err != nil {
		return err
	}
	want := []string{
		"Hello, World!",
		"Goodbye, World!",
		"Notification sent: Completed greeting workflow for World",
	}[:len(steps)]
	if result.Version != version || !slices.Equal(result.Results, want) {
		return fmt.Errorf("version %s result = %+v, want %v", version, result, want)
	}
	if len(result.ActivityVersions) != len(want) {
		return fmt.Errorf("version %s returned %d activity versions, want %d", version, len(result.ActivityVersions), len(want))
	}
	for _, observed := range result.ActivityVersions {
		if observed != version {
			return fmt.Errorf("activity version = %q, orchestration version = %q", observed, version)
		}
	}
	return nil
}
