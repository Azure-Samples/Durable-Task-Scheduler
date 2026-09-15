package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestrationName = "GoFunctionChaining"
	sayHelloName      = "GoFunctionChainingSayHello"
	processName       = "GoFunctionChainingProcessGreeting"
	finalizeName      = "GoFunctionChainingFinalizeResponse"
)

type Greeting struct {
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
}

func sayHello(ctx task.ActivityContext) (any, error) {
	var name string
	if err := ctx.GetInput(&name); err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("recipient must not be empty")
	}
	return Greeting{Recipient: name, Message: "Hello " + name + "!"}, nil
}

func readGreeting(ctx task.ActivityContext) (Greeting, error) {
	var greeting Greeting
	if err := ctx.GetInput(&greeting); err != nil {
		return Greeting{}, err
	}
	if greeting.Recipient == "" || greeting.Message == "" {
		return Greeting{}, errors.New("greeting requires a recipient and message")
	}
	return greeting, nil
}

func processGreeting(ctx task.ActivityContext) (any, error) {
	greeting, err := readGreeting(ctx)
	if err != nil {
		return nil, err
	}
	greeting.Message += " How are you today?"
	return greeting, nil
}

func finalizeResponse(ctx task.ActivityContext) (any, error) {
	greeting, err := readGreeting(ctx)
	if err != nil {
		return nil, err
	}
	greeting.Message += " I hope you're doing well!"
	return greeting, nil
}

func functionChaining(ctx *task.OrchestrationContext) (any, error) {
	var name string
	if err := ctx.GetInput(&name); err != nil {
		return nil, err
	}
	ctx.Logger().Info("Starting greeting pipeline", "recipient", name)

	var greeting Greeting
	if err := ctx.CallActivity(sayHelloName, task.WithActivityInput(name)).Await(&greeting); err != nil {
		return nil, fmt.Errorf("create greeting: %w", err)
	}
	if err := ctx.CallActivity(processName, task.WithActivityInput(greeting)).Await(&greeting); err != nil {
		return nil, fmt.Errorf("process greeting: %w", err)
	}
	if err := ctx.CallActivity(finalizeName, task.WithActivityInput(greeting)).Await(&greeting); err != nil {
		return nil, fmt.Errorf("finalize greeting: %w", err)
	}
	return greeting.Message, nil
}

func newRegistry() (*task.TaskRegistry, error) {
	r := task.NewTaskRegistry()
	return r, errors.Join(
		r.AddOrchestratorN(orchestrationName, functionChaining),
		r.AddActivityN(sayHelloName, sayHello),
		r.AddActivityN(processName, processGreeting),
		r.AddActivityN(finalizeName, finalizeResponse),
	)
}

func run(ctx context.Context) error {
	r, err := newRegistry()
	if err != nil {
		return err
	}
	return sample.WithHost(ctx, r, func(ctx context.Context, c *dts.Client) (err error) {
		id, err := c.ScheduleNewOrchestration(ctx, orchestrationName,
			api.WithInstanceID(sample.ID("function-chaining")), api.WithInput("User"))
		if err != nil {
			return err
		}
		defer stopOnError(c, id, &err)

		var output string
		if err := sample.Wait(ctx, c, id, &output); err != nil {
			return err
		}
		const want = "Hello User! How are you today? I hope you're doing well!"
		if err := sample.Require(output == want, "greeting = %q, want %q", output, want); err != nil {
			return err
		}
		return sample.PrintJSON(struct {
			InstanceID api.InstanceID `json:"instance_id"`
			Output     string         `json:"output"`
		}{id, output})
	})
}

func stopOnError(c *dts.Client, id api.InstanceID, runErr *error) {
	if *runErr == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	state, err := c.FetchOrchestrationMetadata(ctx, id)
	if err == nil && !state.IsComplete() {
		err = c.TerminateOrchestration(ctx, id)
		if err == nil {
			_, err = c.WaitForOrchestrationCompletion(ctx, id)
		}
	}
	*runErr = errors.Join(*runErr, err)
}

func main() {
	sample.Main("function-chaining", run)
}
