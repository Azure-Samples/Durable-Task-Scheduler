package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/exporthistory"
)

func run(ctx context.Context) (err error) {
	batch := newExportBatch()
	store, err := newHistoryStore(batch.Container)
	if err != nil {
		return err
	}
	worker, err := startWorker(ctx, batch, store)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, worker.Close()) }()
	query, err := prepareExport(ctx, worker, batch)
	if err != nil {
		return err
	}
	job, err := jobClient(worker, batch)
	if err != nil {
		return err
	}
	fmt.Printf("Destination: %s/%s\n", batch.Container, batch.Prefix)
	err = withJobCleanup(ctx, worker.lifetime.context, job, func() error {
		description, err := executeExport(ctx, job, query)
		if err != nil {
			return err
		}
		fmt.Printf("Export job %s: %s (%d histories exported)\n",
			job.ID(), description.Status, description.ExportedInstances)
		return nil
	})
	if err == nil {
		fmt.Println("Export job deleted; history blobs retained.")
	}
	return err
}

func jobClient(worker *historyWorker, batch exportBatch) (*exporthistory.JobClient, error) {
	client, err := exporthistory.NewClient(worker.Client.TaskHubGrpcClient, exporthistory.ClientOptions{
		ContainerName: batch.Container, Prefix: batch.Prefix,
	})
	if err != nil {
		return nil, err
	}
	return client.JobClient(batch.JobID)
}

func executeExport(ctx context.Context, job *exporthistory.JobClient, query api.InstanceIDQuery) (*exporthistory.ExportJobDescription, error) {
	if err := job.Create(ctx, exporthistory.JobCreationOptions{
		JobID: job.ID(), Mode: exporthistory.ExportModeBatch,
		CompletedTimeFrom: query.CompletedTimeFrom, CompletedTimeTo: query.CompletedTimeTo,
		RuntimeStatus: query.RuntimeStatus, MaxInstancesPerBatch: query.PageSize,
	}); err != nil {
		return nil, err
	}
	var description *exporthistory.ExportJobDescription
	err := sample.Until(ctx, 250*time.Millisecond, func() (bool, error) {
		var err error
		description, err = job.Describe(ctx)
		if err != nil {
			return false, err
		}
		if description.Status == exporthistory.ExportJobStatusFailed {
			return false, fmt.Errorf("export job %s failed: %s", job.ID(), description.LastError)
		}
		return description.Status == exporthistory.ExportJobStatusCompleted, nil
	})
	return description, err
}
