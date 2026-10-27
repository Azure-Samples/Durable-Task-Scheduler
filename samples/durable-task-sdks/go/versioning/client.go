package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
)

func run(ctx context.Context) (err error) {
	host, err := startWorker(ctx)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, host.Close()) }()

	for _, version := range []string{"1.0.0", "3.0.0"} {
		id := sample.ID("versioning")
		if _, err := host.Client.ScheduleNewOrchestration(ctx, orchestrationName,
			api.WithInstanceID(id), api.WithInput("World"), api.WithVersion(version)); err != nil {
			return err
		}
		var result versionResult
		if err := sample.Wait(ctx, host.Client, id, &result); err != nil {
			return err
		}
		fmt.Printf("Version %s: %s\n", result.Version, strings.Join(result.Results, " | "))
	}
	return nil
}
