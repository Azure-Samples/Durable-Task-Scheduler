package main

import (
	"fmt"

	"github.com/microsoft/durabletask-go/task"
)

type versionResult struct {
	Version          string   `json:"version"`
	Results          []string `json:"results"`
	ActivityVersions []string `json:"activity_versions"`
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
		// Activity versions inherit the execution's version, not the worker's default.
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
