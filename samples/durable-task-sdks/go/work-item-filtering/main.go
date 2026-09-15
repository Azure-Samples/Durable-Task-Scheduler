package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/task"
)

const (
	greetingName = "go-sample-filtering-greeting"
	helloName    = "go-sample-filtering-hello"
	mathName     = "go-sample-filtering-math"
	addName      = "go-sample-filtering-add"
)

type numbers struct {
	A int `json:"a"`
	B int `json:"b"`
}

type greetingResult struct {
	Worker string `json:"worker"`
	Result string `json:"result"`
}

type mathResult struct {
	Worker string `json:"worker"`
	Result int    `json:"result"`
}

func main() {
	sample.Main("work-item-filtering", run)
}

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

	greetingID, mathID := sample.ID("filtering-greeting"), sample.ID("filtering-math")
	if _, err := workerA.Client.ScheduleNewOrchestration(ctx, greetingName,
		api.WithInstanceID(greetingID), api.WithInput("World")); err != nil {
		return err
	}
	// Scheduling through A's client does not select A's worker; task filters route it to B.
	if _, err := workerA.Client.ScheduleNewOrchestration(ctx, mathName,
		api.WithInstanceID(mathID), api.WithInput(numbers{A: 40, B: 2})); err != nil {
		return err
	}
	var greeting greetingResult
	var sum mathResult
	if err := sample.Wait(ctx, workerA.Client, greetingID, &greeting); err != nil {
		return err
	}
	if err := sample.Wait(ctx, workerA.Client, mathID, &sum); err != nil {
		return err
	}
	if greeting != (greetingResult{Worker: "A", Result: "Hello, World!"}) {
		return fmt.Errorf("greeting routed incorrectly: %+v", greeting)
	}
	if sum != (mathResult{Worker: "B", Result: 42}) {
		return fmt.Errorf("math routed incorrectly: %+v", sum)
	}
	fmt.Printf("Worker %s: %s\nWorker %s: %d\n", greeting.Worker, greeting.Result, sum.Worker, sum.Result)
	return nil
}

func newRegistries() (*task.TaskRegistry, *task.TaskRegistry, error) {
	a, b := task.NewTaskRegistry(), task.NewTaskRegistry()
	if err := a.AddOrchestratorN(greetingName, greetingWorkflow); err != nil {
		return nil, nil, err
	}
	if err := a.AddActivityN(helloName, sayHello); err != nil {
		return nil, nil, err
	}
	if err := b.AddOrchestratorN(mathName, mathWorkflow); err != nil {
		return nil, nil, err
	}
	if err := b.AddActivityN(addName, addNumbers); err != nil {
		return nil, nil, err
	}
	return a, b, nil
}

func greetingWorkflow(ctx *task.OrchestrationContext) (any, error) {
	var name string
	if err := ctx.GetInput(&name); err != nil {
		return nil, err
	}
	var result greetingResult
	if err := ctx.CallActivity(helloName, task.WithActivityInput(name)).Await(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func mathWorkflow(ctx *task.OrchestrationContext) (any, error) {
	var input numbers
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	var result mathResult
	if err := ctx.CallActivity(addName, task.WithActivityInput(input)).Await(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func sayHello(ctx task.ActivityContext) (any, error) {
	var name string
	if err := ctx.GetInput(&name); err != nil {
		return nil, err
	}
	return greetingResult{Worker: "A", Result: fmt.Sprintf("Hello, %s!", name)}, nil
}

func addNumbers(ctx task.ActivityContext) (any, error) {
	var input numbers
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	return mathResult{Worker: "B", Result: input.A + input.B}, nil
}
