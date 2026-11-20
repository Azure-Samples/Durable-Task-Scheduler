package main

import (
	"context"
	"errors"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestrationName = "GoTaskQueuesProcessOrder"
	childName         = "GoTaskQueuesPartnerOrder"
	acceptName        = "GoTaskQueuesAcceptOrder"
	partnerName       = "GoTaskQueuesCallPartner"
)

func newRegistry() (*task.TaskRegistry, error) {
	r := task.NewTaskRegistry()
	return r, errors.Join(
		r.AddOrchestratorN(orchestrationName, processOrder),
		r.AddOrchestratorN(childName, partnerOrder),
		r.AddActivityN(acceptName, acceptOrder),
		r.AddActivityN(partnerName, callPartner),
	)
}

func withWorker(ctx context.Context, run func(context.Context, *dts.Client) error) (err error) {
	r, err := newRegistry()
	if err != nil {
		return err
	}
	options, err := sample.Options()
	if err != nil {
		return err
	}
	c, err := dts.NewClient(ctx, options, sample.Logger())
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, c.Close()) }()
	// Defined task queues require an unfiltered connection. Do not use sample.Start.
	worker, err := dts.NewWorker(options, r, sample.Logger())
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		err = errors.Join(err, worker.Shutdown(shutdownCtx))
	}()
	if err := worker.Start(ctx); err != nil {
		return err
	}
	return run(ctx, c)
}
