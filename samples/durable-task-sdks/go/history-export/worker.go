package main

import (
	"context"
	"errors"

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
	workerCtx    context.Context
	cancelWorker context.CancelFunc
}

func startWorker(ctx context.Context, store exporthistory.Store) (_ *historyWorker, err error) {
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
	registry := task.NewTaskRegistry()
	if err := registry.AddOrchestratorN(orchestratorName, squareOrchestrator); err != nil {
		return nil, err
	}
	if err := registry.AddActivityN(squareName, square); err != nil {
		return nil, err
	}
	if err := exporthistory.Register(registry, exporthistory.WorkerOptions{
		Source:       sourceClient,
		Store:        store,
		HistoryQuery: api.HistoryQuery{MaxEvents: maxHistoryEvents, MaxBytes: maxHistoryBytes},
	}); err != nil {
		return nil, err
	}
	workerCtx, cancelWorker := context.WithCancel(context.WithoutCancel(ctx))
	host, err := sample.StartWithWorkerContext(ctx, workerCtx, registry, options, exporthistory.WithExportHistory())
	if err != nil {
		cancelWorker()
		return nil, err
	}
	return &historyWorker{Host: host, sourceClient: sourceClient, workerCtx: workerCtx, cancelWorker: cancelWorker}, nil
}

func (w *historyWorker) Close() error {
	defer w.cancelWorker()
	return errors.Join(w.Host.Close(), w.sourceClient.Close())
}
