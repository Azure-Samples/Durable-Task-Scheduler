package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

func run(ctx context.Context) (err error) {
	greetingRegistry, mathRegistry, err := newRegistries()
	if err != nil {
		return err
	}
	workerA, err := sample.Start(ctx, greetingRegistry, nil)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, workerA.Close()) }()
	workerB, err := sample.Start(ctx, mathRegistry, nil)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, workerB.Close()) }()

	greeting, sum, err := runWorkloads(ctx, workerA.Client)
	if err != nil {
		return err
	}
	fmt.Printf("Worker %s: %s\nWorker %s: %d\n", greeting.Worker, greeting.Result, sum.Worker, sum.Result)
	return nil
}

func runWorkloads(ctx context.Context, c *dts.Client) (greetingResult, mathResult, error) {
	var greeting greetingResult
	var sum mathResult
	greetingID, mathID := sample.ID("filtering-greeting"), sample.ID("filtering-math")
	if _, err := c.ScheduleNewOrchestration(ctx, greetingName,
		api.WithInstanceID(greetingID), api.WithInput("World")); err != nil {
		return greeting, sum, err
	}
	// Scheduling through A's client does not select A's worker; task filters route it to B.
	if _, err := c.ScheduleNewOrchestration(ctx, mathName,
		api.WithInstanceID(mathID), api.WithInput(numbers{A: 40, B: 2})); err != nil {
		return greeting, sum, err
	}
	if err := sample.Wait(ctx, c, greetingID, &greeting); err != nil {
		return greeting, sum, err
	}
	if err := sample.Wait(ctx, c, mathID, &sum); err != nil {
		return greeting, sum, err
	}
	return greeting, sum, nil
}
