package main

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestrationName = "go-sample-versioning-greeting"
	helloName         = "go-sample-versioning-hello"
	goodbyeName       = "go-sample-versioning-goodbye"
	notificationName  = "go-sample-versioning-notification"
	currentVersion    = "10.0.0"
)

var versions = []string{"1.0.0", "2.0.0", "3.0.0", currentVersion}

type versionResult struct {
	Version          string   `json:"version"`
	Results          []string `json:"results"`
	ActivityVersions []string `json:"activity_versions"`
}

type activityResult struct {
	Message string `json:"message"`
	Version string `json:"version"`
}

func main() {
	sample.Main("versioning", run)
}

func run(ctx context.Context) (err error) {
	registry, err := newRegistry()
	if err != nil {
		return err
	}
	options, err := sample.Options()
	if err != nil {
		return err
	}
	options.Versioning = &task.VersioningOptions{
		Version:         currentVersion,
		DefaultVersion:  currentVersion,
		MatchStrategy:   task.VersionMatchCurrentOrOlder,
		FailureStrategy: task.VersionFailureFail,
	}
	host, err := sample.Start(ctx, registry, options)
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
		if err := sample.PrintJSON(result); err != nil {
			return err
		}
	}
	fmt.Println("SDK CurrentOrOlder worker 10.0.0 accepted 1.0.0, 2.0.0, 3.0.0, and 10.0.0")
	return nil
}

func newRegistry() (*task.TaskRegistry, error) {
	registry := task.NewTaskRegistry()
	for _, version := range versions {
		if err := registry.AddOrchestratorNVersion(orchestrationName, version, versionedGreeting); err != nil {
			return nil, err
		}
		for _, activity := range []struct {
			name   string
			format string
		}{
			{helloName, "Hello, %s!"},
			{goodbyeName, "Goodbye, %s!"},
			{notificationName, "Notification sent: Completed greeting workflow for %s"},
		} {
			if err := registry.AddActivityNVersion(activity.name, version, messageActivity(activity.format)); err != nil {
				return nil, err
			}
		}
	}
	return registry, nil
}

func versionedGreeting(ctx *task.OrchestrationContext) (any, error) {
	var name string
	if err := ctx.GetInput(&name); err != nil {
		return nil, err
	}
	steps, err := stepsForVersion(ctx.Version)
	if err != nil {
		return nil, err
	}
	result := versionResult{Version: ctx.Version}
	for _, step := range steps {
		var output activityResult
		// Activity versions inherit this execution's version, not the worker's default.
		if err := ctx.CallActivity(step, task.WithActivityInput(name)).Await(&output); err != nil {
			return nil, err
		}
		result.Results = append(result.Results, output.Message)
		result.ActivityVersions = append(result.ActivityVersions, output.Version)
	}
	return result, nil
}

func stepsForVersion(version string) ([]string, error) {
	switch version {
	case "1.0.0":
		return []string{helloName}, nil
	case "2.0.0":
		return []string{helloName, goodbyeName}, nil
	case "3.0.0", currentVersion:
		return []string{helloName, goodbyeName, notificationName}, nil
	default:
		return nil, fmt.Errorf("sample has no workflow definition for version %q", version)
	}
}

func messageActivity(format string) task.Activity {
	return func(ctx task.ActivityContext) (any, error) {
		var name string
		if err := ctx.GetInput(&name); err != nil {
			return nil, err
		}
		info, ok := api.ActivityContextInfoFromContext(ctx.Context())
		if !ok {
			return nil, errors.New("activity version metadata is missing")
		}
		return activityResult{Message: fmt.Sprintf(format, name), Version: info.Version}, nil
	}
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
