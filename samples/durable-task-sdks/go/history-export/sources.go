package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

const sourceCount = 5

type sourceInstance struct {
	ID          api.InstanceID
	Input       int
	CreatedAt   time.Time
	CompletedAt time.Time
}

type exportBatch struct {
	JobID     string
	Container string
	Prefix    string
	Sources   []sourceInstance
}

func newExportBatch() exportBatch {
	batch := exportBatch{
		JobID: string(sample.ID("history-export-job")), Container: string(sample.ID("history-export")),
		Sources: make([]sourceInstance, sourceCount),
	}
	batch.Prefix = batch.JobID + "/"
	for i := range batch.Sources {
		batch.Sources[i] = sourceInstance{ID: sample.ID("history-export-source"), Input: i + 1}
	}
	return batch
}

func seedSources(ctx context.Context, client *dts.Client, sources []sourceInstance) error {
	for i := range sources {
		source := &sources[i]
		if _, err := client.ScheduleNewOrchestration(ctx, orchestratorName,
			api.WithInstanceID(source.ID), api.WithInput(source.Input)); err != nil {
			return err
		}
		var output int
		if err := sample.Wait(ctx, client, source.ID, &output); err != nil {
			return err
		}
		metadata, err := client.FetchOrchestrationMetadata(ctx, source.ID)
		if err != nil {
			return err
		}
		if metadata == nil || metadata.CreatedAt.IsZero() {
			return errors.New("completed source is missing its creation time")
		}
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
	return nil
}

func prepareExport(ctx context.Context, worker *historyWorker, batch exportBatch) (api.InstanceIDQuery, error) {
	if err := seedSources(ctx, worker.Client, batch.Sources); err != nil {
		return api.InstanceIDQuery{}, err
	}
	from, to := completionWindow(batch.Sources)
	// Batch exports reject a future upper bound, even with service clock skew.
	if err := sample.Until(ctx, 100*time.Millisecond, func() (bool, error) {
		return !time.Now().UTC().Before(to), nil
	}); err != nil {
		return api.InstanceIDQuery{}, fmt.Errorf("wait for export window's upper bound: %w", err)
	}
	query := api.InstanceIDQuery{
		RuntimeStatus:     []api.OrchestrationStatus{api.RUNTIME_STATUS_COMPLETED},
		CompletedTimeFrom: from,
		CompletedTimeTo:   to,
		PageSize:          2,
	}
	return query, waitUntilListable(ctx, worker.source, query)
}

func completionWindow(sources []sourceInstance) (time.Time, time.Time) {
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
