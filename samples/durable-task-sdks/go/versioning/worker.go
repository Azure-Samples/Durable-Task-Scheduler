package main

import (
	"context"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
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

func startWorker(ctx context.Context) (*sample.Host, error) {
	registry, err := newRegistry()
	if err != nil {
		return nil, err
	}
	options, err := sample.Options()
	if err != nil {
		return nil, err
	}
	options.Versioning = &task.VersioningOptions{
		Version:         currentVersion,
		DefaultVersion:  currentVersion,
		MatchStrategy:   task.VersionMatchCurrentOrOlder,
		FailureStrategy: task.VersionFailureFail,
	}
	return sample.Start(ctx, registry, options)
}
