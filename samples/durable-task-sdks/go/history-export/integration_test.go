package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/exporthistory"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIntegration(t *testing.T) {
	ctx := testutil.IntegrationContext(t)
	if !t.Run("batch", func(t *testing.T) {
		if err := exerciseHistoryExport(ctx); err != nil {
			t.Fatal(err)
		}
	}) {
		return
	}
	t.Run("active-job cancellation", func(t *testing.T) {
		if err := exerciseCanceledExport(ctx); err != nil {
			t.Fatal(err)
		}
	})
}

func exerciseHistoryExport(ctx context.Context) (err error) {
	if err := separateExportWindows(ctx); err != nil {
		return err
	}
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
	expected, err := sourceExpectations(ctx, worker, batch.Sources)
	if err != nil {
		return err
	}
	blobs, err := historyBlobReader(batch.Container)
	if err != nil {
		return err
	}
	job, err := jobClient(worker, batch)
	if err != nil {
		return err
	}
	return withJobCleanup(ctx, worker.lifetime.context, job, func() error {
		description, err := executeExport(ctx, job, query)
		if err != nil {
			return err
		}
		if err := testutil.Require(
			description.ScannedInstances == sourceCount && description.ExportedInstances == sourceCount &&
				description.LastError == "" && description.OrchestratorInstanceID != "",
			"unexpected export progress: %+v", description); err != nil {
			return err
		}
		if err := sample.Wait(ctx, worker.Client, api.InstanceID(description.OrchestratorInstanceID), nil); err != nil {
			return err
		}
		client, err := exporthistory.NewClient(worker.Client.TaskHubGrpcClient, exporthistory.ClientOptions{})
		if err != nil {
			return err
		}
		jobs, err := client.ListJobs(ctx, exporthistory.ExportJobQuery{JobIDPrefix: batch.JobID, PageSize: 2})
		if err != nil {
			return err
		}
		if err := testutil.Require(len(jobs.Jobs) == 1 && jobs.Jobs[0].JobID == batch.JobID,
			"job-scoped listing did not return exactly this export"); err != nil {
			return err
		}
		_, err = verifyExportBlobs(ctx, blobs, batch.Container, batch.Prefix, expected)
		return err
	})
}

func sourceExpectations(ctx context.Context, worker *historyWorker, sources []sourceInstance) ([]sourceExecution, error) {
	expected := make([]sourceExecution, 0, len(sources))
	for _, source := range sources {
		metadata, err := worker.Client.FetchOrchestrationMetadata(ctx, source.ID, api.WithFetchPayloads(true))
		if err != nil {
			return nil, err
		}
		if metadata == nil || metadata.ExecutionID == "" || metadata.RuntimeStatus != api.RUNTIME_STATUS_COMPLETED {
			return nil, fmt.Errorf("source %s has no completed execution", source.ID)
		}
		var output int
		if err := metadata.ReadOutput(&output); err != nil {
			return nil, err
		}
		if err := testutil.Require(output == source.Input*source.Input, "source %s returned %d", source.ID, output); err != nil {
			return nil, err
		}
		expected = append(expected, sourceExecution{
			ID: source.ID, Input: source.Input,
			ExecutionID: metadata.ExecutionID, CompletedAt: source.CompletedAt,
		})
	}
	return expected, nil
}

type pausedHistoryStore struct {
	exporthistory.Store
	beforeWrite func(context.Context) error
}

func (s pausedHistoryStore) Write(ctx context.Context, object exporthistory.ExportObject) error {
	if err := s.beforeWrite(ctx); err != nil {
		return err
	}
	return s.Store.Write(ctx, object)
}

func exerciseCanceledExport(ctx context.Context) (err error) {
	if err := separateExportWindows(ctx); err != nil {
		return err
	}
	scenario, cancel := context.WithCancel(ctx)
	defer cancel()
	batch := newExportBatch()
	store, err := newHistoryStore(batch.Container)
	if err != nil {
		return err
	}
	active := make(chan struct{})
	worker, err := startWorker(scenario, batch, pausedHistoryStore{
		Store: store, beforeWrite: pauseBeforeWrite(scenario, func() { close(active) }),
	})
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, worker.Close()) }()
	query, err := prepareExport(scenario, worker, batch)
	if err != nil {
		return err
	}
	job, err := jobClient(worker, batch)
	if err != nil {
		return err
	}
	finished := make(chan error, 1)
	go func() {
		finished <- withJobCleanup(scenario, worker.lifetime.context, job, func() error {
			_, err := executeExport(scenario, job, query)
			return err
		})
	}()
	select {
	case err := <-finished:
		if err == nil {
			return errors.New("export completed without reaching the cancellation point")
		}
		return fmt.Errorf("export ended before reaching the cancellation point: %w", err)
	case <-ctx.Done():
		cancel()
		return errors.Join(ctx.Err(), <-finished)
	case <-active:
	}
	description, describeErr := job.Describe(ctx)
	cancel()
	canceledErr := <-finished
	if describeErr != nil {
		return errors.Join(describeErr, canceledErr)
	}
	if err := testutil.Require(
		description.Status == exporthistory.ExportJobStatusActive && description.OrchestratorInstanceID != "",
		"cancellation did not target an active export generation"); err != nil {
		return err
	}
	if err := testutil.Require(onlyCancellation(canceledErr),
		"cancellation was not preserved: %v", canceledErr); err != nil {
		return err
	}
	if worker.lifetime.context.Err() != nil {
		return errors.New("worker stopped before cleanup could finish")
	}
	if _, err := job.Describe(ctx); !errors.Is(err, exporthistory.ErrJobNotFound) {
		return fmt.Errorf("canceled job still exists: %v", err)
	}
	_, err = worker.Client.FetchOrchestrationMetadata(ctx, api.InstanceID(description.OrchestratorInstanceID))
	if !errors.Is(err, api.ErrInstanceNotFound) {
		return fmt.Errorf("canceled export generation was not purged: %v", err)
	}
	return nil
}

func onlyCancellation(err error) bool {
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		causes := joined.Unwrap()
		if len(causes) == 0 {
			return false
		}
		for _, cause := range causes {
			if !onlyCancellation(cause) {
				return false
			}
		}
		return true
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return onlyCancellation(wrapped.Unwrap())
	}
	return errors.Is(err, context.Canceled) || status.Code(err) == codes.Canceled
}

func historyBlobReader(container string) (*azblob.Client, error) {
	options, err := storageOptions(container)
	if err != nil {
		return nil, err
	}
	if options.ConnectionString != "" {
		return azblob.NewClientFromConnectionString(options.ConnectionString, nil)
	}
	return azblob.NewClient(options.AccountURL, options.Credential, nil)
}

func separateExportWindows(ctx context.Context) error {
	// The E2E runner starts this test after the demo. Avoid putting its completed
	// control operations in the next batch's second-granularity window.
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
