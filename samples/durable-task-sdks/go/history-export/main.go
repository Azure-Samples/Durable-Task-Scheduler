package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/exporthistory"
	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestratorName = "GoHistoryExportOrchestrator"
	squareName       = "GoHistoryExportSquare"
	sourceCount      = 5
	maxHistoryEvents = 128
	maxHistoryBytes  = 1024 * 1024
)

type sourceExecution struct {
	ID          api.InstanceID
	ExecutionID string
	Input       int
	CreatedAt   time.Time
	CompletedAt time.Time
}

func main() {
	sample.Main("history-export", run)
}

func run(ctx context.Context) (err error) {
	if err := requireIsolatedTaskHub(os.Getenv("HISTORY_EXPORT_ISOLATED_TASKHUB")); err != nil {
		return err
	}
	options, err := sample.Options()
	if err != nil {
		return err
	}
	container := string(sample.ID("history-export"))
	jobID := string(sample.ID("history-export-job"))
	prefix := jobID + "/"
	var beforeWrite func(context.Context) error
	if pause := os.Getenv("HISTORY_EXPORT_PAUSE_BEFORE_WRITE"); pause != "" {
		if pause != "1" {
			return errors.New("HISTORY_EXPORT_PAUSE_BEFORE_WRITE must be unset or 1")
		}
		beforeWrite = pauseBeforeWrite(ctx, func() {
			fmt.Printf("EXPORT_JOB_ACTIVE job_id=%s paused_before_write=true\n", jobID)
		})
	}
	storeOptions, blobClient, err := storageOptions(container)
	if err != nil {
		return err
	}
	store, err := exporthistory.NewAzureBlobHistoryStore(storeOptions)
	if err != nil {
		return err
	}
	sources := make([]sourceExecution, sourceCount)
	allowed := make(map[api.InstanceID]struct{}, sourceCount)
	for i := range sources {
		sources[i] = sourceExecution{ID: sample.ID("history-export-source"), Input: i + 1}
		allowed[sources[i].ID] = struct{}{}
	}

	// Registration needs a management source before the worker starts. This
	// separate connection uses the same task-hub options as the sample host.
	sourceClient, err := dts.NewClient(ctx, options, sample.Logger())
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, sourceClient.Close()) }()
	source := &ownedHistorySource{inner: sourceClient, allowed: allowed}
	registry := task.NewTaskRegistry()
	if err := registry.AddOrchestratorN(orchestratorName, squareOrchestrator); err != nil {
		return err
	}
	if err := registry.AddActivityN(squareName, square); err != nil {
		return err
	}
	if err := exporthistory.Register(registry, exporthistory.WorkerOptions{
		Source: source,
		Store: &ownedHistoryStore{
			inner: store, allowed: allowed, container: container, prefix: prefix,
			beforeWrite: beforeWrite,
		},
		HistoryQuery: api.HistoryQuery{MaxEvents: maxHistoryEvents, MaxBytes: maxHistoryBytes},
	}); err != nil {
		return err
	}
	workerLifetime := newExportWorkerLifetime(ctx)
	host, err := sample.StartWithWorkerContext(ctx, workerLifetime.context, registry, options, exporthistory.WithExportHistory())
	if err != nil {
		workerLifetime.cancel()
		return err
	}
	defer func() { err = errors.Join(err, workerLifetime.close(host.Close)) }()

	for i := range sources {
		source := &sources[i]
		if _, err := host.Client.ScheduleNewOrchestration(ctx, orchestratorName,
			api.WithInstanceID(source.ID), api.WithInput(source.Input)); err != nil {
			return err
		}
		var output int
		if err := sample.Wait(ctx, host.Client, source.ID, &output); err != nil {
			return err
		}
		if output != source.Input*source.Input {
			return fmt.Errorf("source %s output=%d, expected %d", source.ID, output, source.Input*source.Input)
		}
		metadata, err := host.Client.FetchOrchestrationMetadata(ctx, source.ID)
		if err != nil {
			return err
		}
		if metadata == nil || metadata.ExecutionID == "" {
			return errors.New("completed source is missing its execution ID")
		}
		if metadata.CreatedAt.IsZero() {
			return errors.New("completed source is missing its creation time")
		}
		source.ExecutionID = metadata.ExecutionID
		source.CreatedAt = metadata.CreatedAt
		source.CompletedAt = metadata.CompletedAt
		if source.CompletedAt.IsZero() {
			source.CompletedAt = metadata.LastUpdatedAt
		}
		if source.CompletedAt.IsZero() {
			return errors.New("completed source is missing its completion/update time")
		}
		fmt.Printf("Completed %s: %d -> %d\n", source.ID, source.Input, output)
	}

	from, to := completionWindow(sources)
	// Client-side batch validation rejects a future upper bound. Use the actual
	// service timestamps, waiting for our clock if the service runs ahead.
	if err := sample.Until(ctx, 100*time.Millisecond, func() (bool, error) {
		return !time.Now().UTC().Before(to), nil
	}); err != nil {
		return fmt.Errorf("wait for export window's upper bound: %w", err)
	}
	query := api.InstanceIDQuery{
		RuntimeStatus:     []api.OrchestrationStatus{api.RUNTIME_STATUS_COMPLETED},
		CompletedTimeFrom: from,
		CompletedTimeTo:   to,
		PageSize:          2,
	}
	if err := waitUntilListable(ctx, source, query); err != nil {
		return err
	}
	exportClient, err := exporthistory.NewClient(host.Client.TaskHubGrpcClient, exporthistory.ClientOptions{
		ContainerName: container, Prefix: prefix,
	})
	if err != nil {
		return err
	}
	job, err := exportClient.JobClient(jobID)
	if err != nil {
		return err
	}
	fmt.Printf("Export job: %s; destination: %s/%s\n", jobID, container, prefix)
	var description *exporthistory.ExportJobDescription
	var eventCount int
	if err := withJobCleanup(ctx, workerLifetime.context, job, func() error {
		format := exporthistory.DefaultExportFormat()
		if err := job.Create(ctx, exporthistory.JobCreationOptions{
			JobID:                jobID,
			Mode:                 exporthistory.ExportModeBatch,
			CompletedTimeFrom:    from,
			CompletedTimeTo:      to,
			RuntimeStatus:        query.RuntimeStatus,
			Destination:          &exporthistory.ExportDestination{Container: container, Prefix: prefix},
			Format:               &format,
			MaxInstancesPerBatch: 2,
		}); err != nil {
			return err
		}
		if err := sample.Until(ctx, 250*time.Millisecond, func() (bool, error) {
			var err error
			description, err = job.Describe(ctx)
			if err != nil {
				return false, err
			}
			if description.Status == exporthistory.ExportJobStatusFailed {
				return false, fmt.Errorf("export failed: %s", description.LastError)
			}
			return description.Status == exporthistory.ExportJobStatusCompleted, nil
		}); err != nil {
			return err
		}
		if description.ScannedInstances != sourceCount || description.ExportedInstances != sourceCount ||
			description.LastError != "" || description.OrchestratorInstanceID == "" {
			return fmt.Errorf("unexpected batch progress: scanned=%d exported=%d error=%q run=%q",
				description.ScannedInstances, description.ExportedInstances,
				description.LastError, description.OrchestratorInstanceID)
		}
		if err := sample.Wait(ctx, host.Client, api.InstanceID(description.OrchestratorInstanceID), nil); err != nil {
			return err
		}
		jobs, err := exportClient.ListJobs(ctx, exporthistory.ExportJobQuery{JobIDPrefix: jobID, PageSize: 2})
		if err != nil {
			return err
		}
		if len(jobs.Jobs) != 1 || jobs.Jobs[0].JobID != jobID {
			return errors.New("job-scoped listing did not return this export job")
		}
		eventCount, err = verifyExportBlobs(ctx, blobClient, container, prefix, sources)
		return err
	}); err != nil {
		return err
	}
	fmt.Printf("Verified %d gzip JSONL blobs / %d history events; scanned=%d exported=%d; job deleted\n",
		sourceCount, eventCount, description.ScannedInstances, description.ExportedInstances)
	return nil
}

