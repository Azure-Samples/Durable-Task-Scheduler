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
	counterName  = "go-sample-entities-counter"
	workflowName = "go-sample-entities-workflow"
	resetDelay   = 5 * time.Second
)

type counterState struct {
	Value   int       `json:"value"`
	ResetAt time.Time `json:"reset_at,omitempty"`
}

type workflowResult struct {
	Before  int       `json:"before"`
	After   int       `json:"after"`
	ReadAt  time.Time `json:"read_at"`
	DueAt   time.Time `json:"due_at"`
	ResetAt time.Time `json:"reset_at"`
}

func main() {
	sample.Main("entities", run)
}

func run(ctx context.Context) error {
	registry, err := newRegistry()
	if err != nil {
		return err
	}
	return sample.WithHost(ctx, registry, func(ctx context.Context, c *dts.Client) error {
		runID := string(sample.ID("entities"))
		direct := api.NewEntityID(counterName, runID+"-direct")
		fromWorkflow := api.NewEntityID(counterName, runID+"-workflow")
		if err := c.SignalEntity(ctx, direct, "add", api.WithSignalInput(100)); err != nil {
			return err
		}
		if err := waitForValue(ctx, c, direct, 100); err != nil {
			return err
		}
		if err := c.SignalEntity(ctx, direct, "subtract", api.WithSignalInput(25)); err != nil {
			return err
		}
		if err := waitForValue(ctx, c, direct, 75); err != nil {
			return err
		}

		id := sample.ID("entities-workflow")
		if _, err := c.ScheduleNewOrchestration(ctx, workflowName,
			api.WithInstanceID(id), api.WithInput(fromWorkflow)); err != nil {
			return err
		}
		var result workflowResult
		if err := sample.Wait(ctx, c, id, &result); err != nil {
			return err
		}
		if err := validateResult(result); err != nil {
			return err
		}
		if err := waitForValue(ctx, c, fromWorkflow, 0); err != nil {
			return err
		}
		persisted, err := c.GetEntity(ctx, fromWorkflow)
		if err != nil {
			return err
		}
		if persisted == nil {
			return errors.New("workflow entity disappeared before state verification")
		}
		var state counterState
		if err := persisted.ReadState(&state); err != nil {
			return err
		}
		if state.Value != 0 || !state.ResetAt.Equal(result.ResetAt) {
			return fmt.Errorf("persisted counter state = %+v, workflow result = %+v", state, result)
		}
		fmt.Println("Direct signals: 100 - 25 = 75")
		fmt.Println("Orchestration signals and calls: 10 + 5 - 3 = 12; scheduled reset = 0")
		return sample.PrintJSON(result)
	})
}

func newRegistry() (*task.TaskRegistry, error) {
	registry := task.NewTaskRegistry()
	if err := registry.AddEntityN(counterName, counter); err != nil {
		return nil, err
	}
	if err := registry.AddOrchestratorN(workflowName, counterWorkflow); err != nil {
		return nil, err
	}
	return registry, nil
}

func counter(ctx *task.EntityContext) (any, error) {
	var state counterState
	if ctx.HasState() {
		if err := ctx.GetState(&state); err != nil {
			return nil, err
		}
	}
	switch ctx.Operation {
	case "get":
		return state.Value, nil
	case "snapshot":
		return state, nil
	case "delete":
		ctx.DeleteState()
		return nil, nil
	}
	var amount int
	if ctx.Operation == "add" || ctx.Operation == "subtract" {
		if err := ctx.GetInput(&amount); err != nil {
			return nil, err
		}
	}
	if err := state.change(ctx.Operation, amount, ctx.CurrentTimeUTC()); err != nil {
		return nil, err
	}
	if err := ctx.SetState(state); err != nil {
		return nil, err
	}
	return state.Value, nil
}

func (s *counterState) change(operation string, amount int, now time.Time) error {
	switch operation {
	case "add":
		s.Value += amount
	case "subtract":
		s.Value -= amount
	case "reset":
		s.Value = 0
		// Record the entity operation's execution timestamp, not the requested due time.
		s.ResetAt = now
	default:
		return fmt.Errorf("unknown counter operation %q", operation)
	}
	return nil
}

func counterWorkflow(ctx *task.OrchestrationContext) (any, error) {
	var id api.EntityID
	if err := ctx.GetInput(&id); err != nil {
		return nil, err
	}
	for _, operation := range []struct {
		name   string
		amount int
	}{{"add", 10}, {"add", 5}, {"subtract", 3}} {
		if err := ctx.SignalEntity(id, operation.name, task.WithSignalEntityInput(operation.amount)); err != nil {
			return nil, err
		}
	}
	var initial int
	if err := ctx.CallEntity(id, "get").Await(&initial); err != nil {
		return nil, err
	}
	if initial != 12 {
		return nil, fmt.Errorf("counter after immediate signals = %d, want 12", initial)
	}

	due := ctx.CurrentTimeUtc.Add(resetDelay)
	if err := ctx.SignalEntity(id, "reset", task.WithSignalEntityScheduledTime(due)); err != nil {
		return nil, err
	}
	var before int
	if err := ctx.CallEntity(id, "get").Await(&before); err != nil {
		return nil, err
	}
	readAt := ctx.CurrentTimeUtc
	if err := ctx.CreateTimer(due.Sub(ctx.CurrentTimeUtc) + time.Second).Await(nil); err != nil {
		return nil, err
	}

	// Delivery may lag the due time. Poll durably, with a finite retry bound.
	for attempt := 0; attempt < 20; attempt++ {
		var state counterState
		if err := ctx.CallEntity(id, "snapshot").Await(&state); err != nil {
			return nil, err
		}
		if !state.ResetAt.IsZero() {
			var after int
			if err := ctx.CallEntity(id, "get").Await(&after); err != nil {
				return nil, err
			}
			result := workflowResult{Before: before, After: after, ReadAt: readAt, DueAt: due, ResetAt: state.ResetAt}
			return result, validateResult(result)
		}
		if err := ctx.CreateTimer(500 * time.Millisecond).Await(nil); err != nil {
			return nil, err
		}
	}
	return nil, errors.New("scheduled entity signal was not delivered within the bounded observation window")
}

func validateResult(result workflowResult) error {
	if result.Before != 12 || result.After != 0 {
		return fmt.Errorf("counter values = %d -> %d, want 12 -> 0", result.Before, result.After)
	}
	if result.DueAt.IsZero() || result.ReadAt.IsZero() || !result.ReadAt.Before(result.DueAt) {
		return errors.New("the before-reset read was not verified before the scheduled due time")
	}
	if result.ResetAt.IsZero() || result.ResetAt.Before(result.DueAt) {
		return fmt.Errorf("scheduled reset executed at %s before its due time %s", result.ResetAt, result.DueAt)
	}
	return nil
}

func waitForValue(ctx context.Context, c *dts.Client, id api.EntityID, value int) error {
	err := sample.Until(ctx, 100*time.Millisecond, func() (bool, error) {
		metadata, err := c.GetEntity(ctx, id)
		if err != nil || metadata == nil || !metadata.HasState {
			return false, err
		}
		var state counterState
		if err := metadata.ReadState(&state); err != nil {
			return false, err
		}
		return state.Value == value, nil
	})
	if err != nil {
		return fmt.Errorf("wait for entity %s value %d: %w", id, value, err)
	}
	return nil
}
