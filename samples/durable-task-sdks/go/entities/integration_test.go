package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

func TestIntegration(t *testing.T) {
	ctx := testutil.IntegrationContext(t)
	if err := verifyEntities(ctx); err != nil {
		t.Fatal(err)
	}
}

func verifyEntities(ctx context.Context) error {
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
		return testutil.Require(state.Value == 0 && state.ResetAt.Equal(result.ResetAt),
			"persisted counter state = %+v, workflow result = %+v", state, result)
	})
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