func requireIsolatedTaskHub(acknowledgement string) error {
	if acknowledgement != "1" {
		return errors.New("history export requires an isolated task hub with no other export workers: " +
			"the SDK lists whole completion-time windows, not instance prefixes; " +
			"set HISTORY_EXPORT_ISOLATED_TASKHUB=1 only after ensuring isolation")
	}
	return nil
}

func squareOrchestrator(ctx *task.OrchestrationContext) (any, error) {
	var n int
	if err := ctx.GetInput(&n); err != nil {
		return nil, err
	}
	var result int
	if err := ctx.CallActivity(squareName, task.WithActivityInput(n)).Await(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func square(ctx task.ActivityContext) (any, error) {
	var n int
	if err := ctx.GetInput(&n); err != nil {
		return nil, err
	}
	if n < 1 || n > sourceCount {
		return nil, errors.New("this sample accepts only inputs 1 through 5")
	}
	return n * n, nil
}

func completionWindow(sources []sourceExecution) (time.Time, time.Time) {
	from, to := sources[0].CreatedAt, sources[0].CompletedAt
	for _, source := range sources[1:] {
		if source.CreatedAt.Before(from) {
			from = source.CreatedAt
		}
		if source.CompletedAt.After(to) {
			to = source.CompletedAt
		}
	}
	// The completion index can differ from subsecond metadata timestamps.
	// Cover the sources' whole lifetimes; the allow-list still rejects other IDs.
	return from.Truncate(time.Second), to.Truncate(time.Second).Add(time.Second)
}

func waitUntilListable(ctx context.Context, source *ownedHistorySource, query api.InstanceIDQuery) error {
	visible := 0
	err := sample.Until(ctx, 500*time.Millisecond, func() (bool, error) {
		pageQuery := query
		found := make(map[api.InstanceID]struct{}, len(source.allowed))
		tokens := make(map[string]struct{})
		for {
			page, err := source.ListInstanceIDs(ctx, pageQuery)
			if err != nil {
				return false, err
			}
			for _, id := range page.InstanceIDs {
				found[id] = struct{}{}
			}
			if page.ContinuationToken == "" {
				visible = len(found)
				return visible == len(source.allowed), nil
			}
			if _, repeated := tokens[page.ContinuationToken]; repeated {
				return false, errors.New("instance listing returned a repeated continuation token")
			}
			tokens[page.ContinuationToken] = struct{}{}
			pageQuery.ContinuationToken = page.ContinuationToken
		}
	})
	if err != nil {
		return fmt.Errorf("wait for all owned instances to be visible in the completion-time index (%d/%d visible): %w", visible, len(source.allowed), err)
	}
	return nil
}
