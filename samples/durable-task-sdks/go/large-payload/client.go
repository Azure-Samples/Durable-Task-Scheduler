package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
)

const (
	smallRecords = 10
	largeRecords = 300_000
)

func run(ctx context.Context) (err error) {
	worker, err := startWorker(ctx)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, worker.Close()) }()
	fmt.Printf("Payload container: %s\n", worker.container)

	for _, count := range []int{smallRecords, largeRecords} {
		content, err := recordData(count)
		if err != nil {
			return err
		}
		id, result, err := roundTrip(ctx, worker, content)
		if err != nil {
			return err
		}
		fmt.Printf("%s: completed with %d records (%d bytes)\n", id, result.Records, result.Bytes)
	}
	return nil
}

func roundTrip(ctx context.Context, worker *payloadWorker, content string) (api.InstanceID, payloadResult, error) {
	id := sample.ID("large-payload")
	var result payloadResult
	if _, err := worker.Client.ScheduleNewOrchestration(ctx, worker.workflow.orchestrator,
		api.WithInstanceID(id), api.WithInput(content)); err != nil {
		return id, result, err
	}
	err := sample.Wait(ctx, worker.Client, id, &result)
	return id, result, err
}
