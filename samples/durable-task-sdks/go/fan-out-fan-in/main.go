package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestrationName = "GoFanOutFanIn"
	processName       = "GoFanOutFanInProcessWorkItem"
	aggregateName     = "GoFanOutFanInAggregateResults"
	maxItems          = 100
	maxMagnitude      = 1_000_000
)

type WorkResult struct {
	Item   int64 `json:"item"`
	Result int64 `json:"result"`
}

type Summary struct {
	TotalItems int     `json:"total_items"`
	Sum        int64   `json:"sum"`
	Average    float64 `json:"average"`
}

func square(item int64) (WorkResult, error) {
	if item < -maxMagnitude || item > maxMagnitude {
		return WorkResult{}, fmt.Errorf("item %d exceeds the sample's safe arithmetic range", item)
	}
	return WorkResult{Item: item, Result: item * item}, nil
}

func processWorkItem(ctx task.ActivityContext) (any, error) {
	var item int64
	if err := ctx.GetInput(&item); err != nil {
		return nil, err
	}
	return square(item)
}

func summarize(results []WorkResult) (Summary, error) {
	if len(results) > maxItems {
		return Summary{}, fmt.Errorf("batch contains more than %d items", maxItems)
	}
	summary := Summary{TotalItems: len(results)}
	for _, result := range results {
		expected, err := square(result.Item)
		if err != nil {
			return Summary{}, err
		}
		if result != expected {
			return Summary{}, fmt.Errorf("incorrect square for item %d: %d", result.Item, result.Result)
		}
		summary.Sum += result.Result
	}
	if len(results) != 0 {
		summary.Average = float64(summary.Sum) / float64(len(results))
	}
	return summary, nil
}

func aggregateResults(ctx task.ActivityContext) (any, error) {
	var results []WorkResult
	if err := ctx.GetInput(&results); err != nil {
		return nil, err
	}
	return summarize(results)
}

func fanOutFanIn(ctx *task.OrchestrationContext) (any, error) {
	var items []int64
	if err := ctx.GetInput(&items); err != nil {
		return nil, err
	}
	if len(items) > maxItems {
		return nil, fmt.Errorf("batch contains more than %d items", maxItems)
	}
	ctx.Logger().Info("Fanning out work", "items", len(items))
	pending := make([]task.Task, len(items))
	for i, item := range items {
		pending[i] = ctx.CallActivity(processName, task.WithActivityInput(item))
	}
	// All work is scheduled before waiting. WhenAll also drains siblings on failure.
	if err := ctx.WhenAll(pending...); err != nil {
		return nil, fmt.Errorf("process batch: %w", err)
	}
	results := make([]WorkResult, len(pending))
	for i, work := range pending {
		if err := work.Await(&results[i]); err != nil {
			return nil, fmt.Errorf("decode item %d: %w", i, err)
		}
	}
	var summary Summary
	if err := ctx.CallActivity(aggregateName, task.WithActivityInput(results)).Await(&summary); err != nil {
		return nil, fmt.Errorf("aggregate batch: %w", err)
	}
	return summary, nil
}

func newRegistry() (*task.TaskRegistry, error) {
	r := task.NewTaskRegistry()
	return r, errors.Join(
		r.AddOrchestratorN(orchestrationName, fanOutFanIn),
		r.AddActivityN(processName, processWorkItem),
		r.AddActivityN(aggregateName, aggregateResults),
	)
}

func verifyBatch(ctx context.Context, c *dts.Client, items []int64, want Summary) (err error) {
	id, err := c.ScheduleNewOrchestration(ctx, orchestrationName,
		api.WithInstanceID(sample.ID("fan-out-fan-in")), api.WithInput(items))
	if err != nil {
		return err
	}
	defer stopOnError(c, id, &err)

	var got Summary
	if err := sample.Wait(ctx, c, id, &got); err != nil {
		return err
	}
	if err := sample.Require(got == want, "aggregation = %+v, want %+v", got, want); err != nil {
		return err
	}
	return sample.PrintJSON(struct {
		InstanceID api.InstanceID `json:"instance_id"`
		Summary    Summary        `json:"summary"`
	}{id, got})
}

func run(ctx context.Context) error {
	r, err := newRegistry()
	if err != nil {
		return err
	}
	return sample.WithHost(ctx, r, func(ctx context.Context, c *dts.Client) error {
		if err := verifyBatch(ctx, c, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			Summary{TotalItems: 10, Sum: 385, Average: 38.5}); err != nil {
			return err
		}
		return verifyBatch(ctx, c, []int64{}, Summary{})
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
	sample.Main("fan-out-fan-in", run)
}
