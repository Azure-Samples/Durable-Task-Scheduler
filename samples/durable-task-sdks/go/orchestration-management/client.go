package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
)

func run(ctx context.Context) (err error) {
	host, err := startWorker(ctx)
	if err != nil {
		return err
	}
	var owned []api.InstanceID
	defer func() {
		if err != nil {
			err = errors.Join(err, stopOwned(host.Client, owned))
		}
		err = errors.Join(err, host.Close())
	}()

	id := sample.ID("management")
	owned = append(owned, id)
	if _, err := host.Client.ScheduleNewOrchestration(ctx, workflowName,
		api.WithInstanceID(id), api.WithInput(batchInput{BatchID: "batch-1", ItemCount: 10})); err != nil {
		return err
	}
	var result batchResult
	if err := sample.Wait(ctx, host.Client, id, &result); err != nil {
		return err
	}
	fmt.Printf("Completed %s: %d items (%s)\n", result.BatchID, result.ItemsProcessed, result.Status)

	// A new ID preserves the original execution and reuses its input.
	restartedID, err := host.Client.RestartInstance(ctx, id, api.WithRestartNewInstanceID(true))
	if err != nil {
		return err
	}
	owned = append(owned, restartedID)
	if err := sample.Wait(ctx, host.Client, restartedID, &result); err != nil {
		return err
	}
	fmt.Printf("Restarted as %s: %d items (%s)\n", restartedID, result.ItemsProcessed, result.Status)

	purged, err := host.Client.PurgeInstances(ctx, api.PurgeInstancesRequest{
		InstanceIDs: owned, Recursive: false,
	})
	if err != nil {
		return err
	}
	if purged == nil || !purged.IsComplete {
		return errors.New("the service did not complete the exact-ID purge")
	}
	fmt.Printf("Service reports %d deleted instances; only this run's IDs were submitted\n", purged.DeletedInstanceCount)
	return nil
}
