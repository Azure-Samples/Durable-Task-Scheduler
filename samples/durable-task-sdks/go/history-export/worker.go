package main

import (
	"context"
	"errors"
	"os"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/exporthistory"
	"github.com/microsoft/durabletask-go/task"
)

const (
	maxHistoryEvents = 128
	maxHistoryBytes  = 1024 * 1024
)

type historyWorker struct {
	*sample.Host
	sourceClient *dts.Client
	source       *ownedHistorySource
	lifetime     exportWorkerLifetime
}

func startWorker(ctx context.Context, batch exportBatch, store exporthistory.Store) (_ *historyWorker, err error) {
	if err := requireIsolatedTaskHub(os.Getenv("HISTORY_EXPORT_ISOLATED_TASKHUB")); err != nil {
		return nil, err
	}
	options, err := sample.Options()
	if err != nil {
		return nil, err
	}
	sourceClient, err := dts.NewClient(ctx, options, sample.Logger())
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, sourceClient.Close())
		}
	}()
	allowed := make(map[api.InstanceID]struct{}, len(batch.Sources))
	for _, source := range batch.Sources {
		allowed[source.ID] = struct{}{}
	}
	source := &ownedHistorySource{inner: sourceClient, allowed: allowed}
	registry := task.NewTaskRegistry()
	if err := registry.AddOrchestratorN(orchestratorName, squareOrchestrator); err != nil {
		return nil, err
	}
	if err := registry.AddActivityN(squareName, square); err != nil {
		return nil, err
	}
	if err := exporthistory.Register(registry, exporthistory.WorkerOptions{
		Source: source,
		Store: &ownedHistoryStore{
			inner: store, allowed: allowed, container: batch.Container, prefix: batch.Prefix,
		},
		HistoryQuery: api.HistoryQuery{MaxEvents: maxHistoryEvents, MaxBytes: maxHistoryBytes},
	}); err != nil {
		return nil, err
	}
	lifetime := newExportWorkerLifetime(ctx)
	host, err := sample.StartWithWorkerContext(ctx, lifetime.context, registry, options, exporthistory.WithExportHistory())
	if err != nil {
		lifetime.cancel()
		return nil, err
	}
	return &historyWorker{Host: host, sourceClient: sourceClient, source: source, lifetime: lifetime}, nil
}

func (w *historyWorker) Close() error {
	return errors.Join(w.lifetime.close(w.Host.Close), w.sourceClient.Close())
}

func requireIsolatedTaskHub(acknowledgement string) error {
	if acknowledgement != "1" {
		return errors.New("history export requires an isolated task hub with no other export workers: " +
			"the SDK lists whole completion-time windows, not instance prefixes; " +
			"set HISTORY_EXPORT_ISOLATED_TASKHUB=1 only after ensuring isolation")
	}
	return nil
}
